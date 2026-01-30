package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type SchedulerOpt func(*SchedulerOpts)

type SchedulerOpts struct {
	mq          msgqueue.MessageQueue
	l           *zerolog.Logger
	repov1      repov1.Repository
	dv          datautils.DataDecoderValidator
	alerter     hatcheterrors.Alerter
	p           *partition.Partition
	queueLogger *zerolog.Logger
	pool        *v1.SchedulingPool
}

func defaultSchedulerOpts() *SchedulerOpts {
	l := logger.NewDefaultLogger("scheduler")
	alerter := hatcheterrors.NoOpAlerter{}

	queueLogger := logger.NewDefaultLogger("queue")

	return &SchedulerOpts{
		l:           &l,
		dv:          datautils.NewDataDecoderValidator(),
		alerter:     alerter,
		queueLogger: &queueLogger,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.l = l
	}
}

func WithQueueLoggerConfig(lc *shared.LoggerConfigFile) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		l := logger.NewStdErr(lc, "queue")
		opts.queueLogger = &l
	}
}

func WithAlerter(a hatcheterrors.Alerter) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.alerter = a
	}
}

func WithRepository(r repov1.Repository) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.repov1 = r
	}
}

func WithPartition(p *partition.Partition) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.p = p
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.dv = dv
	}
}

func WithSchedulerPool(s *v1.SchedulingPool) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.pool = s
	}
}

type Scheduler struct {
	mq        msgqueue.MessageQueue
	pubBuffer *msgqueue.MQPubBuffer
	l         *zerolog.Logger
	repov1    repov1.Repository
	dv        datautils.DataDecoderValidator
	s         gocron.Scheduler
	a         *hatcheterrors.Wrapped
	p         *partition.Partition

	// a custom queue logger
	ql *zerolog.Logger

	pool *v1.SchedulingPool

	tasksWithNoWorkerCache *expirable.LRU[string, struct{}]
}

func New(
	fs ...SchedulerOpt,
) (*Scheduler, error) {
	opts := defaultSchedulerOpts()

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
		return nil, fmt.Errorf("partition is required. use WithPartition")
	}

	if opts.pool == nil {
		return nil, fmt.Errorf("scheduler is required. use WithSchedulerPool")
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "scheduler"})

	pubBuffer := msgqueue.NewMQPubBuffer(opts.mq)

	// TODO: replace with config or pull into a constant
	tasksWithNoWorkerCache := expirable.NewLRU(10000, func(string, struct{}) {}, 5*time.Minute)

	q := &Scheduler{
		mq:                     opts.mq,
		pubBuffer:              pubBuffer,
		l:                      opts.l,
		repov1:                 opts.repov1,
		dv:                     opts.dv,
		s:                      s,
		a:                      a,
		p:                      opts.p,
		ql:                     opts.queueLogger,
		pool:                   opts.pool,
		tasksWithNoWorkerCache: tasksWithNoWorkerCache,
	}

	return q, nil
}

func (s *Scheduler) Start() (func() error, error) {
	cleanupDLQ, err := s.mq.Subscribe(msgqueue.DISPATCHER_DEAD_LETTER_QUEUE, s.handleDeadLetteredMessages, msgqueue.NoOpHook)

	if err != nil {
		return nil, fmt.Errorf("could not start subscribe to dispatcher dead letter queue: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	_, err = s.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			s.runSetTenants(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule tenant set queues: %w", err)
	}

	s.s.Start()

	postAck := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := s.handleTask(context.Background(), task)
		if err != nil {
			s.l.Error().Err(err).Msg("could not handle job task")
			return s.a.WrapErr(fmt.Errorf("could not handle job task: %w", err), map[string]interface{}{"task_id": task.ID}) // nolint: errcheck
		}

		return nil
	}

	cleanupQueue, err := s.mq.Subscribe(
		msgqueue.QueueTypeFromPartitionIDAndController(s.p.GetSchedulerPartitionId(), msgqueue.Scheduler),
		msgqueue.NoOpHook, // the only handler is to check the queue, so we acknowledge immediately with the NoOpHook
		postAck,
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not subscribe to job processing queue: %w", err)
	}

	queueResults := s.pool.GetResultsCh()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-queueResults:
				s.l.Debug().Msgf("partition: received queue results")

				if res == nil {
					continue
				}

				go func(results *v1.QueueResults) {
					err := s.scheduleStepRuns(ctx, results.TenantId.String(), results)

					if err != nil {
						s.l.Error().Err(err).Msg("could not schedule step runs")
					}
				}(res)
			}
		}
	}()

	concurrencyResults := s.pool.GetConcurrencyResultsCh()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-concurrencyResults:
				s.l.Debug().Msgf("partition: received concurrency results")

				if res == nil {
					continue
				}

				go s.notifyAfterConcurrency(ctx, res.TenantId.String(), res)
			}
		}
	}()

	cleanup := func() error {
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup job processing queue: %w", err)
		}

		if err := cleanupDLQ(); err != nil {
			return fmt.Errorf("could not cleanup message queue buffer: %w", err)
		}

		if err := s.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

		s.pubBuffer.Stop()

		return nil
	}

	return cleanup, nil
}

