package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts/contractsconnect"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/streams"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	schedulingv1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

type Dispatcher interface {
	contractsconnect.DispatcherHandler
	Start() (func() error, error)
}

var _ contractsconnect.DispatcherHandler = (*DispatcherImpl)(nil)

type DispatcherImpl struct {
	contractsconnect.UnimplementedDispatcherHandler
	v                                   validator.Validator
	s                                   gocron.Scheduler
	mqv1                                msgqueue.MessageQueue
	pubsub                              msgqueue.PubSub
	analytics                           analytics.Analytics
	cache                               cache.Cacheable
	repov1                              v1.Repository
	dv                                  datautils.DataDecoderValidator
	sharedBufferedReaderv1              *msgqueue.SharedBufferedTenantReader
	a                                   *hatcheterrors.Wrapped
	sharedNonBufferedReaderv1           *msgqueue.SharedTenantReader
	l                                   *zerolog.Logger
	pubBuffer                           *msgqueue.MQPubBuffer
	serviceV1                           *DispatcherServiceImpl
	streamSessions                      *streams.Registry
	workers                             *workers
	version                             string
	defaultMaxWorkerLockAcquisitionTime time.Duration
	streamEventBufferTimeout            time.Duration
	workflowRunBufferSize               int
	payloadSizeThreshold                int
	dispatcherId                        uuid.UUID
	refreshTimeoutGroup                 singleflight.Group
	refreshTimeoutBuf                   *refreshTimeoutBuffer
}

// CancelStreamSessions hangs up all registered long-lived subscriber streams. It is
// called during shutdown before GracefulStop, which would otherwise block on them
// until the process is killed.
func (d *DispatcherImpl) CancelStreamSessions() {
	d.streamSessions.CancelAll()
	d.serviceV1.CancelStreamSessions()
}

// V1 returns the dispatcher's V1Dispatcher gRPC service, which serves the durable
// task and durable event RPCs.
func (d *DispatcherImpl) V1() *DispatcherServiceImpl {
	return d.serviceV1
}

func (d *DispatcherImpl) TriggerDAGStep(ctx context.Context, tenantId uuid.UUID, req *operator.DAGStepTriggerRequest) (*operator.DAGStepTriggerResult, error) {
	return d.serviceV1.TriggerDAGStep(ctx, tenantId, req)
}

func (d *DispatcherImpl) CancelDAGChildren(ctx context.Context, tenantId uuid.UUID, taskExternalIds []uuid.UUID) error {
	return d.serviceV1.CancelDAGChildren(ctx, tenantId, taskExternalIds)
}

var ErrWorkerNotFound = fmt.Errorf("worker not found")

type workers struct {
	innerMap syncx.Map[uuid.UUID, *syncx.Map[uuid.UUID, *subscribedWorker]]
}

func (w *workers) Range(f func(key uuid.UUID, value *syncx.Map[uuid.UUID, *subscribedWorker]) bool) {
	w.innerMap.Range(f)
}

func (w *workers) Add(workerId uuid.UUID, sessionId uuid.UUID, worker *subscribedWorker) {
	actual, _ := w.innerMap.LoadOrStore(workerId, &syncx.Map[uuid.UUID, *subscribedWorker]{})

	actual.Store(sessionId, worker)
}

func (w *workers) GetForSession(workerId uuid.UUID, sessionId uuid.UUID) (*subscribedWorker, error) {
	actual, ok := w.innerMap.Load(workerId)
	if !ok {
		return nil, ErrWorkerNotFound
	}

	worker, ok := actual.Load(sessionId)
	if !ok {
		return nil, ErrWorkerNotFound
	}

	return worker, nil
}

func (w *workers) Get(workerId uuid.UUID) ([]*subscribedWorker, error) {
	actual, ok := w.innerMap.Load(workerId)

	if !ok {
		return nil, ErrWorkerNotFound
	}

	workers := []*subscribedWorker{}

	actual.Range(func(key uuid.UUID, value *subscribedWorker) bool {
		workers = append(workers, value)
		return true
	})

	return workers, nil
}

func (w *workers) DeleteForSession(workerId uuid.UUID, sessionId uuid.UUID) {
	actual, ok := w.innerMap.Load(workerId)

	if !ok {
		return
	}

	actual.Delete(sessionId)
}

func (w *workers) Delete(workerId uuid.UUID) {
	w.innerMap.Delete(workerId)
}

type DispatcherOpt func(*DispatcherOpts)

type DispatcherOpts struct {
	cache                               cache.Cacheable
	dv                                  datautils.DataDecoderValidator
	repov1                              v1.Repository
	alerter                             hatcheterrors.Alerter
	mqv1                                msgqueue.MessageQueue
	pubsub                              msgqueue.PubSub
	analytics                           analytics.Analytics
	l                                   *zerolog.Logger
	version                             string
	payloadSizeThreshold                int
	defaultMaxWorkerLockAcquisitionTime time.Duration
	workflowRunBufferSize               int
	streamEventBufferTimeout            time.Duration
	dispatcherId                        uuid.UUID
	promGate                            *prometheus.Gate
}

func defaultDispatcherOpts() *DispatcherOpts {
	logger := logger.NewDefaultLogger("dispatcher")
	alerter := hatcheterrors.NoOpAlerter{}

	return &DispatcherOpts{
		l:                                   &logger,
		dv:                                  datautils.NewDataDecoderValidator(),
		dispatcherId:                        uuid.New(),
		alerter:                             alerter,
		analytics:                           analytics.NoOpAnalytics{},
		payloadSizeThreshold:                3 * 1024 * 1024,
		defaultMaxWorkerLockAcquisitionTime: 250 * time.Millisecond,
		workflowRunBufferSize:               1000,
		streamEventBufferTimeout:            5 * time.Second,
	}
}

func WithMessageQueueV1(mqv1 msgqueue.MessageQueue) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.mqv1 = mqv1
	}
}

func WithPubSub(pubsub msgqueue.PubSub) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.pubsub = pubsub
	}
}

