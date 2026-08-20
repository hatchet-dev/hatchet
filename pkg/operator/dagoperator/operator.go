package dagoperator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	telemetry_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/internal/listutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const (
	// workflowPollInterval is how often the operator queries the database for the tenant's DAG
	// workflows to keep its registered actions in sync.
	workflowPollInterval = 5 * time.Second

	// workflowPollTimeout bounds a single workflow-listing poll.
	workflowPollTimeout = 10 * time.Second
)

// DAGOperatorConfig is the stored config for a DAG operator. Unlike the HTTP operator, the
// action set is not configured statically: the operator polls the database for the tenant's
// DAG workflows and registers each as an action (see pollWorkflows).
type DAGOperatorConfig struct {
	// Slots is the number of concurrent task slots the operator's worker advertises. Defaults
	// to the server's configured default (WithSlots) when unset.
	Slots int `json:"slots"`
}

func resolveSlots(cfgSlots, defaultSlots int) int {
	if cfgSlots > 0 {
		return cfgSlots
	}

	return defaultSlots
}

func SlotConfig(op *sqlcv1.V1Operator, defaultSlots int) (map[string]int32, error) {
	var cfg DAGOperatorConfig

	if err := json.Unmarshal(op.Config, &cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal operator config: %w", err)
	}

	slots := resolveSlots(cfg.Slots, defaultSlots)

	return map[string]int32{repository.SlotTypeDurable: int32(slots)}, nil
}

type DAGOperator struct {
	*operator.SharedOperator[DAGOperatorConfig]

	// repo is used to list the tenant's DAG workflows when refreshing registered actions.
	repo repository.Repository

	// ctx and cancel bound the operator's lifetime. Used by run() so that durable task
	// sessions aren't subject to the dispatcher's short per-delivery context deadline.
	ctx    context.Context
	cancel context.CancelFunc

	// lastActions is the most recently registered action set, used to avoid redundant
	// dispatcher writes when the workflow list is unchanged. Only the polling goroutine
	// touches it.
	lastActions []string

	// runCancels lets a CANCEL_STEP_RUN action stop one run without cancelling d.ctx.
	runCancels syncx.Map[string, context.CancelFunc]

	slots int
}

type DAGOperatorOpt func(*DAGOperator)

// WithSlots sets the server-wide default slot count (SERVER_DAG_OPERATOR_DEFAULT_SLOTS) used
// when the operator's stored config doesn't set its own Slots.
func WithSlots(defaultSlots int) DAGOperatorOpt {
	return func(d *DAGOperator) {
		d.slots = defaultSlots
	}
}

// NewDAGOperator constructs a DAG operator and starts a goroutine that polls the database for
// the tenant's DAG workflows, registering each as a worker action so matching tasks are routed
// to it. The action set is data-driven (not static config), so it is refreshed on a ticker the
// same way the HTTP operator refreshes actions from its healthcheck.
func NewDAGOperator(op *sqlcv1.V1Operator, l *zerolog.Logger, repo repository.Repository, taskEventWriter operator.TaskEventWriter, workerId uuid.UUID, opts ...DAGOperatorOpt) (*DAGOperator, error) {
	shared, err := operator.NewSharedOperator(op, l, repo, taskEventWriter, workerId, DAGOperatorConfig{})

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &DAGOperator{
		SharedOperator: shared,
		repo:           repo,
		ctx:            ctx,
		cancel:         cancel,
	}

	for _, opt := range opts {
		opt(d)
	}

	d.slots = resolveSlots(shared.Config().Slots, d.slots)

	go d.pollWorkflows(ctx)

	return d, nil
}

// Cleanup stops the workflow poller in addition to the shared operator's teardown.
func (d *DAGOperator) Cleanup() {
	if d.cancel != nil {
		d.cancel()
	}

	d.SharedOperator.Cleanup()
}

// Drain stops the workflow poller and drains in-flight tasks without pausing the worker (used
// for bulk teardown, where the caller pauses all operator workers in one query).
func (d *DAGOperator) Drain() {
	if d.cancel != nil {
		d.cancel()
	}

	d.SharedOperator.Drain()
}

// pollWorkflows periodically refreshes the worker's registered actions from the tenant's DAG
// workflows in the database.
func (d *DAGOperator) pollWorkflows(ctx context.Context) {
	// Refresh once up front so the worker registers its actions without waiting a full tick.
	d.refreshActions(ctx)

	ticker := time.NewTicker(workflowPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.refreshActions(ctx)
		}
	}
}

