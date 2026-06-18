package dagoperator

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
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

	return map[string]int32{repository.SlotTypeDefault: int32(slots)}, nil
}

type DAGOperator struct {
	*operator.SharedOperator[DAGOperatorConfig]

	// repo is used to list the tenant's DAG workflows when refreshing registered actions.
	repo repository.Repository

	// cancel stops the workflow-polling goroutine on Cleanup/Drain.
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

	workflowIds, err := d.repo.Operators().ListDAGWorkflowIds(pollCtx, d.TenantId())

	if err != nil {
		d.Logger().Error().Err(err).Msg("could not list dag workflows for operator")
		return
	}

	actions := make([]string, len(workflowIds))

	for i, id := range workflowIds {
		actions[i] = id.String()
	}

	if slicesEqualUnordered(actions, d.lastActions) {
		return
	}

	if err := d.UpdateWorkerActions(ctx, actions); err != nil {
		d.Logger().Error().Err(err).Msg("could not update dag operator worker actions")
		return
	}

	d.lastActions = actions

	d.Logger().Debug().Strs("actions", actions).Msg("updated dag operator worker actions from workflows")
}

// slicesEqualUnordered reports whether a and b contain the same elements regardless of order.
func slicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	ac := slices.Clone(a)
	bc := slices.Clone(b)
	slices.Sort(ac)
	slices.Sort(bc)

	return slices.Equal(ac, bc)
}

func (d *DAGOperator) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	// Track this task so Drain/Cleanup wait for it before the operator shuts down.
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

// run opens a durable-task session for the assigned action and drives the DAG to completion.
// The DAG graph itself is built by buildDAG; the orchestration loop (dagDurableTask) is
// unchanged core logic.
func (d *DAGOperator) run(ctx context.Context, action *contracts.AssignedAction) error {
	// Report STARTED so the task is marked running. Best-effort: a failed report shouldn't
	// prevent the actual work.
	if err := d.SendStarted(action); err != nil {
		d.Logger().Error().Err(err).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("could not report task started")
	}

	externalId, err := uuid.Parse(action.TaskRunExternalId)

	if err != nil {
		return d.fail(action, fmt.Errorf("could not parse task run external id %q: %w", action.TaskRunExternalId, err))
	}

	tasks, err := buildDAG(action)

	if err != nil {
		return d.fail(action, fmt.Errorf("could not build dag: %w", err))
	}

	requestCh, responseCh, err := d.RegisterDurableTask(ctx, externalId)

	if err != nil {
		return d.fail(action, fmt.Errorf("could not register durable task: %w", err))
	}

	// dagDurableTask runs the orchestration loop until the DAG completes or ctx is cancelled.
	// Closing requestCh tears down the dispatcher-side session.
	defer close(requestCh)

	dagDurableTask(ctx, tasks, requestCh, responseCh)

	if err := ctx.Err(); err != nil {
		return d.fail(action, fmt.Errorf("dag did not complete: %w", err))
	}

	// The DAG completed. The dispatcher requires valid JSON output.
	if err := d.SendCompleted(action, []byte("{}")); err != nil {
		return fmt.Errorf("could not report task completion: %w", err)
	}

	return nil
}

// fail reports a task failure (retryable) and returns the originating error.
func (d *DAGOperator) fail(action *contracts.AssignedAction, err error) error {
	if reportErr := d.SendFailed(action, err.Error(), false); reportErr != nil {
		d.Logger().Error().Err(reportErr).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("could not report task failure")
	}

	return err
}

// buildDAG constructs the task graph for an assigned action. Parsing the DAG definition out
// of the action payload is part of the core DAG logic and is not yet implemented; this seam
// keeps the operator plumbing wired up until that lands.
func buildDAG(_ *contracts.AssignedAction) ([]*task, error) {
	return nil, fmt.Errorf("buildDAG is not implemented yet")
}
