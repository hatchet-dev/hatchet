package workflows

// import (
// 	"context"
// 	"fmt"
// 	"sync"
// 	"time"

// 	"github.com/go-co-op/gocron/v2"
// 	"github.com/hashicorp/go-multierror"
// 	"github.com/rs/zerolog"
// 	"golang.org/x/sync/errgroup"

// 	"github.com/hatchet-dev/hatchet/internal/cel"
// 	"github.com/hatchet-dev/hatchet/internal/datautils"
// 	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
// 	"github.com/hatchet-dev/hatchet/internal/msgqueue"
// 	"github.com/hatchet-dev/hatchet/internal/queueutils"
// 	"github.com/hatchet-dev/hatchet/internal/services/partition"
// 	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
// 	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
// 	"github.com/hatchet-dev/hatchet/internal/telemetry"
// 	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
// 	"github.com/hatchet-dev/hatchet/pkg/logger"
// 	"github.com/hatchet-dev/hatchet/pkg/repository"
// 	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
// 	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
// 	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
// 	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
// )

// type WorkflowsController interface {
// 	Start(ctx context.Context) error
// }

// type WorkflowsControllerImpl struct {
// 	mq                       msgqueue.MessageQueue
// 	l                        *zerolog.Logger
// 	repo                     repository.EngineRepository
// 	dv                       datautils.DataDecoderValidator
// 	s                        gocron.Scheduler
// 	tenantAlerter            *alerting.TenantAlertManager
// 	a                        *hatcheterrors.Wrapped
// 	p                        *partition.Partition
// 	celParser                *cel.CELParser
// 	processWorkflowEventsOps *queueutils.OperationPool
// 	unpausedWorkflowRunsOps  *queueutils.OperationPool
// 	bumpQueueOps             *queueutils.OperationPool

// 	workflowVersionCache *cache.Cache
// }

// type WorkflowsControllerOpt func(*WorkflowsControllerOpts)

// type WorkflowsControllerOpts struct {
// 	mq      msgqueue.MessageQueue
// 	l       *zerolog.Logger
// 	repo    repository.EngineRepository
// 	dv      datautils.DataDecoderValidator
// 	ta      *alerting.TenantAlertManager
// 	alerter hatcheterrors.Alerter
// 	p       *partition.Partition
// }

// func defaultWorkflowsControllerOpts() *WorkflowsControllerOpts {
// 	logger := logger.NewDefaultLogger("workflows-controller")
// 	alerter := hatcheterrors.NoOpAlerter{}

// 	return &WorkflowsControllerOpts{
// 		l:       &logger,
// 		dv:      datautils.NewDataDecoderValidator(),
// 		alerter: alerter,
// 	}
// }

// func WithMessageQueue(mq msgqueue.MessageQueue) WorkflowsControllerOpt {
// 	return func(opts *WorkflowsControllerOpts) {
// 		opts.mq = mq
// 	}
// }

// func WithLogger(l *zerolog.Logger) WorkflowsControllerOpt {
// 	return func(opts *WorkflowsControllerOpts) {
// 		opts.l = l
// 	}
// }

// func WithRepository(r repository.EngineRepository) WorkflowsControllerOpt {
// 	return func(opts *WorkflowsControllerOpts) {
// 		opts.repo = r
// 	}
// }

// func WithAlerter(a hatcheterrors.Alerter) WorkflowsControllerOpt {
// 	return func(opts *WorkflowsControllerOpts) {
// 		opts.alerter = a
// 	}
// }

// func WithDataDecoderValidator(dv datautils.DataDecoderValidator) WorkflowsControllerOpt {
// 	return func(opts *WorkflowsControllerOpts) {
// 		opts.dv = dv
// 	}
// }

// func WithTenantAlerter(ta *alerting.TenantAlertManager) WorkflowsControllerOpt {
// 	return func(opts *WorkflowsControllerOpts) {
// 		opts.ta = ta
// 	}
// }

// func WithPartition(p *partition.Partition) WorkflowsControllerOpt {
// 	return func(opts *WorkflowsControllerOpts) {
// 		opts.p = p
// 	}
// }