// refreshActions lists the tenant's DAG workflows and registers each workflow id as an action,
// skipping the dispatcher write when the set is unchanged.
func (d *DAGOperator) refreshActions(ctx context.Context) {
	ctx, span := telemetry.NewSpan(ctx, "dagoperator.refreshActions")
	defer span.End()

	pollCtx, cancel := context.WithTimeout(ctx, workflowPollTimeout)
	defer cancel()

	actions, err := d.repo.Operators().ListDAGOrchestrationActions(pollCtx, d.TenantId())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "could not list dag orchestration actions")
		d.Logger().Error().Err(err).Msg("could not list dag orchestration actions for operator")
		return
	}

	span.SetAttributes(attribute.Int("dagoperator.action_count", len(actions)))

	if listutils.AreUnorderedEqual(actions, d.lastActions) {
		span.SetAttributes(attribute.Bool("dagoperator.actions_changed", false))
		return
	}

	if err := d.UpdateWorkerActions(ctx, actions); err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "could not update dag operator worker actions")
		d.Logger().Error().Err(err).Msg("could not update dag operator worker actions")
		return
	}

	span.SetAttributes(attribute.Bool("dagoperator.actions_changed", true))

	d.lastActions = actions

	d.Logger().Debug().Strs("actions", actions).Msg("updated dag operator worker actions from workflows")
}

func (d *DAGOperator) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	ctx, span := telemetry.NewSpan(ctx, "dagoperator.HandleAction")
	defer span.End()

	span.SetAttributes(
		attribute.String("dagoperator.action_type", action.ActionType.String()),
		attribute.String("dagoperator.task_run_external_id", action.TaskRunExternalId),
	)

	switch action.ActionType {
	case contracts.ActionType_START_STEP_RUN:
		return d.startRun(ctx, action)
	case contracts.ActionType_CANCEL_STEP_RUN:
		release := d.RecordTask()
		defer release()

		return d.cancelRun(ctx, action)
	default:
		release := d.RecordTask()
		defer release()

		d.Logger().Warn().
			Str("action_type", action.ActionType.String()).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("dag operator received unsupported action type; skipping")

		return nil
	}
}

func (d *DAGOperator) startRun(ctx context.Context, action *contracts.AssignedAction) error {
	_, span := telemetry.NewSpan(ctx, "dagoperator.startRun")
	defer span.End()

	span.SetAttributes(attribute.String("dagoperator.task_run_external_id", action.TaskRunExternalId))

	release := d.RecordTask()

	go func() {
		defer release()

		if err := d.run(action); err != nil {
			d.Logger().Error().Err(err).
				Str("task_run_external_id", action.TaskRunExternalId).
				Msg("dag orchestration ended with error")
		}
	}()

	return nil
}

func (d *DAGOperator) cancelRun(ctx context.Context, action *contracts.AssignedAction) error {
	_, span := telemetry.NewSpan(ctx, "dagoperator.cancelRun")
	defer span.End()

	span.SetAttributes(attribute.String("dagoperator.task_run_external_id", action.TaskRunExternalId))

	cancel, ok := d.runCancels.Load(action.TaskRunExternalId)
	if !ok {
		span.SetAttributes(attribute.Bool("dagoperator.run_found", false))

		d.Logger().Warn().
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("dag operator received cancel for a run it has no record of; it may have already finished")

		// No in-flight run() goroutine exists to report the terminal event via
		// handleRunCancellation, so report it here instead.
		return d.SendCancelled(action)
	}

	span.SetAttributes(attribute.Bool("dagoperator.run_found", true))

	cancel()

	return nil
}