func WithAlerter(a hatcheterrors.Alerter) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.alerter = a
	}
}

func WithRepositoryV1(r v1.Repository) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.repov1 = r
	}
}

func WithLogger(l *zerolog.Logger) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.l = l
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.dv = dv
	}
}

func WithDispatcherId(dispatcherId uuid.UUID) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.dispatcherId = dispatcherId
	}
}

func WithCache(cache cache.Cacheable) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.cache = cache
	}
}

func WithPayloadSizeThreshold(threshold int) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.payloadSizeThreshold = threshold
	}
}

func WithDefaultMaxWorkerLockAcquisitionTime(t time.Duration) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.defaultMaxWorkerLockAcquisitionTime = t
	}
}

func WithWorkflowRunBufferSize(size int) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.workflowRunBufferSize = size
	}
}

func WithStreamEventBufferTimeout(timeout time.Duration) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.streamEventBufferTimeout = timeout
	}
}

func WithVersion(version string) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.version = version
	}
}

func WithAnalytics(a analytics.Analytics) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.analytics = a
	}
}

func WithPrometheusGate(gate *prometheus.Gate) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.promGate = gate
	}
}

func New(fs ...DispatcherOpt) (*DispatcherImpl, error) {
	opts := defaultDispatcherOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mqv1 == nil {
		return nil, fmt.Errorf("v1 task queue is required. use WithMessageQueueV1")
	}

	if opts.pubsub == nil {
		return nil, fmt.Errorf("pubsub is required. use WithPubSub")
	}

	if opts.repov1 == nil {
		return nil, fmt.Errorf("v1 repository is required. use WithRepositoryV1")
	}

	if opts.cache == nil {
		return nil, fmt.Errorf("cache is required. use WithCache")
	}

	newLogger := opts.l.With().Str("service", "dispatcher").Logger()
	opts.l = &newLogger

	// create a new scheduler
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler for dispatcher: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "dispatcher"})

	pubBuffer := msgqueue.NewMQPubBuffer(opts.mqv1)

	v := validator.NewDefaultValidator()

	return &DispatcherImpl{
		mqv1:                                opts.mqv1,
		pubsub:                              opts.pubsub,
		pubBuffer:                           pubBuffer,
		l:                                   opts.l,
		dv:                                  opts.dv,
		v:                                   v,
		repov1:                              opts.repov1,
		dispatcherId:                        opts.dispatcherId,
		workers:                             &workers{},
		streamSessions:                      streams.NewRegistry(),
		s:                                   s,
		a:                                   a,
		cache:                               opts.cache,
		payloadSizeThreshold:                opts.payloadSizeThreshold,
		defaultMaxWorkerLockAcquisitionTime: opts.defaultMaxWorkerLockAcquisitionTime,
		workflowRunBufferSize:               opts.workflowRunBufferSize,
		analytics:                           opts.analytics,
		streamEventBufferTimeout:            opts.streamEventBufferTimeout,
		version:                             opts.version,
		refreshTimeoutBuf:                   newRefreshTimeoutBuffer(),
		serviceV1:                           newDispatcherService(opts.repov1, opts.mqv1, opts.pubsub, v, opts.l, opts.dispatcherId, opts.analytics, a, opts.promGate),
	}, nil
}

func (d *DispatcherImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())
	d.sharedNonBufferedReaderv1 = msgqueue.NewSharedTenantReader(d.pubsub)
	d.sharedBufferedReaderv1 = msgqueue.NewSharedBufferedTenantReader(d.pubsub)

	// register the dispatcher by creating a new dispatcher in the database
	dispatcher, err := d.repov1.Dispatcher().CreateNewDispatcher(ctx, &v1.CreateDispatcherOpts{
		ID: d.dispatcherId,
	})

	if err != nil {
		cancel()
		return nil, err
	}

	_, err = d.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			d.runUpdateHeartbeat(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule heartbeat update: %w", err)
	}

	d.s.Start()

	wg := sync.WaitGroup{}

	// subscribe to a task queue with the dispatcher id
	dispatcherId := dispatcher.ID

	fv1 := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		if taskErr := d.handleV1Task(ctx, task); taskErr != nil {
			d.l.Error().Ctx(ctx).Err(taskErr).Msgf("could not handle dispatcher task %s", task.ID)
			return taskErr
		}

		return nil
	}

	// subscribe to a task queue with the dispatcher id
	cleanupQueueV1, err := d.mqv1.Subscribe(msgqueue.QueueTypeFromDispatcherID(dispatcherId), fv1, msgqueue.NoOpHook)

	if err != nil {
		cancel()
		return nil, err
	}

	cleanup := func() error {
		d.l.Debug().Ctx(ctx).Msgf("dispatcher is shutting down...")
		cancel()

		if cleanupErr := cleanupQueueV1(); cleanupErr != nil {
			return fmt.Errorf("could not cleanup queue (v1): %w", cleanupErr)
		}

		wg.Wait()

		d.pubBuffer.Stop()
		d.serviceV1.pubBuffer.Stop()
		d.refreshTimeoutBuf.stop()

		// drain the existing connections
		d.l.Debug().Ctx(ctx).Msg("draining existing connections")

		d.workers.Range(func(key uuid.UUID, value *syncx.Map[uuid.UUID, *subscribedWorker]) bool {
			value.Range(func(key uuid.UUID, value *subscribedWorker) bool {
				w := value

				// operator-backed workers have no stream goroutine reading `finished`: the
				// host that opened the session owns their teardown, the in-process host for
				// the operators the claimer runs and the operator service for the sessions
				// it opens
				if w.handler != nil {
					return true
				}

				w.requestFin()

				return true
			})

			return true
		})

		if shutdownErr := d.s.Shutdown(); shutdownErr != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", shutdownErr)
		}

		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer deleteCancel()

		err = d.repov1.Dispatcher().Delete(deleteCtx, dispatcherId)
		if err != nil {
			return fmt.Errorf("could not delete dispatcher: %w", err)
		}

		d.l.Debug().Ctx(ctx).Msgf("deleted dispatcher %s", dispatcherId)

		d.l.Debug().Ctx(ctx).Msgf("dispatcher has shutdown")
		return nil
	}

	return cleanup, nil
}