// func New(fs ...WorkflowsControllerOpt) (*WorkflowsControllerImpl, error) {
// 	opts := defaultWorkflowsControllerOpts()

// 	for _, f := range fs {
// 		f(opts)
// 	}

// 	if opts.mq == nil {
// 		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
// 	}

// 	if opts.repo == nil {
// 		return nil, fmt.Errorf("repository is required. use WithRepository")
// 	}

// 	if opts.ta == nil {
// 		return nil, fmt.Errorf("tenant alerter is required. use WithTenantAlerter")
// 	}

// 	if opts.p == nil {
// 		return nil, fmt.Errorf("partition is required. use WithPartition")
// 	}

// 	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

// 	if err != nil {
// 		return nil, fmt.Errorf("could not create scheduler: %w", err)
// 	}

// 	newLogger := opts.l.With().Str("service", "workflows-controller").Logger()
// 	opts.l = &newLogger

// 	a := hatcheterrors.NewWrapped(opts.alerter)
// 	a.WithData(map[string]interface{}{"service": "workflows-controller"})

// 	w := &WorkflowsControllerImpl{
// 		mq:            opts.mq,
// 		l:             opts.l,
// 		repo:          opts.repo,
// 		dv:            opts.dv,
// 		s:             s,
// 		tenantAlerter: opts.ta,
// 		a:             a,
// 		p:             opts.p,
// 		celParser:     cel.NewCELParser(),
// 	}

// 	w.processWorkflowEventsOps = queueutils.NewOperationPool(w.l, time.Second*5, "process workflow events", w.processWorkflowEvents)
// 	w.unpausedWorkflowRunsOps = queueutils.NewOperationPool(w.l, time.Second*5, "unpause workflow runs", w.unpauseWorkflowRuns)
// 	w.bumpQueueOps = queueutils.NewOperationPool(w.l, time.Second*5, "bump queue", w.runPollActiveQueuesTenant)

// 	return w, nil
// }

// func (wc *WorkflowsControllerImpl) Start() (func() error, error) {
// 	wc.l.Debug().Msg("starting workflows controller")

// 	workflowVersionCache := cache.New(60 * time.Second)

// 	wc.workflowVersionCache = workflowVersionCache

// 	ctx, cancel := context.WithCancel(context.Background())

// 	wg := sync.WaitGroup{}

// 	_, err := wc.s.NewJob(
// 		gocron.DurationJob(time.Second*5),
// 		gocron.NewTask(
// 			wc.runGetGroupKeyRunRequeue(ctx),
// 		),
// 	)

// 	if err != nil {
// 		cancel()
// 		return nil, fmt.Errorf("could not schedule get group key run requeue: %w", err)
// 	}

// 	_, err = wc.s.NewJob(
// 		gocron.DurationJob(time.Second*5),
// 		gocron.NewTask(
// 			wc.runGetGroupKeyRunReassign(ctx),
// 		),
// 	)

// 	if err != nil {
// 		cancel()
// 		return nil, fmt.Errorf("could not schedule get group key run reassign: %w", err)
// 	}

// 	_, err = wc.s.NewJob(
// 		gocron.DurationJob(time.Second*1),
// 		gocron.NewTask(
// 			wc.runPollActiveQueues(ctx),
// 		),
// 	)

// 	if err != nil {
// 		cancel()
// 		return nil, fmt.Errorf("could not poll active queues: %w", err)
// 	}

// 	_, err = wc.s.NewJob(
// 		gocron.DurationJob(time.Second*1),
// 		gocron.NewTask(
// 			wc.runTenantProcessWorkflowRunEvents(ctx),
// 		),
// 	)

// 	if err != nil {
// 		cancel()
// 		return nil, fmt.Errorf("could not schedule process workflow run events: %w", err)
// 	}

// 	_, err = wc.s.NewJob(
// 		gocron.DurationJob(time.Second*1),
// 		gocron.NewTask(
// 			wc.runTenantUnpauseWorkflowRuns(ctx),
// 		),
// 	)

// 	if err != nil {
// 		cancel()
// 		return nil, fmt.Errorf("could not schedule unpause workflow runs: %w", err)
// 	}

// 	wc.s.Start()