func (d *DAGOperator) run(action *contracts.AssignedAction) error {
	// Rooted here rather than derived from the dispatcher's HandleAction context: run() outlives
	// that context, driven instead by d.ctx/runCtx for the lifetime of the durable task session.
	runSpanCtx, span := telemetry.NewSpan(d.ctx, "dagoperator.run")
	defer span.End()

	span.SetAttributes(
		attribute.String("dagoperator.task_run_external_id", action.TaskRunExternalId),
		attribute.String("dagoperator.workflow_version_id", action.GetWorkflowVersionId()),
		attribute.Int("dagoperator.durable_invocation_count", int(action.GetDurableTaskInvocationCount())),
	)

	externalId, err := uuid.Parse(action.TaskRunExternalId)

	if err != nil {
		return d.fail(span, action, fmt.Errorf("could not parse task run external id %q: %w", action.TaskRunExternalId, err), false)
	}

	startedReported := make(chan struct{})

	go func() {
		defer close(startedReported)

		if err := d.SendStarted(action); err != nil {
			d.Logger().Error().Err(err).
				Str("task_run_external_id", action.TaskRunExternalId).
				Msg("could not report task started")
		}
	}()

	awaitStarted := func() { <-startedReported }

	defer awaitStarted()

	// Scoped to this run so a targeted cancel doesn't tear down every run on this operator.
	// Derived from runSpanCtx so buildDAG/RegisterDurableTask/dagDurableTask spans nest under
	// dagoperator.run instead of starting new traces.
	runCtx, runCancel := context.WithCancel(runSpanCtx)
	d.runCancels.Store(action.TaskRunExternalId, runCancel)
	defer func() {
		d.runCancels.Delete(action.TaskRunExternalId)
		runCancel()
	}()

	tasks, onFailureTask, err := d.buildDAG(runCtx, action)

	awaitStarted()

	if err != nil {
		return d.fail(span, action, fmt.Errorf("could not build dag: %w", err), false)
	}

	workflowVersionId, err := uuid.Parse(action.GetWorkflowVersionId())

	if err != nil {
		return d.fail(span, action, fmt.Errorf("invalid workflow_version_id %q: %w", action.GetWorkflowVersionId(), err), false)
	}

	requestCh, responseCh, err := d.RegisterDurableTask(runCtx, externalId)

	if err != nil {
		return d.fail(span, action, fmt.Errorf("could not register durable task: %w", err), false)
	}

	defer close(requestCh)

	select {
	case requestCh <- &v1contracts.DurableTaskRequest{
		Message: &v1contracts.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1contracts.DurableTaskRequestRegisterWorker{
				WorkerId: d.WorkerId().String(),
			},
		},
	}:
	case <-runCtx.Done():
		return d.handleRunCancellation(span, action, nil, fmt.Errorf("run interrupted before register worker request could be sent: %w", runCtx.Err()))
	}

	select {
	case <-runCtx.Done():
		return d.handleRunCancellation(span, action, nil, fmt.Errorf("run interrupted waiting for register worker ack: %w", runCtx.Err()))
	case _, ok := <-responseCh:
		if !ok {
			return d.fail(span, action, fmt.Errorf("response channel closed waiting for register worker ack"), false)
		}
	}

	var payloadWrapper struct {
		Input               json.RawMessage               `json:"input"`
		DesiredWorkerLabels []*sqlcv1.GetDesiredLabelsRow `json:"desired_worker_labels"`
	}
	workflowInput := "{}"
	if err := json.Unmarshal([]byte(action.ActionPayload), &payloadWrapper); err == nil && len(payloadWrapper.Input) > 0 {
		workflowInput = string(payloadWrapper.Input)
	}

	var additionalMetadata []byte
	if meta := action.GetAdditionalMetadata(); meta != "" {
		additionalMetadata = []byte(meta)
	}

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		return d.TriggerDAGStep(ctx, &operator.DAGStepTriggerRequest{
			ParentTaskExternalId: externalId,
			InvocationCount:      action.GetDurableTaskInvocationCount(),
			WorkflowName:         workflowName,
			WorkflowVersionId:    workflowVersionId,
			ActionId:             actionId,
			ChildIndex:           childIndex,
			Input:                workflowInput,
			AdditionalMetadata:   additionalMetadata,
			DagParentTaskRunIds:  parentTaskRunIds,
			IsSkipped:            isSkipped,
			IsCancelled:          isCancelled,
			DesiredWorkerLabels:  payloadWrapper.DesiredWorkerLabels,
			ParentReExecuted:     parentReExecuted,
		})
	}

	dagErr := dagDurableTask(
		runCtx,
		tasks,
		onFailureTask,
		externalId,
		action.GetDurableTaskInvocationCount(),
		action.ActionPayload,
		requestCh,
		responseCh,
		d.repo.Matches().EvalBoolExpr,
		triggerStep,
	)

	// Fold in if triggered, so output collection and cancellation below see it too.
	if onFailureTask != nil && onFailureTask.isTriggered {
		tasks = append(tasks, onFailureTask)
	}

	if dagErr != nil {
		if isDagCancelledErr(dagErr) {
			return d.cancelDAG(span, action, dagErr.Error())
		}
		if errors.Is(dagErr, context.Canceled) {
			return d.handleRunCancellation(span, action, tasks, dagErr)
		}
		// A child task failing is a terminal DAG outcome that replay reproduces deterministically,
		// so it must not be retried; anything else (operational errors) remains retriable.
		return d.fail(span, action, fmt.Errorf("dag failed: %w", dagErr), isDagChildFailedErr(dagErr))
	}

	output := make(map[string]json.RawMessage, len(tasks))

	var completedRefs []repository.TaskExternalIdNodeIdBranchId
	refToReadableId := make(map[repository.TaskExternalIdNodeIdBranchId]string, len(tasks))

	for _, t := range tasks {
		switch {
		case t.isSkipped:
			if b, err := json.Marshal(map[string]interface{}{"skipped": true}); err == nil {
				output[t.readableId] = json.RawMessage(b)
			}
		case t.isCancelled:
			if b, err := json.Marshal(map[string]interface{}{"cancelled": true}); err == nil {
				output[t.readableId] = json.RawMessage(b)
			}
		default:
			ref := repository.TaskExternalIdNodeIdBranchId{
				TaskExternalId: externalId,
				NodeId:         t.nodeId,
				BranchId:       t.branchId,
			}
			completedRefs = append(completedRefs, ref)
			refToReadableId[ref] = t.readableId
		}
	}

	if len(completedRefs) > 0 {
		events, err := d.repo.DurableEvents().GetSatisfiedDurableEvents(d.ctx, d.TenantId(), completedRefs)
		if err != nil {
			return d.fail(span, action, fmt.Errorf("could not fetch completed task outputs: %w", err), false)
		}

		for _, ev := range events {
			readableId, ok := refToReadableId[repository.TaskExternalIdNodeIdBranchId{
				TaskExternalId: ev.TaskExternalId,
				NodeId:         ev.NodeID,
				BranchId:       ev.BranchID,
			}]
			if !ok {
				continue
			}
			output[readableId] = json.RawMessage(ev.Result)
		}
	}

	outputBytes, err := json.Marshal(output)
	if err != nil {
		outputBytes = []byte("{}")
	}

	span.SetAttributes(
		attribute.Int("dagoperator.completed_task_count", len(completedRefs)),
		attribute.Int("dagoperator.output_key_count", len(output)),
	)

	if err := d.SendCompleted(action, outputBytes); err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "could not report task completion")
		return fmt.Errorf("could not report task completion: %w", err)
	}

	return nil
}

