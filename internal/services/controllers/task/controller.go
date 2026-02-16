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
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/operation"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/olap/signal"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const BULK_MSG_BATCH_SIZE = 50

type TasksController interface {
	Start(ctx context.Context) error
}

type TasksControllerImpl struct {
	mq                                    msgqueue.MessageQueue
	pubBuffer                             *msgqueue.MQPubBuffer
	l                                     *zerolog.Logger
	queueLogger                           *zerolog.Logger
	pgxStatsLogger                        *zerolog.Logger
	repov1                                v1.Repository
	dv                                    datautils.DataDecoderValidator
	s                                     gocron.Scheduler
	a                                     *hatcheterrors.Wrapped
	p                                     *partition.Partition
	celParser                             *cel.CELParser
	opsPoolPollInterval                   time.Duration
	opsPoolJitter                         time.Duration
	timeoutTaskOperations                 *operation.TenantOperationPool
	reassignTaskOperations                *operation.TenantOperationPool
	retryTaskOperations                   *operation.TenantOperationPool
	emitSleepOperations                   *operation.TenantOperationPool
	evictExpiredIdempotencyKeysOperations *operation.TenantOperationPool
	replayEnabled                         bool
	analyzeCronInterval                   time.Duration
	signaler                              *signal.OLAPSignaler
	tw                                    *trigger.TriggerWriter
}

type TasksControllerOpt func(*TasksControllerOpts)

type TasksControllerOpts struct {
	mq                  msgqueue.MessageQueue
	l                   *zerolog.Logger
	repov1              v1.Repository
	dv                  datautils.DataDecoderValidator
	alerter             hatcheterrors.Alerter
	p                   *partition.Partition
	queueLogger         *zerolog.Logger
	pgxStatsLogger      *zerolog.Logger
	opsPoolJitter       time.Duration
	opsPoolPollInterval time.Duration
	replayEnabled       bool
	analyzeCronInterval time.Duration
}

func defaultTasksControllerOpts() *TasksControllerOpts {
	l := logger.NewDefaultLogger("tasks-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	queueLogger := logger.NewDefaultLogger("queue")
	pgxStatsLogger := logger.NewDefaultLogger("pgx-stats")

	return &TasksControllerOpts{
		l:                   &l,
		dv:                  datautils.NewDataDecoderValidator(),
		alerter:             alerter,
		queueLogger:         &queueLogger,
		pgxStatsLogger:      &pgxStatsLogger,
		opsPoolJitter:       1500 * time.Millisecond,
		opsPoolPollInterval: 2 * time.Second,
		replayEnabled:       true, // default to enabled for backward compatibility
		analyzeCronInterval: 3 * time.Hour,
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

func WithOpsPoolJitter(cf server.ConfigFileOperations) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.opsPoolJitter = time.Duration(cf.Jitter) * time.Millisecond
		opts.opsPoolPollInterval = time.Duration(cf.PollInterval) * time.Second
	}
}

func WithReplayEnabled(enabled bool) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.replayEnabled = enabled
	}
}