// 	f := func(task *msgqueue.Message) error {
// 		wg.Add(1)
// 		defer wg.Done()

// 		err := wc.handleTask(context.Background(), task)
// 		if err != nil {
// 			wc.l.Error().Err(err).Msg("could not handle workflow task")
// 			return err
// 		}

// 		return nil
// 	}

// 	cleanupQueue, err := wc.mq.Subscribe(msgqueue.WORKFLOW_PROCESSING_QUEUE, f, msgqueue.NoOpHook)

// 	if err != nil {
// 		cancel()
// 		return nil, err
// 	}

// 	f2 := func(task *msgqueue.Message) error {
// 		wg.Add(1)
// 		defer wg.Done()

// 		err := wc.handlePartitionTask(context.Background(), task)
// 		if err != nil {
// 			wc.l.Error().Err(err).Msg("could not handle job task")
// 			return wc.a.WrapErr(fmt.Errorf("could not handle job task: %w", err), map[string]interface{}{"task_id": task.ID}) // nolint: errcheck
// 		}

// 		return nil
// 	}

// 	cleanupQueue2, err := wc.mq.Subscribe(
// 		msgqueue.QueueTypeFromPartitionIDAndController(wc.p.GetControllerPartitionId(), msgqueue.WorkflowController),
// 		msgqueue.NoOpHook, // the only handler is to check the queue, so we acknowledge immediately with the NoOpHook
// 		f2,
// 	)

// 	if err != nil {
// 		cancel()
// 		return nil, fmt.Errorf("could not subscribe to job processing queue: %w", err)
// 	}

// 	cleanup := func() error {
// 		cancel()

// 		if err := cleanupQueue(); err != nil {
// 			return fmt.Errorf("could not cleanup queue: %w", err)
// 		}

// 		if err := cleanupQueue2(); err != nil {
// 			return fmt.Errorf("could not cleanup queue: %w", err)
// 		}

// 		wg.Wait()

// 		if err := wc.s.Shutdown(); err != nil {
// 			return fmt.Errorf("could not shutdown scheduler: %w", err)
// 		}

// 		workflowVersionCache.Stop()

// 		return nil
// 	}

// 	return cleanup, nil
// }

// func (wc *WorkflowsControllerImpl) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			recoverErr := recoveryutils.RecoverWithAlert(wc.l, wc.a, r)

// 			if recoverErr != nil {
// 				err = recoverErr
// 			}
// 		}
// 	}()

// 	switch task.ID {
// 	case "replay-workflow-run":
// 		return wc.handleReplayWorkflowRun(ctx, task)
// 	case "workflow-run-queued":
// 		return wc.handleWorkflowRunQueued(ctx, task)
// 	case "get-group-key-run-started":
// 		return wc.handleGroupKeyRunStarted(ctx, task)
// 	case "get-group-key-run-finished":
// 		return wc.handleGroupKeyRunFinished(ctx, task)
// 	case "get-group-key-run-failed":
// 		return wc.handleGroupKeyRunFailed(ctx, task)
// 	case "get-group-key-run-timed-out":
// 		return wc.handleGetGroupKeyRunTimedOut(ctx, task)
// 	case "workflow-run-finished":
// 		return wc.handleWorkflowRunFinished(ctx, task)
// 	}

// 	return fmt.Errorf("unknown task: %s", task.ID)
// }

// func (wc *WorkflowsControllerImpl) handlePartitionTask(ctx context.Context, task *msgqueue.Message) (err error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			recoverErr := recoveryutils.RecoverWithAlert(wc.l, wc.a, r)

// 			if recoverErr != nil {
// 				err = recoverErr
// 			}
// 		}
// 	}()

// 	if task.ID == "check-tenant-queue" {
// 		return wc.handleCheckQueue(ctx, task)
// 	}

// 	return fmt.Errorf("unknown task: %s", task.ID)
// }

// func (wc *WorkflowsControllerImpl) handleCheckQueue(ctx context.Context, task *msgqueue.Message) error {
// 	_, span := telemetry.NewSpanWithCarrier(ctx, "handle-check-queue", task.OtelCarrier)
// 	defer span.End()