func (d *DispatcherImpl) handleV1Task(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(d.l, d.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch task.ID {
	case msgqueue.MsgIDTaskAssignedBulk:
		err = d.a.WrapErr(d.handleTaskBulkAssignedTask(ctx, task), map[string]interface{}{})
	case msgqueue.MsgIDTaskCancelled:
		err = d.a.WrapErr(d.handleTaskCancelled(ctx, task), map[string]interface{}{})
	case msgqueue.MsgIDDurableCallbackCompleted:
		err = d.a.WrapErr(d.handleDurableCallbackCompleted(ctx, task), map[string]interface{}{})
	case msgqueue.MsgIDBatchStart:
		err = d.a.WrapErr(d.handleBatchStartTask(ctx, task), map[string]interface{}{})
	default:
		err = fmt.Errorf("unknown task: %s", task.ID)
	}

	return err
}

func (d *DispatcherImpl) DispatcherId() uuid.UUID {
	return d.dispatcherId
}

func (d *DispatcherImpl) handleDurableCallbackCompleted(ctx context.Context, task *msgqueue.Message) error {
	payloads := msgqueue.JSONConvert[tasktypesv1.DurableCallbackCompletedPayload](task.Payloads)

	undelivered := make([]tasktypesv1.DurableCallbackCompletedPayload, 0)

	for _, payload := range payloads {
		err := d.serviceV1.DeliverDurableEventLogEntryCompletion(
			task.TenantID,
			payload.TaskExternalId,
			payload.InvocationCount,
			payload.BranchId,
			payload.NodeId,
			payload.Payload,
			payload.SatisfiedOrder,
			payload.ChildTaskIsFailure,
			payload.ChildTaskErrorMessage,
		)

		if err != nil {
			d.l.Warn().Err(err).Msgf("could not deliver callback completion for task %s; redelivering via the dead-letter queue", payload.TaskExternalId)
			undelivered = append(undelivered, *payload)
		}
	}

	if len(undelivered) == 0 {
		return nil
	}

	msg, err := msgqueue.NewTenantMessage(task.TenantID, msgqueue.MsgIDDurableCallbackCompleted, false, true, undelivered...)

	if err != nil {
		return fmt.Errorf("could not create dead-letter message for undelivered durable callbacks: %w", err)
	}

	if err := d.mqv1.SendMessage(ctx, msgqueue.DISPATCHER_DEAD_LETTER_QUEUE, msg); err != nil {
		return fmt.Errorf("could not publish undelivered durable callbacks to the dead-letter queue: %w", err)
	}

	return nil
}

func (d *DispatcherImpl) runUpdateHeartbeat(ctx context.Context) func() {
	return func() {
		d.l.Debug().Ctx(ctx).Msgf("dispatcher: updating heartbeat")

		now := time.Now().UTC()

		// update the heartbeat
		_, err := d.repov1.Dispatcher().UpdateDispatcher(ctx, d.dispatcherId, &v1.UpdateDispatcherOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			d.l.Err(err).Ctx(ctx).Msg("dispatcher: could not update heartbeat")
		}
	}
}

func (d *DispatcherImpl) handleTaskBulkAssignedTask(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "task-assigned-bulk", msg.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)

	msgs := msgqueue.JSONConvert[tasktypesv1.TaskAssignedBulkTaskPayload](msg.Payloads)
	outerEg := errgroup.Group{}

	toRetry := []*sqlcv1.V1Task{}
	toRetryMu := sync.Mutex{}

	requeue := func(task *sqlcv1.V1Task) {
		toRetryMu.Lock()
		toRetry = append(toRetry, task)
		toRetryMu.Unlock()
	}

	for _, innerMsg := range msgs {
		taskIds := make([]int64, 0)

		for _, tasks := range innerMsg.WorkerIdToTaskIds {
			taskIds = append(taskIds, tasks...)
		}

		attempts := assignedAttemptsFromPayload(innerMsg)

		dispatchData, err := d.repov1.Tasks().ListTasksForDispatch(ctx, msg.TenantID, taskIds)

		if err != nil {
			d.l.Error().Ctx(ctx).Err(err).Msgf("could not bulk list step run data:")
			d.requeueTasksAfterFailedDispatchRead(ctx, requeue, msg.TenantID, attempts)
			continue
		}

		taskIdToData, err := d.populateTaskData(ctx, requeue, msg.TenantID, onlyAssignedAttempts(dispatchData, attempts))

		if err != nil {
			// we've already handled the requeue in populateTaskData, and we've logged the error, so we just continue
			continue
		}

		for workerId, taskIds := range innerMsg.WorkerIdToTaskIds {
			workerId := workerId

			outerEg.Go(func() error {
				return d.sendTasksToWorker(ctx, requeue, msg.TenantID, workerId, taskIds, taskIdToData)
			})
		}
	}

	// we spawn a goroutine to wait for the outer error group to finish and handle retries, because sending over the gRPC stream
	// can occasionally take a long time and we don't want to block the RabbitMQ queue processing
	go func() {
		defer cancel()

		outerErr := outerEg.Wait()

		if err := d.handleRetries(ctx, msg.TenantID, toRetry); err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not retry failed tasks: %w", err))
		}

		if outerErr != nil {
			d.l.Warn().Ctx(ctx).Err(outerErr).Msg("failed to handle task assigned bulk message")
		}
	}()

	return nil
}

func (d *DispatcherImpl) GetLocalWorkerIds() map[uuid.UUID]struct{} {
	workerIds := make(map[uuid.UUID]struct{})

	d.workers.Range(func(workerId uuid.UUID, value *syncx.Map[uuid.UUID, *subscribedWorker]) bool {
		workerIds[workerId] = struct{}{}

		return true
	})

	return workerIds
}

