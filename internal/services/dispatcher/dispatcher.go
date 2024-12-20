package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/telemetry/servertel"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

type Dispatcher interface {
	contracts.DispatcherServer
	Start() (func() error, error)
}

type DispatcherImpl struct {
	contracts.UnimplementedDispatcherServer

	s            gocron.Scheduler
	mq           msgqueue.MessageQueue
	sharedReader *msgqueue.SharedTenantReader
	l            *zerolog.Logger
	dv           datautils.DataDecoderValidator
	v            validator.Validator
	repo         repository.EngineRepository
	cache        cache.Cacheable

	entitlements repository.EntitlementsRepository

	dispatcherId string
	workers      *workers
	a            *hatcheterrors.Wrapped
}

var ErrWorkerNotFound = fmt.Errorf("worker not found")

type workers struct {
	innerMap sync.Map
}

func (w *workers) Range(f func(key, value interface{}) bool) {
	w.innerMap.Range(f)
}

func (w *workers) Add(workerId, sessionId string, worker *subscribedWorker) {
	actual, _ := w.innerMap.LoadOrStore(workerId, &sync.Map{})

	actual.(*sync.Map).Store(sessionId, worker)
}

func (w *workers) GetForSession(workerId, sessionId string) (*subscribedWorker, error) {
	actual, ok := w.innerMap.Load(workerId)
	if !ok {
		return nil, ErrWorkerNotFound
	}

	worker, ok := actual.(*sync.Map).Load(sessionId)
	if !ok {
		return nil, ErrWorkerNotFound
	}

	return worker.(*subscribedWorker), nil
}

func (w *workers) Get(workerId string) ([]*subscribedWorker, error) {
	actual, ok := w.innerMap.Load(workerId)

	if !ok {
		return nil, ErrWorkerNotFound
	}

	workers := []*subscribedWorker{}

	actual.(*sync.Map).Range(func(key, value interface{}) bool {
		workers = append(workers, value.(*subscribedWorker))
		return true
	})

	return workers, nil
}

func (w *workers) DeleteForSession(workerId, sessionId string) {
	actual, ok := w.innerMap.Load(workerId)

	if !ok {
		return
	}

	actual.(*sync.Map).Delete(sessionId)
}

func (w *workers) Delete(workerId string) {
	w.innerMap.Delete(workerId)
}

type DispatcherOpt func(*DispatcherOpts)

type DispatcherOpts struct {
	mq           msgqueue.MessageQueue
	l            *zerolog.Logger
	dv           datautils.DataDecoderValidator
	repo         repository.EngineRepository
	entitlements repository.EntitlementsRepository
	dispatcherId string
	alerter      hatcheterrors.Alerter
	cache        cache.Cacheable
}

func defaultDispatcherOpts() *DispatcherOpts {
	logger := logger.NewDefaultLogger("dispatcher")
	alerter := hatcheterrors.NoOpAlerter{}

	return &DispatcherOpts{
		l:            &logger,
		dv:           datautils.NewDataDecoderValidator(),
		dispatcherId: uuid.New().String(),
		alerter:      alerter,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.mq = mq
	}
}

func WithAlerter(a hatcheterrors.Alerter) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.alerter = a
	}
}

func WithRepository(r repository.EngineRepository) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.repo = r
	}
}

func WithEntitlementsRepository(r repository.EntitlementsRepository) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.entitlements = r
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

func WithDispatcherId(dispatcherId string) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.dispatcherId = dispatcherId
	}
}

func WithCache(cache cache.Cacheable) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.cache = cache
	}
}

