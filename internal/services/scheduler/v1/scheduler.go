package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

var (
	eventTypeWaitingForBatch = sqlcv1.V1EventTypeOlapWAITINGFORBATCH
	eventTypeBatchFlushed    = sqlcv1.V1EventTypeOlapBATCHFLUSHED
)

func describeFlushReason(reason batchFlushReason, batchSize int, interval time.Duration) string {
	switch reason {
	case flushReasonBatchSizeReached:
		if batchSize > 0 {
			return fmt.Sprintf("batch size threshold %d reached", batchSize)
		}

		return "batch size threshold reached"
	case flushReasonWorkerChanged:
		return "assigned worker changed"
	case flushReasonDispatcherChanged:
		return "dispatcher changed"
	case flushReasonIntervalElapsed:
		if interval > 0 {
			return fmt.Sprintf("flush interval %s elapsed", interval)
		}

		return "flush interval elapsed"
	case flushReasonBufferDrained:
		return "buffer drained during shutdown"
	default:
		return string(reason)
	}
}

type SchedulerOpt func(*SchedulerOpts)

type SchedulerOpts struct {
	mq          msgqueue.MessageQueue
	l           *zerolog.Logger
	repo        repository.EngineRepository
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

func WithRepository(r repository.EngineRepository) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.repo = r
	}
}

func WithV2Repository(r repov1.Repository) SchedulerOpt {
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
	repo      repository.EngineRepository
	repov1    repov1.Repository
	dv        datautils.DataDecoderValidator
	s         gocron.Scheduler
	a         *hatcheterrors.Wrapped
	p         *partition.Partition

	// a custom queue logger
	ql *zerolog.Logger

	pool *v1.SchedulingPool

	tasksWithNoWorkerCache *expirable.LRU[string, struct{}]

	batchManager *batchBufferManager
	batchConfigs sync.Map // stepID -> *batchConfig (nil indicates no batch configuration)

	batchCoordinator *schedulingBatchCoordinator
}

type schedulingBatchCoordinator struct {
	scheduler *Scheduler
	mu        sync.Mutex
	states    map[string]*batchCoordinatorState
}

type batchCoordinatorState struct {
	workerID string
	active   bool
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
		repo:                   opts.repo,
		repov1:                 opts.repov1,
		dv:                     opts.dv,
		s:                      s,
		a:                      a,
		p:                      opts.p,
		ql:                     opts.queueLogger,
		pool:                   opts.pool,
		tasksWithNoWorkerCache: tasksWithNoWorkerCache,
	}

	q.batchManager = newBatchBufferManager(q.l, q.batchFlush, q.shouldFlushBatch)
	q.batchCoordinator = newSchedulingBatchCoordinator(q)

	return q, nil
}

func newSchedulingBatchCoordinator(s *Scheduler) *schedulingBatchCoordinator {
	return &schedulingBatchCoordinator{
		scheduler: s,
		states:    make(map[string]*batchCoordinatorState),
	}
}

func batchCoordinatorStateKey(tenantID, stepID, batchKey string) string {
	return tenantID + ":" + stepID + ":" + batchKey
}

