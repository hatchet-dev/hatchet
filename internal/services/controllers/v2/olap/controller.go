package olap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type OLAPController interface {
	Start(ctx context.Context) error
}

type OLAPControllerImpl struct {
	mq     msgqueue.MessageQueue
	l      *zerolog.Logger
	repo   repository.OLAPEventRepository
	v2repo v2.TaskRepository
	dv     datautils.DataDecoderValidator
	a      *hatcheterrors.Wrapped
}

type OLAPControllerOpt func(*OLAPControllerOpts)

type OLAPControllerOpts struct {
	mq      msgqueue.MessageQueue
	l       *zerolog.Logger
	repo    repository.OLAPEventRepository
	v2repo  v2.TaskRepository
	dv      datautils.DataDecoderValidator
	alerter hatcheterrors.Alerter
}

func defaultOLAPControllerOpts() *OLAPControllerOpts {
	l := logger.NewDefaultLogger("olap-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &OLAPControllerOpts{
		l:       &l,
		dv:      datautils.NewDataDecoderValidator(),
		alerter: alerter,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.l = l
	}
}

func WithAlerter(a hatcheterrors.Alerter) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.alerter = a
	}
}

func WithRepository(r repository.OLAPEventRepository) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.repo = r
	}
}

func WithV2Repository(r v2.TaskRepository) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.v2repo = r
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.dv = dv
	}
}

func New(fs ...OLAPControllerOpt) (*OLAPControllerImpl, error) {
	opts := defaultOLAPControllerOpts()

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
		return nil, fmt.Errorf("v2repository is required. use WithRepository")
	}

	newLogger := opts.l.With().Str("service", "olap-controller").Logger()
	opts.l = &newLogger

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "olap-controller"})

	return &OLAPControllerImpl{
		mq:     opts.mq,
		l:      opts.l,
		repo:   opts.repo,
		v2repo: opts.v2repo,
		dv:     opts.dv,
		a:      a,
	}, nil
}

func (tc *OLAPControllerImpl) Start() (func() error, error) {
	cleanupHeavyReadMQ, heavyReadMQ := tc.mq.Clone()
	heavyReadMQ.SetQOS(2000)

	mqBuffer := msgqueue.NewMQSubBuffer(msgqueue.OLAP_QUEUE, heavyReadMQ, tc.handleBufferedMsgs)
	wg := sync.WaitGroup{}

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	cleanup := func() error {
		if err := cleanupBuffer(); err != nil {
			return err
		}

		if err := cleanupHeavyReadMQ(); err != nil {
			return err
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (tc *OLAPControllerImpl) handleBufferedMsgs(tenantId, msgId string, payloads [][]byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(tc.l, tc.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch msgId {
	case "created-task":
		return tc.handleCreatedTask(context.Background(), tenantId, payloads)
	case "create-monitoring-event":
		return tc.handleCreateMonitoringEvent(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

// handleCreatedTask is responsible for flushing a created task to the OLAP repository
func (tc *OLAPControllerImpl) handleCreatedTask(ctx context.Context, tenantId string, payloads [][]byte) error {
	opts := make([]olap.Task, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CreatedTaskPayload](payloads)

	for _, msg := range msgs {
		var priority int32 = 1

		if msg.Priority != nil {
			priority = int32(*msg.Priority)
		}

		// TODO: ADD STICKY AND DESIRED WORKER ID

		// TODO: ADD ADDITIONAL METADATA

		opts = append(opts, olap.Task{
			Id:              uuid.MustParse(msg.ExternalId),
			TenantId:        uuid.MustParse(tenantId),
			Queue:           msg.Queue,
			ActionId:        msg.ActionId,
			ScheduleTimeout: msg.ScheduleTimeout,
			StepTimeout:     msg.StepTimeout,
			Priority:        priority,
			Sticky:          olap.STICKY_NONE,
			DisplayName:     msg.DisplayName,
			Input:           string(msg.Input),
		})
	}

	return tc.repo.CreateTasks(opts)
}

// handleCreateMonitoringEvent is responsible for sending a group of monitoring events to the OLAP repository
func (tc *OLAPControllerImpl) handleCreateMonitoringEvent(ctx context.Context, tenantId string, payloads [][]byte) error {
	taskIds := make([]int64, 0)
	retryCounts := make([]int32, 0)
	workerIds := make([]string, 0)
	eventTypes := make([]contracts.StepActionEventType, 0)
	eventPayloads := make([]string, 0)
	timestamps := make([]time.Time, 0)
	msgs := msgqueue.JSONConvert[tasktypes.CreateMonitoringEventPayload](payloads)

	for _, msg := range msgs {
		eventTimeAt, err := time.Parse(time.RFC3339, msg.EventTimestamp)

		if err != nil {
			eventTimeAt = time.Now()
		}

		enumVal, ok := contracts.StepActionEventType_value[msg.EventType]

		if !ok {
			continue
		}

		taskIds = append(taskIds, msg.TaskId)
		retryCounts = append(retryCounts, msg.RetryCount)
		workerIds = append(workerIds, msg.WorkerId)
		eventTypes = append(eventTypes, contracts.StepActionEventType(enumVal))
		eventPayloads = append(eventPayloads, msg.EventPayload)
		timestamps = append(timestamps, eventTimeAt)
	}

	metas, err := tc.v2repo.ListTaskMetas(ctx, tenantId, taskIds)

	if err != nil {
		return err
	}

	taskIdsToMeta := make(map[int64]*sqlcv2.ListTaskMetasRow)

	for _, taskMeta := range metas {
		taskIdsToMeta[taskMeta.ID] = taskMeta
	}

	opts := make([]olap.TaskEvent, 0)

	for i, taskId := range taskIds {

		taskMeta, ok := taskIdsToMeta[taskId]

		if !ok {
			continue
		}

		var workerId *uuid.UUID

		if workerIds[i] != "" {
			parsedWorkerId, parseErr := uuid.Parse(workerIds[i])

			if parseErr == nil {
				workerId = &parsedWorkerId
			}
		}

		event := olap.TaskEvent{
			TaskId:     uuid.MustParse(sqlchelpers.UUIDToStr(taskMeta.ExternalID)),
			TenantId:   uuid.MustParse(tenantId),
			Timestamp:  timestamps[i],
			RetryCount: uint32(retryCounts[i]),
			WorkerId:   workerId,
		}

		switch eventTypes[i] {
		case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
			event.EventType = olap.EVENT_TYPE_FINISHED
			event.Output = eventPayloads[i]
		case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
			event.EventType = olap.EVENT_TYPE_FAILED
			event.ErrorMsg = eventPayloads[i]
		case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
			event.EventType = olap.EVENT_TYPE_STARTED
		default:
			continue
		}

		opts = append(opts, event)
	}

	return tc.repo.CreateTaskEvents(opts)
}
