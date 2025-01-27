package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	repov2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
	v2 "github.com/hatchet-dev/hatchet/pkg/scheduling/v2"
)

type SchedulerOpt func(*SchedulerOpts)

type SchedulerOpts struct {
	mq          msgqueue.MessageQueue
	l           *zerolog.Logger
	repo        repository.EngineRepository
	repov2      repov2.Repository
	dv          datautils.DataDecoderValidator
	alerter     hatcheterrors.Alerter
	p           *partition.Partition
	queueLogger *zerolog.Logger
	pool        *v2.SchedulingPool
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

func WithRepository(r repository.EngineRepository) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.repo = r
	}
}

func WithV2Repository(r repov2.Repository) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.repov2 = r
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

func WithSchedulerPool(s *v2.SchedulingPool) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.pool = s
	}
}

type Scheduler struct {
	mq     msgqueue.MessageQueue
	l      *zerolog.Logger
	repo   repository.EngineRepository
	repov2 repov2.Repository
	dv     datautils.DataDecoderValidator
	s      gocron.Scheduler
	a      *hatcheterrors.Wrapped
	p      *partition.Partition

	// a custom queue logger
	ql *zerolog.Logger

	pool *v2.SchedulingPool
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

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.repov2 == nil {
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

	q := &Scheduler{
		mq:     opts.mq,
		l:      opts.l,
		repo:   opts.repo,
		repov2: opts.repov2,
		dv:     opts.dv,
		s:      s,
		a:      a,
		p:      opts.p,
		ql:     opts.queueLogger,
		pool:   opts.pool,
	}

	return q, nil
}

func (s *Scheduler) Start() (func() error, error) {

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	_, err := s.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			s.runTenantSetQueues(ctx),
		),
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

				err = s.scheduleStepRuns(ctx, sqlchelpers.UUIDToStr(res.TenantId), res)

				if err != nil {
					s.l.Error().Err(err).Msg("could not schedule step runs")
				}
			}
		}
	}()

	cleanup := func() error {
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup job processing queue: %w", err)
		}

		if err := s.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

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

	if task.ID == "check-tenant-queue" {
		return s.handleCheckQueue(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (s *Scheduler) handleCheckQueue(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "handle-check-queue", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueue.JSONConvert[tasktypes.CheckTenantQueuePayload](msg.Payloads)

	for _, payload := range payloads {
		switch {
		case payload.IsStepQueued && payload.QueueName != "":
			s.pool.Queue(ctx, msg.TenantID, payload.QueueName)
		case payload.IsSlotReleased:
			if payload.QueueName != "" {
				s.pool.Queue(ctx, msg.TenantID, payload.QueueName)
			}

			s.pool.Replenish(ctx, msg.TenantID)
		default:
			s.pool.RefreshAll(ctx, msg.TenantID)
		}
	}

	return nil
}

func (s *Scheduler) runTenantSetQueues(ctx context.Context) func() {
	return func() {
		s.l.Debug().Msgf("partition: checking step run requeue")

		// list all tenants
		tenants, err := s.repo.Tenant().ListTenantsBySchedulerPartition(ctx, s.p.GetSchedulerPartitionId())

		if err != nil {
			s.l.Err(err).Msg("could not list tenants")
			return
		}

		s.pool.SetTenants(tenants)
	}
}

func (s *Scheduler) scheduleStepRuns(ctx context.Context, tenantId string, res *v2.QueueResults) error {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-runs")
	defer span.End()

	var err error

	// bulk assign step runs
	if len(res.Assigned) > 0 {
		dispatcherIdToWorkerIdsToStepRuns := make(map[string]map[string][]int64)

		workerIds := make([]string, 0)

		for _, assigned := range res.Assigned {
			workerIds = append(workerIds, sqlchelpers.UUIDToStr(assigned.WorkerId))
		}

		var dispatcherIdWorkerIds map[string][]string

		dispatcherIdWorkerIds, err = s.repo.Worker().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

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

		for _, bulkAssigned := range res.Assigned {
			dispatcherId, ok := workerIdToDispatcherId[sqlchelpers.UUIDToStr(bulkAssigned.WorkerId)]

			if !ok {
				s.l.Error().Msg("could not assign step run to worker: no dispatcher id. attempting internal retry.")

				s.internalRetry(ctx, tenantId, bulkAssigned)

				continue
			}

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId] = make(map[string][]int64)
			}

			workerId := sqlchelpers.UUIDToStr(bulkAssigned.WorkerId)

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = make([]int64, 0)
			}

			dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = append(dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId], bulkAssigned.QueueItem.TaskID)
		}

		// for each dispatcher, send a bulk assigned task
		for dispatcherId, workerIdsToStepRuns := range dispatcherIdToWorkerIdsToStepRuns {
			msg, err := taskBulkAssignedTask(tenantId, dispatcherId, workerIdsToStepRuns)

			if err != nil {
				err = multierror.Append(err, fmt.Errorf("could not create bulk assigned task: %w", err))
			}

			err = s.mq.SendMessage(
				ctx,
				msgqueue.QueueTypeFromDispatcherID(dispatcherId),
				msg,
			)

			if err != nil {
				err = multierror.Append(err, fmt.Errorf("could not send bulk assigned task: %w", err))
			}
		}
	}

	// TODO: FIXME
	// for _, toCancel := range res.SchedulingTimedOut {
	// 	innerErr := s.mq.SendMessage(
	// 		ctx,
	// 		msgqueue.JOB_PROCESSING_QUEUE,
	// 		getStepRunCancelTask(
	// 			tenantId,
	// 			toCancel,
	// 			"SCHEDULING_TIMED_OUT",
	// 		),
	// 	)

	// 	if innerErr != nil {
	// 		err = multierror.Append(err, fmt.Errorf("could not send cancel step run event: %w", innerErr))
	// 	}
	// }

	return err
}

func (s *Scheduler) internalRetry(ctx context.Context, tenantId string, assigned ...*repov2.AssignedItem) {
	opts := []repov2.FailTaskOpts{}

	for _, a := range assigned {
		opts = append(opts, repov2.FailTaskOpts{
			TaskIdRetryCount: &repov2.TaskIdRetryCount{
				Id:         a.QueueItem.TaskID,
				RetryCount: a.QueueItem.RetryCount,
			},
			IsAppError: false,
		})
	}

	err := s.repov2.Tasks().FailTasks(ctx, tenantId, opts)

	if err != nil {
		s.l.Error().Err(err).Msg("could not retry failed tasks")
	}
}

// func getStepRunCancelTask(tenantId, stepRunId, reason string) *msgqueue.Message {
// 	payload, _ := datautils.ToJSONMap(tasktypes.StepRunCancelTaskPayload{
// 		StepRunId:           stepRunId,
// 		CancelledReason:     reason,
// 		PropagateToChildren: true,
// 	})

// 	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunCancelTaskMetadata{
// 		TenantId: tenantId,
// 	})

// 	return &msgqueue.Message{
// 		ID:       "step-run-cancel",
// 		Payload:  payload,
// 		Metadata: metadata,
// 		Retries:  3,
// 	}
// }

func taskBulkAssignedTask(tenantId, dispatcherId string, workerIdsToTaskIds map[string][]int64) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-assigned-bulk",
		tasktypes.TaskAssignedBulkTaskPayload{
			WorkerIdToTaskIds: workerIdsToTaskIds,
		},
		false,
	)
}