func (d *DAGOperator) abortForShutdown(span trace.Span, action *contracts.AssignedAction, err error) error {
	span.RecordError(err)
	span.SetStatus(telemetry_codes.Error, "operator shutting down mid-run")

	d.Logger().Warn().Err(err).
		Str("task_run_external_id", action.TaskRunExternalId).
		Msg("dag operator shutting down mid-run; leaving task for reassignment")
	return err
}

func (d *DAGOperator) handleRunCancellation(span trace.Span, action *contracts.AssignedAction, tasks []*task, err error) error {
	if d.ctx.Err() != nil {
		return d.abortForShutdown(span, action, fmt.Errorf("dag orchestration interrupted by operator shutdown: %w", err))
	}

	// Already-triggered children keep running unless cancelled explicitly.
	var childExternalIds []uuid.UUID
	for _, t := range tasks {
		if t.isTriggered && !t.isCompleted && t.workflowRunExternalId != nil {
			childExternalIds = append(childExternalIds, *t.workflowRunExternalId)
		}
	}

	span.SetAttributes(attribute.Int("dagoperator.cancelled_children_count", len(childExternalIds)))

	if len(childExternalIds) > 0 {
		// runCtx/d.ctx may already be cancelled, but this cleanup should still run.
		cancelCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if cancelErr := d.CancelDAGChildren(cancelCtx, childExternalIds); cancelErr != nil {
			span.RecordError(cancelErr)

			d.Logger().Error().Err(cancelErr).
				Str("task_run_external_id", action.TaskRunExternalId).
				Msg("could not cancel dag children after run cancellation")
		}
	}

	return d.cancelDAG(span, action, err.Error())
}

