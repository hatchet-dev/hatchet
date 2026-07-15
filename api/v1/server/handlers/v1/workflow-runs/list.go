package workflowruns

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func allOlapStatuses(runningFilter *gen.V1RunningFilter) []sqlcv1.V1ReadableStatusOlap {
	statuses := []sqlcv1.V1ReadableStatusOlap{
		sqlcv1.V1ReadableStatusOlapQUEUED,
		sqlcv1.V1ReadableStatusOlapFAILED,
		sqlcv1.V1ReadableStatusOlapCOMPLETED,
		sqlcv1.V1ReadableStatusOlapCANCELLED,
	}

	rf := gen.ALL
	if runningFilter != nil {
		rf = *runningFilter
	}
	switch rf {
	case gen.EVICTED:
		statuses = append(statuses, sqlcv1.V1ReadableStatusOlapEVICTED)
	case gen.ONWORKER:
		statuses = append(statuses, sqlcv1.V1ReadableStatusOlapRUNNING)
	default:
		statuses = append(statuses, sqlcv1.V1ReadableStatusOlapRUNNING, sqlcv1.V1ReadableStatusOlapEVICTED)
	}

	return statuses
}

var taskStatusToOlapStatus = map[gen.V1TaskStatus]sqlcv1.V1ReadableStatusOlap{
	gen.V1TaskStatusQUEUED:    sqlcv1.V1ReadableStatusOlapQUEUED,
	gen.V1TaskStatusRUNNING:   sqlcv1.V1ReadableStatusOlapRUNNING,
	gen.V1TaskStatusFAILED:    sqlcv1.V1ReadableStatusOlapFAILED,
	gen.V1TaskStatusCOMPLETED: sqlcv1.V1ReadableStatusOlapCOMPLETED,
	gen.V1TaskStatusCANCELLED: sqlcv1.V1ReadableStatusOlapCANCELLED,
}

func normalizeWorkflowRunStatuses(statuses []gen.V1TaskStatus, runningFilter *gen.V1RunningFilter) []sqlcv1.V1ReadableStatusOlap {
	normalized := make([]sqlcv1.V1ReadableStatusOlap, 0, len(statuses))
	seen := make(map[sqlcv1.V1ReadableStatusOlap]struct{}, len(statuses))

	for _, status := range statuses {
		if status == gen.V1TaskStatusRUNNING {
			rf := gen.ALL
			if runningFilter != nil {
				rf = *runningFilter
			}
			switch rf {
			case gen.EVICTED:
				if _, exists := seen[sqlcv1.V1ReadableStatusOlapEVICTED]; !exists {
					seen[sqlcv1.V1ReadableStatusOlapEVICTED] = struct{}{}
					normalized = append(normalized, sqlcv1.V1ReadableStatusOlapEVICTED)
				}
			case gen.ONWORKER:
				if _, exists := seen[sqlcv1.V1ReadableStatusOlapRUNNING]; !exists {
					seen[sqlcv1.V1ReadableStatusOlapRUNNING] = struct{}{}
					normalized = append(normalized, sqlcv1.V1ReadableStatusOlapRUNNING)
				}
			default:
				if _, exists := seen[sqlcv1.V1ReadableStatusOlapRUNNING]; !exists {
					seen[sqlcv1.V1ReadableStatusOlapRUNNING] = struct{}{}
					normalized = append(normalized, sqlcv1.V1ReadableStatusOlapRUNNING)
				}
				if _, exists := seen[sqlcv1.V1ReadableStatusOlapEVICTED]; !exists {
					seen[sqlcv1.V1ReadableStatusOlapEVICTED] = struct{}{}
					normalized = append(normalized, sqlcv1.V1ReadableStatusOlapEVICTED)
				}
			}
			continue
		}

		mapped, ok := taskStatusToOlapStatus[status]
		if !ok {
			continue
		}

		if _, exists := seen[mapped]; exists {
			continue
		}

		seen[mapped] = struct{}{}
		normalized = append(normalized, mapped)
	}

	return normalized
}

