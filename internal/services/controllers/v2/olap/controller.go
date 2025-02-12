package olap

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

type OLAPController interface {
	Start(ctx context.Context) error
}

type OLAPControllerImpl struct {
	mq                         msgqueue.MessageQueue
	l                          *zerolog.Logger
	repo                       repository.OLAPEventRepository
	v2repo                     v2.TaskRepository
	dv                         datautils.DataDecoderValidator
	a                          *hatcheterrors.Wrapped
	p                          *partition.Partition
	s                          gocron.Scheduler
	updateTaskStatusOperations *queueutils.OperationPool
	updateDAGStatusOperations  *queueutils.OperationPool
}

type OLAPControllerOpt func(*OLAPControllerOpts)

type OLAPControllerOpts struct {
	mq      msgqueue.MessageQueue
	l       *zerolog.Logger
	repo    repository.OLAPEventRepository
	v2repo  v2.TaskRepository
	dv      datautils.DataDecoderValidator
	alerter hatcheterrors.Alerter
	p       *partition.Partition
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

func WithPartition(p *partition.Partition) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.p = p
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

	if opts.p == nil {
		return nil, errors.New("partition is required. use WithPartition")
	}

	newLogger := opts.l.With().Str("service", "olap-controller").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "olap-controller"})

	o := &OLAPControllerImpl{
		mq:     opts.mq,
		l:      opts.l,
		s:      s,
		p:      opts.p,
		repo:   opts.repo,
		v2repo: opts.v2repo,
		dv:     opts.dv,
		a:      a,
	}

	o.updateTaskStatusOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "update task statuses", o.updateTaskStatuses)
	o.updateDAGStatusOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "update dag statuses", o.updateDAGStatuses)

	return o, nil
}