// 	metadata := tasktypes.CheckTenantQueueMetadata{}

// 	err := wc.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode check queue metadata: %w", err)
// 	}

// 	// if this tenant is registered, then we should check the queue
// 	wc.bumpQueueOps.RunOrContinue(metadata.TenantId)

// 	return nil
// }

// func (wc *WorkflowsControllerImpl) checkTenantQueue(ctx context.Context, tenantId string) {
// 	// send a message to the tenant partition queue that a step run is ready to be scheduled
// 	tenant, err := wc.repo.Tenant().GetTenantByID(ctx, tenantId)

// 	if err != nil {
// 		wc.l.Err(err).Msg("could not add message to tenant partition queue")
// 		return
// 	}

// 	if tenant.ControllerPartitionId.Valid {
// 		err = wc.mq.SendMessage(
// 			ctx,
// 			msgqueue.QueueTypeFromPartitionIDAndController(tenant.ControllerPartitionId.String, msgqueue.WorkflowController),
// 			tasktypes.CheckTenantQueueToTask(tenantId, "", false, false),
// 		)

// 		if err != nil {
// 			wc.l.Err(err).Msg("could not add message to tenant partition queue")
// 		}
// 	}
// }

// func (wc *WorkflowsControllerImpl) handleReplayWorkflowRun(ctx context.Context, task *msgqueue.Message) error {
// 	ctx, span := telemetry.NewSpan(ctx, "replay-workflow-run") // nolint:ineffassign
// 	defer span.End()

// 	payload := tasktypes.ReplayWorkflowRunTaskPayload{}
// 	metadata := tasktypes.ReplayWorkflowRunTaskMetadata{}

// 	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

// 	if err != nil {
// 		return fmt.Errorf("could not decode replay workflow run task payload: %w", err)
// 	}

// 	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode replay workflow run task metadata: %w", err)
// 	}

// 	_, err = wc.repo.WorkflowRun().ReplayWorkflowRun(ctx, metadata.TenantId, payload.WorkflowRunId)

// 	if err != nil {
// 		return fmt.Errorf("could not replay workflow run: %w", err)
// 	}

// 	// push a task that the workflow run is queued
// 	return wc.mq.SendMessage(
// 		ctx,
// 		msgqueue.WORKFLOW_PROCESSING_QUEUE,
// 		tasktypes.WorkflowRunQueuedToTask(metadata.TenantId, payload.WorkflowRunId),
// 	)
// }

// func (ec *WorkflowsControllerImpl) handleGroupKeyRunStarted(ctx context.Context, task *msgqueue.Message) error {
// 	ctx, span := telemetry.NewSpan(ctx, "get-group-key-run-started") // nolint:ineffassign
// 	defer span.End()

// 	payload := tasktypes.GetGroupKeyRunStartedTaskPayload{}
// 	metadata := tasktypes.GetGroupKeyRunStartedTaskMetadata{}

// 	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

// 	if err != nil {
// 		return fmt.Errorf("could not decode group key run started task payload: %w", err)
// 	}

// 	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode group key run started task metadata: %w", err)
// 	}

// 	// update the get group key run in the database
// 	startedAt, err := time.Parse(time.RFC3339, payload.StartedAt)

// 	if err != nil {
// 		return fmt.Errorf("could not parse started at: %w", err)
// 	}

// 	_, err = ec.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
// 		StartedAt: &startedAt,
// 		Status:    repository.StepRunStatusPtr(db.StepRunStatusRunning),
// 	})

// 	return err
// }

// func (wc *WorkflowsControllerImpl) handleGroupKeyRunFinished(ctx context.Context, task *msgqueue.Message) error {
// 	ctx, span := telemetry.NewSpan(ctx, "handle-group-key-run-finished")
// 	defer span.End()

// 	payload := tasktypes.GetGroupKeyRunFinishedTaskPayload{}
// 	metadata := tasktypes.GetGroupKeyRunFinishedTaskMetadata{}

// 	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

// 	if err != nil {
// 		return fmt.Errorf("could not decode group key run finished task payload: %w", err)
// 	}

// 	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode group key run finished task metadata: %w", err)
// 	}

