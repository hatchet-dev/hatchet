package workflows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type WorkflowsController interface {
	Start(ctx context.Context) error
}

type WorkflowsControllerImpl struct {
	mq            msgqueue.MessageQueue
	l             *zerolog.Logger
	repo          repository.EngineRepository
	dv            datautils.DataDecoderValidator
	s             gocron.Scheduler
	tenantAlerter *alerting.TenantAlertManager
	a             *hatcheterrors.Wrapped
	p             *partition.Partition
}

type WorkflowsControllerOpt func(*WorkflowsControllerOpts)

type WorkflowsControllerOpts struct {
	mq      msgqueue.MessageQueue
	l       *zerolog.Logger
	repo    repository.EngineRepository
	dv      datautils.DataDecoderValidator
	ta      *alerting.TenantAlertManager
	alerter hatcheterrors.Alerter
	p       *partition.Partition
}

func defaultWorkflowsControllerOpts() *WorkflowsControllerOpts {
	logger := logger.NewDefaultLogger("workflows-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &WorkflowsControllerOpts{
		l:       &logger,
		dv:      datautils.NewDataDecoderValidator(),
		alerter: alerter,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.l = l
	}
}

func WithRepository(r repository.EngineRepository) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.repo = r
	}
}

func WithAlerter(a hatcheterrors.Alerter) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.alerter = a
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.dv = dv
	}
}

func WithTenantAlerter(ta *alerting.TenantAlertManager) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.ta = ta
	}
}

func WithPartition(p *partition.Partition) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.p = p
	}
}

func New(fs ...WorkflowsControllerOpt) (*WorkflowsControllerImpl, error) {
	opts := defaultWorkflowsControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.ta == nil {
		return nil, fmt.Errorf("tenant alerter is required. use WithTenantAlerter")
	}

	if opts.p == nil {
		return nil, fmt.Errorf("partition is required. use WithPartition")
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	newLogger := opts.l.With().Str("service", "workflows-controller").Logger()
	opts.l = &newLogger

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "workflows-controller"})

	return &WorkflowsControllerImpl{
		mq:            opts.mq,
		l:             opts.l,
		repo:          opts.repo,
		dv:            opts.dv,
		s:             s,
		tenantAlerter: opts.ta,
		a:             a,
		p:             opts.p,
	}, nil
}

func (wc *WorkflowsControllerImpl) Start() (func() error, error) {
	wc.l.Debug().Msg("starting workflows controller")

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	_, err := wc.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			wc.runGetGroupKeyRunRequeue(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule get group key run requeue: %w", err)
	}

	_, err = wc.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			wc.runGetGroupKeyRunReassign(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule get group key run reassign: %w", err)
	}

	_, err = wc.s.NewJob(
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(
			wc.runPollActiveQueues(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not poll active queues: %w", err)
	}

	wc.s.Start()

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := wc.handleTask(context.Background(), task)
		if err != nil {
			wc.l.Error().Err(err).Msg("could not handle workflow task")
			return err
		}

		return nil
	}

	cleanupQueue, err := wc.mq.Subscribe(msgqueue.WORKFLOW_PROCESSING_QUEUE, f, msgqueue.NoOpHook)

	if err != nil {
		cancel()
		return nil, err
	}

	cleanup := func() error {
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup queue: %w", err)
		}

		wg.Wait()

		if err := wc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		return nil
	}

	return cleanup, nil
}

func (wc *WorkflowsControllerImpl) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(wc.l, wc.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch task.ID {
	case "replay-workflow-run":
		return wc.handleReplayWorkflowRun(ctx, task)
	case "workflow-run-queued":
		return wc.handleWorkflowRunQueued(ctx, task)
	case "get-group-key-run-started":
		return wc.handleGroupKeyRunStarted(ctx, task)
	case "get-group-key-run-finished":
		return wc.handleGroupKeyRunFinished(ctx, task)
	case "get-group-key-run-failed":
		return wc.handleGroupKeyRunFailed(ctx, task)
	case "get-group-key-run-timed-out":
		return wc.handleGetGroupKeyRunTimedOut(ctx, task)
	case "workflow-run-finished":
		return wc.handleWorkflowRunFinished(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (wc *WorkflowsControllerImpl) handleReplayWorkflowRun(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "replay-workflow-run") // nolint:ineffassign
	defer span.End()

	payload := tasktypes.ReplayWorkflowRunTaskPayload{}
	metadata := tasktypes.ReplayWorkflowRunTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode replay workflow run task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode replay workflow run task metadata: %w", err)
	}

	_, err = wc.repo.WorkflowRun().ReplayWorkflowRun(ctx, metadata.TenantId, payload.WorkflowRunId)

	if err != nil {
		return fmt.Errorf("could not replay workflow run: %w", err)
	}

	// push a task that the workflow run is queued
	return wc.mq.AddMessage(
		ctx,
		msgqueue.WORKFLOW_PROCESSING_QUEUE,
		tasktypes.WorkflowRunQueuedToTask(metadata.TenantId, payload.WorkflowRunId),
	)
}

func (ec *WorkflowsControllerImpl) handleGroupKeyRunStarted(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "get-group-key-run-started") // nolint:ineffassign
	defer span.End()

	payload := tasktypes.GetGroupKeyRunStartedTaskPayload{}
	metadata := tasktypes.GetGroupKeyRunStartedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode group key run started task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode group key run started task metadata: %w", err)
	}

	// update the get group key run in the database
	startedAt, err := time.Parse(time.RFC3339, payload.StartedAt)

	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	_, err = ec.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
		StartedAt: &startedAt,
		Status:    repository.StepRunStatusPtr(db.StepRunStatusRunning),
	})

	return err
}

