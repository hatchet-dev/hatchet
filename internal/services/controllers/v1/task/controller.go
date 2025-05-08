package task

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

const BULK_MSG_BATCH_SIZE = 50

type TasksController interface {
	Start(ctx context.Context) error
}

type TasksControllerImpl struct {
	mq                     msgqueue.MessageQueue
	pubBuffer              *msgqueue.MQPubBuffer
	l                      *zerolog.Logger
	queueLogger            *zerolog.Logger
	pgxStatsLogger         *zerolog.Logger
	repo                   repository.EngineRepository
	repov1                 v1.Repository
	dv                     datautils.DataDecoderValidator
	s                      gocron.Scheduler
	a                      *hatcheterrors.Wrapped
	p                      *partition.Partition
	celParser              *cel.CELParser
	timeoutTaskOperations  *queueutils.OperationPool
	reassignTaskOperations *queueutils.OperationPool
	retryTaskOperations    *queueutils.OperationPool
	emitSleepOperations    *queueutils.OperationPool
}

type TasksControllerOpt func(*TasksControllerOpts)

type TasksControllerOpts struct {
	mq             msgqueue.MessageQueue
	l              *zerolog.Logger
	repo           repository.EngineRepository
	repov1         v1.Repository
	dv             datautils.DataDecoderValidator
	alerter        hatcheterrors.Alerter
	p              *partition.Partition
	queueLogger    *zerolog.Logger
	pgxStatsLogger *zerolog.Logger
}

func defaultTasksControllerOpts() *TasksControllerOpts {
	l := logger.NewDefaultLogger("tasks-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	queueLogger := logger.NewDefaultLogger("queue")
	pgxStatsLogger := logger.NewDefaultLogger("pgx-stats")

	return &TasksControllerOpts{
		l:              &l,
		dv:             datautils.NewDataDecoderValidator(),
		alerter:        alerter,
		queueLogger:    &queueLogger,
		pgxStatsLogger: &pgxStatsLogger,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.l = l
	}
}

func WithQueueLoggerConfig(lc *shared.LoggerConfigFile) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		l := logger.NewStdErr(lc, "queue")
		opts.queueLogger = &l
	}
}

func WithPgxStatsLoggerConfig(lc *shared.LoggerConfigFile) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		l := logger.NewStdErr(lc, "pgx-stats")
		opts.pgxStatsLogger = &l
	}
}

func WithAlerter(a hatcheterrors.Alerter) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.alerter = a
	}
}

func WithRepository(r repository.EngineRepository) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.repo = r
	}
}

func WithV1Repository(r v1.Repository) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.repov1 = r
	}
}

func WithPartition(p *partition.Partition) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.p = p
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.dv = dv
	}
}

func New(fs ...TasksControllerOpt) (*TasksControllerImpl, error) {
	opts := defaultTasksControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repov1 == nil {
		return nil, fmt.Errorf("v2 repository is required. use WithV2Repository")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.p == nil {
		return nil, errors.New("partition is required. use WithPartition")
	}

	newLogger := opts.l.With().Str("service", "tasks-controller").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "tasks-controller"})

	pubBuffer := msgqueue.NewMQPubBuffer(opts.mq)

	t := &TasksControllerImpl{
		mq:             opts.mq,
		pubBuffer:      pubBuffer,
		l:              opts.l,
		queueLogger:    opts.queueLogger,
		pgxStatsLogger: opts.pgxStatsLogger,
		repo:           opts.repo,
		repov1:         opts.repov1,
		dv:             opts.dv,
		s:              s,
		a:              a,
		p:              opts.p,
		celParser:      cel.NewCELParser(),
	}

	t.timeoutTaskOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "timeout step runs", t.processTaskTimeouts)
	t.emitSleepOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "emit sleep step runs", t.processSleeps)
	t.reassignTaskOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "reassign step runs", t.processTaskReassignments)
	t.retryTaskOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "retry step runs", t.processTaskRetryQueueItems)

	return t, nil
}

