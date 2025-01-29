package olap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
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
	createTaskEventOpts := make([]olap.TaskEvent, 0)
	createTaskOpts := make([]olap.Task, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CreatedTaskPayload](payloads)

	for _, msg := range msgs {
		var priority int32 = 1

		if msg.Priority != nil {
			priority = int32(*msg.Priority)
		}

		// TODO: ADD STICKY AND DESIRED WORKER ID

		// TODO: ADD ADDITIONAL METADATA

		createTaskOpts = append(createTaskOpts, olap.Task{
			Id:              uuid.MustParse(msg.ExternalId),
			SourceId:        msg.SourceId,
			InsertedAt:      msg.InsertedAt,
			TenantId:        uuid.MustParse(tenantId),
			Queue:           msg.Queue,
			ActionId:        msg.ActionId,
			ScheduleTimeout: msg.ScheduleTimeout,
			StepTimeout:     msg.StepTimeout,
			WorkflowId:      uuid.MustParse(msg.WorkflowId),
			Priority:        priority,
			Sticky:          olap.STICKY_NONE,
			DisplayName:     msg.DisplayName,
			Input:           string(msg.Input),
		})

		createTaskEventOpts = append(createTaskEventOpts, olap.TaskEvent{
			TaskId:     uuid.MustParse(msg.ExternalId),
			TenantId:   uuid.MustParse(tenantId),
			Timestamp:  msg.InsertedAt,
			RetryCount: 0,
			EventType:  olap.EVENT_TYPE_CREATED,
		})
	}

	eg := errgroup.Group{}

	eg.Go(func() error {
		return tc.repo.CreateTasks(createTaskOpts)
	})

	eg.Go(func() error {
		return tc.repo.CreateTaskEvents(createTaskEventOpts)
	})

	return eg.Wait()
}

// handleCreateMonitoringEvent is responsible for sending a group of monitoring events to the OLAP repository
func (tc *OLAPControllerImpl) handleCreateMonitoringEvent(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.CreateMonitoringEventPayload](payloads)

	taskIdsToLookup := make([]int64, len(msgs))

	for i, msg := range msgs {
		taskId := msg.TaskId
		if taskId != nil {
			taskIdsToLookup[i] = *taskId
		}
	}

	metas, err := tc.v2repo.ListTaskMetas(ctx, tenantId, taskIdsToLookup)

	if err != nil {
		return err
	}

	taskIdsToExternalIds := make(map[int64]string)

	for _, taskMeta := range metas {
		taskIdsToExternalIds[taskMeta.ID] = sqlchelpers.UUIDToStr(taskMeta.ExternalID)
	}

	taskExternalIds := make([]string, 0)
	retryCounts := make([]int32, 0)
	workerIds := make([]string, 0)
	eventTypes := make([]olap.EventType, 0)
	eventPayloads := make([]string, 0)
	eventMessages := make([]string, 0)
	timestamps := make([]time.Time, 0)

	for _, msg := range msgs {
		var externalId string

		if msg.TaskExternalId != nil {
			externalId = *msg.TaskExternalId
		} else if msg.TaskId != nil {
			externalIdLookup, ok := taskIdsToExternalIds[*msg.TaskId]

			if !ok {
				tc.l.Error().Msgf("could not find external id for task id %d", *msg.TaskId)
				continue
			}

			externalId = externalIdLookup
		} else {
			tc.l.Error().Msg("task id or external id must be set")
			continue
		}

		taskExternalIds = append(taskExternalIds, externalId)
		retryCounts = append(retryCounts, msg.RetryCount)
		eventTypes = append(eventTypes, msg.EventType)
		eventPayloads = append(eventPayloads, msg.EventPayload)
		eventMessages = append(eventMessages, msg.EventMessage)
		timestamps = append(timestamps, msg.EventTimestamp)

		if msg.WorkerId != nil {
			workerIds = append(workerIds, *msg.WorkerId)
		} else {
			workerIds = append(workerIds, "")
		}
	}

	opts := make([]olap.TaskEvent, 0)

	for i, taskId := range taskExternalIds {
		var workerId *uuid.UUID

		if workerIds[i] != "" {
			parsedWorkerId, parseErr := uuid.Parse(workerIds[i])

			if parseErr == nil {
				workerId = &parsedWorkerId
			}
		}

		event := olap.TaskEvent{
			TaskId:     uuid.MustParse(taskId),
			TenantId:   uuid.MustParse(tenantId),
			Timestamp:  timestamps[i],
			RetryCount: uint32(retryCounts[i]),
			WorkerId:   workerId,
			EventType:  eventTypes[i],
		}

		switch eventTypes[i] {
		case olap.EVENT_TYPE_FINISHED:
			event.Output = eventPayloads[i]
		case olap.EVENT_TYPE_FAILED:
			event.ErrorMsg = eventPayloads[i]
		case olap.EVENT_TYPE_CANCELLED:
			event.AdditionalEventMessage = eventMessages[i]
		}

		opts = append(opts, event)
	}

	return tc.repo.CreateTaskEvents(opts)
}