func (wc *WorkflowsControllerImpl) handleGroupKeyRunFinished(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-group-key-run-finished")
	defer span.End()

	payload := tasktypes.GetGroupKeyRunFinishedTaskPayload{}
	metadata := tasktypes.GetGroupKeyRunFinishedTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode group key run finished task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode group key run finished task metadata: %w", err)
	}

	// update the group key run in the database
	finishedAt, err := time.Parse(time.RFC3339, payload.FinishedAt)

	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	groupKeyRun, err := wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
		FinishedAt: &finishedAt,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusSucceeded),
		Output:     &payload.GroupKey,
	})

	if err != nil {
		return fmt.Errorf("could not update group key run: %w", err)
	}

	errGroup := new(errgroup.Group)

	errGroup.Go(func() error {
		workflowVersionId := sqlchelpers.UUIDToStr(groupKeyRun.WorkflowVersionId)

		err := wc.bumpQueue(ctx, metadata.TenantId, workflowVersionId, &payload.GroupKey)

		return err
	})

	return errGroup.Wait()
}

func (wc *WorkflowsControllerImpl) runPollActiveQueues(ctx context.Context) func() {
	return func() {

		wc.l.Debug().Msg("polling active queues")

		toQueueList, err := wc.repo.WorkflowRun().ListActiveQueuedWorkflowVersions(ctx)

		if err != nil {
			wc.l.Error().Err(err).Msg("could not list active queued workflow versions")
			return
		}

		errGroup := new(errgroup.Group)

		for i := range toQueueList {
			toQueue := toQueueList[i]
			errGroup.Go(func() error {
				workflowVersionId := sqlchelpers.UUIDToStr(toQueue.WorkflowVersionId)
				tenantId := sqlchelpers.UUIDToStr(toQueue.TenantId)
				var key *string
				if toQueue.ConcurrencyGroupId.Valid {
					key = &toQueue.ConcurrencyGroupId.String
				}
				err := wc.bumpQueue(ctx, tenantId, workflowVersionId, key)
				return err
			})
		}

		err = errGroup.Wait()

		if err != nil {
			wc.l.Error().Err(err)
		}
	}
}
func (wc *WorkflowsControllerImpl) bumpQueue(ctx context.Context, tenantId string, workflowVersionId string, groupKey *string) error {

	workflowVersion, err := wc.repo.Workflow().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

	if err != nil {
		return fmt.Errorf("could not get workflow version: %w", err)
	}

	if workflowVersion.ConcurrencyLimitStrategy.Valid {
		switch workflowVersion.ConcurrencyLimitStrategy.ConcurrencyLimitStrategy {
		case dbsqlc.ConcurrencyLimitStrategyCANCELINPROGRESS:
			if groupKey == nil {
				return fmt.Errorf("group key is required for cancel in progress strategy")
			}
			err = wc.queueByCancelInProgress(ctx, tenantId, *groupKey, workflowVersion)
		case dbsqlc.ConcurrencyLimitStrategyGROUPROUNDROBIN:
			err = wc.queueByGroupRoundRobin(ctx, tenantId, workflowVersion)
		default:
			return fmt.Errorf("unimplemented concurrency limit strategy: %s", workflowVersion.ConcurrencyLimitStrategy.ConcurrencyLimitStrategy)
		}
	}

	return err
}

func (wc *WorkflowsControllerImpl) handleGroupKeyRunFailed(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-group-key-run-failed") // nolint: ineffassign
	defer span.End()

	payload := tasktypes.GetGroupKeyRunFailedTaskPayload{}
	metadata := tasktypes.GetGroupKeyRunFailedTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode group key run failed task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode group key run failed task metadata: %w", err)
	}

	// update the group key run in the database
	failedAt, err := time.Parse(time.RFC3339, payload.FailedAt)
	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	_, err = wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
		FinishedAt: &failedAt,
		Error:      &payload.Error,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusFailed),
	})

	if err != nil {
		return fmt.Errorf("could not update get group key run: %w", err)
	}

	return nil
}

func (wc *WorkflowsControllerImpl) handleGetGroupKeyRunTimedOut(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-get-group-key-run-timed-out")
	defer span.End()

	payload := tasktypes.GetGroupKeyRunTimedOutTaskPayload{}
	metadata := tasktypes.GetGroupKeyRunTimedOutTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode get group key run run timed out task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode get group key run run timed out task metadata: %w", err)
	}

	return wc.cancelGetGroupKeyRun(ctx, metadata.TenantId, payload.GetGroupKeyRunId, "TIMED_OUT")
}

func (wc *WorkflowsControllerImpl) cancelGetGroupKeyRun(ctx context.Context, tenantId, getGroupKeyRunId, reason string) error {
	ctx, span := telemetry.NewSpan(ctx, "cancel-get-group-key-run") // nolint: ineffassign
	defer span.End()

	// cancel current step run
	now := time.Now().UTC()

	groupKeyRun, err := wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(ctx, tenantId, getGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
		CancelledAt:     &now,
		CancelledReason: repository.StringPtr(reason),
		Status:          repository.StepRunStatusPtr(db.StepRunStatusCancelled),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	// cancel all existing jobs on the workflow run
	workflowRunId := sqlchelpers.UUIDToStr(groupKeyRun.WorkflowRunId)

	workflowRun, err := wc.repo.WorkflowRun().GetWorkflowRunById(ctx, tenantId, workflowRunId)

	if err != nil {
		return fmt.Errorf("could not get workflow run: %w", err)
	}

	return wc.cancelWorkflowRunJobs(ctx, workflowRun)
}
