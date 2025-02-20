package workflows

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) WorkflowRunGetMetrics(ctx echo.Context, request gen.WorkflowRunGetMetricsRequestObject) (gen.WorkflowRunGetMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	listOpts := &repository.WorkflowRunsMetricsOpts{}

	if request.Params.WorkflowId != nil {
		workflowIdStr := request.Params.WorkflowId.String()
		listOpts.WorkflowId = &workflowIdStr
	}

	if request.Params.CreatedAfter != nil {
		listOpts.CreatedAfter = request.Params.CreatedAfter
	}

	if request.Params.CreatedBefore != nil {
		listOpts.CreatedBefore = request.Params.CreatedBefore
	}

	if request.Params.EventId != nil {
		eventIdStr := request.Params.EventId.String()
		listOpts.EventId = &eventIdStr
	}

	if request.Params.ParentWorkflowRunId != nil {
		parentWorkflowRunIdStr := request.Params.ParentWorkflowRunId.String()
		listOpts.ParentId = &parentWorkflowRunIdStr
	}

	if request.Params.ParentStepRunId != nil {
		parentStepRunIdStr := request.Params.ParentStepRunId.String()
		listOpts.ParentStepRunId = &parentStepRunIdStr
	}

	if request.Params.AdditionalMetadata != nil {
		additionalMetadata := make(map[string]interface{}, len(*request.Params.AdditionalMetadata))

		for _, v := range *request.Params.AdditionalMetadata {
			splitValue := strings.Split(fmt.Sprintf("%v", v), ":")

			if len(splitValue) == 2 {
				additionalMetadata[splitValue[0]] = splitValue[1]
			} else {
				return gen.WorkflowRunGetMetrics400JSONResponse(apierrors.NewAPIErrors("Additional metadata filters must be in the format key:value.")), nil

			}
		}

		listOpts.AdditionalMetadata = additionalMetadata
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	workflowRunsMetricsCount, err := t.config.APIRepository.WorkflowRun().WorkflowRunMetricsCount(dbCtx, tenant.ID, listOpts)

	if err != nil {
		return nil, err
	}

	failed := int(workflowRunsMetricsCount.FAILED)
	pending := int(workflowRunsMetricsCount.PENDING)
	queued := int(workflowRunsMetricsCount.QUEUED)
	running := int(workflowRunsMetricsCount.RUNNING)
	succeeded := int(workflowRunsMetricsCount.SUCCEEDED)
	cancelled := int(workflowRunsMetricsCount.CANCELLED)

	return gen.WorkflowRunGetMetrics200JSONResponse(
		gen.WorkflowRunsMetrics{
			Counts: &gen.WorkflowRunsMetricsCounts{
				FAILED:    &failed,
				PENDING:   &pending,
				QUEUED:    &queued,
				RUNNING:   &running,
				SUCCEEDED: &succeeded,
				CANCELLED: &cancelled,
			},
		},
	), nil
}