// Note: this is very similar to handleTaskBulkAssignedTask, with some differences in what's sync vs run in a goroutine
// In this method, we wait until all tasks have been sent to the worker before returning
func (d *DispatcherImpl) HandleLocalAssignments(ctx context.Context, tenantId, workerId uuid.UUID, tasks []*schedulingv1.AssignedItemWithTask) error {
	ctx, span := telemetry.NewSpan(ctx, "DispatcherImpl.HandleLocalAssignments")
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	toRetry := []*sqlcv1.V1Task{}
	toRetryMu := sync.Mutex{}

	requeue := func(task *sqlcv1.V1Task) {
		toRetryMu.Lock()
		toRetry = append(toRetry, task)
		toRetryMu.Unlock()
	}

	// we already have payloads; no lookups necessary. we can just send them to the worker
	taskIdToData := make(map[int64]*V1TaskWithPayloadAndInvocationCount)
	taskIds := make([]int64, 0, len(tasks))

	getDurableInvocationCountOpts := make([]v1.IdInsertedAt, 0)

	for _, assigned := range tasks {
		taskIdToData[assigned.Task.ID] = &V1TaskWithPayloadAndInvocationCount{
			V1TaskWithPayload: assigned.Task,
		}
		taskIds = append(taskIds, assigned.Task.ID)

		if assigned.Task.IsDurable.Valid && assigned.Task.IsDurable.Bool {
			getDurableInvocationCountOpts = append(getDurableInvocationCountOpts, v1.IdInsertedAt{
				ID:                   assigned.Task.ID,
				InsertedAtUnixMicros: assigned.Task.InsertedAt.Time.UnixMicro(),
			})
		}
	}

	if len(getDurableInvocationCountOpts) > 0 {
		invocationCounts, err := d.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, tenantId, getDurableInvocationCountOpts)

		if err != nil {
			d.l.Error().Err(err).Msgf("could not get durable task invocation counts for %d tasks", len(getDurableInvocationCountOpts))
		} else {
			for _, assigned := range tasks {
				if assigned.Task.IsDurable.Valid && assigned.Task.IsDurable.Bool {
					count := invocationCounts[v1.IdInsertedAt{
						ID:                   assigned.Task.ID,
						InsertedAtUnixMicros: assigned.Task.InsertedAt.Time.UnixMicro(),
					}]
					taskIdToData[assigned.Task.ID].InvocationCount = count
				}
			}
		}
	}

	// this is one of the core differences from handleTaskBulkAssignedTask: we run this synchronously
	// so that we continue to use an optimistic scheduling semaphore slot until all tasks have been sent
	// to the worker
	err := d.sendTasksToWorker(ctx, requeue, tenantId, workerId, taskIds, taskIdToData)

	if retryErr := d.handleRetries(ctx, tenantId, toRetry); retryErr != nil {
		err = multierror.Append(err, fmt.Errorf("could not retry failed tasks: %w", retryErr))
	}

	return err
}

type V1TaskWithPayloadAndInvocationCount struct {
	*v1.V1TaskWithPayload
	InvocationCount *int32 // only used for durable tasks
}

