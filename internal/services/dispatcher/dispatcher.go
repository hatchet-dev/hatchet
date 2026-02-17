package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

type Dispatcher interface {
	contracts.DispatcherServer
	Start() (func() error, error)
}

type DispatcherImpl struct {
	contracts.UnimplementedDispatcherServer

	s                           gocron.Scheduler
	mqv1                        msgqueue.MessageQueue
	pubBuffer                   *msgqueue.MQPubBuffer
	sharedNonBufferedReaderv1   *msgqueue.SharedTenantReader
	sharedBufferedReaderv1      *msgqueue.SharedBufferedTenantReader
	l                           *zerolog.Logger
	dv                          datautils.DataDecoderValidator
	v                           validator.Validator
	repov1                      v1.Repository
	cache                       cache.Cacheable
	payloadSizeThreshold        int
	defaultMaxWorkerBacklogSize int64
	workflowRunBufferSize       int

	dispatcherId uuid.UUID
	workers      *workers
	a            *hatcheterrors.Wrapped

	durableCallbackFn func(taskExternalId uuid.UUID, nodeId int64, payload []byte) error
	version           string
}

var ErrWorkerNotFound = fmt.Errorf("worker not found")

type workers struct {
	innerMap syncx.Map[uuid.UUID, *syncx.Map[string, *subscribedWorker]]
}

func (w *workers) Range(f func(key uuid.UUID, value *syncx.Map[string, *subscribedWorker]) bool) {
	w.innerMap.Range(f)
}

func (w *workers) Add(workerId uuid.UUID, sessionId string, worker *subscribedWorker) {
	actual, _ := w.innerMap.LoadOrStore(workerId, &syncx.Map[string, *subscribedWorker]{})

	actual.Store(sessionId, worker)
}

func (w *workers) GetForSession(workerId uuid.UUID, sessionId string) (*subscribedWorker, error) {
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

	actual.Range(func(key string, value *subscribedWorker) bool {
		workers = append(workers, value)
		return true
	})

	return workers, nil
}

func (w *workers) DeleteForSession(workerId uuid.UUID, sessionId string) {
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
	mqv1                        msgqueue.MessageQueue
	l                           *zerolog.Logger
	dv                          datautils.DataDecoderValidator
	repov1                      v1.Repository
	dispatcherId                uuid.UUID
	alerter                     hatcheterrors.Alerter
	cache                       cache.Cacheable
	payloadSizeThreshold        int
	defaultMaxWorkerBacklogSize int64
	workflowRunBufferSize       int
	version                     string
}

func defaultDispatcherOpts() *DispatcherOpts {
	logger := logger.NewDefaultLogger("dispatcher")
	alerter := hatcheterrors.NoOpAlerter{}

	return &DispatcherOpts{
		l:                           &logger,
		dv:                          datautils.NewDataDecoderValidator(),
		dispatcherId:                uuid.New(),
		alerter:                     alerter,
		payloadSizeThreshold:        3 * 1024 * 1024,
		defaultMaxWorkerBacklogSize: 20,
		workflowRunBufferSize:       1000,
	}
}

func WithMessageQueueV1(mqv1 msgqueue.MessageQueue) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.mqv1 = mqv1
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

func WithDefaultMaxWorkerBacklogSize(size int64) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.defaultMaxWorkerBacklogSize = size
	}
}

func WithWorkflowRunBufferSize(size int) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.workflowRunBufferSize = size
	}
}

func WithVersion(version string) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.version = version
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

	return &DispatcherImpl{
		mqv1:                        opts.mqv1,
		pubBuffer:                   pubBuffer,
		l:                           opts.l,
		dv:                          opts.dv,
		v:                           validator.NewDefaultValidator(),
		repov1:                      opts.repov1,
		dispatcherId:                opts.dispatcherId,
		workers:                     &workers{},
		s:                           s,
		a:                           a,
		cache:                       opts.cache,
		payloadSizeThreshold:        opts.payloadSizeThreshold,
		defaultMaxWorkerBacklogSize: opts.defaultMaxWorkerBacklogSize,
		workflowRunBufferSize:       opts.workflowRunBufferSize,
		version:                     opts.version,
	}, nil
}

func (d *DispatcherImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())
	d.sharedNonBufferedReaderv1 = msgqueue.NewSharedTenantReader(d.mqv1)
	d.sharedBufferedReaderv1 = msgqueue.NewSharedBufferedTenantReader(d.mqv1)

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

		err := d.handleV1Task(ctx, task)
		if err != nil {
			d.l.Error().Err(err).Msgf("could not handle dispatcher task %s", task.ID)
			return err
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
		d.l.Debug().Msgf("dispatcher is shutting down...")
		cancel()

		if err := cleanupQueueV1(); err != nil {
			return fmt.Errorf("could not cleanup queue (v1): %w", err)
		}

		wg.Wait()

		d.pubBuffer.Stop()

		// drain the existing connections
		d.l.Debug().Msg("draining existing connections")

		d.workers.Range(func(key uuid.UUID, value *syncx.Map[string, *subscribedWorker]) bool {
			value.Range(func(key string, value *subscribedWorker) bool {
				w := value

				w.finished <- true

				return true
			})

			return true
		})

		if err := d.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer deleteCancel()

		err = d.repov1.Dispatcher().Delete(deleteCtx, dispatcherId)
		if err != nil {
			return fmt.Errorf("could not delete dispatcher: %w", err)
		}

		d.l.Debug().Msgf("deleted dispatcher %s", dispatcherId)

		d.l.Debug().Msgf("dispatcher has shutdown")
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
	case "task-assigned-bulk":
		err = d.a.WrapErr(d.handleTaskBulkAssignedTask(ctx, task), map[string]interface{}{})
	case "task-cancelled":
		err = d.a.WrapErr(d.handleTaskCancelled(ctx, task), map[string]interface{}{})
	case msgqueue.MsgIDDurableCallbackCompleted:
		err = d.a.WrapErr(d.handleDurableCallbackCompleted(ctx, task), map[string]interface{}{})
	default:
		err = fmt.Errorf("unknown task: %s", task.ID)
	}

	return err
}

func (d *DispatcherImpl) DispatcherId() uuid.UUID {
	return d.dispatcherId
}

func (d *DispatcherImpl) SetDurableCallbackHandler(fn func(uuid.UUID, int64, []byte) error) {
	d.durableCallbackFn = fn
}

func (d *DispatcherImpl) handleDurableCallbackCompleted(ctx context.Context, task *msgqueue.Message) error {
	if d.durableCallbackFn == nil {
		return nil
	}

	payloads := msgqueue.JSONConvert[tasktypes.DurableCallbackCompletedPayload](task.Payloads)

	for _, payload := range payloads {
		err := d.durableCallbackFn(
			payload.TaskExternalId,
			payload.NodeId,
			payload.Payload,
		)

		if err != nil {
			d.l.Error().Err(err).Msgf("failed to deliver callback completion for task %s", payload.TaskExternalId)
		}
	}

	return nil
}

func (d *DispatcherImpl) runUpdateHeartbeat(ctx context.Context) func() {
	return func() {
		d.l.Debug().Msgf("dispatcher: updating heartbeat")

		now := time.Now().UTC()

		// update the heartbeat
		_, err := d.repov1.Dispatcher().UpdateDispatcher(ctx, d.dispatcherId, &v1.UpdateDispatcherOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			d.l.Err(err).Msg("dispatcher: could not update heartbeat")
		}
	}
}