func (s *Scheduler) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(s.l, s.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	if task.ID == msgqueue.MsgIDCheckTenantQueue {
		return s.handleCheckQueue(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (s *Scheduler) handleCheckQueue(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "handle-check-queue", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueue.JSONConvert[tasktypes.CheckTenantQueuesPayload](msg.Payloads)

	for _, payload := range payloads {
		if len(payload.StrategyIds) > 0 {
			s.pool.NotifyConcurrency(ctx, msg.TenantID, payload.StrategyIds)
		}

		if len(payload.QueueNames) > 0 {
			s.pool.NotifyQueues(ctx, msg.TenantID, payload.QueueNames)
		}

		if payload.SlotsReleased {
			s.pool.Replenish(ctx, msg.TenantID)
		}
	}

	return nil
}

func (s *Scheduler) runSetTenants(ctx context.Context) func() {
	return func() {
		s.l.Debug().Msgf("partition: checking step run requeue")

		// list all tenants
		tenants, err := s.repov1.Tenant().ListTenantsBySchedulerPartition(ctx, s.p.GetSchedulerPartitionId(), sqlcv1.TenantMajorEngineVersionV1)

		if err != nil {
			s.l.Err(err).Msg("could not list tenants")
			return
		}

		s.pool.SetTenants(tenants)
	}
}

func (s *Scheduler) scheduleStepRuns(ctx context.Context, tenantId string, res *v1.QueueResults) error {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-runs")
	defer span.End()

	var outerErr error

	// bulk assign step runs
	if len(res.Assigned) > 0 {
		dispatcherIdToWorkerIdsToStepRuns := make(map[string]map[string][]int64)

		workerIds := make([]string, 0)

		for _, assigned := range res.Assigned {
			workerIds = append(workerIds, assigned.WorkerId.String())
		}

		var dispatcherIdWorkerIds map[string][]string

		dispatcherIdWorkerIds, err := s.repov1.Workers().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

		if err != nil {
			s.internalRetry(ctx, tenantId, res.Assigned...)

			return fmt.Errorf("could not list dispatcher ids for workers: %w. attempting internal retry", err)
		}

		workerIdToDispatcherId := make(map[string]string)

		for dispatcherId, workerIds := range dispatcherIdWorkerIds {
			for _, workerId := range workerIds {
				workerIdToDispatcherId[workerId] = dispatcherId
			}
		}

		assignedMsgs := make([]*msgqueue.Message, 0)

		for _, bulkAssigned := range res.Assigned {
			dispatcherId, ok := workerIdToDispatcherId[bulkAssigned.WorkerId.String()]

			if !ok {
				s.l.Error().Msg("could not assign step run to worker: no dispatcher id. attempting internal retry.")

				s.internalRetry(ctx, tenantId, bulkAssigned)

				continue
			}

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId] = make(map[string][]int64)
			}

			workerId := bulkAssigned.WorkerId.String()

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = make([]int64, 0)
			}

			dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = append(dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId], bulkAssigned.QueueItem.TaskID)

			taskId := bulkAssigned.QueueItem.TaskID

			assignedMsg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:         taskId,
					RetryCount:     bulkAssigned.QueueItem.RetryCount,
					WorkerId:       &workerId,
					EventType:      sqlcv1.V1EventTypeOlapASSIGNED,
					EventTimestamp: time.Now(),
				},
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create monitoring event message: %w", err))
				continue
			}

			assignedMsgs = append(assignedMsgs, assignedMsg)
		}

		// for each dispatcher, send a bulk assigned task
		for dispatcherId, workerIdsToStepRuns := range dispatcherIdToWorkerIdsToStepRuns {
			msg, err := taskBulkAssignedTask(tenantId, workerIdsToStepRuns)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create bulk assigned task: %w", err))
				continue
			}

			err = s.mq.SendMessage(
				ctx,
				msgqueue.QueueTypeFromDispatcherID(dispatcherId),
				msg,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send bulk assigned task: %w", err))
			}
		}

		for _, assignedMsg := range assignedMsgs {
			err = s.pubBuffer.Pub(
				ctx,
				msgqueue.OLAP_QUEUE,
				assignedMsg,
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send monitoring event message: %w", err))
			}
		}
	}

	if len(res.RateLimited) > 0 {
		for _, rateLimited := range res.RateLimited {
			message := fmt.Sprintf(
				"Rate limit exceeded for key %s, attempting to consume %d units, but only had %d remaining",
				rateLimited.ExceededKey,
				rateLimited.ExceededUnits,
				rateLimited.ExceededVal,
			)

			msg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:         rateLimited.TaskId,
					RetryCount:     rateLimited.RetryCount,
					EventType:      sqlcv1.V1EventTypeOlapREQUEUEDRATELIMIT,
					EventTimestamp: time.Now(),
					EventMessage:   message,
				},
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create cancelled task: %w", err))
				continue
			}

			err = s.pubBuffer.Pub(
				ctx,
				msgqueue.OLAP_QUEUE,
				msg,
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send cancelled task: %w", err))
			}
		}
	}

	if len(res.SchedulingTimedOut) > 0 {
		for _, schedulingTimedOut := range res.SchedulingTimedOut {
			msg, err := tasktypes.CancelledTaskMessage(
				tenantId,
				schedulingTimedOut.TaskID,
				schedulingTimedOut.TaskInsertedAt,
				schedulingTimedOut.ExternalID.String(),
				schedulingTimedOut.WorkflowRunID.String(),
				schedulingTimedOut.RetryCount,
				sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT,
				"",
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create cancelled task: %w", err))
				continue
			}

			err = s.mq.SendMessage(
				ctx,
				msgqueue.TASK_PROCESSING_QUEUE,
				msg,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send cancelled task: %w", err))
			}
		}
	}

	if len(res.Unassigned) > 0 {
		for _, unassigned := range res.Unassigned {
			taskExternalId := unassigned.ExternalID.String()

			// if we have seen this task recently, don't send it again
			if _, ok := s.tasksWithNoWorkerCache.Get(taskExternalId); ok {
				s.l.Debug().Msgf("skipping unassigned task %s as it was recently unassigned", taskExternalId)
				continue
			}

			taskId := unassigned.TaskID

			msg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:         taskId,
					RetryCount:     unassigned.RetryCount,
					EventType:      sqlcv1.V1EventTypeOlapREQUEUEDNOWORKER,
					EventTimestamp: time.Now(),
				},
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create cancelled task: %w", err))
				continue
			}

			err = s.pubBuffer.Pub(
				ctx,
				msgqueue.OLAP_QUEUE,
				msg,
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send cancelled task: %w", err))
			}

			s.tasksWithNoWorkerCache.Add(taskExternalId, struct{}{})
		}
	}

	return outerErr
}