func (c *schedulingBatchCoordinator) DecideBatch(ctx context.Context, qi *sqlcv1.V1QueueItem, workerID pgtype.UUID) v1.BatchDecision {
	if c == nil || c.scheduler == nil {
		return v1.BatchDecision{}
	}

	tenantID := sqlchelpers.UUIDToStr(qi.TenantID)
	stepID := sqlchelpers.UUIDToStr(qi.StepID)

	cfg, err := c.scheduler.getBatchConfig(ctx, tenantID, stepID)
	if err != nil {
		c.scheduler.l.Error().Err(err).Str("tenant_id", tenantID).Str("step_id", stepID).Msg("batch coordinator: getBatchConfig failed")
		return v1.BatchDecision{}
	}

	if cfg == nil {
		return v1.BatchDecision{}
	}

	workerStr := ""

	if workerID.Valid {
		workerStr = sqlchelpers.UUIDToStr(workerID)
	}

	batchKey := ""

	if qi.BatchKey.Valid {
		batchKey = strings.TrimSpace(qi.BatchKey.String)
	}

	if cfg.maxRuns > 0 && batchKey != "" {
		activeRuns, countErr := c.scheduler.repov1.Tasks().CountActiveTaskBatchRuns(ctx, tenantID, stepID, batchKey)
		if countErr != nil {
			c.scheduler.l.Error().
				Err(countErr).
				Str("tenant_id", tenantID).
				Str("step_id", stepID).
				Str("batch_key", batchKey).
				Msg("batch coordinator: count active batch runs failed")
		} else if activeRuns >= cfg.maxRuns {
			return v1.BatchDecision{
				Action:      v1.BatchActionDefer,
				ReleaseSlot: true,
			}
		}
	}

	key := batchCoordinatorStateKey(tenantID, stepID, batchKey)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.states == nil {
		c.states = make(map[string]*batchCoordinatorState)
	}

	state, ok := c.states[key]
	if !ok {
		state = &batchCoordinatorState{}
		c.states[key] = state
	}

	if !state.active || state.workerID == "" {
		state.active = true
		state.workerID = workerStr

		if workerStr == "" {
			return v1.BatchDecision{}
		}

		return v1.BatchDecision{
			Action:      v1.BatchActionBuffer,
			ReleaseSlot: true,
		}
	}

	if state.workerID != "" && workerStr != "" && state.workerID != workerStr {
		state.workerID = workerStr
		state.active = true

		return v1.BatchDecision{
			Action:      v1.BatchActionBuffer,
			ReleaseSlot: true,
		}
	}

	return v1.BatchDecision{
		Action:      v1.BatchActionBuffer,
		ReleaseSlot: true,
	}
}

func (c *schedulingBatchCoordinator) setBufferState(tenantID, stepID, batchKey, workerID string, active bool) {
	if c == nil {
		return
	}

	key := batchCoordinatorStateKey(tenantID, stepID, batchKey)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.states == nil {
		c.states = make(map[string]*batchCoordinatorState)
	}

	if !active {
		delete(c.states, key)
		return
	}

	state, ok := c.states[key]
	if !ok {
		state = &batchCoordinatorState{}
		c.states[key] = state
	}

	state.active = true
	state.workerID = workerID
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
					err := s.scheduleStepRuns(ctx, sqlchelpers.UUIDToStr(results.TenantId), results)

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

				go s.notifyAfterConcurrency(ctx, sqlchelpers.UUIDToStr(res.TenantId), res)
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

		if s.batchManager != nil {
			if err := s.batchManager.FlushAll(context.Background()); err != nil {
				s.l.Error().Err(err).Msg("failed to flush batch buffers during shutdown")
			}
		}

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

	if task.ID == "check-tenant-queue" {
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
		tenants, err := s.repo.Tenant().ListTenantsBySchedulerPartition(ctx, s.p.GetSchedulerPartitionId(), dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			s.l.Err(err).Msg("could not list tenants")
			return
		}

		s.pool.SetTenants(tenants)

		if s.batchCoordinator != nil {
			for _, tenant := range tenants {
				s.pool.SetBatchCoordinator(sqlchelpers.UUIDToStr(tenant.ID), s.batchCoordinator)
			}
		}
	}
}

