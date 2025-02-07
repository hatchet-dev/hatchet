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
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

type TriggerController interface {
	Start(ctx context.Context) error
}

type TriggerControllerImpl struct {
	mq        msgqueue.MessageQueue
	pubBuffer *msgqueue.MQPubBuffer
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

	pubBuffer := msgqueue.NewMQPubBuffer(opts.mq)

	return &TriggerControllerImpl{
		mq:        opts.mq,
		pubBuffer: pubBuffer,
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
	case "user-event":
		return tc.handleProcessEventTrigger(context.Background(), tenantId, payloads)
	case "task-trigger":
		return tc.handleProcessTaskTrigger(context.Background(), tenantId, payloads)
	case "internal-event":
		return tc.handleProcessInternalEventMatches(context.Background(), tenantId, payloads)
	}

	tc.l.Error().Msgf("unknown message id: %s", msgId)

	return nil
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TriggerControllerImpl) handleProcessEventTrigger(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.UserEventTaskPayload](payloads)

	eg := &errgroup.Group{}

	// TODO: RUN IN THE SAME TRANSACTION
	eg.Go(func() error {
		return tc.handleProcessUserEventTrigger(ctx, tenantId, msgs)
	})

	eg.Go(func() error {
		return tc.handleProcessUserEventMatches(ctx, tenantId, msgs)
	})

	return eg.Wait()
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TriggerControllerImpl) handleProcessUserEventTrigger(ctx context.Context, tenantId string, msgs []*tasktypes.UserEventTaskPayload) error {
	opts := make([]v2.EventTriggerOpts, 0, len(msgs))

	for _, msg := range msgs {
		opts = append(opts, v2.EventTriggerOpts{
			EventId:            msg.EventId,
			Key:                msg.EventKey,
			Data:               msg.EventData,
			AdditionalMetadata: msg.EventAdditionalMetadata,
		})
	}

	tasks, err := tc.v2repo.Triggers().TriggerFromEvents(ctx, tenantId, opts)

	if err != nil {
		return fmt.Errorf("could not trigger tasks from events: %w", err)
	}

	return tc.signalTasksCreated(ctx, tenantId, tasks)
}

// handleProcessUserEventMatches is responsible for triggering tasks based on user event matches.
func (tc *TriggerControllerImpl) handleProcessUserEventMatches(ctx context.Context, tenantId string, payloads []*tasktypes.UserEventTaskPayload) error {
	tc.l.Error().Msg("not implemented")
	return nil
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TriggerControllerImpl) handleProcessTaskTrigger(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.TriggerTaskPayload](payloads)

	opts := make([]v2.WorkflowNameTriggerOpts, 0, len(msgs))

	for _, msg := range msgs {
		opts = append(opts, v2.WorkflowNameTriggerOpts{
			WorkflowName:       msg.WorkflowName,
			ExternalId:         msg.TaskExternalId,
			Data:               msg.Data,
			AdditionalMetadata: msg.AdditionalMetadata,
		})
	}

	tasks, err := tc.v2repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)

	if err != nil {
		return fmt.Errorf("could not query workflows for events: %w", err)
	}

	return tc.signalTasksCreated(ctx, tenantId, tasks)
}

// handleProcessUserEventMatches is responsible for triggering tasks based on user event matches.
func (tc *TriggerControllerImpl) handleProcessInternalEventMatches(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.InternalEventTaskPayload](payloads)
	candidateMatches := make([]v2.CandidateEventMatch, 0)

	for _, msg := range msgs {
		candidateMatches = append(candidateMatches, v2.CandidateEventMatch{
			ID:             uuid.NewString(),
			EventTimestamp: msg.EventTimestamp,
			Key:            msg.EventKey,
			Data:           msg.EventData,
		})
	}

	tasks, err := tc.v2repo.Matches().ProcessInternalEventMatches(ctx, tenantId, candidateMatches)

	if err != nil {
		return fmt.Errorf("could not process internal event matches: %w", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	return tc.signalTasksCreated(ctx, tenantId, tasks)
}

func (tc *TriggerControllerImpl) signalTasksCreated(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error {
	// get all unique queues and notify them
	queues := make(map[string]struct{})

	for _, task := range tasks {
		queues[task.Queue] = struct{}{}
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
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		taskCp := task
		msg, err := tasktypes.CreatedTaskMessage(tenantId, taskCp)

		if err != nil {
			tc.l.Err(err).Msg("could not create message for olap queue")
			continue
		}

		err = tc.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			msg,
			false,
		)

		if err != nil {
			tc.l.Err(err).Msg("could not add message to olap queue")
			continue
		}

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     0,
				EventType:      timescalev2.V2EventTypeOlapQUEUED,
				EventTimestamp: time.Now(),
			},
		)

		if err != nil {
			tc.l.Err(err).Msg("could not create monitoring event message")
			continue
		}

		err = tc.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
			false,
		)

		if err != nil {
			tc.l.Err(err).Msg("could not add monitoring event message to olap queue")
			continue
		}
	}

	return nil
}