func (s *Scheduler) internalRetry(ctx context.Context, tenantId string, assigned ...*repov1.AssignedItem) {
	for _, a := range assigned {
		msg, err := tasktypes.FailedTaskMessage(
			tenantId,
			a.QueueItem.TaskID,
			a.QueueItem.TaskInsertedAt,
			a.QueueItem.ExternalID.String(),
			a.QueueItem.WorkflowRunID.String(),
			a.QueueItem.RetryCount,
			false,
			"could not assign step run to worker",
			false,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not create failed task")
			continue
		}

		err = s.mq.SendMessage(
			ctx,
			msgqueue.TASK_PROCESSING_QUEUE,
			msg,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not send failed task")
			continue
		}
	}
}

func (s *Scheduler) notifyAfterConcurrency(ctx context.Context, tenantId string, res *v1.ConcurrencyResults) {
	uniqueQueueNames := make(map[string]struct{}, 0)

	for _, task := range res.Queued {
		uniqueQueueNames[task.Queue] = struct{}{}
	}

	uniqueNextStrategies := make(map[int64]struct{}, len(res.NextConcurrencyStrategies))

	for _, id := range res.NextConcurrencyStrategies {
		uniqueNextStrategies[id] = struct{}{}
	}

	strategies := make([]int64, 0, len(uniqueNextStrategies))

	for stratId := range uniqueNextStrategies {
		strategies = append(strategies, stratId)
	}

	s.pool.NotifyConcurrency(ctx, tenantId, strategies)

	queues := make([]string, 0, len(uniqueQueueNames))

	for queueName := range uniqueQueueNames {
		queues = append(queues, queueName)
	}

	s.pool.NotifyQueues(ctx, tenantId, queues)

	// handle cancellations
	for _, cancelled := range res.Cancelled {
		eventType := sqlcv1.V1EventTypeOlapCANCELLED
		eventMessage := ""
		shouldNotify := true

		if cancelled.CancelledReason == "SCHEDULING_TIMED_OUT" {
			eventType = sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT
			shouldNotify = false
		} else {
			eventMessage = "Cancelled due to concurrency strategy"
		}

		msg, err := tasktypes.CancelledTaskMessage(
			tenantId,
			cancelled.TaskIdInsertedAtRetryCount.Id,
			cancelled.TaskIdInsertedAtRetryCount.InsertedAt,
			cancelled.TaskExternalId,
			cancelled.WorkflowRunId,
			cancelled.TaskIdInsertedAtRetryCount.RetryCount,
			eventType,
			eventMessage,
			shouldNotify,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not create cancelled task")
			continue
		}

		err = s.mq.SendMessage(
			ctx,
			msgqueue.TASK_PROCESSING_QUEUE,
			msg,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not send cancelled task")
			continue
		}
	}
}