func (d *DispatcherImpl) populateTaskData(
	ctx context.Context,
	requeue func(task *sqlcv1.V1Task),
	tenantId uuid.UUID,
	dispatchData *v1.TasksForDispatch,
) (map[int64]*V1TaskWithPayloadAndInvocationCount, error) {
	bulkDatas := dispatchData.Tasks

	var err error

	getInvocationCountOpts := make([]v1.IdInsertedAt, 0)

	for _, task := range bulkDatas {
		if task.IsDurable.Valid && task.IsDurable.Bool {
			getInvocationCountOpts = append(getInvocationCountOpts, v1.IdInsertedAt{
				ID:                   task.ID,
				InsertedAtUnixMicros: task.InsertedAt.Time.UnixMicro(),
			})
		}
	}

	invocationCounts := make(map[v1.IdInsertedAt]*int32)

	if len(getInvocationCountOpts) > 0 {
		invocationCounts, err = d.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, tenantId, getInvocationCountOpts)

		if err != nil {
			for _, task := range bulkDatas {
				requeue(task)
			}

			d.l.Error().Err(err).Msgf("could not get durable task invocation counts for %d tasks", len(getInvocationCountOpts))
			return nil, err
		}
	}

	parentDataMap, err := d.repov1.Tasks().ListTaskParentOutputs(ctx, tenantId, bulkDatas)

	if err != nil {
		for _, task := range bulkDatas {
			requeue(task)
		}

		d.l.Error().Ctx(ctx).Err(err).Msgf("could not list parent data for %d tasks", len(bulkDatas))
		return nil, err
	}

	retrievePayloadOpts := make([]v1.PayloadRef[v1.TaskInputShape], len(bulkDatas))

	for i, task := range bulkDatas {
		retrievePayloadOpts[i] = v1.PayloadRef[v1.TaskInputShape]{
			Id:                 task.ID,
			InsertedAt:         task.InsertedAt,
			TenantId:           task.TenantID,
			ExternalId:         task.ExternalID,
			PayloadReferenceId: task.PayloadReferenceID,
		}
	}

	inputs, err := d.repov1.Payloads().TaskInputs().Resolve(ctx, nil, dispatchData.PayloadRows, retrievePayloadOpts...)

	// FIXME: we should differentiate between a retryable error and a non-retryable error here;
	// for example, if we're hitting an S3 rate limit for payloads that exist in S3, we should retry;
	// however, if the payloads simply don't exist, we should fail the tasks instead of requeuing them.
	// The tasks will eventually fail but the extra retries are wasteful.
	if err != nil {
		for _, task := range bulkDatas {
			requeue(task)
		}

		d.l.Error().Ctx(ctx).Err(err).Msgf("could not bulk retrieve inputs for %d tasks", len(bulkDatas))
		return nil, err
	}

	taskIdToStepRunData := make(map[int64]*v1.V1StepRunData, len(bulkDatas))

	// dagChildStepRunData holds the tasks whose parents need to be resolved via GetDagParentOutputs
	// (durable DAG-operator children); we resolve all of them with a single batched lookup below
	// instead of one DB round trip per task.
	dagChildStepRunData := make([]*v1.V1StepRunData, 0)
	dagParentExternalIdSet := make(map[uuid.UUID]struct{})

	for _, task := range bulkDatas {
		stepRunData, err := v1.NewV1StepRunDataFromTaskAndInputPayload(task, taskInputPayloadOrFallback(inputs, task).Bytes())

		if err != nil {
			d.l.Warn().Ctx(ctx).Err(err).Msg("failed to unmarshal input")
			continue
		}

		taskIdToStepRunData[task.ID] = stepRunData

		if len(stepRunData.DagParentTaskRunIds) > 0 {
			dagChildStepRunData = append(dagChildStepRunData, stepRunData)

			for _, parentExternalId := range stepRunData.DagParentTaskRunIds {
				dagParentExternalIdSet[parentExternalId] = struct{}{}
			}
		} else if parentData, ok := parentDataMap[task.ID]; ok {
			parentOutputs, stepRunErrors := splitParentOutputsAndStepRunErrors(ctx, d.l, task.ID, parentData)

			stepRunData.Parents = parentOutputs
			stepRunData.StepRunErrors = stepRunErrors
		}
	}

	if len(dagChildStepRunData) > 0 {
		dagParentExternalIds := make([]uuid.UUID, 0, len(dagParentExternalIdSet))

		for parentExternalId := range dagParentExternalIdSet {
			dagParentExternalIds = append(dagParentExternalIds, parentExternalId)
		}

		dagParentOutputs, err := d.repov1.Tasks().GetDagParentOutputs(ctx, tenantId, dagParentExternalIds)

		if err != nil {
			// if we failed to get all of them at once in a batch, do it 1 by 1. Unfortunately results in an N+1 query pattern,
			// but not much we can do.
			d.l.Warn().Ctx(ctx).Err(err).Msgf("failed to batch look up dag parent outputs for %d tasks, falling back to per-task lookups", len(dagChildStepRunData))

			dagParentOutputs = make(map[uuid.UUID]*v1.TaskOutputEvent)

			for _, stepRunData := range dagChildStepRunData {
				perTaskOutputs, perTaskErr := d.repov1.Tasks().GetDagParentOutputs(ctx, tenantId, stepRunData.DagParentTaskRunIds)

				if perTaskErr != nil {
					d.l.Warn().Ctx(ctx).Err(perTaskErr).Msg("failed to look up dag parent outputs")
					continue
				}

				for parentExternalId, output := range perTaskOutputs {
					dagParentOutputs[parentExternalId] = output
				}
			}
		}

		resolveDagParentOutputs(ctx, d.l, dagChildStepRunData, dagParentOutputs)
	}

	runtimes := dispatchData.Runtimes

	taskIdToData := make(map[int64]*V1TaskWithPayloadAndInvocationCount)

	for _, task := range bulkDatas {
		var dispatchedPayload []byte

		if stepRunData, ok := taskIdToStepRunData[task.ID]; ok {
			dispatchedPayload = stepRunData.Bytes()
		} else {
			dispatchedPayload = taskInputPayloadOrFallback(inputs, task).Bytes()
		}

		invocationCount := invocationCounts[v1.IdInsertedAt{
			ID:                   task.ID,
			InsertedAtUnixMicros: task.InsertedAt.Time.UnixMicro(),
		}]

		taskIdToData[task.ID] = &V1TaskWithPayloadAndInvocationCount{
			&v1.V1TaskWithPayload{
				V1Task:  task,
				Runtime: runtimes[task.ID],
				Payload: dispatchedPayload,
			},
			invocationCount,
		}
	}

	return taskIdToData, nil
}

func taskInputPayloadOrFallback(inputs map[v1.PayloadRef[v1.TaskInputShape]]v1.Payload[v1.TaskInputShape], task *sqlcv1.V1Task) v1.Payload[v1.TaskInputShape] {
	return v1.RetrievePayloadOrFallback(inputs, v1.PayloadRef[v1.TaskInputShape]{
		Id:                 task.ID,
		InsertedAt:         task.InsertedAt,
		TenantId:           task.TenantID,
		ExternalId:         task.ExternalID,
		PayloadReferenceId: task.PayloadReferenceID,
	}, task.Input)
}

func splitParentOutputsAndStepRunErrors(
	ctx context.Context,
	l *zerolog.Logger,
	taskId int64,
	terminalEvents []*v1.TaskOutputEvent,
) (map[string]map[string]interface{}, map[string]string) {
	parentOutputs := make(map[string]map[string]interface{})
	stepRunErrors := make(map[string]string)

	for _, event := range terminalEvents {
		if event.IsFailed() {
			if event.TaskId != taskId {
				stepRunErrors[event.StepReadableID] = event.ErrorMessage
			}

			continue
		}

		outputMap, err := parentOutputMapFromEvent(event)

		if err != nil {
			l.Warn().Ctx(ctx).Err(err).Msg("failed to unmarshal output")
			continue
		}

		parentOutputs[event.StepReadableID] = outputMap
	}

	return parentOutputs, stepRunErrors
}

// A task that returns nothing has an empty output, which the child must still see as an empty object.
func parentOutputMapFromEvent(event *v1.TaskOutputEvent) (map[string]interface{}, error) {
	outputMap := make(map[string]interface{})

	if len(event.Output) == 0 {
		return outputMap, nil
	}

	if err := json.Unmarshal(event.Output, &outputMap); err != nil {
		return nil, err
	}

	return outputMap, nil
}