func New(fs ...DispatcherOpt) (*DispatcherImpl, error) {
	opts := defaultDispatcherOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.entitlements == nil {
		return nil, fmt.Errorf("entitlements repository is required. use WithEntitlementsRepository")
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

	return &DispatcherImpl{
		mq:           opts.mq,
		l:            opts.l,
		dv:           opts.dv,
		v:            validator.NewDefaultValidator(),
		repo:         opts.repo,
		entitlements: opts.entitlements,
		dispatcherId: opts.dispatcherId,
		workers:      &workers{},
		s:            s,
		a:            a,
		cache:        opts.cache,
	}, nil
}

func (d *DispatcherImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())
	mqCleanup, heavyReadMQ := d.mq.Clone()
	heavyReadMQ.SetQOS(1000)

	d.sharedReader = msgqueue.NewSharedTenantReader(heavyReadMQ)

	// register the dispatcher by creating a new dispatcher in the database
	dispatcher, err := d.repo.Dispatcher().CreateNewDispatcher(ctx, &repository.CreateDispatcherOpts{
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

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := d.handleTask(ctx, task)
		if err != nil {
			d.l.Error().Err(err).Msgf("could not handle dispatcher task %s", task.ID)
			return err
		}

		return nil
	}

	// subscribe to a task queue with the dispatcher id
	dispatcherId := sqlchelpers.UUIDToStr(dispatcher.ID)
	cleanupQueue, err := d.mq.Subscribe(msgqueue.QueueTypeFromDispatcherID(dispatcherId), f, msgqueue.NoOpHook)

	if err != nil {
		cancel()
		return nil, err
	}

	cleanup := func() error {
		d.l.Debug().Msgf("dispatcher is shutting down...")
		cancel()

		if err := mqCleanup(); err != nil {
			return fmt.Errorf("could not cleanup queue: %w", err)
		}

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup queue: %w", err)
		}

		wg.Wait()

		// drain the existing connections
		d.l.Debug().Msg("draining existing connections")

		d.workers.Range(func(key, value interface{}) bool {
			value.(*sync.Map).Range(func(key, value interface{}) bool {
				w := value.(*subscribedWorker)

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

		err = d.repo.Dispatcher().Delete(deleteCtx, dispatcherId)
		if err != nil {
			return fmt.Errorf("could not delete dispatcher: %w", err)
		}

		d.l.Debug().Msgf("deleted dispatcher %s", dispatcherId)

		d.l.Debug().Msgf("dispatcher has shutdown")
		return nil
	}

	return cleanup, nil
}

func (d *DispatcherImpl) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(d.l, d.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch task.ID {
	case "group-key-action-assigned":
		err = d.a.WrapErr(d.handleGroupKeyActionAssignedTask(ctx, task), map[string]interface{}{})
	case "step-run-assigned-bulk":
		err = d.a.WrapErr(d.handleStepRunBulkAssignedTask(ctx, task), map[string]interface{}{})
	case "step-run-cancelled":
		err = d.a.WrapErr(d.handleStepRunCancelled(ctx, task), map[string]interface{}{})
	default:
		err = fmt.Errorf("unknown task: %s", task.ID)
	}

	return err
}

func (d *DispatcherImpl) handleGroupKeyActionAssignedTask(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "group-key-action-assigned", task.OtelCarrier)
	defer span.End()

	payload := tasktypes.GroupKeyActionAssignedTaskPayload{}
	metadata := tasktypes.GroupKeyActionAssignedTaskMetadata{}

	err := d.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task payload: %w", err)
	}

	err = d.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task metadata: %w", err)
	}

	// get the worker for this task
	workers, err := d.workers.Get(payload.WorkerId)

	if err != nil {
		return fmt.Errorf("could not get worker: %w", err)
	}

	// load the workflow run from the database
	workflowRun, err := d.repo.WorkflowRun().GetWorkflowRunById(ctx, metadata.TenantId, payload.WorkflowRunId)

	if err != nil {
		return fmt.Errorf("could not get workflow run: %w", err)
	}

	servertel.WithWorkflowRunModel(span, workflowRun)

	groupKeyRunId := sqlchelpers.UUIDToStr(workflowRun.GetGroupKeyRunId)

	if groupKeyRunId == "" {
		return fmt.Errorf("could not get group key run")
	}

	sqlcGroupKeyRun, err := d.repo.GetGroupKeyRun().GetGroupKeyRunForEngine(ctx, metadata.TenantId, groupKeyRunId)

	if err != nil {
		return fmt.Errorf("could not get group key run for engine: %w", err)
	}

	var multiErr error
	var success bool

	for _, w := range workers {
		err = w.StartGroupKeyAction(ctx, metadata.TenantId, sqlcGroupKeyRun)

		if err != nil {
			multiErr = multierror.Append(multiErr, fmt.Errorf("could not send group key action to worker: %w", err))
		} else {
			success = true
		}
	}

	if success {
		return nil
	}

	return multiErr
}

func (d *DispatcherImpl) handleStepRunBulkAssignedTask(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "step-run-assigned-bulk", task.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	payload := tasktypes.StepRunAssignedBulkTaskPayload{}
	metadata := tasktypes.StepRunAssignedBulkTaskMetadata{}

	err := d.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task payload: %w", err)
	}

	err = d.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task metadata: %w", err)
	}

	// load the step runs from the database
	stepRunIds := make([]string, 0)

	for _, srs := range payload.WorkerIdToStepRunIds {
		stepRunIds = append(stepRunIds, srs...)
	}

	bulkDatas, err := d.repo.StepRun().GetStepRunBulkDataForEngine(ctx, metadata.TenantId, stepRunIds)

	if err != nil {
		return fmt.Errorf("could not bulk list step run data: %w", err)
	}

	stepRunIdToData := make(map[string]*dbsqlc.GetStepRunBulkDataForEngineRow)

	for _, sr := range bulkDatas {

		stepRunIdToData[sqlchelpers.UUIDToStr(sr.SRID)] = sr
	}

	outerEg := errgroup.Group{}

	for workerId, stepRunIds := range payload.WorkerIdToStepRunIds {
		workerId := workerId

		outerEg.Go(func() error {
			d.l.Debug().Msgf("worker %s has %d step runs", workerId, len(stepRunIds))

			// get the worker for this task
			workers, err := d.workers.Get(workerId)

			if err != nil && !errors.Is(err, ErrWorkerNotFound) {
				return fmt.Errorf("could not get worker: %w", err)
			}

			innerEg := errgroup.Group{}

			toRetry := []string{}
			toRetryMu := sync.Mutex{}

			for _, stepRunId := range stepRunIds {
				stepRunId := stepRunId

				innerEg.Go(func() error {
					stepRun := stepRunIdToData[stepRunId]

					requeue := func() {
						toRetryMu.Lock()
						toRetry = append(toRetry, stepRunId)
						toRetryMu.Unlock()
					}

					// if we've reached the context deadline, this should be requeued
					if ctx.Err() != nil {
						requeue()
						return nil
					}

					// if the step run has a job run in a non-running state, we should not send it to the worker
					if repository.IsFinalJobRunStatus(stepRun.JobRunStatus) {
						d.l.Debug().Msgf("job run %s is in a final state %s, ignoring", sqlchelpers.UUIDToStr(stepRun.JobRunId), string(stepRun.JobRunStatus))

						// release the semaphore
						return d.repo.StepRun().ReleaseStepRunSemaphore(ctx, metadata.TenantId, stepRunId, false)
					}

					// if the step run is in a final state, we should not send it to the worker
					if repository.IsFinalStepRunStatus(stepRun.Status) {
						d.l.Warn().Msgf("step run %s is in a final state %s, ignoring", stepRunId, string(stepRun.Status))

						return d.repo.StepRun().ReleaseStepRunSemaphore(ctx, metadata.TenantId, stepRunId, false)
					}

					var multiErr error
					var success bool

					for i, w := range workers {
						err := w.StartStepRunFromBulk(ctx, metadata.TenantId, stepRun)

						if err != nil {
							d.l.Err(err).Msgf("could not send step run to worker (%d)", i)
							multiErr = multierror.Append(multiErr, fmt.Errorf("could not send step action to worker (%d): %w", i, err))
						} else {
							success = true
							break
						}
					}

					now := time.Now().UTC()

					if success {
						defer d.repo.StepRun().DeferredStepRunEvent(
							metadata.TenantId,
							repository.CreateStepRunEventOpts{
								StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
								EventMessage:  repository.StringPtr("Sent step run to the assigned worker"),
								EventReason:   repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonSENTTOWORKER),
								EventSeverity: repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityINFO),
								Timestamp:     &now,
								EventData:     map[string]interface{}{"worker_id": workerId},
							},
						)

						return nil
					}

					defer d.repo.StepRun().DeferredStepRunEvent(
						metadata.TenantId,
						repository.CreateStepRunEventOpts{
							StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
							EventMessage:  repository.StringPtr("Could not send step run to assigned worker"),
							EventReason:   repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonREASSIGNED),
							EventSeverity: repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityWARNING),
							Timestamp:     &now,
							EventData:     map[string]interface{}{"worker_id": workerId},
						},
					)

					requeue()

					return multiErr
				})
			}

			innerErr := innerEg.Wait()

			if len(toRetry) > 0 {
				retryCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				_, stepRunsToFail, err := d.repo.StepRun().InternalRetryStepRuns(retryCtx, metadata.TenantId, toRetry)

				if err != nil {
					innerErr = multierror.Append(innerErr, fmt.Errorf("could not requeue step runs: %w", err))
				}

				if len(stepRunsToFail) > 0 {
					now := time.Now()

					batchErr := queueutils.BatchConcurrent(50, stepRunsToFail, func(stepRuns []*dbsqlc.GetStepRunForEngineRow) error {
						var innerBatchErr error

						for _, stepRun := range stepRuns {
							err := d.mq.AddMessage(
								retryCtx,
								msgqueue.JOB_PROCESSING_QUEUE,
								tasktypes.StepRunFailedToTask(
									stepRun,
									"Could not send step run to worker",
									&now,
								),
							)

							if err != nil {
								innerBatchErr = multierror.Append(innerBatchErr, err)
							}
						}

						return innerBatchErr
					})

					if batchErr != nil {
						innerErr = multierror.Append(innerErr, fmt.Errorf("could not fail step runs: %w", batchErr))
					}
				}
			}

			return innerErr
		})
	}

	return outerEg.Wait()
}