func (tc *TasksControllerImpl) Start() (func() error, error) {
	mqBuffer := msgqueue.NewMQSubBuffer(msgqueue.TASK_PROCESSING_QUEUE, tc.mq, tc.handleBufferedMsgs)

	tc.s.Start()

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	startupPartitionCtx, cancelStartupPartition := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStartupPartition()

	// always create table partition on startup
	if err := tc.createTablePartition(startupPartitionCtx); err != nil {
		return nil, fmt.Errorf("could not create table partition: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	_, err = tc.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			tc.runTenantTimeoutTasks(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run timeout: %w", err)
	}

	_, err = tc.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			tc.runTenantSleepEmitter(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run emit sleep: %w", err)
	}

	_, err = tc.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			tc.runTenantReassignTasks(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run reassignment: %w", err)
	}

	_, err = tc.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			tc.runTenantRetryQueueItems(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run reassignment: %w", err)
	}

	_, err = tc.s.NewJob(
		gocron.DurationJob(time.Minute*15),
		gocron.NewTask(
			tc.runTaskTablePartition(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule task partition method: %w", err)
	}

	cleanup := func() error {
		cancel()

		if err := cleanupBuffer(); err != nil {
			return err
		}

		tc.pubBuffer.Stop()

		if err := tc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		return nil
	}

	return cleanup, nil
}

func (tc *TasksControllerImpl) handleBufferedMsgs(tenantId, msgId string, payloads [][]byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(tc.l, tc.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch msgId {
	case "task-completed":
		return tc.handleTaskCompleted(context.Background(), tenantId, payloads)
	case "task-failed":
		return tc.handleTaskFailed(context.Background(), tenantId, payloads)
	case "task-cancelled":
		return tc.handleTaskCancelled(context.Background(), tenantId, payloads)
	case "cancel-tasks":
		return tc.handleCancelTasks(context.Background(), tenantId, payloads)
	case "replay-tasks":
		return tc.handleReplayTasks(context.Background(), tenantId, payloads)
	case "user-event":
		return tc.handleProcessUserEvents(context.Background(), tenantId, payloads)
	case "internal-event":
		return tc.handleProcessInternalEvents(context.Background(), tenantId, payloads)
	case "task-trigger":
		return tc.handleProcessTaskTrigger(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

func (tc *TasksControllerImpl) handleTaskCompleted(ctx context.Context, tenantId string, payloads [][]byte) error {
	opts := make([]v1.CompleteTaskOpts, 0)
	idsToData := make(map[int64][]byte)

	msgs := msgqueue.JSONConvert[tasktypes.CompletedTaskPayload](payloads)

	for _, msg := range msgs {
		opts = append(opts, v1.CompleteTaskOpts{
			TaskIdInsertedAtRetryCount: &v1.TaskIdInsertedAtRetryCount{
				Id:         msg.TaskId,
				InsertedAt: msg.InsertedAt,
				RetryCount: msg.RetryCount,
			},
			Output: msg.Output,
		})

		idsToData[msg.TaskId] = msg.Output
	}

	res, err := tc.repov1.Tasks().CompleteTasks(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	// instrumentation
	for range res.ReleasedTasks {
		prometheus.SucceededTasks.Inc()
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, res.ReleasedTasks)

	return tc.sendInternalEvents(ctx, tenantId, res.InternalEvents)
}

func (tc *TasksControllerImpl) handleTaskFailed(ctx context.Context, tenantId string, payloads [][]byte) error {
	opts := make([]v1.FailTaskOpts, 0)

	msgs := msgqueue.JSONConvert[tasktypes.FailedTaskPayload](payloads)
	idsToErrorMsg := make(map[int64]string)

	for _, msg := range msgs {
		opts = append(opts, v1.FailTaskOpts{
			TaskIdInsertedAtRetryCount: &v1.TaskIdInsertedAtRetryCount{
				Id:         msg.TaskId,
				InsertedAt: msg.InsertedAt,
				RetryCount: msg.RetryCount,
			},
			IsAppError:     msg.IsAppError,
			ErrorMessage:   msg.ErrorMsg,
			IsNonRetryable: msg.IsNonRetryable,
		})

		if msg.ErrorMsg != "" {
			idsToErrorMsg[msg.TaskId] = msg.ErrorMsg
		}

		// send failed tasks to the olap repository
		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         msg.TaskId,
				RetryCount:     msg.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapFAILED,
				EventTimestamp: time.Now().UTC(),
				EventPayload:   msg.ErrorMsg,
			},
		)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not create monitoring event message")
			continue
		}

		err = tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, olapMsg, false)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not create monitoring event message")
			continue
		}
	}

	res, err := tc.repov1.Tasks().FailTasks(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	return tc.processFailTasksResponse(ctx, tenantId, res)
}

func (tc *TasksControllerImpl) processFailTasksResponse(ctx context.Context, tenantId string, res *v1.FailTasksResponse) error {
	retriedTaskIds := make(map[int64]struct{})

	for _, task := range res.RetriedTasks {
		retriedTaskIds[task.Id] = struct{}{}
	}

	internalEventsWithoutRetries := make([]v1.InternalTaskEvent, 0)

	for _, e := range res.InternalEvents {
		// if the task is retried, don't send a message to the trigger queue
		if _, ok := retriedTaskIds[e.TaskID]; ok {
			prometheus.RetriedTasks.Inc()
			continue
		}

		internalEventsWithoutRetries = append(internalEventsWithoutRetries, e)
		prometheus.FailedTasks.Inc()
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, res.ReleasedTasks)

	// TODO: MOVE THIS TO THE DATA LAYER?
	err := tc.sendInternalEvents(ctx, tenantId, internalEventsWithoutRetries)

	if err != nil {
		return err
	}

	var outerErr error

	// send retried tasks to the olap repository
	for _, task := range res.RetriedTasks {
		if task.IsAppError {
			err = tc.pubRetryEvent(ctx, tenantId, task)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not publish retry event: %w", err))
			}
		}
	}

	return outerErr
}

func (tc *TasksControllerImpl) handleTaskCancelled(ctx context.Context, tenantId string, payloads [][]byte) error {
	opts := make([]v1.TaskIdInsertedAtRetryCount, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CancelledTaskPayload](payloads)
	shouldTasksNotify := make(map[int64]bool)

	for _, msg := range msgs {
		opts = append(opts, v1.TaskIdInsertedAtRetryCount{
			Id:         msg.TaskId,
			InsertedAt: msg.InsertedAt,
			RetryCount: msg.RetryCount,
		})

		shouldTasksNotify[msg.TaskId] = msg.ShouldNotify
	}

	res, err := tc.repov1.Tasks().CancelTasks(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	tasksToSendToDispatcher := make([]tasktypes.SignalTaskCancelledPayload, 0)

	for _, task := range res.ReleasedTasks {
		if shouldTasksNotify[task.ID] {
			tasksToSendToDispatcher = append(tasksToSendToDispatcher, tasktypes.SignalTaskCancelledPayload{
				TaskId:     task.ID,
				WorkerId:   sqlchelpers.UUIDToStr(task.WorkerID),
				RetryCount: task.RetryCount,
			})
		}
	}

	// send task cancellations to the dispatcher
	err = tc.sendTaskCancellationsToDispatcher(ctx, tenantId, tasksToSendToDispatcher)

	if err != nil {
		return fmt.Errorf("could not send task cancellations to dispatcher: %w", err)
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, res.ReleasedTasks)

	// TODO: MOVE THIS TO THE DATA LAYER?
	err = tc.sendInternalEvents(ctx, tenantId, res.InternalEvents)

	if err != nil {
		return err
	}

	var outerErr error

	for _, msg := range msgs {
		taskId := msg.TaskId

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         taskId,
				RetryCount:     msg.RetryCount,
				EventType:      msg.EventType,
				EventTimestamp: time.Now(),
				EventMessage:   msg.EventMessage,
			},
		)

		if err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not create monitoring event message: %w", err))
			continue
		}

		err = tc.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
			false,
		)

		if err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not publish monitoring event message: %w", err))
		}
	}

	// instrumentation
	for range res.ReleasedTasks {
		prometheus.CancelledTasks.Inc()
	}

	return err
}