func resolveDagParentOutputs(
	ctx context.Context,
	l *zerolog.Logger,
	dagChildStepRunData []*v1.V1StepRunData,
	dagParentOutputs map[uuid.UUID]*v1.TaskOutputEvent,
) {
	for _, stepRunData := range dagChildStepRunData {
		parents := make(map[string]map[string]interface{})
		stepRunErrors := make(map[string]string)

		for _, parentExternalId := range stepRunData.DagParentTaskRunIds {
			parentOutput, ok := dagParentOutputs[parentExternalId]

			if !ok {
				continue
			}

			if parentOutput.IsFailed() {
				stepRunErrors[parentOutput.StepReadableID] = parentOutput.ErrorMessage
				continue
			}

			outputMap, err := parentOutputMapFromEvent(parentOutput)

			if err != nil {
				l.Warn().Ctx(ctx).Err(err).Msgf("failed to unmarshal dag parent output for %s", parentOutput.StepReadableID)
				continue
			}

			parents[parentOutput.StepReadableID] = outputMap
		}

		stepRunData.Parents = parents
		stepRunData.StepRunErrors = stepRunErrors
	}
}

func (d *DispatcherImpl) sendTasksToWorker(
	ctx context.Context,
	requeue func(task *sqlcv1.V1Task),
	tenantId, workerId uuid.UUID,
	taskIds []int64,
	tasks map[int64]*V1TaskWithPayloadAndInvocationCount,
) error {
	// get the worker for this task
	workers, err := d.workers.Get(workerId)

	if err != nil && !errors.Is(err, ErrWorkerNotFound) {
		return fmt.Errorf("could not get worker: %w", err)
	}

	innerEg := errgroup.Group{}

	for _, taskId := range taskIds {
		task, ok := tasks[taskId]

		if !ok {
			d.l.Debug().Ctx(ctx).Msgf("task %d is no longer assigned to worker %s, skipping", taskId, workerId)
			continue
		}

		innerEg.Go(func() error {
			// if we've reached the context deadline, this should be requeued
			if ctx.Err() != nil {
				requeue(task.V1Task)
				return nil
			}

			var multiErr error
			var success bool

			for i, w := range workers {
				err := w.StartTaskFromBulk(ctx, tenantId, task.V1TaskWithPayload, task.InvocationCount)

				if err != nil {
					multiErr = multierror.Append(
						multiErr,
						fmt.Errorf("could not send action for task %s to worker %s (%d / %d): %w", task.ExternalID.String(), workerId, i+1, len(workers), err),
					)
				} else {
					success = true
					break
				}
			}

			if success {
				var durableInvCount int32
				if task.InvocationCount != nil {
					durableInvCount = *task.InvocationCount
				}

				msg, err := tasktypesv1.MonitoringEventMessageFromInternal(
					task.TenantID,
					tasktypesv1.CreateMonitoringEventPayload{
						TaskId:                 task.ID,
						RetryCount:             task.RetryCount,
						DurableInvocationCount: durableInvCount,
						WorkerId:               &workerId,
						EventType:              sqlcv1.V1EventTypeOlapSENTTOWORKER,
						EventTimestamp:         time.Now().UTC(),
						EventMessage:           "Sent task run to the assigned worker",
					},
				)

				if err != nil {
					d.l.Error().Ctx(ctx).Err(err).Int64("task_id", task.ID).Msg("could not create monitoring event")
				} else {
					defer func() {
						if err := d.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false); err != nil {
							d.l.Error().Ctx(ctx).Err(err).Msg("could not publish monitoring event")
						}
					}()
				}

				return nil
			}

			requeue(task.V1Task)

			return multiErr
		})
	}

	return innerEg.Wait()
}

type assignedAttempt struct {
	WorkerId      uuid.UUID
	RetryCount    int32
	HasRetryCount bool
}

func assignedAttemptsFromPayload(payload *tasktypesv1.TaskAssignedBulkTaskPayload) map[int64]assignedAttempt {
	attempts := make(map[int64]assignedAttempt)

	for workerId, taskIds := range payload.WorkerIdToTaskIds {
		retryCounts := payload.WorkerIdToTaskRetryCounts[workerId]

		for _, taskId := range taskIds {
			retryCount, hasRetryCount := retryCounts[taskId]

			attempts[taskId] = assignedAttempt{
				WorkerId:      workerId,
				RetryCount:    retryCount,
				HasRetryCount: hasRetryCount,
			}
		}
	}

	return attempts
}

// isCurrentAttempt only checks the worker for messages from schedulers that predate retry counts in
// the payload.
func (a assignedAttempt) isCurrentAttempt(runtime *sqlcv1.V1TaskRuntime) bool {
	return runtime != nil &&
		runtime.WorkerID != nil &&
		*runtime.WorkerID == a.WorkerId &&
		(!a.HasRetryCount || runtime.RetryCount == a.RetryCount)
}

// onlyAssignedAttempts drops tasks reassigned since the message was published (the reassign keeps the
// retry count), so a delayed message can't send a task to a worker it was taken away from.
func onlyAssignedAttempts(dispatchData *v1.TasksForDispatch, attempts map[int64]assignedAttempt) *v1.TasksForDispatch {
	return &v1.TasksForDispatch{
		Tasks: enumerables.Filter(dispatchData.Tasks, func(task *sqlcv1.V1Task) bool {
			return attempts[task.ID].isCurrentAttempt(dispatchData.Runtimes[task.ID])
		}),
		Runtimes:    dispatchData.Runtimes,
		PayloadRows: dispatchData.PayloadRows,
	}
}

// requeueTasksAfterFailedDispatchRead retries the read under a fresh context, since the first read may
// have failed because the handler's deadline expired, and requeues only tasks still on the attempt the
// message named. Anything else is left to the timeout path.
func (d *DispatcherImpl) requeueTasksAfterFailedDispatchRead(
	ctx context.Context,
	requeue func(task *sqlcv1.V1Task),
	tenantId uuid.UUID,
	attempts map[int64]assignedAttempt,
) {
	recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	taskIds := make([]int64, 0, len(attempts))

	for taskId, attempt := range attempts {
		if attempt.HasRetryCount {
			taskIds = append(taskIds, taskId)
		}
	}

	if len(taskIds) == 0 {
		return
	}

	dispatchData, err := d.repov1.Tasks().ListTasksForDispatch(recoveryCtx, tenantId, taskIds)

	if err != nil {
		d.l.Error().Ctx(ctx).Err(err).Msgf("could not list %d tasks to requeue after a failed dispatch read", len(taskIds))
		return
	}

	for _, task := range dispatchData.Tasks {
		if !attempts[task.ID].isCurrentAttempt(dispatchData.Runtimes[task.ID]) {
			continue
		}

		requeue(task)
	}
}

