package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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

// DynamicDagNodeOpts is one node of a dynamic DAG as the caller submits it: a registered
// single-task workflow and the nodes (by task name) it depends on.
type DynamicDagNodeOpts struct {
	TaskName        string   `validate:"required,hatchetName"`
	ParentTaskNames []string `validate:"dive,hatchetName"`
}

// DynamicDagNode is a dynamic DAG node resolved at trigger time. The workflow version is pinned
// so that replays and deploys of the task mid-run cannot change what the node runs.
type DynamicDagNode struct {
	TaskName          string    `json:"task_name"`
	WorkflowVersionId uuid.UUID `json:"workflow_version_id"`
	StepId            uuid.UUID `json:"step_id"`
	ActionId          string    `json:"action_id"`
	StepReadableId    string    `json:"step_readable_id"`
	ParentTaskNames   []string  `json:"parent_task_names,omitempty"`
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

// ResolveDynamicDagNodes checks that the nodes form a DAG of registered single-task workflows and
// pins each node to its workflow's latest version. The first time a dynamic DAG is triggered
// under dagName, it registers the orchestrator-only workflow its runs belong to.
func (r *workflowRepository) ResolveDynamicDagNodes(ctx context.Context, tenantId uuid.UUID, dagName string, nodes []DynamicDagNodeOpts) ([]DynamicDagNode, error) {
	invalid := func(format string, args ...any) error {
		return &TriggerOptInvalidArgumentError{Err: fmt.Errorf(format, args...)}
	}

	taskNames := make([]string, 0, len(nodes))
	stepsForCycleCheck := make([]CreateStepOpts, 0, len(nodes))

	for _, node := range nodes {
		if err := r.v.Validate(node); err != nil {
			return nil, invalid("invalid node %q: %w", node.TaskName, err)
		}

		if slices.Contains(taskNames, node.TaskName) {
			return nil, invalid("task %q appears in more than one node", node.TaskName)
		}

		taskNames = append(taskNames, node.TaskName)
		stepsForCycleCheck = append(stepsForCycleCheck, CreateStepOpts{ReadableId: node.TaskName, Parents: node.ParentTaskNames})
	}

	for _, node := range nodes {
		for _, parent := range node.ParentTaskNames {
			if !slices.Contains(taskNames, parent) {
				return nil, invalid("node %q has parent %q, which is not a node of the dag", node.TaskName, parent)
			}
		}
	}

	if hasCycleV1(stepsForCycleCheck) {
		return nil, invalid("the dag's nodes contain a cycle")
	}

	db := r.pool.ForTenant(tenantId)

	workflows, err := r.listWorkflowsByNames(ctx, db, tenantId, append([]string{dagName}, taskNames...))

	if err != nil {
		return nil, err
	}

	workflowNameToVersionId := make(map[string]uuid.UUID, len(workflows))
	versionIds := make([]uuid.UUID, 0, len(workflows))

	for _, w := range workflows {
		workflowNameToVersionId[w.WorkflowName] = w.WorkflowVersionId
		versionIds = append(versionIds, w.WorkflowVersionId)
	}

	versionIdToSteps, err := r.listStepsByWorkflowVersionIds(ctx, db, tenantId, versionIds)

	if err != nil {
		return nil, err
	}

	resolved := make([]DynamicDagNode, 0, len(nodes))

	for _, node := range nodes {
		versionId, ok := workflowNameToVersionId[node.TaskName]

		if !ok {
			return nil, invalid("task %q is not registered", node.TaskName)
		}

		steps := versionIdToSteps[versionId]

		if len(steps) != 1 || steps[0].IsDagOrchestrator {
			return nil, invalid("%q is not a standalone task; dynamic dag nodes must be standalone tasks", node.TaskName)
		}

		resolved = append(resolved, DynamicDagNode{
			TaskName:          node.TaskName,
			WorkflowVersionId: versionId,
			StepId:            steps[0].ID,
			ActionId:          steps[0].ActionId,
			StepReadableId:    steps[0].ReadableId.String,
			ParentTaskNames:   node.ParentTaskNames,
		})
	}

	if dagVersionId, ok := workflowNameToVersionId[dagName]; ok {
		if len(regularUserSteps(versionIdToSteps[dagVersionId])) > 0 {
			return nil, invalid("workflow %q already exists and is not a dynamic dag", dagName)
		}
	} else if _, err := r.PutWorkflowVersion(ctx, tenantId, &CreateWorkflowVersionOpts{Name: dagName, IsDynamicDag: true}); err != nil {
		return nil, fmt.Errorf("could not register dynamic dag workflow %q: %w", dagName, err)
	}

	return resolved, nil
}

func (r *TaskRepositoryImpl) GetDynamicDagShape(ctx context.Context, tenantId, workflowRunExternalId uuid.UUID) ([]*sqlcv1.GetWorkflowShapeRow, error) {
	orchestrator, err := r.queries.GetTaskByExternalId(ctx, r.pool.ForTenant(tenantId), sqlcv1.GetTaskByExternalIdParams{
		Tenantid:   tenantId,
		Externalid: workflowRunExternalId,
	})

	// core retention can drop the orchestrator task while OLAP still holds the run
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get orchestrator task: %w", err)
	}

	if len(orchestrator.DynamicDagNodes) == 0 {
		return nil, nil
	}

	var nodes []DynamicDagNode

	if err := json.Unmarshal(orchestrator.DynamicDagNodes, &nodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dynamic dag nodes: %w", err)
	}

	shape := make([]*sqlcv1.GetWorkflowShapeRow, len(nodes))
	taskNameToShapeIndex := make(map[string]int, len(nodes))

	for i, node := range nodes {
		shape[i] = &sqlcv1.GetWorkflowShapeRow{
			Parentstepid:    node.StepId,
			Stepname:        pgtype.Text{String: node.TaskName, Valid: true},
			Childrenstepids: []uuid.UUID{},
		}
		taskNameToShapeIndex[node.TaskName] = i
	}

	for _, node := range nodes {
		for _, parentTaskName := range node.ParentTaskNames {
			if parentIndex, ok := taskNameToShapeIndex[parentTaskName]; ok {
				shape[parentIndex].Childrenstepids = append(shape[parentIndex].Childrenstepids, node.StepId)
			}
		}
	}

	return shape, nil
}
