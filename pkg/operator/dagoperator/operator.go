package dagoperator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/listutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// defaultOperatorSlots is the worker slot count used when a DAG operator does not configure one.
const defaultOperatorSlots = 100

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
	// to defaultOperatorSlots when unset.
	Slots int `json:"slots"`
}

// SlotConfig returns the worker slot config (slot_type -> max units) for a DAG operator,
// derived from its stored config. It is used by the manager to provision the operator's
// worker and may vary between operators.
func SlotConfig(op *sqlcv1.V1Operator) (map[string]int32, error) {
	var cfg DAGOperatorConfig

	if err := json.Unmarshal(op.Config, &cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal operator config: %w", err)
	}

	slots := cfg.Slots

	if slots <= 0 {
		slots = defaultOperatorSlots
	}

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
}

// NewDAGOperator constructs a DAG operator and starts a goroutine that polls the database for
// the tenant's DAG workflows, registering each as a worker action so matching tasks are routed
// to it. The action set is data-driven (not static config), so it is refreshed on a ticker the
// same way the HTTP operator refreshes actions from its healthcheck.
func NewDAGOperator(op *sqlcv1.V1Operator, l *zerolog.Logger, repo repository.Repository, taskEventWriter operator.TaskEventWriter, workerId uuid.UUID) (*DAGOperator, error) {
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
	pollCtx, cancel := context.WithTimeout(ctx, workflowPollTimeout)
	defer cancel()

	actions, err := d.repo.Operators().ListDAGOrchestrationActions(pollCtx, d.TenantId())

	if err != nil {
		d.Logger().Error().Err(err).Msg("could not list dag orchestration actions for operator")
		return
	}

	if listutils.AreUnorderedEqual(actions, d.lastActions) {
		return
	}

	if err := d.UpdateWorkerActions(ctx, actions); err != nil {
		d.Logger().Error().Err(err).Msg("could not update dag operator worker actions")
		return
	}

	d.lastActions = actions

	d.Logger().Debug().Strs("actions", actions).Msg("updated dag operator worker actions from workflows")
}

func (d *DAGOperator) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	release := d.RecordTask()
	defer release()

	switch action.ActionType {
	case contracts.ActionType_START_STEP_RUN:
		return d.run(ctx, action)
	default:
		// TODO: support CANCEL_STEP_RUN and START_GET_GROUP_KEY. Until then, acknowledge
		// without doing anything.
		d.Logger().Warn().
			Str("action_type", action.ActionType.String()).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("dag operator received unsupported action type; skipping")

		return nil
	}
}