// requeueBatchItems skips items no longer on this batch and worker, so a late batch start can't fail
// an attempt that has since been assigned elsewhere.
func (d *DispatcherImpl) requeueBatchItems(ctx context.Context, payload *tasktypesv1.StartBatchTaskPayload) error {
	if len(payload.Items) == 0 {
		return nil
	}

	batchId, err := uuid.Parse(payload.BatchId)

	if err != nil {
		return fmt.Errorf("could not parse batch id %s: %w", payload.BatchId, err)
	}

	taskIds := make([]int64, 0, len(payload.Items))

	for _, item := range payload.Items {
		taskIds = append(taskIds, item.TaskID)
	}

	dispatchData, err := d.repov1.Tasks().ListTasksForDispatch(ctx, payload.TenantId, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks to requeue batch %s: %w", payload.BatchId, err)
	}

	attempt := assignedAttempt{WorkerId: payload.WorkerId}

	toRetry := enumerables.Filter(dispatchData.Tasks, func(task *sqlcv1.V1Task) bool {
		runtime := dispatchData.Runtimes[task.ID]

		return attempt.isCurrentAttempt(runtime) && runtime.BatchID != nil && *runtime.BatchID == batchId
	})

	d.l.Warn().Ctx(ctx).Str("batch_id", payload.BatchId).Str("worker_id", payload.WorkerId.String()).Int("tasks", len(toRetry)).Msg("worker for batch is not connected to this dispatcher, requeueing its tasks")

	// handleRetries sleeps with backoff before publishing, so it must not hold up the message handler
	go func() {
		if err := d.handleRetries(ctx, payload.TenantId, toRetry); err != nil {
			d.l.Error().Ctx(ctx).Err(err).Str("batch_id", payload.BatchId).Msg("could not requeue batch items")
		}
	}()

	return nil
}

func (d *DispatcherImpl) handleRetries(
	ctx context.Context,
	tenantId uuid.UUID,
	toRetry []*sqlcv1.V1Task,
) error {
	if len(toRetry) == 0 {
		return nil
	}

	retryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	retryGroup := errgroup.Group{}

	for _, _task := range toRetry {
		tenantId := tenantId
		task := _task

		retryGroup.Go(func() error {
			msg, err := tasktypesv1.FailedTaskMessage(
				tenantId,
				task.ID,
				task.InsertedAt,
				task.ExternalID,
				task.WorkflowRunID,
				task.RetryCount,
				false,
				"Could not send task to worker. "+
					"This likely means that too many slots have been configured for the number of workers "+
					"or the network latency between engine and worker is unusually high.",
				false,
				nil,
			)

			if err != nil {
				return fmt.Errorf("could not create failed task message: %w", err)
			}

			queueutils.SleepWithExponentialBackoff(100*time.Millisecond, 5*time.Second, int(task.InternalRetryCount))

			err = msgqueue.PubTenantMessage(retryCtx, d.l, d.mqv1, d.pubsub, msgqueue.TASK_PROCESSING_QUEUE, msg)

			if err != nil {
				return fmt.Errorf("could not send failed task message: %w", err)
			}

			return nil
		})
	}

	return retryGroup.Wait()
}

type batchItemPayload struct {
	WorkflowRunID uuid.UUID       `json:"workflow_run_id"`
	Payload       json.RawMessage `json:"payload"`
}

func (d *DispatcherImpl) sendBatchStartFromPayload(ctx context.Context, payload *tasktypesv1.StartBatchTaskPayload) error {
	if payload == nil {
		return nil
	}

	if payload.BatchId == "" {
		return fmt.Errorf("batch start payload missing batch id")
	}

	if payload.ActionId == "" {
		return fmt.Errorf("batch start payload missing action id for batch %s", payload.BatchId)
	}

	workers, err := d.workers.Get(payload.WorkerId)
	if err != nil {
		if errors.Is(err, ErrWorkerNotFound) {
			return d.requeueBatchItems(ctx, payload)
		}
		return fmt.Errorf("could not get worker for batch %s: %w", payload.BatchId, err)
	}

	// Fetch all task payloads and embed them in the action payload.
	var actionPayload string
	if len(payload.Items) > 0 {
		taskIds := make([]int64, 0, len(payload.Items))
		externalIDByTaskID := make(map[int64]uuid.UUID, len(payload.Items))
		workflowRunIDByTaskID := make(map[int64]uuid.UUID, len(payload.Items))

		for _, item := range payload.Items {
			taskIds = append(taskIds, item.TaskID)
			externalIDByTaskID[item.TaskID] = item.ExternalID
			workflowRunIDByTaskID[item.TaskID] = item.WorkflowRunID
		}

		dispatchData, err := d.repov1.Tasks().ListTasksForDispatch(ctx, payload.TenantId, taskIds)
		if err != nil {
			return fmt.Errorf("could not list tasks for batch %s: %w", payload.BatchId, err)
		}

		taskIdToData, populateErr := d.populateTaskData(ctx, func(_ *sqlcv1.V1Task) {}, payload.TenantId, dispatchData)
		if populateErr != nil {
			return fmt.Errorf("could not populate task data for batch %s: %w", payload.BatchId, populateErr)
		}

		batchItems := make(map[uuid.UUID]batchItemPayload, len(payload.Items))
		for taskID, data := range taskIdToData {
			extID, ok := externalIDByTaskID[taskID]
			if !ok {
				continue
			}
			payloadBytes := data.Payload
			if payloadBytes == nil {
				payloadBytes = []byte("{}")
			}
			batchItems[extID] = batchItemPayload{
				Payload:       json.RawMessage(payloadBytes),
				WorkflowRunID: workflowRunIDByTaskID[taskID],
			}
		}

		payloadJSON, err := json.Marshal(batchItems)
		if err != nil {
			return fmt.Errorf("could not marshal batch items payload for batch %s: %w", payload.BatchId, err)
		}
		actionPayload = string(payloadJSON)
	}

	triggerTime := payload.TriggerTime
	if triggerTime.IsZero() {
		triggerTime = time.Now().UTC()
	}

	expectedSize := payload.ExpectedSize
	if expectedSize < 0 {
		return fmt.Errorf("batch item payload has negative expected size %s", payload.BatchId)
	}

	batchID := payload.BatchId
	batchStart := &contracts.BatchStartPayload{
		TriggerTime:  timestamppb.New(triggerTime),
		ExpectedSize: int32(expectedSize),
	}

	if payload.TriggerReason != "" {
		batchStart.TriggerReason = payload.TriggerReason
	}

	action := &contracts.AssignedAction{
		TenantId:          payload.TenantId.String(),
		ActionType:        contracts.ActionType_START_BATCH,
		ActionId:          payload.ActionId,
		ActionPayload:     actionPayload,
		BatchStartPayload: batchStart,
		BatchId:           &batchID,
	}
	key := strings.TrimSpace(payload.BatchKey)
	if strings.TrimSpace(key) != "" {
		action.BatchKey = &key
	}

	var sendErr error
	var success bool

	for i, w := range workers {
		if err := w.StartBatch(ctx, action); err != nil {
			sendErr = multierror.Append(sendErr, fmt.Errorf("could not send batch start to worker %s (%d): %w", payload.WorkerId, i, err))
		} else {
			success = true
			break
		}
	}

	if !success {
		return sendErr
	}

	return nil
}