func (s *Scheduler) scheduleStepRuns(ctx context.Context, tenantId string, res *v1.QueueResults) error {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-runs")
	defer span.End()

	var outerErr error

	// bulk assign step runs
	if len(res.Assigned) > 0 || len(res.Buffered) > 0 {
		dispatcherIdToWorkerIdsToStepRuns := make(map[string]map[string][]int64)

		workerIdSet := make(map[string]struct{})
		workerIds := make([]string, 0, len(res.Assigned)+len(res.Buffered))

		for _, assigned := range res.Assigned {
			workerId := sqlchelpers.UUIDToStr(assigned.WorkerId)

			if workerId == "" {
				continue
			}

			if _, exists := workerIdSet[workerId]; !exists {
				workerIdSet[workerId] = struct{}{}
				workerIds = append(workerIds, workerId)
			}
		}

		for _, buffered := range res.Buffered {
			workerId := sqlchelpers.UUIDToStr(buffered.WorkerId)

			if workerId == "" {
				continue
			}

			if _, exists := workerIdSet[workerId]; !exists {
				workerIdSet[workerId] = struct{}{}
				workerIds = append(workerIds, workerId)
			}
		}

		var dispatcherIdWorkerIds map[string][]string

		dispatcherIdWorkerIds, err := s.repo.Worker().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

		if err != nil {
			retryItems := append([]*repov1.AssignedItem{}, res.Assigned...)
			retryItems = append(retryItems, res.Buffered...)

			s.internalRetry(ctx, tenantId, retryItems...)

			return fmt.Errorf("could not list dispatcher ids for workers: %w. attempting internal retry", err)
		}

		workerIdToDispatcherId := make(map[string]string)

		for dispatcherId, workerIds := range dispatcherIdWorkerIds {
			for _, workerId := range workerIds {
				workerIdToDispatcherId[workerId] = dispatcherId
			}
		}

		assignedMsgs := make([]*msgqueue.Message, 0)

		type queueAssignment struct {
			item     *repov1.AssignedItem
			buffered bool
		}

		allAssignments := make([]queueAssignment, 0, len(res.Assigned)+len(res.Buffered))

		for _, assigned := range res.Assigned {
			allAssignments = append(allAssignments, queueAssignment{
				item:     assigned,
				buffered: false,
			})
		}

		for _, buffered := range res.Buffered {
			allAssignments = append(allAssignments, queueAssignment{
				item:     buffered,
				buffered: true,
			})
		}

		for _, assignment := range allAssignments {
			if assignment.item == nil || assignment.item.QueueItem == nil {
				continue
			}

			dispatcherId, ok := workerIdToDispatcherId[sqlchelpers.UUIDToStr(assignment.item.WorkerId)]

			if !ok {
				s.l.Error().Msg("could not assign step run to worker: no dispatcher id. attempting internal retry.")

				s.internalRetry(ctx, tenantId, assignment.item)

				continue
			}

			workerId := sqlchelpers.UUIDToStr(assignment.item.WorkerId)

			taskId := assignment.item.QueueItem.TaskID

			stepId := sqlchelpers.UUIDToStr(assignment.item.QueueItem.StepID)
			actionId := assignment.item.QueueItem.ActionID

			payload := tasktypes.CreateMonitoringEventPayload{
				TaskId:         taskId,
				RetryCount:     assignment.item.QueueItem.RetryCount,
				WorkerId:       &workerId,
				EventType:      sqlcv1.V1EventTypeOlapASSIGNED,
				EventTimestamp: time.Now(),
			}

			batchKey := ""
			if assignment.item.QueueItem != nil && assignment.item.QueueItem.BatchKey.Valid {
				batchKey = strings.TrimSpace(assignment.item.QueueItem.BatchKey.String)
			}

			cfg, cfgErr := s.getBatchConfig(ctx, tenantId, stepId)

			if cfgErr != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not fetch batch configuration for step %s: %w", stepId, cfgErr))
				// fall back to regular dispatch
			} else if cfg != nil {
				if cfg.maxRuns > 0 && batchKey != "" {
					activeRuns, err := s.repov1.Tasks().CountActiveTaskBatchRuns(ctx, tenantId, stepId, batchKey)
					if err != nil {
						outerErr = multierror.Append(outerErr, fmt.Errorf("could not count active batches for step %s: %w", stepId, err))
					} else if activeRuns >= cfg.maxRuns {
						waitingPayload := payload
						waitingPayload.EventType = eventTypeWaitingForBatch
						if batchKey != "" {
							waitingPayload.EventMessage = fmt.Sprintf("Waiting for batch capacity (%d/%d active) for key %s.", activeRuns, cfg.maxRuns, batchKey)
						} else {
							waitingPayload.EventMessage = fmt.Sprintf("Waiting for batch capacity (%d/%d active).", activeRuns, cfg.maxRuns)
						}

						waitingPayload.EventPayload = buildBatchEventPayload(map[string]interface{}{
							"status":     "waiting_for_capacity",
							"batchKey":   batchKey,
							"maxRuns":    cfg.maxRuns,
							"activeRuns": activeRuns,
						})

						msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, waitingPayload)
						if err != nil {
							outerErr = multierror.Append(outerErr, fmt.Errorf("could not create capacity waiting event: %w", err))
						} else {
							assignedMsgs = append(assignedMsgs, msg)
						}

						s.l.Debug().
							Str("tenant_id", tenantId).
							Str("step_id", stepId).
							Str("batch_key", batchKey).
							Int("active_runs", activeRuns).
							Int("max_runs", cfg.maxRuns).
							Msg("deferring batch dispatch until capacity is available")
					}
				}

				addResult, addErr := s.batchManager.Add(
					ctx,
					tenantId,
					stepId,
					actionId,
					dispatcherId,
					workerId,
					batchKey,
					*cfg,
					assignment.item,
				)

				if addErr != nil {
					s.internalRetry(ctx, tenantId, assignment.item)
					outerErr = multierror.Append(outerErr, fmt.Errorf("could not buffer batch task: %w", addErr))
					continue
				}

				displayBatchSize := cfg.batchSize
				if displayBatchSize <= 0 {
					displayBatchSize = 1
				}

				waitingPayload := payload
				waitingPayload.EventType = eventTypeWaitingForBatch
				if addResult.Flushed {
					reasonText := "batch flushed"
					if addResult.FlushReason != nil {
						reasonText = describeFlushReason(*addResult.FlushReason, cfg.batchSize, cfg.flushInterval)
					}

					batchID := addResult.FlushedBatchID
					if batchID == "" {
						batchID = "unknown"
					}

					message := fmt.Sprintf("Batch %s flushed immediately because %s.", batchID, reasonText)
					if batchKey != "" {
						message += fmt.Sprintf(" Batch key: %s.", batchKey)
					}
					if cfg.maxRuns > 0 {
						message += fmt.Sprintf(" Max concurrent batches per key: %d.", cfg.maxRuns)
					}
					waitingPayload.EventMessage = message
					waitingPayload.EventPayload = buildBatchEventPayload(map[string]interface{}{
						"status":   "flushed_immediately",
						"batchId":  batchID,
						"batchKey": batchKey,
						"maxRuns":  cfg.maxRuns,
					})
				} else {
					batchID := addResult.PendingBatchID
					if batchID == "" {
						batchID = "pending"
					}

					var builder strings.Builder
					fmt.Fprintf(&builder, "Waiting for batch %s (%d/%d).", batchID, addResult.Pending, displayBatchSize)
					if batchKey != "" {
						fmt.Fprintf(&builder, " Batch key: %s.", batchKey)
					}
					fmt.Fprintf(&builder, " Flush when size reaches %d", displayBatchSize)

					if cfg.flushInterval > 0 {
						if addResult.NextFlushAt != nil {
							fmt.Fprintf(&builder, " or at %s (interval %s)", addResult.NextFlushAt.UTC().Format(time.RFC3339), cfg.flushInterval)
						} else {
							fmt.Fprintf(&builder, " or after %s interval", cfg.flushInterval)
						}
					}

					if cfg.maxRuns > 0 {
						fmt.Fprintf(&builder, " Max concurrent batches per key: %d.", cfg.maxRuns)
					}

					builder.WriteString(".")
					waitingPayload.EventMessage = builder.String()
					waitingPayload.EventPayload = buildBatchEventPayload(map[string]interface{}{
						"status":       "waiting_for_batch",
						"batchId":      batchID,
						"batchKey":     batchKey,
						"pending":      addResult.Pending,
						"expectedSize": displayBatchSize,
						"maxRuns":      cfg.maxRuns,
					})
				}

				msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, waitingPayload)
				if err != nil {
					outerErr = multierror.Append(outerErr, fmt.Errorf("could not create buffered assignment monitoring event: %w", err))
				} else {
					assignedMsgs = append(assignedMsgs, msg)
				}

				if s.batchCoordinator != nil {
					bufferActive := addResult.Pending > 0
					s.batchCoordinator.setBufferState(tenantId, stepId, batchKey, workerId, bufferActive)
				}

				// already handled via batch manager
				continue
			}

			if assignment.buffered {
				s.l.Warn().Str("step_id", stepId).Msg("buffered queue item encountered without batch configuration")
			}

			msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, payload)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create monitoring event message: %w", err))
				continue
			}

			assignedMsgs = append(assignedMsgs, msg)

			if s.batchCoordinator != nil {
				s.batchCoordinator.setBufferState(tenantId, stepId, batchKey, workerId, false)
			}

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId] = make(map[string][]int64)
			}

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = make([]int64, 0)
			}

			dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = append(dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId], assignment.item.QueueItem.TaskID)
		}

		// for each dispatcher, send a bulk assigned task
		for dispatcherId, workerIdsToStepRuns := range dispatcherIdToWorkerIdsToStepRuns {
			workerBatches := make(map[string][]tasktypes.TaskAssignedBatch, len(workerIdsToStepRuns))

			for workerId, taskIds := range workerIdsToStepRuns {
				for _, taskId := range taskIds {
					workerBatches[workerId] = append(workerBatches[workerId], tasktypes.TaskAssignedBatch{
						BatchID:   "",
						BatchSize: 1,
						TaskIds:   []int64{taskId},
					})
				}
			}

			msg, err := taskAssignedBatchMessage(tenantId, workerBatches)

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
				sqlchelpers.UUIDToStr(schedulingTimedOut.ExternalID),
				sqlchelpers.UUIDToStr(schedulingTimedOut.WorkflowRunID),
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
			taskExternalId := sqlchelpers.UUIDToStr(unassigned.ExternalID)

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
			sqlchelpers.UUIDToStr(a.QueueItem.ExternalID),
			sqlchelpers.UUIDToStr(a.QueueItem.WorkflowRunID),
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
	case "task-assigned-bulk":
		err = s.handleDeadLetteredTaskBulkAssigned(ctx, msg)
	case "task-cancelled":
		err = s.handleDeadLetteredTaskCancelled(ctx, msg)
	default:
		err = fmt.Errorf("unknown task: %s", msg.ID)
	}

	return err
}