func (d *DispatcherImpl) handleStepRunCancelled(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "step-run-cancelled", task.OtelCarrier)
	defer span.End()

	payload := tasktypes.StepRunCancelledTaskPayload{}
	metadata := tasktypes.StepRunCancelledTaskMetadata{}

	err := d.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task payload: %w", err)
	}

	err = d.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task metadata: %w", err)
	}

	// get the worker for this task
	workers, err := d.workers.Get(payload.WorkerId)

	if err != nil && !errors.Is(err, ErrWorkerNotFound) {
		return fmt.Errorf("could not get worker: %w", err)
	} else if errors.Is(err, ErrWorkerNotFound) {
		// if the worker is not found, we can ignore this task
		d.l.Debug().Msgf("worker %s not found, ignoring task", payload.WorkerId)
		return nil
	}

	// load the step run from the database
	stepRun, err := d.repo.StepRun().GetStepRunForEngine(ctx, metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	servertel.WithStepRunModel(span, stepRun)

	var multiErr error
	var success bool

	for _, w := range workers {
		err = w.CancelStepRun(ctx, metadata.TenantId, stepRun)

		if err != nil {
			multiErr = multierror.Append(multiErr, fmt.Errorf("could not send job to worker: %w", err))
		} else {
			success = true
		}
	}

	if success {
		return nil
	}

	return multiErr
}

func (d *DispatcherImpl) runUpdateHeartbeat(ctx context.Context) func() {
	return func() {
		d.l.Debug().Msgf("dispatcher: updating heartbeat")

		now := time.Now().UTC()

		// update the heartbeat
		_, err := d.repo.Dispatcher().UpdateDispatcher(ctx, d.dispatcherId, &repository.UpdateDispatcherOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			d.l.Err(err).Msg("dispatcher: could not update heartbeat")
		}
	}
}