func (o *OLAPControllerImpl) Start() (func() error, error) {
	cleanupHeavyReadMQ, heavyReadMQ := o.mq.Clone()
	heavyReadMQ.SetQOS(2000)

	o.s.Start()

	mqBuffer := msgqueue.NewMQSubBuffer(msgqueue.OLAP_QUEUE, heavyReadMQ, o.handleBufferedMsgs)
	wg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())

	_, err := o.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			o.runTenantTaskStatusUpdates(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule task status updates: %w", err)
	}

	_, err = o.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			o.runTenantDAGStatusUpdates(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule dag status updates: %w", err)
	}

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	cleanup := func() error {
		cancel()

		if err := cleanupBuffer(); err != nil {
			return err
		}

		if err := cleanupHeavyReadMQ(); err != nil {
			return err
		}

		if err := o.s.Shutdown(); err != nil {
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
	case "created-dag":
		return tc.handleCreatedDAG(context.Background(), tenantId, payloads)
	case "create-monitoring-event":
		return tc.handleCreateMonitoringEvent(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

// handleCreatedTask is responsible for flushing a created task to the OLAP repository
func (tc *OLAPControllerImpl) handleCreatedTask(ctx context.Context, tenantId string, payloads [][]byte) error {
	createTaskOpts := make([]*sqlcv2.V2Task, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CreatedTaskPayload](payloads)

	for _, msg := range msgs {
		createTaskOpts = append(createTaskOpts, msg.V2Task)
	}

	return tc.repo.CreateTasks(ctx, tenantId, createTaskOpts)
}

// handleCreatedTask is responsible for flushing a created task to the OLAP repository
func (tc *OLAPControllerImpl) handleCreatedDAG(ctx context.Context, tenantId string, payloads [][]byte) error {
	createDAGOpts := make([]*v2.DAGWithData, 0)
	msgs := msgqueue.JSONConvert[tasktypes.CreatedDAGPayload](payloads)

	for _, msg := range msgs {
		createDAGOpts = append(createDAGOpts, msg.DAGWithData)
	}

	return tc.repo.CreateDAGs(ctx, tenantId, createDAGOpts)
}

// handleCreateMonitoringEvent is responsible for sending a group of monitoring events to the OLAP repository
func (tc *OLAPControllerImpl) handleCreateMonitoringEvent(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.CreateMonitoringEventPayload](payloads)

	taskIdsToLookup := make([]int64, len(msgs))

	for i, msg := range msgs {
		taskIdsToLookup[i] = msg.TaskId
	}

	metas, err := tc.v2repo.ListTaskMetas(ctx, tenantId, taskIdsToLookup)

	if err != nil {
		return err
	}

	taskIdsToMetas := make(map[int64]*sqlcv2.ListTaskMetasRow)

	for _, taskMeta := range metas {
		taskIdsToMetas[taskMeta.ID] = taskMeta
	}

	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)
	retryCounts := make([]int32, 0)
	workerIds := make([]string, 0)
	workflowIds := make([]pgtype.UUID, 0)
	eventTypes := make([]timescalev2.V2EventTypeOlap, 0)
	readableStatuses := make([]timescalev2.V2ReadableStatusOlap, 0)
	eventPayloads := make([]string, 0)
	eventMessages := make([]string, 0)
	timestamps := make([]pgtype.Timestamptz, 0)

	for _, msg := range msgs {
		taskMeta := taskIdsToMetas[msg.TaskId]

		if taskMeta == nil {
			tc.l.Error().Msgf("could not find task meta for task id %d", msg.TaskId)
			continue
		}

		taskIds = append(taskIds, msg.TaskId)
		taskInsertedAts = append(taskInsertedAts, taskMeta.InsertedAt)
		workflowIds = append(workflowIds, taskMeta.WorkflowID)
		retryCounts = append(retryCounts, msg.RetryCount)
		eventTypes = append(eventTypes, msg.EventType)
		eventPayloads = append(eventPayloads, msg.EventPayload)
		eventMessages = append(eventMessages, msg.EventMessage)
		timestamps = append(timestamps, sqlchelpers.TimestamptzFromTime(msg.EventTimestamp))

		if msg.WorkerId != nil {
			workerIds = append(workerIds, *msg.WorkerId)
		} else {
			workerIds = append(workerIds, "")
		}

		switch msg.EventType {
		case timescalev2.V2EventTypeOlapRETRYING:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapQUEUED)
		case timescalev2.V2EventTypeOlapREASSIGNED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapQUEUED)
		case timescalev2.V2EventTypeOlapRETRIEDBYUSER:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapQUEUED)
		case timescalev2.V2EventTypeOlapCREATED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapQUEUED)
		case timescalev2.V2EventTypeOlapQUEUED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapQUEUED)
		case timescalev2.V2EventTypeOlapREQUEUEDNOWORKER:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapQUEUED)
		case timescalev2.V2EventTypeOlapREQUEUEDRATELIMIT:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapQUEUED)
		case timescalev2.V2EventTypeOlapASSIGNED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapRUNNING)
		case timescalev2.V2EventTypeOlapACKNOWLEDGED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapRUNNING)
		case timescalev2.V2EventTypeOlapSENTTOWORKER:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapRUNNING)
		case timescalev2.V2EventTypeOlapSLOTRELEASED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapRUNNING)
		case timescalev2.V2EventTypeOlapSTARTED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapRUNNING)
		case timescalev2.V2EventTypeOlapTIMEOUTREFRESHED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapRUNNING)
		case timescalev2.V2EventTypeOlapSCHEDULINGTIMEDOUT:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapFAILED)
		case timescalev2.V2EventTypeOlapFINISHED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapCOMPLETED)
		case timescalev2.V2EventTypeOlapFAILED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapFAILED)
		case timescalev2.V2EventTypeOlapCANCELLED:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapCANCELLED)
		case timescalev2.V2EventTypeOlapTIMEDOUT:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapFAILED)
		case timescalev2.V2EventTypeOlapRATELIMITERROR:
			readableStatuses = append(readableStatuses, timescalev2.V2ReadableStatusOlapFAILED)
		}
	}

	opts := make([]timescalev2.CreateTaskEventsOLAPParams, 0)

	for i, taskId := range taskIds {
		var workerId pgtype.UUID

		if workerIds[i] != "" {
			workerId = sqlchelpers.UUIDFromStr(workerIds[i])
		}

		event := timescalev2.CreateTaskEventsOLAPParams{
			TenantID:       sqlchelpers.UUIDFromStr(tenantId),
			TaskID:         taskId,
			TaskInsertedAt: taskInsertedAts[i],
			WorkflowID:     workflowIds[i],
			EventType:      eventTypes[i],
			EventTimestamp: timestamps[i],
			ReadableStatus: readableStatuses[i],
			RetryCount:     retryCounts[i],
			WorkerID:       workerId,
		}

		switch eventTypes[i] {
		case timescalev2.V2EventTypeOlapFINISHED:
			event.Output = []byte(eventPayloads[i])
		case timescalev2.V2EventTypeOlapFAILED:
			event.ErrorMessage = sqlchelpers.TextFromStr(eventPayloads[i])
		case timescalev2.V2EventTypeOlapCANCELLED:
			event.AdditionalEventMessage = sqlchelpers.TextFromStr(eventMessages[i])
		}

		opts = append(opts, event)
	}

	return tc.repo.CreateTaskEvents(ctx, tenantId, opts)
}