func (tc *TasksControllerImpl) handleCancelTasks(ctx context.Context, tenantId string, payloads [][]byte) error {
	// sure would be nice if we could use our own durable execution primitives here, but that's a bootstrapping
	// problem that we don't have a clean way to solve (yet)
	msgs := msgqueue.JSONConvert[tasktypes.CancelTasksPayload](payloads)
	pubPayloads := make([]tasktypes.CancelledTaskPayload, 0)

	for _, msg := range msgs {
		for _, task := range msg.Tasks {
			pubPayloads = append(pubPayloads, tasktypes.CancelledTaskPayload{
				TaskId:       task.Id,
				InsertedAt:   task.InsertedAt,
				RetryCount:   task.RetryCount,
				EventType:    sqlcv1.V1EventTypeOlapCANCELLED,
				ShouldNotify: true,
			})
		}
	}

	// Batch tasks to cancel in groups of 50 and publish to the message queue. This is a form of backpressure
	// as we don't want to run out of RabbitMQ memory if we publish a very large number of tasks to cancel.
	return queueutils.BatchLinear(BULK_MSG_BATCH_SIZE, pubPayloads, func(pubPayloads []tasktypes.CancelledTaskPayload) error {
		msg, err := msgqueue.NewTenantMessage(
			tenantId,
			"task-cancelled",
			false,
			true,
			pubPayloads...,
		)

		if err != nil {
			return fmt.Errorf("could not create message for task cancellation: %w", err)
		}

		return tc.mq.SendMessage(
			ctx,
			msgqueue.TASK_PROCESSING_QUEUE,
			msg,
		)
	})
}