func taskAssignedBatchMessage(tenantId string, workerBatches map[string][]tasktypes.TaskAssignedBatch) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-assigned-bulk",
		false,
		true,
		tasktypes.TaskAssignedBulkTaskPayload{
			WorkerBatches: workerBatches,
		},
	)
}

func buildBatchEventPayload(fields map[string]interface{}) string {
	data := make(map[string]interface{}, len(fields))

	for key, value := range fields {
		switch v := value.(type) {
		case string:
			if v != "" {
				data[key] = v
			}
		case int:
			if v != 0 {
				data[key] = v
			}
		case int64:
			if v != 0 {
				data[key] = v
			}
		case float64:
			if v != 0 {
				data[key] = v
			}
		default:
			if value != nil {
				data[key] = value
			}
		}
	}

	if len(data) == 0 {
		return ""
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	return string(bytes)
}

func (s *Scheduler) getBatchConfig(ctx context.Context, tenantId string, stepId string) (*batchConfig, error) {
	if cached, ok := s.batchConfigs.Load(stepId); ok {
		if cached == nil {
			return nil, nil
		}

		cfg := cached.(*batchConfig)
		copied := *cfg
		return &copied, nil
	}

	configs, err := s.repov1.Workflows().ListStepBatchConfigs(ctx, tenantId, []string{stepId})

	if err != nil {
		return nil, err
	}

	cfg, ok := configs[stepId]

	if !ok || cfg == nil {
		s.batchConfigs.Store(stepId, nil)
		return nil, nil
	}

	converted := &batchConfig{
		batchSize: int(cfg.BatchSize),
	}

	if cfg.FlushIntervalMs != nil && *cfg.FlushIntervalMs > 0 {
		converted.flushInterval = time.Duration(*cfg.FlushIntervalMs) * time.Millisecond
	}

	if cfg.MaxRuns != nil && *cfg.MaxRuns > 0 {
		converted.maxRuns = int(*cfg.MaxRuns)
	}

	s.batchConfigs.Store(stepId, converted)

	copied := *converted

	return &copied, nil
}

func (s *Scheduler) shouldFlushBatch(ctx context.Context, req *batchFlushRequest) (bool, error) {
	switch req.FlushReason {
	case flushReasonWorkerChanged, flushReasonDispatcherChanged, flushReasonBufferDrained:
		return true, nil
	}

	if req.BatchKey == "" || req.StepID == "" || req.MaxRuns <= 0 {
		return true, nil
	}

	reserved, err := s.repov1.Tasks().ReserveTaskBatchRun(
		ctx,
		req.TenantID,
		req.StepID,
		req.ActionID,
		req.BatchKey,
		req.BatchID,
		req.MaxRuns,
	)

	if err != nil {
		return false, err
	}

	return reserved, nil
}

func (s *Scheduler) batchFlush(ctx context.Context, req *batchFlushRequest) (err error) {
	if req == nil || len(req.Items) == 0 {
		return nil
	}

	reserved := req.BatchKey != "" && req.MaxRuns > 0

	defer func() {
		if err != nil && reserved {
			if completeErr := s.repov1.Tasks().CompleteTaskBatchRun(ctx, req.TenantID, req.BatchID); completeErr != nil {
				s.l.Error().Err(completeErr).Str("tenant_id", req.TenantID).Str("batch_id", req.BatchID).Msg("failed to release reserved batch run after error")
			}
		}
	}()

	assignments := make([]repov1.TaskBatchAssignment, 0, len(req.Items))
	taskIds := make([]int64, 0, len(req.Items))
	var actionID string

	for _, assigned := range req.Items {
		if assigned == nil || assigned.QueueItem == nil || !assigned.QueueItem.TaskInsertedAt.Valid {
			continue
		}

		assignments = append(assignments, repov1.TaskBatchAssignment{
			TaskID:         assigned.QueueItem.TaskID,
			TaskInsertedAt: assigned.QueueItem.TaskInsertedAt.Time,
			BatchIndex:     len(assignments),
		})
		taskIds = append(taskIds, assigned.QueueItem.TaskID)

		if actionID == "" && assigned.QueueItem.ActionID != "" {
			actionID = assigned.QueueItem.ActionID
		}
	}

	if len(assignments) == 0 {
		return nil
	}

	batchSize := len(assignments)

	if actionID == "" {
		return fmt.Errorf("batch flush missing action id for batch %s", req.BatchID)
	}

	triggeredAt := req.TriggeredAt
	if triggeredAt.IsZero() {
		triggeredAt = time.Now().UTC()
	}

	reasonText := describeFlushReason(req.FlushReason, req.ConfiguredBatchSize, req.ConfiguredFlushInterval)

	if err := s.repov1.Tasks().UpdateTaskBatchMetadata(ctx, req.TenantID, req.BatchID, req.WorkerID, req.BatchKey, batchSize, assignments); err != nil {
		s.internalRetry(ctx, req.TenantID, req.Items...)

		return fmt.Errorf("could not persist batch metadata: %w", err)
	}

	startPayload := tasktypes.StartBatchTaskPayload{
		TenantId:      req.TenantID,
		WorkerId:      req.WorkerID,
		ActionId:      actionID,
		BatchId:       req.BatchID,
		ExpectedSize:  batchSize,
		BatchKey:      req.BatchKey,
		TriggerReason: reasonText,
		TriggerTime:   triggeredAt,
	}

	if req.MaxRuns > 0 {
		maxRuns := req.MaxRuns
		startPayload.MaxRuns = &maxRuns
	}

	startMsg, err := tasktypes.StartBatchMessage(req.TenantID, startPayload)

	if err != nil {
		s.internalRetry(ctx, req.TenantID, req.Items...)

		return fmt.Errorf("could not create batch start message: %w", err)
	}

	if err := s.mq.SendMessage(
		ctx,
		msgqueue.QueueTypeFromDispatcherID(req.DispatcherID),
		startMsg,
	); err != nil {
		s.internalRetry(ctx, req.TenantID, req.Items...)

		return fmt.Errorf("could not send batch start message: %w", err)
	}

	workerBatches := map[string][]tasktypes.TaskAssignedBatch{
		req.WorkerID: {
			{
				BatchID:   req.BatchID,
				BatchSize: batchSize,
				TaskIds:   taskIds,
			},
		},
	}

	msg, err := taskAssignedBatchMessage(req.TenantID, workerBatches)

	if err != nil {
		s.internalRetry(ctx, req.TenantID, req.Items...)

		return fmt.Errorf("could not create bulk assigned task message: %w", err)
	}

	err = s.mq.SendMessage(
		ctx,
		msgqueue.QueueTypeFromDispatcherID(req.DispatcherID),
		msg,
	)

	if err != nil {
		s.internalRetry(ctx, req.TenantID, req.Items...)

		return fmt.Errorf("could not send bulk assigned task: %w", err)
	}

	var workerIdPtr *string

	if req.WorkerID != "" {
		id := req.WorkerID
		workerIdPtr = &id
	}

	keyDisplay := req.BatchKey
	if strings.TrimSpace(keyDisplay) == "" {
		keyDisplay = "(none)"
	}

	batchMessage := fmt.Sprintf(
		"Batch %s flushed (%d tasks) at %s because %s.",
		req.BatchID,
		batchSize,
		triggeredAt.UTC().Format(time.RFC3339),
		reasonText,
	)

	batchMessage += fmt.Sprintf(" Batch key: %s.", keyDisplay)

	if req.MaxRuns > 0 {
		batchMessage += fmt.Sprintf(" Max concurrent batches per key: %d.", req.MaxRuns)
	}
	batchPayload := buildBatchEventPayload(map[string]interface{}{
		"status":        "flushed",
		"batchId":       req.BatchID,
		"batchKey":      req.BatchKey,
		"batchSize":     batchSize,
		"expectedSize":  req.ConfiguredBatchSize,
		"triggerReason": reasonText,
		"triggeredAt":   triggeredAt.UTC().Format(time.RFC3339),
		"maxRuns":       req.MaxRuns,
	})
	monitoringPayloads := make([]tasktypes.CreateMonitoringEventPayload, 0, len(assignments))

	for _, assigned := range req.Items {
		if assigned == nil || assigned.QueueItem == nil {
			continue
		}

		monitoringPayloads = append(monitoringPayloads, tasktypes.CreateMonitoringEventPayload{
			TaskId:         assigned.QueueItem.TaskID,
			RetryCount:     assigned.QueueItem.RetryCount,
			WorkerId:       workerIdPtr,
			EventType:      eventTypeBatchFlushed,
			EventTimestamp: triggeredAt,
			EventMessage:   batchMessage,
			EventPayload:   batchPayload,
		})
	}

	if len(monitoringPayloads) > 0 {
		flushMsg, err := msgqueue.NewTenantMessage(
			req.TenantID,
			"create-monitoring-event",
			false,
			true,
			monitoringPayloads...,
		)

		if err != nil {
			return fmt.Errorf("could not create batch flushed monitoring events: %w", err)
		}

		if err := s.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, flushMsg, false); err != nil {
			return fmt.Errorf("could not send batch flushed monitoring events: %w", err)
		}
	}

	if s.batchCoordinator != nil && len(req.Items) > 0 {
		stepID := ""

		for _, item := range req.Items {
			if item != nil && item.QueueItem != nil {
				stepID = sqlchelpers.UUIDToStr(item.QueueItem.StepID)
				break
			}
		}

		if stepID != "" {
			s.batchCoordinator.setBufferState(req.TenantID, stepID, req.BatchKey, req.WorkerID, false)
		}
	}

	return nil
}