// run uses d.ctx (the operator's lifetime context) rather than the dispatcher's delivery
// context, which has a short timeout that would cancel long-running DAGs mid-flight.
func (d *DAGOperator) run(deliveryCtx context.Context, action *contracts.AssignedAction) error {
	if err := d.SendStarted(action); err != nil {
		d.Logger().Error().Err(err).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("could not report task started")
	}

	externalId, err := uuid.Parse(action.TaskRunExternalId)

	if err != nil {
		return d.fail(action, fmt.Errorf("could not parse task run external id %q: %w", action.TaskRunExternalId, err), false)
	}

	tasks, err := d.buildDAG(d.ctx, action)

	if err != nil {
		return d.fail(action, fmt.Errorf("could not build dag: %w", err), false)
	}

	requestCh, responseCh, err := d.RegisterDurableTask(d.ctx, externalId)

	if err != nil {
		return d.fail(action, fmt.Errorf("could not register durable task: %w", err), false)
	}

	defer close(requestCh)

	requestCh <- &v1contracts.DurableTaskRequest{
		Message: &v1contracts.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1contracts.DurableTaskRequestRegisterWorker{
				WorkerId: d.WorkerId().String(),
			},
		},
	}

	select {
	case <-d.ctx.Done():
		return d.fail(action, fmt.Errorf("operator shutting down waiting for register worker ack: %w", d.ctx.Err()), false)
	case _, ok := <-responseCh:
		if !ok {
			return d.fail(action, fmt.Errorf("response channel closed waiting for register worker ack"), false)
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

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled bool) (*operator.DAGStepTriggerResult, error) {
		return d.TriggerDAGStep(ctx, &operator.DAGStepTriggerRequest{
			ParentTaskExternalId: externalId,
			InvocationCount:      action.GetDurableTaskInvocationCount(),
			WorkflowName:         workflowName,
			ActionId:             actionId,
			ChildIndex:           childIndex,
			Input:                workflowInput,
			AdditionalMetadata:   additionalMetadata,
			DagParentTaskRunIds:  parentTaskRunIds,
			IsSkipped:            isSkipped,
			IsCancelled:          isCancelled,
			DesiredWorkerLabels:  payloadWrapper.DesiredWorkerLabels,
		})
	}

	dagErr := dagDurableTask(
		d.ctx,
		tasks,
		externalId,
		action.GetDurableTaskInvocationCount(),
		action.ActionPayload,
		requestCh,
		responseCh,
		d.repo.Matches().EvalBoolExpr,
		triggerStep,
	)

	if dagErr != nil {
		if isDagCancelledErr(dagErr) {
			return d.cancelDAG(action, dagErr.Error())
		}
		// A child task failing is a terminal DAG outcome that replay reproduces deterministically,
		// so it must not be retried; anything else (operational errors) remains retriable.
		return d.fail(action, fmt.Errorf("dag failed: %w", dagErr), isDagChildFailedErr(dagErr))
	}

	output := make(map[string]json.RawMessage, len(tasks))
	for _, t := range tasks {
		if t.output != nil {
			b, err := json.Marshal(t.output)
			if err == nil {
				output[t.readableId] = json.RawMessage(b)
			}
		}
	}

	outputBytes, err := json.Marshal(output)
	if err != nil {
		outputBytes = []byte("{}")
	}

	if err := d.SendCompleted(action, outputBytes); err != nil {
		return fmt.Errorf("could not report task completion: %w", err)
	}

	return nil
}

func (d *DAGOperator) fail(action *contracts.AssignedAction, err error, shouldNotRetry bool) error {
	if reportErr := d.SendFailed(action, err.Error(), shouldNotRetry); reportErr != nil {
		d.Logger().Error().Err(reportErr).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("could not report task failure")
		return err
	}

	return nil
}

func (d *DAGOperator) cancelDAG(action *contracts.AssignedAction, msg string) error {
	if reportErr := d.SendCancelled(action, msg); reportErr != nil {
		return fmt.Errorf("could not report task cancellation for task run id %s: %w", action.TaskRunExternalId, reportErr)
	}

	return nil
}

func (d *DAGOperator) buildDAG(ctx context.Context, action *contracts.AssignedAction) ([]*task, error) {
	versionIdStr := action.GetWorkflowVersionId()

	if versionIdStr == "" {
		return nil, fmt.Errorf("action is missing workflow_version_id")
	}

	versionId, err := uuid.Parse(versionIdStr)

	if err != nil {
		return nil, fmt.Errorf("invalid workflow_version_id %q: %w", versionIdStr, err)
	}

	steps, err := d.repo.Workflows().ListStepsByWorkflowVersionId(ctx, d.TenantId(), versionId)

	if err != nil {
		return nil, fmt.Errorf("could not list steps for workflow version %s: %w", versionId, err)
	}

	tasksByStepId := make(map[uuid.UUID]*task, len(steps))
	tasks := make([]*task, 0, len(steps))
	stepIds := make([]uuid.UUID, 0, len(steps))

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
		tasksByStepId[s.ID] = t
		tasks = append(tasks, t)
		stepIds = append(stepIds, s.ID)
		taskIndex++
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
			return nil, fmt.Errorf("could not list step match conditions for workflow version %s: %w", versionId, err)
		}

		for _, cond := range stepConditions {
			if t, ok := tasksByStepId[cond.StepID]; ok {
				t.stepConditions = append(t.stepConditions, cond)
			}
		}
	}

	return tasks, nil
}