func (tc *TasksControllerImpl) handleReplayTasks(ctx context.Context, tenantId string, payloads [][]byte) error {
	// sure would be nice if we could use our own durable execution primitives here, but that's a bootstrapping
	// problem that we don't have a clean way to solve (yet)
	msgs := msgqueue.JSONConvert[tasktypes.ReplayTasksPayload](payloads)

	taskIdRetryCounts := make([]v1.TaskIdInsertedAtRetryCount, 0)

	for _, msg := range msgs {
		for _, task := range msg.Tasks {
			taskIdRetryCounts = append(taskIdRetryCounts, v1.TaskIdInsertedAtRetryCount{
				Id:         task.Id,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			})
		}
	}

	replayRes, err := tc.repov1.Tasks().ReplayTasks(ctx, tenantId, taskIdRetryCounts)

	if err != nil {
		return fmt.Errorf("could not replay tasks: %w", err)
	}

	eg := &errgroup.Group{}

	if len(replayRes.ReplayedTasks) > 0 {
		eg.Go(func() error {
			err = tc.signalTasksReplayed(ctx, tenantId, replayRes.ReplayedTasks)

			if err != nil {
				return fmt.Errorf("could not signal replayed tasks: %w", err)
			}

			return nil
		})
	}

	if len(replayRes.UpsertedTasks) > 0 {
		eg.Go(func() error {
			err = tc.signalTasksUpdated(ctx, tenantId, replayRes.UpsertedTasks)

			if err != nil {
				return fmt.Errorf("could not signal queued tasks: %w", err)
			}

			return nil
		})
	}

	if len(replayRes.InternalEventResults.CreatedTasks) > 0 {
		eg.Go(func() error {
			err = tc.signalTasksCreated(ctx, tenantId, replayRes.InternalEventResults.CreatedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (tc *TasksControllerImpl) sendTaskCancellationsToDispatcher(ctx context.Context, tenantId string, releasedTasks []tasktypes.SignalTaskCancelledPayload) error {
	workerIds := make([]string, 0)

	for _, task := range releasedTasks {
		workerIds = append(workerIds, task.WorkerId)
	}

	dispatcherIdWorkerIds, err := tc.repo.Worker().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for workers: %w", err)
	}

	workerIdToDispatcherId := make(map[string]string)

	for dispatcherId, workerIds := range dispatcherIdWorkerIds {
		for _, workerId := range workerIds {
			workerIdToDispatcherId[workerId] = dispatcherId
		}
	}

	// assemble messages
	dispatcherIdsToPayloads := make(map[string][]tasktypes.SignalTaskCancelledPayload)

	for _, task := range releasedTasks {
		workerId := task.WorkerId
		dispatcherId := workerIdToDispatcherId[workerId]

		dispatcherIdsToPayloads[dispatcherId] = append(dispatcherIdsToPayloads[dispatcherId], task)
	}

	// send messages
	for dispatcherId, payloads := range dispatcherIdsToPayloads {
		msg, err := msgqueue.NewTenantMessage(
			tenantId,
			"task-cancelled",
			false,
			true,
			payloads...,
		)

		if err != nil {
			return fmt.Errorf("could not create message for task cancellation: %w", err)
		}

		err = tc.mq.SendMessage(
			ctx,
			msgqueue.QueueTypeFromDispatcherID(dispatcherId),
			msg,
		)

		if err != nil {
			return fmt.Errorf("could not send message for task cancellation: %w", err)
		}
	}

	return nil
}

func (tc *TasksControllerImpl) notifyQueuesOnCompletion(ctx context.Context, tenantId string, releasedTasks []*sqlcv1.ReleaseTasksRow) {
	if len(releasedTasks) == 0 {
		return
	}

	tenant, err := tc.repo.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		tc.l.Err(err).Msg("could not get tenant")
		return
	}

	if tenant.SchedulerPartitionId.Valid {
		msg, err := tasktypes.NotifyTaskReleased(tenantId, releasedTasks)

		if err != nil {
			tc.l.Err(err).Msg("could not create message for scheduler partition queue")
		} else {
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

	payloads := make([]tasktypes.CandidateFinalizedPayload, 0, len(releasedTasks))

	for _, releasedTask := range releasedTasks {
		payloads = append(payloads, tasktypes.CandidateFinalizedPayload{
			WorkflowRunId: sqlchelpers.UUIDToStr(releasedTask.WorkflowRunID),
		})
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		"workflow-run-finished-candidate",
		true,
		false,
		payloads...,
	)

	if err != nil {
		tc.l.Err(err).Msg("could not create message for workflow run finished candidate")
		return
	}

	err = tc.mq.SendMessage(
		ctx,
		msgqueue.TenantEventConsumerQueue(tenantId),
		msg,
	)

	if err != nil {
		tc.l.Err(err).Msg("could not send workflow-run-finished-candidate message")
		return
	}
}

// handleProcessUserEvents is responsible for inserting tasks into the database based on event triggers.
func (tc *TasksControllerImpl) handleProcessUserEvents(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.UserEventTaskPayload](payloads)

	eg := &errgroup.Group{}

	// TODO: run these in the same tx or send as separate messages?
	eg.Go(func() error {
		return tc.handleProcessUserEventTrigger(ctx, tenantId, msgs)
	})

	eg.Go(func() error {
		return tc.handleProcessUserEventMatches(ctx, tenantId, msgs)
	})

	return eg.Wait()
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TasksControllerImpl) handleProcessUserEventTrigger(ctx context.Context, tenantId string, msgs []*tasktypes.UserEventTaskPayload) error {
	opts := make([]v1.EventTriggerOpts, 0, len(msgs))

	for _, msg := range msgs {
		opts = append(opts, v1.EventTriggerOpts{
			EventId:            msg.EventId,
			Key:                msg.EventKey,
			Data:               msg.EventData,
			AdditionalMetadata: msg.EventAdditionalMetadata,
			Priority:           msg.EventPriority,
		})
	}

	result, err := tc.repov1.Triggers().TriggerFromEvents(ctx, tenantId, opts)

	if err != nil {
		return fmt.Errorf("could not trigger tasks from events: %w", err)
	}

	eventTriggerOpts := make([]tasktypes.CreatedEventTriggerPayloadSingleton, 0)
	eventSeenAt := time.Now()

	for _, runsAndOpts := range *result.EventIdToRunsAndOpts {
		opts := runsAndOpts.Opts
		runs := runsAndOpts.Runs

		if len(runs) == 0 {
			eventTriggerOpts = append(eventTriggerOpts, tasktypes.CreatedEventTriggerPayloadSingleton{
				// FIXME: Should `SeenAt` be set on the SDK when the event is created?
				EventSeenAt:             eventSeenAt,
				EventKey:                opts.Key,
				EventId:                 opts.EventId,
				EventPayload:            opts.Data,
				EventAdditionalMetadata: opts.AdditionalMetadata,
			})

			continue
		}

		for _, run := range runs {
			eventTriggerOpts = append(eventTriggerOpts, tasktypes.CreatedEventTriggerPayloadSingleton{
				MaybeRunId:         &run.Id,
				MaybeRunInsertedAt: &run.InsertedAt,
				// FIXME: Should `SeenAt` be set on the SDK when the event is created?
				EventSeenAt:             eventSeenAt,
				EventKey:                opts.Key,
				EventId:                 opts.EventId,
				EventPayload:            opts.Data,
				EventAdditionalMetadata: opts.AdditionalMetadata,
			})
		}
	}

	msg, err := tasktypes.CreatedEventTriggerMessage(
		tenantId,
		tasktypes.CreatedEventTriggerPayload{
			Payloads: eventTriggerOpts,
		},
	)

	if err != nil {
		return fmt.Errorf("could not create event trigger message: %w", err)
	}

	err = tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false)

	if err != nil {
		return fmt.Errorf("could not trigger tasks from events: %w", err)
	}

	eg := &errgroup.Group{}

	eg.Go(func() error {
		return tc.signalTasksCreated(ctx, tenantId, result.Tasks)
	})

	eg.Go(func() error {
		return tc.signalDAGsCreated(ctx, tenantId, result.Dags)
	})

	return eg.Wait()
}

// handleProcessUserEventMatches is responsible for signaling or creating tasks based on user event matches.
func (tc *TasksControllerImpl) handleProcessUserEventMatches(ctx context.Context, tenantId string, payloads []*tasktypes.UserEventTaskPayload) error {
	return tc.processUserEventMatches(ctx, tenantId, payloads)
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TasksControllerImpl) handleProcessInternalEvents(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[v1.InternalTaskEvent](payloads)

	return tc.processInternalEvents(ctx, tenantId, msgs)
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TasksControllerImpl) handleProcessTaskTrigger(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[v1.WorkflowNameTriggerOpts](payloads)
	tasks, dags, err := tc.repov1.Triggers().TriggerFromWorkflowNames(ctx, tenantId, msgs)

	if err != nil {
		return fmt.Errorf("could not trigger workflows from names: %w", err)
	}

	eg := &errgroup.Group{}

	eg.Go(func() error {
		return tc.signalTasksCreated(ctx, tenantId, tasks)
	})

	eg.Go(func() error {
		return tc.signalDAGsCreated(ctx, tenantId, dags)
	})

	return eg.Wait()
}

func (tc *TasksControllerImpl) sendInternalEvents(ctx context.Context, tenantId string, events []v1.InternalTaskEvent) error {
	if len(events) == 0 {
		return nil
	}

	msg, err := tasktypes.NewInternalEventMessage(tenantId, time.Now(), events...)

	if err != nil {
		return fmt.Errorf("could not create internal event message: %w", err)
	}

	return tc.mq.SendMessage(
		ctx,
		msgqueue.TASK_PROCESSING_QUEUE,
		msg,
	)
}

// processUserEventMatches looks for user event matches
func (tc *TasksControllerImpl) processUserEventMatches(ctx context.Context, tenantId string, events []*tasktypes.UserEventTaskPayload) error {
	candidateMatches := make([]v1.CandidateEventMatch, 0)

	for _, event := range events {
		candidateMatches = append(candidateMatches, v1.CandidateEventMatch{
			ID:             event.EventId,
			EventTimestamp: time.Now(),
			// NOTE: the event type of the V1TaskEvent is the event key for the match condition
			Key:  event.EventKey,
			Data: event.EventData,
		})
	}

	matchResult, err := tc.repov1.Matches().ProcessUserEventMatches(ctx, tenantId, candidateMatches)

	if err != nil {
		return fmt.Errorf("could not process user event matches: %w", err)
	}

	if len(matchResult.CreatedTasks) > 0 {
		err = tc.signalTasksCreated(ctx, tenantId, matchResult.CreatedTasks)

		if err != nil {
			return fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	return nil
}

func (tc *TasksControllerImpl) processInternalEvents(ctx context.Context, tenantId string, events []*v1.InternalTaskEvent) error {
	candidateMatches := make([]v1.CandidateEventMatch, 0)

	for _, event := range events {
		candidateMatches = append(candidateMatches, v1.CandidateEventMatch{
			ID:             uuid.NewString(),
			EventTimestamp: time.Now(),
			// NOTE: the event type of the V1TaskEvent is the event key for the match condition
			Key:          string(event.EventType),
			Data:         event.Data,
			ResourceHint: &event.TaskExternalID,
		})
	}

	matchResult, err := tc.repov1.Matches().ProcessInternalEventMatches(ctx, tenantId, candidateMatches)

	if err != nil {
		return fmt.Errorf("could not process internal event matches: %w", err)
	}

	if len(matchResult.CreatedTasks) > 0 {
		err = tc.signalTasksCreated(ctx, tenantId, matchResult.CreatedTasks)

		if err != nil {
			return fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	if len(matchResult.ReplayedTasks) > 0 {
		err = tc.signalTasksReplayedFromMatch(ctx, tenantId, matchResult.ReplayedTasks)

		if err != nil {
			return fmt.Errorf("could not signal replayed tasks: %w", err)
		}
	}

	return nil
}

func (tc *TasksControllerImpl) signalDAGsCreated(ctx context.Context, tenantId string, dags []*v1.DAGWithData) error {
	// notify that tasks have been created
	// TODO: make this transactionally safe?
	for _, dag := range dags {
		dagCp := dag
		msg, err := tasktypes.CreatedDAGMessage(tenantId, dagCp)

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
	}

	return nil
}

func (tc *TasksControllerImpl) signalTasksCreated(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	// group tasks by initial states
	queuedTasks := make([]*sqlcv1.V1Task, 0)
	failedTasks := make([]*sqlcv1.V1Task, 0)
	cancelledTasks := make([]*sqlcv1.V1Task, 0)
	skippedTasks := make([]*sqlcv1.V1Task, 0)

	for _, task := range tasks {
		switch task.InitialState {
		case sqlcv1.V1TaskInitialStateQUEUED:
			queuedTasks = append(queuedTasks, task)
		case sqlcv1.V1TaskInitialStateFAILED:
			failedTasks = append(failedTasks, task)
		case sqlcv1.V1TaskInitialStateCANCELLED:
			cancelledTasks = append(cancelledTasks, task)
		case sqlcv1.V1TaskInitialStateSKIPPED:
			skippedTasks = append(skippedTasks, task)
		}

		msg, err := tasktypes.CreatedTaskMessage(tenantId, task)

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
	}

	eg := &errgroup.Group{}

	if len(queuedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndQueued(ctx, tenantId, queuedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(failedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndFailed(ctx, tenantId, failedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(cancelledTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndCancelled(ctx, tenantId, cancelledTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(skippedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndSkipped(ctx, tenantId, skippedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (tc *TasksControllerImpl) signalTasksReplayedFromMatch(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	// group tasks by initial states
	queuedTasks := make([]*sqlcv1.V1Task, 0)
	failedTasks := make([]*sqlcv1.V1Task, 0)
	cancelledTasks := make([]*sqlcv1.V1Task, 0)
	skippedTasks := make([]*sqlcv1.V1Task, 0)

	for _, task := range tasks {
		switch task.InitialState {
		case sqlcv1.V1TaskInitialStateQUEUED:
			queuedTasks = append(queuedTasks, task)
		case sqlcv1.V1TaskInitialStateFAILED:
			failedTasks = append(failedTasks, task)
		case sqlcv1.V1TaskInitialStateCANCELLED:
			cancelledTasks = append(cancelledTasks, task)
		case sqlcv1.V1TaskInitialStateSKIPPED:
			skippedTasks = append(skippedTasks, task)
		}
	}

	eg := &errgroup.Group{}

	if len(queuedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndQueued(ctx, tenantId, queuedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(failedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndFailed(ctx, tenantId, failedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(cancelledTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndCancelled(ctx, tenantId, cancelledTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(skippedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndSkipped(ctx, tenantId, skippedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (tc *TasksControllerImpl) signalTasksUpdated(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	// group tasks by initial states
	queuedTasks := make([]*sqlcv1.V1Task, 0)
	failedTasks := make([]*sqlcv1.V1Task, 0)
	cancelledTasks := make([]*sqlcv1.V1Task, 0)
	skippedTasks := make([]*sqlcv1.V1Task, 0)

	for _, task := range tasks {
		switch task.InitialState {
		case sqlcv1.V1TaskInitialStateQUEUED:
			queuedTasks = append(queuedTasks, task)
		case sqlcv1.V1TaskInitialStateFAILED:
			failedTasks = append(failedTasks, task)
		case sqlcv1.V1TaskInitialStateCANCELLED:
			cancelledTasks = append(cancelledTasks, task)
		case sqlcv1.V1TaskInitialStateSKIPPED:
			skippedTasks = append(skippedTasks, task)
		}
	}

	eg := &errgroup.Group{}

	if len(queuedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndQueued(ctx, tenantId, queuedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(failedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndFailed(ctx, tenantId, failedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(cancelledTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndCancelled(ctx, tenantId, cancelledTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(skippedTasks) > 0 {
		eg.Go(func() error {
			err := tc.signalTasksCreatedAndSkipped(ctx, tenantId, skippedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (tc *TasksControllerImpl) signalTasksCreatedAndQueued(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	// get all unique queues and notify them
	queues := make(map[string]struct{})

	for _, task := range tasks {
		queues[task.Queue] = struct{}{}
	}

	tenant, err := tc.repo.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		return err
	}

	if tenant.SchedulerPartitionId.Valid {
		msg, err := tasktypes.NotifyTaskCreated(tenantId, tasks)

		if err != nil {
			tc.l.Err(err).Msg("could not create message for scheduler partition queue")
		} else {
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

	// notify that tasks have been created
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg := ""

		if len(task.ConcurrencyKeys) > 0 {
			msg = "concurrency keys evaluated as:"

			for _, key := range task.ConcurrencyKeys {
				msg += fmt.Sprintf(" %s", key)
			}
		}

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapQUEUED,
				EventTimestamp: time.Now(),
				EventMessage:   msg,
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

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
		}
	}()

	return nil
}

func (tc *TasksControllerImpl) signalTasksCreatedAndCancelled(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	internalEvents := make([]v1.InternalTaskEvent, 0)

	for _, task := range tasks {
		taskExternalId := sqlchelpers.UUIDToStr(task.ExternalID)

		dataBytes := v1.NewCancelledTaskOutputEventFromTask(task).Bytes()

		internalEvents = append(internalEvents, v1.InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: taskExternalId,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeCANCELLED,
			Data:           dataBytes,
		})
	}

	err := tc.sendInternalEvents(ctx, tenantId, internalEvents)

	if err != nil {
		return err
	}

	// notify that tasks have been cancelled
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapCANCELLED,
			EventTimestamp: time.Now(),
		})

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
	}

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
			prometheus.CancelledTasks.Inc()
		}
	}()

	return nil
}

func (tc *TasksControllerImpl) signalTasksCreatedAndFailed(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	internalEvents := make([]v1.InternalTaskEvent, 0)

	for _, task := range tasks {
		taskExternalId := sqlchelpers.UUIDToStr(task.ExternalID)

		dataBytes := v1.NewFailedTaskOutputEventFromTask(task).Bytes()

		internalEvents = append(internalEvents, v1.InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: taskExternalId,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeFAILED,
			Data:           dataBytes,
		})
	}

	err := tc.sendInternalEvents(ctx, tenantId, internalEvents)

	if err != nil {
		return err
	}

	// notify that tasks have been cancelled
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapFAILED,
			EventPayload:   task.InitialStateReason.String,
			EventTimestamp: time.Now(),
		})

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
	}

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
			prometheus.FailedTasks.Inc()
		}
	}()

	return nil
}

func (tc *TasksControllerImpl) signalTasksCreatedAndSkipped(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	internalEvents := make([]v1.InternalTaskEvent, 0)

	for _, task := range tasks {
		taskExternalId := sqlchelpers.UUIDToStr(task.ExternalID)

		dataBytes := v1.NewSkippedTaskOutputEventFromTask(task).Bytes()

		internalEvents = append(internalEvents, v1.InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: taskExternalId,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeCOMPLETED,
			Data:           dataBytes,
		})
	}

	err := tc.sendInternalEvents(ctx, tenantId, internalEvents)

	if err != nil {
		return err
	}

	// notify that tasks have been cancelled
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapSKIPPED,
			EventTimestamp: time.Now(),
		})

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
	}

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
			prometheus.SkippedTasks.Inc()
		}
	}()

	return nil
}

func (tc *TasksControllerImpl) signalTasksReplayed(ctx context.Context, tenantId string, tasks []v1.TaskIdInsertedAtRetryCount) error {
	// notify that tasks have been created
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg := "Task was replayed, resetting task result."

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.Id,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapRETRIEDBYUSER,
				EventTimestamp: time.Now(),
				EventMessage:   msg,
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

func (tc *TasksControllerImpl) pubRetryEvent(ctx context.Context, tenantId string, task v1.RetriedTask) error {
	taskId := task.Id

	retryMsg := fmt.Sprintf("This is retry number %d.", task.AppRetryCount)

	if task.RetryBackoffFactor.Valid && task.RetryMaxBackoff.Valid {
		maxBackoffSeconds := int(task.RetryMaxBackoff.Int32)
		backoffFactor := task.RetryBackoffFactor.Float64

		// compute the backoff duration
		durationMilliseconds := 1000 * min(float64(maxBackoffSeconds), math.Pow(backoffFactor, float64(task.AppRetryCount)))
		retryDur := time.Duration(int(durationMilliseconds)) * time.Millisecond
		retryTime := time.Now().Add(retryDur)

		retryMsg = fmt.Sprintf("%s Retrying in %s (%s).", retryMsg, retryDur.String(), retryTime.Format(time.RFC3339))
	}

	olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
		tenantId,
		tasktypes.CreateMonitoringEventPayload{
			TaskId:         taskId,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapRETRYING,
			EventTimestamp: time.Now(),
			EventMessage:   retryMsg,
		},
	)

	if err != nil {
		return fmt.Errorf("could not create monitoring event message: %w", err)
	}

	err = tc.pubBuffer.Pub(
		ctx,
		msgqueue.OLAP_QUEUE,
		olapMsg,
		false,
	)

	if err != nil {
		return fmt.Errorf("could not publish monitoring event message: %w", err)
	}

	if !task.RetryBackoffFactor.Valid {
		olapMsg, err = tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         taskId,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapQUEUED,
				EventTimestamp: time.Now(),
			},
		)

		if err != nil {
			return fmt.Errorf("could not create monitoring event message: %w", err)
		}

		err = tc.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
			false,
		)

		if err != nil {
			return fmt.Errorf("could not publish monitoring event message: %w", err)
		}
	}

	return nil
}