func (s *Scheduler) handleDeadLetteredTaskBulkAssigned(ctx context.Context, msg *msgqueue.Message) error {
	msgs := msgqueue.JSONConvert[tasktypes.TaskAssignedBulkTaskPayload](msg.Payloads)

	taskIds := make([]int64, 0)

	for _, innerMsg := range msgs {
		for _, workerBatches := range innerMsg.WorkerBatches {
			for _, taskBatch := range workerBatches {
				if len(taskBatch.TaskIds) == 0 {
					continue
				}

				s.l.Error().Msgf("handling dead-lettered task assignments for tenant %s, tasks: %v. This indicates an abrupt shutdown of a dispatcher and should be investigated.", msg.TenantID, taskBatch.TaskIds)
				taskIds = append(taskIds, taskBatch.TaskIds...)
			}
		}
	}

	toFail, err := s.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks for dead lettered bulk assigned message: %w", err)
	}

	for _, _task := range toFail {
		tenantId := msg.TenantID
		task := _task.Task

		msg, err := tasktypes.FailedTaskMessage(
			tenantId,
			task.ID,
			task.InsertedAt,
			sqlchelpers.UUIDToStr(task.ExternalID),
			sqlchelpers.UUIDToStr(task.WorkflowRunID),
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
	dispatcherIdWorkerIds, err := s.repo.Worker().GetDispatcherIdsForWorkers(ctx, msg.TenantID, workerIds)

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
			"task-cancelled",
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