func (d *DAGOperator) fail(span trace.Span, action *contracts.AssignedAction, err error, shouldNotRetry bool) error {
	span.RecordError(err)
	span.SetStatus(telemetry_codes.Error, "dag run failed")
	span.SetAttributes(attribute.Bool("dagoperator.should_not_retry", shouldNotRetry))

	if reportErr := d.SendFailed(action, err.Error(), shouldNotRetry); reportErr != nil {
		span.RecordError(reportErr)

		d.Logger().Error().Err(reportErr).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("could not report task failure")
		return err
	}

	return nil
}

func (d *DAGOperator) cancelDAG(span trace.Span, action *contracts.AssignedAction, msg string) error {
	span.SetStatus(telemetry_codes.Error, "dag run cancelled")
	span.SetAttributes(attribute.String("dagoperator.cancel_message", msg))

	if reportErr := d.SendCancelledWithMessage(action, msg); reportErr != nil {
		span.RecordError(reportErr)
		return fmt.Errorf("could not report task cancellation for task run id %s: %w", action.TaskRunExternalId, reportErr)
	}

	return nil
}

func (d *DAGOperator) buildDAG(ctx context.Context, action *contracts.AssignedAction) ([]*task, *task, error) {
	ctx, span := telemetry.NewSpan(ctx, "dagoperator.buildDAG")
	defer span.End()

	versionIdStr := action.GetWorkflowVersionId()

	if versionIdStr == "" {
		return nil, nil, fmt.Errorf("action is missing workflow_version_id")
	}

	versionId, err := uuid.Parse(versionIdStr)

	if err != nil {
		return nil, nil, fmt.Errorf("invalid workflow_version_id %q: %w", versionIdStr, err)
	}

	span.SetAttributes(attribute.String("dagoperator.workflow_version_id", versionId.String()))

	steps, err := d.repo.Workflows().ListStepsByWorkflowVersionId(ctx, d.TenantId(), versionId)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "could not list steps for workflow version")
		return nil, nil, fmt.Errorf("could not list steps for workflow version %s: %w", versionId, err)
	}

	span.SetAttributes(attribute.Int("dagoperator.step_count", len(steps)))

	tasksByStepId := make(map[uuid.UUID]*task, len(steps))
	tasks := make([]*task, 0, len(steps))
	stepIds := make([]uuid.UUID, 0, len(steps))
	var onFailureTask *task

	taskIndex := 0
	for _, s := range steps {
		if s.IsDagOrchestrator {
			continue
		}

		t := &task{
			id:           s.ID,
			actionId:     s.ActionId,
			workflowName: s.WorkflowName,
			readableId:   s.ReadableId.String,
			index:        int32(taskIndex), // nolint:gosec
		}
		taskIndex++

		// Keep it out of tasksByStepId/tasks/stepIds so it isn't gated by normal readiness.
		if s.JobKind == sqlcv1.JobKindONFAILURE {
			onFailureTask = t
			continue
		}

		tasksByStepId[s.ID] = t
		tasks = append(tasks, t)
		stepIds = append(stepIds, s.ID)
	}

	for _, s := range steps {
		t, ok := tasksByStepId[s.ID]
		if !ok {
			continue
		}
		for _, parentId := range s.Parents {
			if parent, ok := tasksByStepId[parentId]; ok {
				t.parents = append(t.parents, parent)
			}
		}
	}

	if len(stepIds) > 0 {
		stepConditions, err := d.repo.Workflows().ListStepMatchConditions(ctx, d.TenantId(), stepIds)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(telemetry_codes.Error, "could not list step match conditions")
			return nil, nil, fmt.Errorf("could not list step match conditions for workflow version %s: %w", versionId, err)
		}

		span.SetAttributes(attribute.Int("dagoperator.step_condition_count", len(stepConditions)))

		for _, cond := range stepConditions {
			if t, ok := tasksByStepId[cond.StepID]; ok {
				t.stepConditions = append(t.stepConditions, cond)
			}
		}
	}

	span.SetAttributes(
		attribute.Int("dagoperator.task_count", len(tasks)),
		attribute.Bool("dagoperator.has_on_failure", onFailureTask != nil),
	)

	return tasks, onFailureTask, nil
}