func WithAnalyzeCronInterval(interval time.Duration) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.analyzeCronInterval = interval
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

	signaler := signal.NewOLAPSignaler(opts.mq, opts.repov1, opts.l, pubBuffer)
	tw := trigger.NewTriggerWriter(opts.mq, opts.repov1, opts.l, pubBuffer, 0)

	t := &TasksControllerImpl{
		mq:                  opts.mq,
		pubBuffer:           pubBuffer,
		l:                   opts.l,
		queueLogger:         opts.queueLogger,
		pgxStatsLogger:      opts.pgxStatsLogger,
		repov1:              opts.repov1,
		dv:                  opts.dv,
		s:                   s,
		a:                   a,
		p:                   opts.p,
		celParser:           cel.NewCELParser(),
		opsPoolJitter:       opts.opsPoolJitter,
		opsPoolPollInterval: opts.opsPoolPollInterval,
		replayEnabled:       opts.replayEnabled,
		analyzeCronInterval: opts.analyzeCronInterval,
		signaler:            signaler,
		tw:                  tw,
	}

	jitter := t.opsPoolJitter
	timeout := time.Second * 30

	t.timeoutTaskOperations = operation.NewTenantOperationPool(opts.p, opts.l, "timeout-step-runs", timeout, "timeout step runs", t.processTaskTimeouts, operation.WithPoolInterval(
		opts.repov1.IntervalSettings(),
		jitter,
		1*time.Second,
		30*time.Second,
		3,
		opts.repov1.Tasks().DefaultTaskActivityGauge,
	))

	t.emitSleepOperations = operation.NewTenantOperationPool(opts.p, opts.l, "emit-sleep-step-runs", timeout, "emit sleep step runs", t.processSleeps, operation.WithPoolInterval(
		opts.repov1.IntervalSettings(),
		jitter,
		1*time.Second,
		30*time.Second,
		3,
		opts.repov1.Tasks().DefaultTaskActivityGauge,
	))

	t.reassignTaskOperations = operation.NewTenantOperationPool(opts.p, opts.l, "reassign-step-runs", timeout, "reassign step runs", t.processTaskReassignments, operation.WithPoolInterval(
		opts.repov1.IntervalSettings(),
		jitter,
		1*time.Second,
		30*time.Second,
		3,
		opts.repov1.Tasks().DefaultTaskActivityGauge,
	))

	t.retryTaskOperations = operation.NewTenantOperationPool(opts.p, opts.l, "retry-step-runs", timeout, "retry step runs", t.processTaskRetryQueueItems, operation.WithPoolInterval(
		opts.repov1.IntervalSettings(),
		jitter,
		1*time.Second,
		30*time.Second,
		3,
		opts.repov1.Tasks().DefaultTaskActivityGauge,
	))

	t.evictExpiredIdempotencyKeysOperations = operation.NewTenantOperationPool(opts.p, opts.l, "evict-expired-idempotency-keys", timeout, "evict expired idempotency keys", t.evictExpiredIdempotencyKeys, operation.WithPoolInterval(
		opts.repov1.IntervalSettings(),
		jitter,
		1*time.Second,
		30*time.Second,
		3,
		opts.repov1.Tasks().DefaultTaskActivityGauge,
	))

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

	spanContext, span := telemetry.NewSpan(ctx, "TasksControllerImpl.Start")

	_, err = tc.s.NewJob(
		gocron.DurationJob(time.Minute*15),
		gocron.NewTask(
			tc.runTaskTablePartition(spanContext),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		wrappedErr := fmt.Errorf("could not schedule task partition method: %w", err)

		cancel()
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not schedule task partition method")
		span.End()

		return nil, wrappedErr
	}

	_, err = tc.s.NewJob(
		gocron.DurationJob(tc.repov1.Payloads().ExternalCutoverProcessInterval()),
		gocron.NewTask(
			tc.processPayloadExternalCutovers(spanContext),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		wrappedErr := fmt.Errorf("could not schedule process payload external cutovers: %w", err)

		cancel()
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not run process payload external cutovers")
		span.End()

		return nil, wrappedErr
	}

	_, err = tc.s.NewJob(
		gocron.DurationJob(tc.analyzeCronInterval),
		gocron.NewTask(
			tc.runAnalyze(spanContext),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		wrappedErr := fmt.Errorf("could not run analyze: %w", err)

		cancel()
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not run analyze")
		span.End()

		return nil, wrappedErr
	}

	_, err = tc.s.NewJob(
		gocron.DurationJob(1*time.Minute),
		gocron.NewTask(
			tc.runCleanup(spanContext),
		),
	)

	if err != nil {
		wrappedErr := fmt.Errorf("could not run cleanup: %w", err)

		cancel()
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not run cleanup")
		span.End()

		return nil, wrappedErr
	}

	cleanup := func() error {
		cancel()

		if err := cleanupBuffer(); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not cleanup buffer")
			return err
		}

		tc.timeoutTaskOperations.Cleanup()
		tc.reassignTaskOperations.Cleanup()
		tc.retryTaskOperations.Cleanup()
		tc.emitSleepOperations.Cleanup()
		tc.evictExpiredIdempotencyKeysOperations.Cleanup()

		tc.pubBuffer.Stop()

		if err := tc.s.Shutdown(); err != nil {
			err := fmt.Errorf("could not shutdown scheduler: %w", err)

			span.RecordError(err)
			span.SetStatus(codes.Error, "could not shutdown scheduler")
			return err
		}

		span.End()

		return nil
	}

	return cleanup, nil
}

func (tc *TasksControllerImpl) handleBufferedMsgs(tenantId uuid.UUID, msgId string, payloads [][]byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(tc.l, tc.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch msgId {
	case msgqueue.MsgIDTaskCompleted:
		return tc.handleTaskCompleted(context.Background(), tenantId, payloads)
	case msgqueue.MsgIDTaskFailed:
		return tc.handleTaskFailed(context.Background(), tenantId, payloads)
	case msgqueue.MsgIDTaskCancelled:
		return tc.handleTaskCancelled(context.Background(), tenantId, payloads)
	case msgqueue.MsgIDCancelTasks:
		return tc.handleCancelTasks(context.Background(), tenantId, payloads)
	case msgqueue.MsgIDReplayTasks:
		return tc.handleReplayTasks(context.Background(), tenantId, payloads)
	case msgqueue.MsgIDUserEvent:
		return tc.handleProcessUserEvents(context.Background(), tenantId, payloads)
	case msgqueue.MsgIDInternalEvent:
		return tc.handleProcessInternalEvents(context.Background(), tenantId, payloads)
	case msgqueue.MsgIDTaskTrigger:
		return tc.handleProcessTaskTrigger(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

func (tc *TasksControllerImpl) handleTaskCompleted(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.handleTaskCompleted")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

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
		err = fmt.Errorf("could not complete tasks: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not complete tasks")
		return err
	}

	// instrumentation
	for range res.ReleasedTasks {
		prometheus.SucceededTasks.Inc()
		prometheus.TenantSucceededTasks.WithLabelValues(tenantId.String()).Inc()
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, res.ReleasedTasks)

	return tc.signaler.SendInternalEvents(ctx, tenantId, res.InternalEvents)
}

func (tc *TasksControllerImpl) handleTaskFailed(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.handleTaskFailed")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

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
			err = fmt.Errorf("could not create monitoring event message: %w", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not create monitoring event message")
			continue
		}

		err = tc.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, olapMsg, false)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not publish monitoring event message")
			err = fmt.Errorf("could not publish monitoring event message: %w", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not publish monitoring event message")
			continue
		}
	}

	res, err := tc.repov1.Tasks().FailTasks(ctx, tenantId, opts)

	if err != nil {
		err = fmt.Errorf("could not fail tasks: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not fail tasks")
		return err
	}

	err = tc.processFailTasksResponse(ctx, tenantId, res)

	if err != nil {
		err = fmt.Errorf("could not process fail tasks response: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not process fail tasks response")
		return err
	}

	return nil
}

func (tc *TasksControllerImpl) processFailTasksResponse(ctx context.Context, tenantId uuid.UUID, res *v1.FailTasksResponse) error {
	ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.processFailTasksResponse")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	retriedTaskIds := make(map[int64]struct{})

	for _, task := range res.RetriedTasks {
		retriedTaskIds[task.Id] = struct{}{}
	}

	internalEventsWithoutRetries := make([]v1.InternalTaskEvent, 0)

	for _, e := range res.InternalEvents {
		// if the task is retried, don't send a message to the trigger queue
		if _, ok := retriedTaskIds[e.TaskID]; ok {
			prometheus.RetriedTasks.Inc()
			prometheus.TenantRetriedTasks.WithLabelValues(tenantId.String()).Inc()
			continue
		}

		internalEventsWithoutRetries = append(internalEventsWithoutRetries, e)
		prometheus.FailedTasks.Inc()
		prometheus.TenantFailedTasks.WithLabelValues(tenantId.String()).Inc()
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, res.ReleasedTasks)

	// TODO: MOVE THIS TO THE DATA LAYER?
	err := tc.signaler.SendInternalEvents(ctx, tenantId, internalEventsWithoutRetries)

	if err != nil {
		err = fmt.Errorf("could not send internal events: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not send internal events")
		return err
	}

	var outerErr error

	// send retried tasks to the olap repository
	for _, task := range res.RetriedTasks {
		if task.IsAppError {
			err = tc.pubRetryEvent(ctx, tenantId, task)

			if err != nil {
				err = fmt.Errorf("could not publish retry event: %w", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, "could not publish retry event")
				outerErr = multierror.Append(outerErr, err)
			}
		}
	}

	return outerErr
}

func (tc *TasksControllerImpl) handleTaskCancelled(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.handleTaskCancelled")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

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
		err = fmt.Errorf("could not cancel tasks: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not cancel tasks")
		return err
	}

	tasksToSendToDispatcher := make([]tasktypes.SignalTaskCancelledPayload, 0)

	for _, task := range res.ReleasedTasks {
		if shouldTasksNotify[task.ID] {
			tasksToSendToDispatcher = append(tasksToSendToDispatcher, tasktypes.SignalTaskCancelledPayload{
				TaskId:     task.ID,
				WorkerId:   task.WorkerID,
				RetryCount: task.RetryCount,
			})
		}
	}

	// send task cancellations to the dispatcher
	err = tc.sendTaskCancellationsToDispatcher(ctx, tenantId, tasksToSendToDispatcher)

	if err != nil {
		err = fmt.Errorf("could not send task cancellations to dispatcher: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not send task cancellations to dispatcher")
		return err
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, res.ReleasedTasks)

	// TODO: MOVE THIS TO THE DATA LAYER?
	err = tc.signaler.SendInternalEvents(ctx, tenantId, res.InternalEvents)

	if err != nil {
		err = fmt.Errorf("could not send internal events: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not send internal events")
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
			tc.l.Error().Err(err).Msg("could not create monitoring event message")
			err = fmt.Errorf("could not create monitoring event message: %w", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not create monitoring event message")
			outerErr = multierror.Append(outerErr, err)
			continue
		}

		err = tc.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
			false,
		)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not publish monitoring event message")
			err = fmt.Errorf("could not publish monitoring event message: %w", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not publish monitoring event message")
			outerErr = multierror.Append(outerErr, err)
		}
	}

	// instrumentation
	for range res.ReleasedTasks {
		prometheus.CancelledTasks.Inc()
		prometheus.TenantCancelledTasks.WithLabelValues(tenantId.String()).Inc()
	}

	return err
}

