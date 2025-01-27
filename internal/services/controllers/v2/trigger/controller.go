package trigger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
)

type TriggerController interface {
	Start(ctx context.Context) error
}

type TriggerControllerImpl struct {
	mq        msgqueue.MessageQueue
	l         *zerolog.Logger
	repo      repository.EngineRepository
	v2repo    v2.Repository
	dv        datautils.DataDecoderValidator
	s         gocron.Scheduler
	a         *hatcheterrors.Wrapped
	p         *partition.Partition
	celParser *cel.CELParser
}

type TriggerControllerOpt func(*TriggerControllerOpts)

type TriggerControllerOpts struct {
	mq      msgqueue.MessageQueue
	l       *zerolog.Logger
	repo    repository.EngineRepository
	v2repo  v2.Repository
	dv      datautils.DataDecoderValidator
	alerter hatcheterrors.Alerter
	p       *partition.Partition
}

func defaultTriggerControllerOpts() *TriggerControllerOpts {
	l := logger.NewDefaultLogger("trigger-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &TriggerControllerOpts{
		l:       &l,
		dv:      datautils.NewDataDecoderValidator(),
		alerter: alerter,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) TriggerControllerOpt {
	return func(opts *TriggerControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) TriggerControllerOpt {
	return func(opts *TriggerControllerOpts) {
		opts.l = l
	}
}

func WithAlerter(a hatcheterrors.Alerter) TriggerControllerOpt {
	return func(opts *TriggerControllerOpts) {
		opts.alerter = a
	}
}

func WithRepository(r repository.EngineRepository) TriggerControllerOpt {
	return func(opts *TriggerControllerOpts) {
		opts.repo = r
	}
}

func WithV2Repository(r v2.Repository) TriggerControllerOpt {
	return func(opts *TriggerControllerOpts) {
		opts.v2repo = r
	}
}

func WithPartition(p *partition.Partition) TriggerControllerOpt {
	return func(opts *TriggerControllerOpts) {
		opts.p = p
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) TriggerControllerOpt {
	return func(opts *TriggerControllerOpts) {
		opts.dv = dv
	}
}

func New(fs ...TriggerControllerOpt) (*TriggerControllerImpl, error) {
	opts := defaultTriggerControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.v2repo == nil {
		return nil, fmt.Errorf("v2repo is required. use WithV2Repository")
	}

	if opts.p == nil {
		return nil, errors.New("partition is required. use WithPartition")
	}

	newLogger := opts.l.With().Str("service", "trigger-controller").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "trigger-controller"})

	return &TriggerControllerImpl{
		mq:        opts.mq,
		l:         opts.l,
		repo:      opts.repo,
		v2repo:    opts.v2repo,
		dv:        opts.dv,
		s:         s,
		a:         a,
		p:         opts.p,
		celParser: cel.NewCELParser(),
	}, nil
}

func (tc *TriggerControllerImpl) Start() (func() error, error) {
	mqBuffer := msgqueue.NewMQSubBuffer(msgqueue.TRIGGER_QUEUE, tc.mq, tc.handleBufferedMsgs)
	wg := sync.WaitGroup{}

	tc.s.Start()

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	cleanup := func() error {
		if err := cleanupBuffer(); err != nil {
			return err
		}

		if err := tc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (tc *TriggerControllerImpl) handleBufferedMsgs(tenantId, msgId string, payloads [][]byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(tc.l, tc.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch msgId {
	case "process-trigger":
		return tc.handleProcessTrigger(context.Background(), tenantId, payloads)
	}

	tc.l.Error().Msgf("unknown message id: %s", msgId)

	return nil
}

// handleProcessTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TriggerControllerImpl) handleProcessTrigger(ctx context.Context, tenantId string, payloads [][]byte) error {
	// parse out event ids from the messages
	tuples := make([]v2.EventIdKey, 0, len(payloads))
	idsToData := make(map[string][]byte, len(payloads))

	msgs := msgqueue.JSONConvert[tasktypes.EventTaskPayload](payloads)

	for _, msg := range msgs {
		tuples = append(tuples, v2.EventIdKey{
			EventId: msg.EventId,
			Key:     msg.EventKey,
		})

		idsToData[msg.EventId] = []byte(msg.EventData)
	}

	// get a list of workflow versions which correspond to events
	startDatas, err := tc.v2repo.Events().ListTriggeredWorkflowsForEvents(ctx, tenantId, tuples)

	if err != nil {
		return fmt.Errorf("could not query workflows for events: %w", err)
	}

	// parse the workflow versions into a list of CreateTaskOpts
	opts, err := tc.getTaskCreateOpts(startDatas, idsToData)

	if err != nil {
		return fmt.Errorf("could not get task create options: %w", err)
	}

	// create the tasks
	err = tc.v2repo.Tasks().CreateTasks(ctx, tenantId, opts)

	if err != nil {
		return fmt.Errorf("could not create tasks: %w", err)
	}

	// get all unique queues and notify them
	queues := make(map[string]struct{})

	for _, opt := range opts {
		queues[opt.Queue] = struct{}{}
	}

	tenant, err := tc.repo.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		return err
	}

	for queue := range queues {
		if tenant.SchedulerPartitionId.Valid {
			msg, err := tasktypes.CheckTenantQueueToTask(tenantId, queue, true, false)

			if err != nil {
				tc.l.Err(err).Msg("could not create message for scheduler partition queue")
				continue
			}

			err = tc.mq.SendMessage(
				ctx,
				msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler),
				msg,
			)

			if err != nil {
				tc.l.Err(err).Msg("could not add message to scheduler partition queue")
			}
		}
	}

	// notify the OLAP processor that tasks have been created
	// TODO: make this transactionally safe
	for _, opt := range opts {
		msg, err := tasktypes.TaskOptToMessage(tenantId, opt)

		if err != nil {
			tc.l.Err(err).Msg("could not create message for olap queue")
			continue
		}

		err = tc.mq.SendMessage(ctx, msgqueue.OLAP_QUEUE, msg)

		if err != nil {
			tc.l.Err(err).Msg("could not add message to olap queue")
		}
	}

	return nil
}

func (tc *TriggerControllerImpl) getTaskCreateOpts(startDatas []*v2.WorkflowVersionWithTriggeringEvent, idsToData map[string][]byte) ([]v2.CreateTaskOpts, error) {
	opts := make([]v2.CreateTaskOpts, 0, len(startDatas))

	for _, startData := range startDatas {
		// parse the start data into a CreateTaskOpts
		// if startData.WorkflowStartData.IsDAG {
		// 	tc.l.Error().Msgf("DAG workflows are not supported in v2 at the moment")
		// }

		id := uuid.New().String()

		eventData := idsToData[startData.EventId]

		// parse the start data into a CreateTaskOpts
		opt := v2.CreateTaskOpts{
			ExternalId:      id,
			Queue:           startData.WorkflowStartData.ActionId,
			ActionId:        startData.WorkflowStartData.ActionId,
			StepId:          sqlchelpers.UUIDToStr(startData.WorkflowStartData.ID),
			ScheduleTimeout: startData.WorkflowStartData.ScheduleTimeout,
			StepTimeout:     startData.WorkflowStartData.Timeout.String,
			DisplayName:     startData.WorkflowStartData.WorkflowName,
			Input:           eventData,
			// TODO: OTHER RELEVANT FIELDS
		}

		opts = append(opts, opt)
	}

	return opts, nil
}