// 	// update the group key run in the database
// 	finishedAt, err := time.Parse(time.RFC3339, payload.FinishedAt)

// 	if err != nil {
// 		return fmt.Errorf("could not parse started at: %w", err)
// 	}

// 	_, err = wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
// 		FinishedAt: &finishedAt,
// 		Status:     repository.StepRunStatusPtr(db.StepRunStatusSucceeded),
// 		Output:     &payload.GroupKey,
// 	})

// 	if err != nil {
// 		return fmt.Errorf("could not update group key run: %w", err)
// 	}

// 	wc.checkTenantQueue(ctx, metadata.TenantId)

// 	return nil
// }

// func (wc *WorkflowsControllerImpl) runPollActiveQueues(ctx context.Context) func() {
// 	return func() {
// 		wc.l.Debug().Msg("polling active queues")

// 		// list all tenants
// 		tenants, err := wc.repo.Tenant().ListTenantsByControllerPartition(ctx, wc.p.GetControllerPartitionId())

// 		if err != nil {
// 			wc.l.Err(err).Msg("could not list tenants")
// 			return
// 		}

// 		for i := range tenants {
// 			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)
// 			wc.bumpQueueOps.RunOrContinue(tenantId)
// 		}
// 	}
// }

// func (wc *WorkflowsControllerImpl) runPollActiveQueuesTenant(ctx context.Context, tenantId string) (bool, error) {
// 	wc.l.Debug().Msgf("polling active queues for tenant: %s", tenantId)

// 	toQueueList, err := wc.repo.WorkflowRun().ListActiveQueuedWorkflowVersions(ctx, tenantId)

// 	if err != nil {
// 		wc.l.Error().Err(err).Msgf("could not list active queued workflow versions for tenant: %s", tenantId)
// 		return false, err
// 	}

// 	errGroup := new(errgroup.Group)

// 	for i := range toQueueList {
// 		toQueue := toQueueList[i]
// 		errGroup.Go(func() error {
// 			workflowVersionId := sqlchelpers.UUIDToStr(toQueue.WorkflowVersionId)
// 			tenantId := sqlchelpers.UUIDToStr(toQueue.TenantId)
// 			err := wc.bumpQueue(ctx, tenantId, workflowVersionId)
// 			return err
// 		})
// 	}

// 	return false, errGroup.Wait()
// }

// func (wc *WorkflowsControllerImpl) bumpQueue(ctx context.Context, tenantId string, workflowVersionId string) error {
// 	workflowVersion, err := wc.getWorkflowVersion(ctx, tenantId, workflowVersionId)

// 	if err != nil {
// 		return fmt.Errorf("could not get workflow version: %w", err)
// 	}

// 	if workflowVersion.ConcurrencyLimitStrategy.Valid {
// 		switch workflowVersion.ConcurrencyLimitStrategy.ConcurrencyLimitStrategy {
// 		case dbsqlc.ConcurrencyLimitStrategyCANCELINPROGRESS:
// 			err = wc.queueByCancelInProgress(ctx, tenantId, workflowVersion)
// 		case dbsqlc.ConcurrencyLimitStrategyGROUPROUNDROBIN:
// 			err = wc.queueByGroupRoundRobin(ctx, tenantId, workflowVersion)
// 		case dbsqlc.ConcurrencyLimitStrategyCANCELNEWEST:
// 			err = wc.queueByCancelNewest(ctx, tenantId, workflowVersion)
// 		default:
// 			return fmt.Errorf("unimplemented concurrency limit strategy: %s", workflowVersion.ConcurrencyLimitStrategy.ConcurrencyLimitStrategy)
// 		}
// 	}

// 	return err
// }

// func (wc *WorkflowsControllerImpl) getWorkflowVersion(ctx context.Context, tenantId, workflowVersionId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
// 	cachedWorkflowVersion, ok := wc.workflowVersionCache.Get(workflowVersionId)

// 	if ok {
// 		return cachedWorkflowVersion.(*dbsqlc.GetWorkflowVersionForEngineRow), nil
// 	}

// 	workflowVersion, err := wc.repo.Workflow().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

// 	if err != nil {
// 		return nil, fmt.Errorf("could not get workflow version: %w", err)
// 	}

