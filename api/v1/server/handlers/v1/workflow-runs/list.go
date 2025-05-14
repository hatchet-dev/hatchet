package workflowruns

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *V1WorkflowRunsService) WithDags(ctx context.Context, request gen.V1WorkflowRunListRequestObject, tenantId string, includeInputAndOutput bool) (*gen.V1TaskSummaryList, error) {
	ctx, span := telemetry.NewSpan(ctx, "v1-workflow-runs-list-only-tasks")
	defer span.End()

	var (
		statuses = []sqlcv1.V1ReadableStatusOlap{
			sqlcv1.V1ReadableStatusOlapQUEUED,
			sqlcv1.V1ReadableStatusOlapRUNNING,
			sqlcv1.V1ReadableStatusOlapFAILED,
			sqlcv1.V1ReadableStatusOlapCOMPLETED,
			sqlcv1.V1ReadableStatusOlapCANCELLED,
		}
		since             = request.Params.Since
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}
			for _, status := range *request.Params.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
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

	opts := v1.ListWorkflowRunOpts{
		CreatedAfter: since,
		Statuses:     statuses,
		WorkflowIds:  workflowIds,
		Limit:        limit,
		Offset:       offset,
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.Split(v, ":")
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

	if request.Params.Until != nil {
		opts.FinishedBefore = request.Params.Until
	}

	if request.Params.ParentTaskExternalId != nil {
		parentTaskExternalId := request.Params.ParentTaskExternalId.String()
		id := sqlchelpers.UUIDFromStr(parentTaskExternalId)
		opts.ParentTaskExternalId = &id
	}

	if request.Params.TriggeringEventExternalId != nil {
		id := sqlchelpers.UUIDFromStr(request.Params.TriggeringEventExternalId.String())
		opts.TriggeringEventExternalId = &id
	}

	dags, total, err := t.config.V1.OLAP().ListWorkflowRuns(
		ctx,
		tenantId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	dagExternalIds := make([]pgtype.UUID, 0)

	for _, dag := range dags {
		if dag.Kind == sqlcv1.V1RunKindDAG {
			dagExternalIds = append(dagExternalIds, dag.ExternalID)
		}
	}

	tasks, taskIdToDagExternalId, err := t.config.V1.OLAP().ListTasksByDAGId(
		ctx,
		tenantId,
		dagExternalIds,
		includeInputAndOutput,
	)

	if err != nil {
		return nil, err
	}

	pgWorkflowIds := make([]pgtype.UUID, 0)

	for _, wf := range dags {
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

	for _, task := range tasks {
		if name, ok := workflowNames[task.WorkflowID]; ok {
			taskIdToWorkflowName[task.ID] = name
		}
	}

	taskIdToActionId := make(map[int64]string)

	for _, task := range tasks {
		taskIdToActionId[task.ID] = task.ActionID
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

	result := transformers.ToWorkflowRunMany(dags, dagChildren, taskIdToActionId, workflowNames, total, limit, offset)

	return &result, nil
}

func (t *V1WorkflowRunsService) OnlyTasks(ctx context.Context, request gen.V1WorkflowRunListRequestObject, tenantId string, includeInputAndOutput bool) (*gen.V1TaskSummaryList, error) {
	ctx, span := telemetry.NewSpan(ctx, "v1-workflow-runs-list-only-tasks")
	defer span.End()

	var (
		statuses = []sqlcv1.V1ReadableStatusOlap{
			sqlcv1.V1ReadableStatusOlapQUEUED,
			sqlcv1.V1ReadableStatusOlapRUNNING,
			sqlcv1.V1ReadableStatusOlapFAILED,
			sqlcv1.V1ReadableStatusOlapCOMPLETED,
			sqlcv1.V1ReadableStatusOlapCANCELLED,
		}
		since             = request.Params.Since
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}
			for _, status := range *request.Params.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
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

	opts := v1.ListTaskRunOpts{
		CreatedAfter:          since,
		Statuses:              statuses,
		WorkflowIds:           workflowIds,
		Limit:                 limit,
		Offset:                offset,
		WorkerId:              request.Params.WorkerId,
		IncludeInputAndOutput: includeInputAndOutput,
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.Split(v, ":")
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

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

	taskIdToWorkflowName := make(map[int64]string)

	result := transformers.TaskRunDataRowToWorkflowRunsMany(tasks, taskIdToWorkflowName, total, limit, offset)

	return &result, nil
}

func (t *V1WorkflowRunsService) V1WorkflowRunList(ctx echo.Context, request gen.V1WorkflowRunListRequestObject) (gen.V1WorkflowRunListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	spanContext, span := telemetry.NewSpan(ctx.Request().Context(), "v1-workflow-runs-list")
	defer span.End()

	var (
		taskSummaryList *gen.V1TaskSummaryList
		err             error
	)

	if request.Params.OnlyTasks {
		taskSummaryList, err = t.OnlyTasks(spanContext, request, tenantId, true)
	} else {
		taskSummaryList, err = t.WithDags(spanContext, request, tenantId, true)
	}

	if err != nil || taskSummaryList == nil {
		return nil, err
	}

	return gen.V1WorkflowRunList200JSONResponse(
		*taskSummaryList,
	), nil
}

func v1TaskSummaryToV2TaskSummary(taskSummary gen.V1TaskSummary) gen.V2TaskSummary {
	children := make([]gen.V2TaskSummary, 0)

	if taskSummary.Children != nil {
		for _, child := range *taskSummary.Children {
			children = append(children, v1TaskSummaryToV2TaskSummary(child))
		}
	}

	return gen.V2TaskSummary{
		ActionId:              taskSummary.ActionId,
		AdditionalMetadata:    taskSummary.AdditionalMetadata,
		Attempt:               taskSummary.Attempt,
		Children:              &children,
		CreatedAt:             taskSummary.CreatedAt,
		DisplayName:           taskSummary.DisplayName,
		Duration:              taskSummary.Duration,
		ErrorMessage:          taskSummary.ErrorMessage,
		FinishedAt:            taskSummary.FinishedAt,
		Metadata:              taskSummary.Metadata,
		NumSpawnedChildren:    taskSummary.NumSpawnedChildren,
		RetryCount:            taskSummary.RetryCount,
		StartedAt:             taskSummary.StartedAt,
		Status:                taskSummary.Status,
		StepId:                taskSummary.StepId,
		TaskExternalId:        taskSummary.TaskExternalId,
		TaskId:                taskSummary.TaskId,
		TaskInsertedAt:        taskSummary.TaskInsertedAt,
		TenantId:              taskSummary.TenantId,
		Type:                  taskSummary.Type,
		WorkflowId:            taskSummary.WorkflowId,
		WorkflowName:          taskSummary.WorkflowName,
		WorkflowRunExternalId: taskSummary.WorkflowRunExternalId,
		WorkflowVersionId:     taskSummary.WorkflowVersionId,
	}
}

func (t *V1WorkflowRunsService) V2WorkflowRunList(ctx echo.Context, request gen.V2WorkflowRunListRequestObject) (gen.V2WorkflowRunListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	spanContext, span := telemetry.NewSpan(ctx.Request().Context(), "v2-workflow-runs-list")
	defer span.End()

	v1Request := gen.V1WorkflowRunListRequestObject{
		Tenant: request.Tenant,
		Params: gen.V1WorkflowRunListParams{
			Offset:                    request.Params.Offset,
			Limit:                     request.Params.Limit,
			Statuses:                  request.Params.Statuses,
			Since:                     request.Params.Since,
			Until:                     request.Params.Until,
			AdditionalMetadata:        request.Params.AdditionalMetadata,
			WorkflowIds:               request.Params.WorkflowIds,
			WorkerId:                  request.Params.WorkerId,
			OnlyTasks:                 request.Params.OnlyTasks,
			ParentTaskExternalId:      request.Params.ParentTaskExternalId,
			TriggeringEventExternalId: request.Params.TriggeringEventExternalId,
		},
	}

	var (
		taskSummaryList *gen.V1TaskSummaryList
		err             error
	)

	if request.Params.OnlyTasks {
		taskSummaryList, err = t.OnlyTasks(spanContext, v1Request, tenantId, false)
	} else {
		taskSummaryList, err = t.WithDags(spanContext, v1Request, tenantId, false)
	}

	if err != nil || taskSummaryList == nil {
		return nil, err
	}

	v2TaskSummaries := make([]gen.V2TaskSummary, len(taskSummaryList.Rows))

	for i, taskSummary := range taskSummaryList.Rows {
		v2TaskSummaries[i] = v1TaskSummaryToV2TaskSummary(taskSummary)
	}

	v2TaskSummaryList := gen.V2TaskSummaryList{
		Pagination: taskSummaryList.Pagination,
		Rows:       v2TaskSummaries,
	}

	return gen.V2WorkflowRunList200JSONResponse(
		v2TaskSummaryList,
	), nil
}

func (t *V1WorkflowRunsService) V1WorkflowRunDisplayNamesList(ctx echo.Context, request gen.V1WorkflowRunDisplayNamesListRequestObject) (gen.V1WorkflowRunDisplayNamesListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	externalIds := make([]pgtype.UUID, len(request.Params.ExternalIds))

	for i, id := range request.Params.ExternalIds {
		externalIds[i] = sqlchelpers.UUIDFromStr(id.String())
	}

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