func (d *DispatcherImpl) handleTaskCancelled(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "tasks-cancelled", msg.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	msgs := msgqueue.JSONConvert[tasktypesv1.SignalTaskCancelledPayload](msg.Payloads)

	taskIdsToRetryCounts := make(map[int64][]int32)

	for _, innerMsg := range msgs {
		taskIdsToRetryCounts[innerMsg.TaskId] = append(taskIdsToRetryCounts[innerMsg.TaskId], innerMsg.RetryCount)
	}

	taskIds := make([]int64, 0)
	for taskId := range taskIdsToRetryCounts {
		taskIds = append(taskIds, taskId)
	}

	tasks, err := d.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks: %w", err)
	}

	taskIdToData := make(map[int64]*V1TaskWithPayloadAndInvocationCount)
	durableTaskIds := make([]v1.IdInsertedAt, 0)

	for _, task := range tasks {
		taskIdToData[task.ID] = &V1TaskWithPayloadAndInvocationCount{
			V1TaskWithPayload: &v1.V1TaskWithPayload{
				V1Task: task,
			},
		}

		if task.IsDurable.Valid && task.IsDurable.Bool {
			durableTaskIds = append(durableTaskIds, v1.IdInsertedAt{
				ID:                   task.ID,
				InsertedAtUnixMicros: task.InsertedAt.Time.UnixMicro(),
			})
		}
	}

	if len(durableTaskIds) > 0 {
		invocationCounts, err := d.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, msg.TenantID, durableTaskIds)

		if err != nil {
			return fmt.Errorf("could not get durable task invocation counts: %w", err)
		}
		for _, id := range durableTaskIds {
			taskIdToData[id.ID].InvocationCount = invocationCounts[id]
		}
	}

	// group by worker id
	workerIdToTasks := make(map[uuid.UUID][]*V1TaskWithPayloadAndInvocationCount)

	for _, msg := range msgs {
		task, ok := taskIdToData[msg.TaskId]

		if !ok {
			d.l.Warn().Ctx(ctx).Msgf("task %d not found", msg.TaskId)
			continue
		}

		workerIdToTasks[msg.WorkerId] = append(workerIdToTasks[msg.WorkerId], task)
	}

	var multiErr error

	for workerId, tasks := range workerIdToTasks {
		// get the worker for this task
		workers, err := d.workers.Get(workerId)

		if err != nil && !errors.Is(err, ErrWorkerNotFound) {
			return fmt.Errorf("could not get worker: %w", err)
		} else if errors.Is(err, ErrWorkerNotFound) {
			// if the worker is not found, we can ignore this task
			d.l.Debug().Ctx(ctx).Msgf("worker %s not found, ignoring task", workerId)
			continue
		}

		for _, w := range workers {
			for _, task := range tasks {
				retryCounts, ok := taskIdsToRetryCounts[task.ID]

				if !ok {
					d.l.Warn().Ctx(ctx).Msgf("task %d not found in retry counts", task.ID)
					continue
				}

				for _, retryCount := range retryCounts {
					err = w.CancelTask(ctx, msg.TenantID, task.V1Task, retryCount, task.InvocationCount)

					if err != nil {
						multiErr = multierror.Append(multiErr, fmt.Errorf("could not send job to worker: %w", err))
					}
				}
			}
		}
	}

	return multiErr
}

func (d *DispatcherImpl) handleBatchStartTask(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "batch-start", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueue.JSONConvert[tasktypesv1.StartBatchTaskPayload](msg.Payloads)

	var result error

	for _, payload := range payloads {
		if err := d.sendBatchStartFromPayload(ctx, payload); err != nil {
			if errors.Is(err, ErrWorkerNotFound) {
				continue
			}
			result = multierror.Append(result, err)
		}
	}

	return result
}

// MarkDraining is the first step of shutdown: it flags this dispatcher inactive so the
// operator claim query stops claiming for it and other dispatchers take its operators over
// while it is still draining.
func (d *DispatcherImpl) MarkDraining(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return d.repov1.Dispatcher().MarkInactive(ctx, d.dispatcherId)
}