// 	wc.workflowVersionCache.Set(workflowVersionId, workflowVersion)

// 	return workflowVersion, nil
// }

// func (wc *WorkflowsControllerImpl) handleGroupKeyRunFailed(ctx context.Context, task *msgqueue.Message) error {
// 	ctx, span := telemetry.NewSpan(ctx, "handle-group-key-run-failed") // nolint: ineffassign
// 	defer span.End()

// 	payload := tasktypes.GetGroupKeyRunFailedTaskPayload{}
// 	metadata := tasktypes.GetGroupKeyRunFailedTaskMetadata{}

// 	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

// 	if err != nil {
// 		return fmt.Errorf("could not decode group key run failed task payload: %w", err)
// 	}

// 	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode group key run failed task metadata: %w", err)
// 	}

// 	// update the group key run in the database
// 	failedAt, err := time.Parse(time.RFC3339, payload.FailedAt)
// 	if err != nil {
// 		return fmt.Errorf("could not parse started at: %w", err)
// 	}

// 	_, err = wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
// 		FinishedAt: &failedAt,
// 		Error:      &payload.Error,
// 		Status:     repository.StepRunStatusPtr(db.StepRunStatusFailed),
// 	})

// 	if err != nil {
// 		return fmt.Errorf("could not update get group key run: %w", err)
// 	}

// 	return nil
// }

// func (wc *WorkflowsControllerImpl) handleGetGroupKeyRunTimedOut(ctx context.Context, task *msgqueue.Message) error {
// 	ctx, span := telemetry.NewSpan(ctx, "handle-get-group-key-run-timed-out")
// 	defer span.End()

// 	payload := tasktypes.GetGroupKeyRunTimedOutTaskPayload{}
// 	metadata := tasktypes.GetGroupKeyRunTimedOutTaskMetadata{}

// 	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

// 	if err != nil {
// 		return fmt.Errorf("could not decode get group key run run timed out task payload: %w", err)
// 	}

// 	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

// 	if err != nil {
// 		return fmt.Errorf("could not decode get group key run run timed out task metadata: %w", err)
// 	}

// 	return wc.cancelGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, "TIMED_OUT")
// }

// func (wc *WorkflowsControllerImpl) cancelGetGroupKeyRun(ctx context.Context, tenantId, getGroupKeyRunId, reason string) error {
// 	ctx, span := telemetry.NewSpan(ctx, "cancel-get-group-key-run") // nolint: ineffassign
// 	defer span.End()

// 	// cancel current step run
// 	now := time.Now().UTC()

// 	groupKeyRun, err := wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, tenantId, getGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
// 		CancelledAt:     &now,
// 		CancelledReason: repository.StringPtr(reason),
// 		Status:          repository.StepRunStatusPtr(db.StepRunStatusCancelled),
// 	})

// 	if err != nil {
// 		return fmt.Errorf("could not update get group key run: %w", err)
// 	}
// 	return wc.cancelWorkflowRunJobs(ctx, tenantId, sqlchelpers.UUIDToStr(groupKeyRun.WorkflowRunId), reason)
// }

// func (wc *WorkflowsControllerImpl) cancelWorkflowRunJobs(ctx context.Context, tenantId string, workflowRunId string, reason string) error {

// 	workflowRun, err := wc.repo.WorkflowRun().GetWorkflowRunById(ctx, tenantId, workflowRunId)
// 	if err != nil {
// 		return fmt.Errorf("could not get workflow run (%s) : %w", workflowRunId, err)
// 	}

// 	jobRuns, err := wc.repo.JobRun().GetJobRunsByWorkflowRunId(ctx, tenantId, workflowRunId)

// 	if err != nil {
// 		return fmt.Errorf("could not cancel workflow run jobs: %w", err)
// 	}
// 	var returnErr error

// 	for i := range jobRuns {
// 		// don't cancel job runs that are onFailure

// 		if workflowRun.WorkflowVersion.OnFailureJobId.Valid && jobRuns[i].JobId == workflowRun.WorkflowVersion.OnFailureJobId {
// 			continue
// 		}

// 		jobRunId := sqlchelpers.UUIDToStr(jobRuns[i].ID)

