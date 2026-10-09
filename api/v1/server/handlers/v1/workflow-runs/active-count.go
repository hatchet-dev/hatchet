package workflowruns

import (
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// activeOlapStatuses returns the requested statuses that are still active (QUEUED or RUNNING), mapped to OLAP
// statuses the same way the run list maps them.
func activeOlapStatuses(statuses *[]gen.V1TaskStatus, runningFilter *gen.V1RunningFilter) []sqlcv1.V1ReadableStatusOlap {
	requested := []gen.V1TaskStatus{gen.V1TaskStatusQUEUED, gen.V1TaskStatusRUNNING}

	if statuses != nil && len(*statuses) > 0 {
		requested = *statuses
	}

	active := make([]gen.V1TaskStatus, 0, len(requested))

	for _, status := range requested {
		if status == gen.V1TaskStatusQUEUED || status == gen.V1TaskStatusRUNNING {
			active = append(active, status)
		}
	}

	return normalizeWorkflowRunStatuses(active, runningFilter)
}

// activeCountOpts maps the active-count params to run list options. It returns false when none of the requested
// statuses are active, so nothing can match.
func activeCountOpts(params gen.V1WorkflowRunActiveCountParams, useGinIndex bool) (v1.ListWorkflowRunOpts, bool) {
	statuses := activeOlapStatuses(params.Statuses, params.RunningFilter)

	if len(statuses) == 0 {
		return v1.ListWorkflowRunOpts{}, false
	}

	// The run list bounds inserted_at inclusively by FinishedBefore; Postgres timestamps have microsecond
	// precision, so one microsecond earlier makes `before` exclusive.
	until := params.Before.Add(-time.Microsecond)

	opts := v1.ListWorkflowRunOpts{
		CreatedAfter:    params.Since,
		FinishedBefore:  &until,
		Statuses:        statuses,
		IdempotencyKeys: params.IdempotencyKeys,
	}

	if params.WorkflowIds != nil {
		opts.WorkflowIds = *params.WorkflowIds
	}

	if params.AdditionalMetadata != nil {
		additionalMetadataFilters := make(map[string]interface{})

		for _, v := range *params.AdditionalMetadata {
			kv_pairs := strings.SplitN(v, ":", 2)
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

	opts.AdditionalMetadataOperator = additionalMetadataOperator(params.AdditionalMetadataOperator, len(opts.AdditionalMetadata), useGinIndex)

	return opts, true
}

func (t *V1WorkflowRunsService) V1WorkflowRunActiveCount(ctx echo.Context, request gen.V1WorkflowRunActiveCountRequestObject) (gen.V1WorkflowRunActiveCountResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	spanContext, span := telemetry.NewSpan(ctx.Request().Context(), "v1-workflow-runs-active-count")
	defer span.End()

	if !request.Params.Before.After(request.Params.Since) {
		return gen.V1WorkflowRunActiveCount400JSONResponse(apierrors.NewAPIErrors("before must be later than since")), nil
	}

	useGinIndex, err := t.useGinIndex(spanContext, tenantId, request.Params.AdditionalMetadata)
	if err != nil {
		return nil, err
	}

	opts, ok := activeCountOpts(request.Params, useGinIndex)

	if !ok {
		return gen.V1WorkflowRunActiveCount200JSONResponse(gen.V1WorkflowRunActiveCount{}), nil
	}

	count, capped, err := t.config.V1.OLAP().CountWorkflowRuns(spanContext, tenantId, opts)
	if err != nil {
		return nil, err
	}

	return gen.V1WorkflowRunActiveCount200JSONResponse(gen.V1WorkflowRunActiveCount{
		Count:  count,
		Capped: capped,
	}), nil
}