func taskBulkAssignedTask(tenantId string, workerIdsToTaskIds map[string][]int64) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskAssignedBulk,
		false,
		true,
		tasktypes.TaskAssignedBulkTaskPayload{
			WorkerIdToTaskIds: workerIdsToTaskIds,
		},
	)
}

func (s *Scheduler) handleDeadLetteredMessages(msg *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(s.l, s.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	switch msg.ID {
	case msgqueue.MsgIDTaskAssignedBulk:
		err = s.handleDeadLetteredTaskBulkAssigned(ctx, msg)
	case msgqueue.MsgIDTaskCancelled:
		err = s.handleDeadLetteredTaskCancelled(ctx, msg)
	default:
		err = fmt.Errorf("unknown task: %s", msg.ID)
	}

	return err
}

func (s *Scheduler) handleDeadLetteredTaskBulkAssigned(ctx context.Context, msg *msgqueue.Message) error {
	msgs := msgqueue.JSONConvert[tasktypes.TaskAssignedBulkTaskPayload](msg.Payloads)

	taskIds := make([]int64, 0)

	for _, innerMsg := range msgs {
		for _, tasks := range innerMsg.WorkerIdToTaskIds {
			s.l.Error().Msgf("handling dead-lettered task assignments for tenant %s, tasks: %v. This indicates an abrupt shutdown of a dispatcher and should be investigated.", msg.TenantID, tasks)
			taskIds = append(taskIds, tasks...)
		}
	}

	toFail, err := s.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks for dead lettered bulk assigned message: %w", err)
	}

	for _, _task := range toFail {
		tenantId := msg.TenantID
		task := _task

		msg, err := tasktypes.FailedTaskMessage(
			tenantId,
			task.ID,
			task.InsertedAt,
			task.ExternalID.String(),
			task.WorkflowRunID.String(),
			task.RetryCount,
			false,
			"Could not send task to worker",
			false,
		)

		if err != nil {
			return fmt.Errorf("could not create failed task message: %w", err)
		}

		err = s.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

		if err != nil {
			// NOTE: failure to send on the MQ is likely not transient; ideally we could only retry individual
			// tasks but since this message has the tasks in a batch, we retry all of them instead. we're banking
			// on the downstream `task-failed` processing to be idempotent.
			return fmt.Errorf("could not send failed task message: %w", err)
		}
	}

	return nil
}

func (s *Scheduler) handleDeadLetteredTaskCancelled(ctx context.Context, msg *msgqueue.Message) error {
	payloads := msgqueue.JSONConvert[tasktypes.SignalTaskCancelledPayload](msg.Payloads)

	// try to resend the cancellation signal to the impacted worker.
	workerIds := make([]string, 0)

	for _, p := range payloads {
		s.l.Error().Msgf("handling dead-lettered task cancellations for tenant %s, task %d. This indicates an abrupt shutdown of a dispatcher and should be investigated.", msg.TenantID, p.TaskId)
		workerIds = append(workerIds, p.WorkerId)
	}

	// since the dispatcher IDs may have changed since the previous send, we need to query them again
	dispatcherIdWorkerIds, err := s.repov1.Workers().GetDispatcherIdsForWorkers(ctx, msg.TenantID, workerIds)

	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for workers: %w", err)
	}

	workerIdToDispatcherId := make(map[string]string)

	for dispatcherId, workerIds := range dispatcherIdWorkerIds {
		for _, workerId := range workerIds {
			workerIdToDispatcherId[workerId] = dispatcherId
		}
	}

	dispatcherIdsToPayloads := make(map[string][]tasktypes.SignalTaskCancelledPayload)

	for _, p := range payloads {
		// if we no longer have the worker attached to a dispatcher, discard the message
		if _, ok := workerIdToDispatcherId[p.WorkerId]; !ok {
			continue
		}

		pcp := *p
		dispatcherId := workerIdToDispatcherId[pcp.WorkerId]

		dispatcherIdsToPayloads[dispatcherId] = append(dispatcherIdsToPayloads[dispatcherId], pcp)
	}

	for dispatcherId, payloads := range dispatcherIdsToPayloads {
		msg, err := msgqueue.NewTenantMessage(
			msg.TenantID,
			msgqueue.MsgIDTaskCancelled,
			false,
			true,
			payloads...,
		)

		if err != nil {
			return fmt.Errorf("could not create message for task cancellation: %w", err)
		}

		err = s.mq.SendMessage(
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