func (tc *TasksControllerImpl) handleCancelTasks(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
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
			msgqueue.MsgIDTaskCancelled,
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

func (tc *TasksControllerImpl) handleReplayTasks(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	if !tc.replayEnabled {
		tc.l.Debug().Msg("replay is disabled, skipping handleReplayTasks")
		return nil
	}

	// sure would be nice if we could use our own durable execution primitives here, but that's a bootstrapping
	// problem that we don't have a clean way to solve (yet)
	msgs := msgqueue.JSONConvert[tasktypes.ReplayTasksPayload](payloads)

	taskIdRetryCounts := make([]tasktypes.TaskIdInsertedAtRetryCountWithExternalId, 0)

	for _, msg := range msgs {
		opts := make([]v1.TaskIdInsertedAtRetryCount, len(msg.Tasks))

		for i, task := range msg.Tasks {
			opts[i] = v1.TaskIdInsertedAtRetryCount{
				Id:         task.Id,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			}
		}

		validTasks, err := tc.repov1.Tasks().FilterValidTasks(ctx, tenantId, opts)
		if err != nil {
			return fmt.Errorf("failed to filter valid tasks for replay: %w", err)
		}

		for _, task := range msg.Tasks {
			if _, ok := validTasks[task.Id]; !ok {
				continue
			}

			taskIdRetryCounts = append(taskIdRetryCounts, tasktypes.TaskIdInsertedAtRetryCountWithExternalId{
				TaskIdInsertedAtRetryCount: v1.TaskIdInsertedAtRetryCount{
					Id:         task.Id,
					InsertedAt: task.InsertedAt,
					RetryCount: task.RetryCount,
				},
				WorkflowRunExternalId: task.WorkflowRunExternalId,
			})
		}
	}

	workflowRunIdToTasks := make(map[string][]v1.TaskIdInsertedAtRetryCount)
	for _, task := range taskIdRetryCounts {
		if task.WorkflowRunExternalId == uuid.Nil {
			// Use a random uuid to effectively send tasks one at a time
			randomUuid := uuid.NewString()
			workflowRunIdToTasks[randomUuid] = append(workflowRunIdToTasks[randomUuid], task.TaskIdInsertedAtRetryCount)
		} else {
			workflowRunIdToTasks[task.WorkflowRunExternalId.String()] = append(workflowRunIdToTasks[task.WorkflowRunExternalId.String()], task.TaskIdInsertedAtRetryCount)
		}
	}

	eg := &errgroup.Group{}

	for _, tasks := range workflowRunIdToTasks {
		replayRes, err := tc.repov1.Tasks().ReplayTasks(ctx, tenantId, tasks)

		if err != nil {
			return fmt.Errorf("failed to replay task: %w", err)
		}

		if len(replayRes.ReplayedTasks) > 0 {
			eg.Go(func() error {
				err := tc.signaler.SignalTasksReplayed(ctx, tenantId, replayRes.ReplayedTasks)

				if err != nil {
					return fmt.Errorf("could not signal replayed tasks: %w", err)
				}

				return nil
			})
		}

		if len(replayRes.UpsertedTasks) > 0 {
			eg.Go(func() error {
				err := tc.signaler.SignalTasksUpdated(ctx, tenantId, replayRes.UpsertedTasks)

				if err != nil {
					return fmt.Errorf("could not signal queued tasks: %w", err)
				}

				return nil
			})
		}

		if len(replayRes.InternalEventResults.CreatedTasks) > 0 {
			eg.Go(func() error {
				err := tc.signaler.SignalTasksCreated(ctx, tenantId, replayRes.InternalEventResults.CreatedTasks)

				if err != nil {
					return fmt.Errorf("could not signal created tasks: %w", err)
				}

				return nil
			})
		}
	}

	return eg.Wait()
}

func (tc *TasksControllerImpl) sendTaskCancellationsToDispatcher(ctx context.Context, tenantId uuid.UUID, releasedTasks []tasktypes.SignalTaskCancelledPayload) error {
	workerIds := make([]uuid.UUID, 0)

	for _, task := range releasedTasks {
		workerIds = append(workerIds, task.WorkerId)
	}

	workerIdToDispatcherId, _, err := tc.repov1.Workers().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for workers: %w", err)
	}

	// assemble messages
	dispatcherIdsToPayloads := make(map[uuid.UUID][]tasktypes.SignalTaskCancelledPayload)

	for _, task := range releasedTasks {
		workerId := task.WorkerId
		dispatcherId := workerIdToDispatcherId[workerId]

		dispatcherIdsToPayloads[dispatcherId] = append(dispatcherIdsToPayloads[dispatcherId], task)
	}

	// send messages
	for dispatcherId, payloads := range dispatcherIdsToPayloads {
		msg, err := msgqueue.NewTenantMessage(
			tenantId,
			msgqueue.MsgIDTaskCancelled,
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

func (tc *TasksControllerImpl) notifyQueuesOnCompletion(ctx context.Context, tenantId uuid.UUID, releasedTasks []*sqlcv1.ReleaseTasksRow) {
	if len(releasedTasks) == 0 {
		return
	}

	tenant, err := tc.repov1.Tenant().GetTenantByID(ctx, tenantId)

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
			WorkflowRunId: releasedTask.WorkflowRunID,
		})
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDWorkflowRunFinishedCandidate,
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
func (tc *TasksControllerImpl) handleProcessUserEvents(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.handleProcessUserEvents")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

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
func (tc *TasksControllerImpl) handleProcessUserEventTrigger(ctx context.Context, tenantId uuid.UUID, msgs []*tasktypes.UserEventTaskPayload) error {
	opts := make([]v1.EventTriggerOpts, 0, len(msgs))
	eventIdToOpts := make(map[uuid.UUID]v1.EventTriggerOpts)

	for _, msg := range msgs {
		if msg.WasProcessedLocally {
			continue
		}

		opt := v1.EventTriggerOpts{
			ExternalId:            msg.EventExternalId,
			Key:                   msg.EventKey,
			Data:                  msg.EventData,
			AdditionalMetadata:    msg.EventAdditionalMetadata,
			Priority:              msg.EventPriority,
			Scope:                 msg.EventScope,
			TriggeringWebhookName: msg.TriggeringWebhookName,
		}

		opts = append(opts, opt)

		eventIdToOpts[msg.EventExternalId] = opt
	}

	return tc.tw.TriggerFromEvents(ctx, tenantId, eventIdToOpts)
}

// handleProcessUserEventMatches is responsible for signaling or creating tasks based on user event matches.
func (tc *TasksControllerImpl) handleProcessUserEventMatches(ctx context.Context, tenantId uuid.UUID, payloads []*tasktypes.UserEventTaskPayload) error {
	return tc.processUserEventMatches(ctx, tenantId, payloads)
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TasksControllerImpl) handleProcessInternalEvents(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.handleProcessInternalEvents")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	msgs := msgqueue.JSONConvert[v1.InternalTaskEvent](payloads)

	return tc.processInternalEvents(ctx, tenantId, msgs)
}

// handleProcessEventTrigger is responsible for inserting tasks into the database based on event triggers.
func (tc *TasksControllerImpl) handleProcessTaskTrigger(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	return tc.tw.TriggerFromWorkflowNames(ctx, tenantId, msgqueue.JSONConvert[v1.WorkflowNameTriggerOpts](payloads))
}

// processUserEventMatches looks for user event matches
func (tc *TasksControllerImpl) processUserEventMatches(ctx context.Context, tenantId uuid.UUID, events []*tasktypes.UserEventTaskPayload) error {
	candidateMatches := make([]v1.CandidateEventMatch, 0)

	for _, event := range events {
		candidateMatches = append(candidateMatches, v1.CandidateEventMatch{
			ID:             event.EventExternalId,
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
		err = tc.signaler.SignalTasksCreated(ctx, tenantId, matchResult.CreatedTasks)

		if err != nil {
			return fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	if len(matchResult.SatisfiedCallbacks) > 0 {
		if err := tc.processSatisfiedCallbacks(ctx, tenantId, matchResult.SatisfiedCallbacks); err != nil {
			tc.l.Error().Err(err).Msg("could not process satisfied callbacks")
		}
	}

	return nil
}

func (tc *TasksControllerImpl) processInternalEvents(ctx context.Context, tenantId uuid.UUID, events []*v1.InternalTaskEvent) error {
	candidateMatches := make([]v1.CandidateEventMatch, 0)

	for _, event := range events {
		resourceHint := event.TaskExternalID.String()
		candidateMatches = append(candidateMatches, v1.CandidateEventMatch{
			ID:             uuid.New(),
			EventTimestamp: time.Now(),
			// NOTE: the event type of the V1TaskEvent is the event key for the match condition
			Key:          string(event.EventType),
			Data:         event.Data,
			ResourceHint: &resourceHint,
		})
	}

	matchResult, err := tc.repov1.Matches().ProcessInternalEventMatches(ctx, tenantId, candidateMatches)

	if err != nil {
		return fmt.Errorf("could not process internal event matches: %w", err)
	}

	if len(matchResult.CreatedTasks) > 0 {
		err = tc.signaler.SignalTasksCreated(ctx, tenantId, matchResult.CreatedTasks)

		if err != nil {
			return fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	if len(matchResult.ReplayedTasks) > 0 {
		err = tc.signaler.SignalTasksReplayedFromMatch(ctx, tenantId, matchResult.ReplayedTasks)

		if err != nil {
			return fmt.Errorf("could not signal replayed tasks: %w", err)
		}
	}

	if len(matchResult.SatisfiedCallbacks) > 0 {
		if err := tc.processSatisfiedCallbacks(ctx, tenantId, matchResult.SatisfiedCallbacks); err != nil {
			tc.l.Error().Err(err).Msg("could not process satisfied callbacks")
		}
	}

	return nil
}

func (tc *TasksControllerImpl) pubRetryEvent(ctx context.Context, tenantId uuid.UUID, task v1.RetriedTask) error {
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