// 		err := wc.mq.SendMessage(
// 			context.Background(),
// 			msgqueue.JOB_PROCESSING_QUEUE,
// 			tasktypes.JobRunCancelledToTask(tenantId, jobRunId, &reason),
// 		)

// 		if err != nil {
// 			returnErr = multierror.Append(err, fmt.Errorf("could not add job run to task queue: %w", err))
// 		}
// 	}

// 	return returnErr
// }

// func (wc *WorkflowsControllerImpl) runTenantProcessWorkflowRunEvents(ctx context.Context) func() {
// 	return func() {
// 		wc.l.Debug().Msgf("partition: processing workflow run events")

// 		// list all tenants
// 		tenants, err := wc.repo.Tenant().ListTenantsByControllerPartition(ctx, wc.p.GetControllerPartitionId())

// 		if err != nil {
// 			wc.l.Err(err).Msg("could not list tenants")
// 			return
// 		}

// 		for i := range tenants {
// 			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

// 			wc.processWorkflowEventsOps.RunOrContinue(tenantId)
// 		}
// 	}
// }

// func (wc *WorkflowsControllerImpl) runTenantUnpauseWorkflowRuns(ctx context.Context) func() {
// 	return func() {
// 		wc.l.Debug().Msgf("partition: processing unpaused workflow runs")

// 		// list all tenants
// 		tenants, err := wc.repo.Tenant().ListTenantsByControllerPartition(ctx, wc.p.GetControllerPartitionId())

// 		if err != nil {
// 			wc.l.Err(err).Msg("could not list tenants")
// 			return
// 		}

// 		for i := range tenants {
// 			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

// 			wc.unpausedWorkflowRunsOps.RunOrContinue(tenantId)
// 		}
// 	}
// }

// func (wc *WorkflowsControllerImpl) processWorkflowEvents(ctx context.Context, tenantId string) (bool, error) {
// 	ctx, span := telemetry.NewSpan(ctx, "process-workflow-events")
// 	defer span.End()

// 	dbCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
// 	defer cancel()

// 	res, err := wc.repo.WorkflowRun().ProcessWorkflowRunUpdates(dbCtx, tenantId)

// 	if err != nil {
// 		return false, fmt.Errorf("could not process step run updates: %w", err)
// 	}

// 	return res, nil
// }

// func (wc *WorkflowsControllerImpl) unpauseWorkflowRuns(ctx context.Context, tenantId string) (bool, error) {
// 	ctx, span := telemetry.NewSpan(ctx, "unpause-workflow-runs")
// 	defer span.End()

// 	dbCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
// 	defer cancel()

// 	toQueue, res, err := wc.repo.WorkflowRun().ProcessUnpausedWorkflowRuns(dbCtx, tenantId)

// 	if err != nil {
// 		return false, fmt.Errorf("could not process unpaused workflow runs: %w", err)
// 	}
// 	if toQueue != nil {
// 		errGroup := new(errgroup.Group)

// 		for i := range toQueue {
// 			row := toQueue[i]

// 			errGroup.Go(func() error {
// 				workflowRunId := sqlchelpers.UUIDToStr(row.WorkflowRun.ID)

// 				wc.l.Info().Msgf("popped workflow run %s", workflowRunId)

// 				ssr, err := wc.repo.WorkflowRun().QueueWorkflowRunJobs(ctx, tenantId, workflowRunId)

// 				if err != nil {
// 					return fmt.Errorf("could not queue workflow run jobs: %w", err)
// 				}

// 				for _, stepRunCp := range ssr {
// 					err = wc.mq.SendMessage(
// 						ctx,
// 						msgqueue.JOB_PROCESSING_QUEUE,
// 						tasktypes.StepRunQueuedToTask(stepRunCp),
// 					)
// 					if err != nil {
// 						return fmt.Errorf("could not queue step run: %w", err)
// 					}
// 				}

// 				return nil
// 			})
// 		}

// 		if err := errGroup.Wait(); err != nil {
// 			return false, fmt.Errorf("unpauseWorkflows could not start step runs: %w", err)
// 		}

// 	}

// 	return res, nil
// }
