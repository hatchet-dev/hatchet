package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// dagOrchestratorActionSuffix ends the action id of the orchestrator step the engine adds to
// a DAG workflow when the tenant runs the DAG operator.
const dagOrchestratorActionSuffix = "_orchestrator"

// dagOrchestratorActionRegex is the form DAGOrchestratorActionId produces: a lower-cased
// workflow name (the hatchetName alphabet) followed by the suffix. It admits no ":", so it is
// disjoint from the "service:verb" action ids workers register, and no ";", the separator
// the worker action hash frames ids with.
var dagOrchestratorActionRegex = regexp.MustCompile(`^[a-z0-9.\-_]+_orchestrator$`)

// DAGOrchestratorActionId is the action id of the orchestrator step of the named workflow.
// It has no ":", so ParseActionID rejects it and no SDK or GRPC worker can register it: only
// the engine's DAG operator holds orchestrator actions, and the operator service admits them
// on DAG operator sessions alone (see IsDAGOrchestratorActionId).
func DAGOrchestratorActionId(workflowName string) string {
	return strings.ToLower(workflowName + dagOrchestratorActionSuffix)
}

// IsDAGOrchestratorActionId reports whether id is an orchestrator action id as
// DAGOrchestratorActionId produces it.
func IsDAGOrchestratorActionId(id string) bool {
	return dagOrchestratorActionRegex.MatchString(id)
}

// Step names must be unique within a workflow, so the orchestrator can only take the workflow
// name when no task in the workflow already has it.
func dagOrchestratorReadableId(workflowName string, taskReadableIds []string) string {
	readableId := workflowName

	for slices.Contains(taskReadableIds, readableId) {
		readableId += dagOrchestratorActionSuffix
	}

	return readableId
}

// PutDynamicDagWorkflowVersion registers dagName as a workflow whose steps are copies of the
// nodes' standalone tasks as they are registered now, and returns the version to run. Sending
// the same nodes again returns the existing version.
func (r *workflowRepository) PutDynamicDagWorkflowVersion(ctx context.Context, tenantId uuid.UUID, dagName string, nodes []*v1contracts.DynamicDagNode) (uuid.UUID, error) {
	invalid := func(format string, args ...any) error {
		return &TriggerOptInvalidArgumentError{Err: fmt.Errorf(format, args...)}
	}

	workflowNames := []string{dagName}

	for _, node := range nodes {
		workflowNames = append(workflowNames, node.GetTaskName())
	}

	db := r.pool.ForTenant(tenantId)

	workflows, err := r.listWorkflowsByNames(ctx, db, tenantId, workflowNames)

	if err != nil {
		return uuid.Nil, err
	}

	versionIds := make([]uuid.UUID, len(workflows))

	for i, w := range workflows {
		versionIds[i] = w.WorkflowVersionId
	}

	versions, err := r.queries.GetWorkflowVersionForEngine(ctx, db, sqlcv1.GetWorkflowVersionForEngineParams{
		Tenantid: tenantId,
		Ids:      versionIds,
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get workflow versions: %w", err)
	}

	workflowNameToRegisteredOpts := make(map[string]CreateWorkflowVersionOpts, len(versions))

	for _, version := range versions {
		// versions created before the options were stored have none
		if len(version.WorkflowVersion.CreateWorkflowVersionOpts) == 0 {
			return uuid.Nil, invalid("%q was registered before workflow options were stored; change and re-register it to use it in a dynamic dag", version.WorkflowName)
		}

		var registeredOpts CreateWorkflowVersionOpts

		if err := json.Unmarshal(version.WorkflowVersion.CreateWorkflowVersionOpts, &registeredOpts); err != nil {
			return uuid.Nil, fmt.Errorf("failed to unmarshal the registered options of %q: %w", version.WorkflowName, err)
		}

		workflowNameToRegisteredOpts[version.WorkflowName] = registeredOpts
	}

	if registeredOpts, ok := workflowNameToRegisteredOpts[dagName]; ok && !registeredOpts.IsDynamicDag {
		return uuid.Nil, invalid("workflow %q already exists and is not a dynamic dag", dagName)
	}

	steps := make([]CreateStepOpts, len(nodes))

	for i, node := range nodes {
		registeredOpts, ok := workflowNameToRegisteredOpts[node.GetTaskName()]

		if !ok {
			return uuid.Nil, invalid("task %q is not registered", node.GetTaskName())
		}

		if len(registeredOpts.Tasks) != 1 || registeredOpts.OnFailure != nil {
			return uuid.Nil, invalid("%q is not a standalone task; dynamic dag nodes must be standalone tasks", node.GetTaskName())
		}

		step := registeredOpts.Tasks[0]
		step.ReadableId = node.GetTaskName()
		step.Parents = node.GetParentTaskNames()

		// OrGroupId is not stored with the registered options, only the index that conditions
		// of the same or-group share
		orGroupIdIndexToOrGroupId := make(map[int32]uuid.UUID)

		for j, condition := range step.TriggerConditions {
			if _, ok := orGroupIdIndexToOrGroupId[condition.OrGroupIdIndex]; !ok {
				orGroupIdIndexToOrGroupId[condition.OrGroupIdIndex] = uuid.New()
			}

			step.TriggerConditions[j].OrGroupId = orGroupIdIndexToOrGroupId[condition.OrGroupIdIndex]
		}

		steps[i] = step
	}

	// rejects duplicate nodes, parents that are not nodes, and cycles
	if _, err := orderWorkflowStepsV1(steps); err != nil {
		return uuid.Nil, invalid("%w", err)
	}

	description := ""

	version, err := r.PutWorkflowVersion(ctx, tenantId, &CreateWorkflowVersionOpts{
		Name:         dagName,
		Description:  &description,
		Tasks:        steps,
		IsDynamicDag: true,
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("could not register dynamic dag workflow %q: %w", dagName, err)
	}

	return version.WorkflowVersion.ID, nil
}
