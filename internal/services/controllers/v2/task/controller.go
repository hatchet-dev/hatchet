package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
)

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
	repov2                 v2.TaskRepository
	dv                     datautils.DataDecoderValidator
	s                      gocron.Scheduler
	a                      *hatcheterrors.Wrapped
	p                      *partition.Partition
	celParser              *cel.CELParser
	timeoutTaskOperations  *queueutils.OperationPool
	reassignTaskOperations *queueutils.OperationPool
}

type TasksControllerOpt func(*TasksControllerOpts)

type TasksControllerOpts struct {
	mq             msgqueue.MessageQueue
	l              *zerolog.Logger
	repo           repository.EngineRepository
	repov2         v2.TaskRepository
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

func WithV2Repository(r v2.TaskRepository) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.repov2 = r
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

	if opts.repov2 == nil {
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
		repov2:         opts.repov2,
		dv:             opts.dv,
		s:              s,
		a:              a,
		p:              opts.p,
		celParser:      cel.NewCELParser(),
	}

	t.timeoutTaskOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "timeout step runs", t.processTaskTimeouts)
	t.reassignTaskOperations = queueutils.NewOperationPool(opts.l, time.Second*5, "reassign step runs", t.processTaskReassignments)

	return t, nil
}

func (tc *TasksControllerImpl) Start() (func() error, error) {
	mqBuffer := msgqueue.NewMQSubBuffer(msgqueue.TASK_PROCESSING_QUEUE, tc.mq, tc.handleBufferedMsgs)
	wg := sync.WaitGroup{}

	tc.s.Start()

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	// always create table partition on startup
	if err := tc.createTablePartition(context.Background()); err != nil {
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
			tc.runTenantReassignTasks(ctx),
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

		if err := tc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

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
	case "replay-task":
		// return ec.handleStepRunReplay(ctx, task)
	case "task-cancelled":
		return tc.handleTaskCancelled(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

func (tc *TasksControllerImpl) handleTaskCompleted(ctx context.Context, tenantId string, payloads [][]byte) error {
	opts := make([]v2.TaskIdRetryCount, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CompletedTaskPayload](payloads)

	for _, msg := range msgs {
		opts = append(opts, v2.TaskIdRetryCount{
			Id:         msg.TaskId,
			RetryCount: msg.RetryCount,
		})
	}

	queues, err := tc.repov2.CompleteTasks(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, queues)

	return nil
}

func (tc *TasksControllerImpl) handleTaskFailed(ctx context.Context, tenantId string, payloads [][]byte) error {
	opts := make([]v2.FailTaskOpts, 0)

	msgs := msgqueue.JSONConvert[tasktypes.FailedTaskPayload](payloads)

	for _, msg := range msgs {
		opts = append(opts, v2.FailTaskOpts{
			TaskIdRetryCount: &v2.TaskIdRetryCount{
				Id:         msg.TaskId,
				RetryCount: msg.RetryCount,
			},
			IsAppError: msg.IsAppError,
		})
	}

	retriedTasks, queues, err := tc.repov2.FailTasks(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, queues)

	var outerErr error

	// send retried tasks to the olap repository
	for _, task := range retriedTasks {
		taskId := task.Id

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         &taskId,
				RetryCount:     task.RetryCount,
				EventType:      olap.EVENT_TYPE_QUEUED,
				EventTimestamp: time.Now(),
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

	return outerErr
}

func (tc *TasksControllerImpl) handleTaskCancelled(ctx context.Context, tenantId string, payloads [][]byte) error {
	opts := make([]v2.TaskIdRetryCount, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CancelledTaskPayload](payloads)

	for _, msg := range msgs {
		opts = append(opts, v2.TaskIdRetryCount{
			Id:         msg.TaskId,
			RetryCount: msg.RetryCount,
		})
	}

	queues, err := tc.repov2.CancelTasks(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	tc.notifyQueuesOnCompletion(ctx, tenantId, queues)

	var outerErr error

	for _, msg := range msgs {
		taskId := msg.TaskId

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         &taskId,
				RetryCount:     msg.RetryCount,
				EventType:      msg.EventType,
				EventTimestamp: time.Now(),
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

	return err
}

func (tc *TasksControllerImpl) notifyQueuesOnCompletion(ctx context.Context, tenantId string, queues []string) {
	tenant, err := tc.repo.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		tc.l.Err(err).Msg("could not get tenant")
		return
	}

	uniqueQueues := make(map[string]struct{})

	for _, queue := range queues {
		uniqueQueues[queue] = struct{}{}
	}

	for queue := range uniqueQueues {
		if tenant.SchedulerPartitionId.Valid {
			msg, err := tasktypes.CheckTenantQueueToTask(tenantId, queue, false, true)

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
}