func (t *V1WorkflowRunsService) WithDags(ctx context.Context, request gen.V1WorkflowRunListRequestObject, tenantId uuid.UUID, useGinIndex bool) (gen.V1WorkflowRunListResponseObject, error) {
	ctx, span := telemetry.NewSpan(ctx, "v1-workflow-runs-list-with-dags-tasks")
	defer span.End()

	var (
		statuses       = allOlapStatuses(request.Params.RunningFilter)
		since          = request.Params.Since
		limit    int64 = 50
		offset   int64
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = normalizeWorkflowRunStatuses(*request.Params.Statuses, request.Params.RunningFilter)
		}
	}

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	workflowIds := make([]uuid.UUID, 0)
	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	includePayloads := false
	if request.Params.IncludePayloads != nil {
		includePayloads = *request.Params.IncludePayloads
	}

	opts := v1.ListWorkflowRunOpts{
		CreatedAfter:    since,
		Statuses:        statuses,
		WorkflowIds:     workflowIds,
		Limit:           limit,
		Offset:          offset,
		IncludePayloads: includePayloads,
		UseGinIndex:     useGinIndex,
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.SplitN(v, ":", 2)
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

	opts.AdditionalMetadataOperator = additionalMetadataOperator(request.Params.AdditionalMetadataOperator, len(opts.AdditionalMetadata))

	if request.Params.Until != nil {
		opts.FinishedBefore = request.Params.Until
	}

	if request.Params.ParentTaskExternalId != nil {
		opts.ParentTaskExternalId = request.Params.ParentTaskExternalId
	}

	if request.Params.TriggeringEventExternalId != nil {
		opts.TriggeringEventExternalId = request.Params.TriggeringEventExternalId
	}

	dags, total, err := t.config.V1.OLAP().ListWorkflowRuns(
		ctx,
		tenantId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	dagExternalIds := make([]uuid.UUID, 0)
	dagDurableTaskExternalIds := make(map[uuid.UUID]struct{}, 0)

	for _, dag := range dags {
		if dag.Kind == sqlcv1.V1RunKindDAG {
			dagExternalIds = append(dagExternalIds, dag.ExternalID)
		}

		if dag.Kind == sqlcv1.V1RunKindDAG && dag.IsDagOrchestratorTask {
			dagDurableTaskExternalIds[dag.ExternalID] = struct{}{}
		}
	}

	orchestratorChildExternalIds := make(map[uuid.UUID]struct{})
	idsInsertedAtsForDagChildren := make([]v1.IdInsertedAt, 0)
	for _, dag := range dags {
		if dag.IsDagOrchestratorTask || dag.ParentTaskExternalId == nil {
			continue
		}
		if _, parentIsOrchestrator := dagDurableTaskExternalIds[*dag.ParentTaskExternalId]; parentIsOrchestrator {
			orchestratorChildExternalIds[dag.ExternalID] = struct{}{}
			idsInsertedAtsForDagChildren = append(idsInsertedAtsForDagChildren, v1.IdInsertedAt{
				ID:         dag.ID,
				InsertedAt: dag.InsertedAt,
			})
		}
	}

	filteredDags := make([]*v1.WorkflowRunData, 0, len(dags))
	for _, dag := range dags {
		if _, isChild := orchestratorChildExternalIds[dag.ExternalID]; !isChild {
			filteredDags = append(filteredDags, dag)
		}
	}

	tasks, taskIdToDagExternalId, err := t.config.V1.OLAP().ListTasksByDAGId(
		ctx,
		tenantId,
		dagExternalIds,
		includePayloads,
		idsInsertedAtsForDagChildren,
	)

	if err != nil {
		return nil, err
	}

	pgWorkflowIds := make([]uuid.UUID, 0)

	for _, wf := range filteredDags {
		pgWorkflowIds = append(pgWorkflowIds, wf.WorkflowID)
	}

	workflowNames, err := t.config.V1.Workflows().ListWorkflowNamesByIds(
		ctx,
		tenantId,
		pgWorkflowIds,
	)

	if err != nil {
		return nil, err
	}

	taskIdToWorkflowName := make(map[int64]string)
	taskIdToActionId := make(map[int64]string)

	for _, task := range tasks {
		taskIdToActionId[task.ID] = task.ActionID
		if name, ok := workflowNames[task.WorkflowID]; ok {
			taskIdToWorkflowName[task.ID] = name
		}
	}

	parsedTasks := transformers.TaskRunDataRowToWorkflowRunsMany(tasks, taskIdToWorkflowName, total, limit, offset)

	dagChildren := make(map[uuid.UUID][]gen.V1TaskSummary)

	for _, task := range parsedTasks.Rows {
		dagExternalId := taskIdToDagExternalId[int64(task.TaskId)]
		existing, ok := dagChildren[dagExternalId]

		if ok {
			dagChildren[dagExternalId] = append(existing, task)
		} else {
			dagChildren[dagExternalId] = []gen.V1TaskSummary{task}
		}
	}

	result := transformers.ToWorkflowRunMany(filteredDags, dagChildren, taskIdToActionId, workflowNames, total, limit, offset)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunList200JSONResponse(
		result,
	), nil
}

func (t *V1WorkflowRunsService) OnlyTasks(ctx context.Context, request gen.V1WorkflowRunListRequestObject, tenantId uuid.UUID, useGinIndex bool) (gen.V1WorkflowRunListResponseObject, error) {
	ctx, span := telemetry.NewSpan(ctx, "v1-workflow-runs-list-only-tasks")
	defer span.End()

	var (
		statuses          = allOlapStatuses(request.Params.RunningFilter)
		since             = request.Params.Since
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = normalizeWorkflowRunStatuses(*request.Params.Statuses, request.Params.RunningFilter)
		}
	}

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	includePayloads := false
	if request.Params.IncludePayloads != nil {
		includePayloads = *request.Params.IncludePayloads
	}

	opts := v1.ListTaskRunOpts{
		CreatedAfter:    since,
		Statuses:        statuses,
		WorkflowIds:     workflowIds,
		Limit:           limit,
		Offset:          offset,
		WorkerId:        request.Params.WorkerId,
		IncludePayloads: includePayloads,
		UseGinIndex:     useGinIndex,
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.SplitN(v, ":", 2)
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

	opts.AdditionalMetadataOperator = additionalMetadataOperator(request.Params.AdditionalMetadataOperator, len(opts.AdditionalMetadata))

	if request.Params.Until != nil {
		opts.FinishedBefore = request.Params.Until
	}

	if request.Params.TriggeringEventExternalId != nil {
		opts.TriggeringEventExternalId = request.Params.TriggeringEventExternalId
	}

	tasks, total, err := t.config.V1.OLAP().ListTasks(
		ctx,
		tenantId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	workflowIdsForNames := make([]uuid.UUID, 0)
	for _, task := range tasks {
		workflowIdsForNames = append(workflowIdsForNames, task.WorkflowID)
	}

	workflowIdToName, err := t.config.V1.Workflows().ListWorkflowNamesByIds(
		ctx,
		tenantId,
		workflowIdsForNames,
	)

	if err != nil {
		return nil, err
	}

	taskIdToWorkflowName := make(map[int64]string)

	for _, task := range tasks {
		if name, ok := workflowIdToName[task.WorkflowID]; ok {
			taskIdToWorkflowName[task.ID] = name
		}
	}

	result := transformers.TaskRunDataRowToWorkflowRunsMany(tasks, taskIdToWorkflowName, total, limit, offset)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunList200JSONResponse(
		result,
	), nil
}

func (t *V1WorkflowRunsService) V1WorkflowRunList(ctx echo.Context, request gen.V1WorkflowRunListRequestObject) (gen.V1WorkflowRunListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	spanContext, span := telemetry.NewSpan(ctx.Request().Context(), "v1-workflow-runs-list")
	defer span.End()

	useGinIndex := false

	if request.Params.AdditionalMetadata != nil && len(*request.Params.AdditionalMetadata) > 0 {
		enabled, err := t.config.V1.TenantEntitlement().IsStrictAdditionalMetadataFiltersEnabled(spanContext, tenantId)

		if err != nil {
			return nil, err
		}

		// if there's only one filter, we should always use the `AND` path with the index, since it's the most performant and all methods are equivalent in that case
		useGinIndex = enabled || len(*request.Params.AdditionalMetadata) == 1
	}

	if request.Params.OnlyTasks {
		return t.OnlyTasks(spanContext, request, tenantId, useGinIndex)
	} else {
		return t.WithDags(spanContext, request, tenantId, useGinIndex)
	}
}

// additionalMetadataOperator maps the optional additional_metadata_operator query
// param to the repository operator, defaulting to OR
func additionalMetadataOperator(param *gen.V1AdditionalMetadataOperator, numFilters int) v1.AdditionalMetadataOperator {
	// if we only have one filter, always use the `AND` since it's the most performant way, and both methods are equivalent
	if numFilters <= 1 {
		return v1.AdditionalMetadataOperatorAnd
	}

	if param != nil && *param == gen.AND {
		return v1.AdditionalMetadataOperatorAnd
	}

	return v1.AdditionalMetadataOperatorOr
}

func (t *V1WorkflowRunsService) V1WorkflowRunDisplayNamesList(ctx echo.Context, request gen.V1WorkflowRunDisplayNamesListRequestObject) (gen.V1WorkflowRunDisplayNamesListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	externalIds := request.Params.ExternalIds

	displayNames, err := t.config.V1.OLAP().ListWorkflowRunDisplayNames(
		ctx.Request().Context(),
		tenant.ID,
		externalIds,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToWorkflowRunDisplayNamesList(displayNames)

	return gen.V1WorkflowRunDisplayNamesList200JSONResponse(
		result,
	), nil
}

func (t *V1WorkflowRunsService) V1WorkflowRunExternalIdsList(ctx echo.Context, request gen.V1WorkflowRunExternalIdsListRequestObject) (gen.V1WorkflowRunExternalIdsListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	spanCtx, span := telemetry.NewSpan(ctx.Request().Context(), "v1-workflow-runs-list-external-ids")
	defer span.End()

	var (
		statuses    = allOlapStatuses(request.Params.RunningFilter)
		since       = request.Params.Since
		workflowIds = []uuid.UUID{}
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = normalizeWorkflowRunStatuses(*request.Params.Statuses, request.Params.RunningFilter)
		}
	}

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	opts := v1.ListWorkflowRunOpts{
		CreatedAfter: since,
		Statuses:     statuses,
		WorkflowIds:  workflowIds,
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.SplitN(v, ":", 2)
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

	if request.Params.Until != nil {
		opts.FinishedBefore = request.Params.Until
	}

	externalIds, err := t.config.V1.OLAP().ListWorkflowRunExternalIds(
		spanCtx,
		tenantId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	return gen.V1WorkflowRunExternalIdsList200JSONResponse(
		externalIds,
	), nil
}
