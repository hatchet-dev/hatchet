package tenants

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) TenantGetQueueMetrics(ctx echo.Context, request gen.TenantGetQueueMetricsRequestObject) (gen.TenantGetQueueMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	opts := repository.GetQueueMetricsOpts{}

	if request.Params.AdditionalMetadata != nil {
		additionalMetadata := make(map[string]interface{}, len(*request.Params.AdditionalMetadata))

		for _, v := range *request.Params.AdditionalMetadata {
			splitValue := strings.Split(fmt.Sprintf("%v", v), ":")

			if len(splitValue) == 2 {
				additionalMetadata[splitValue[0]] = splitValue[1]
			} else {
				return gen.TenantGetQueueMetrics400JSONResponse(apierrors.NewAPIErrors("Additional metadata filters must be in the format key:value.")), nil

			}
		}

		opts.AdditionalMetadata = additionalMetadata
	}

	if request.Params.Workflows != nil {
		opts.WorkflowIds = *request.Params.Workflows
	}

	metrics, err := t.config.APIRepository.Tenant().GetQueueMetrics(ctx.Request().Context(), tenant.ID, &opts)

	if err != nil {
		return nil, err
	}

	stepRunQueueCounts, err := t.config.EngineRepository.StepRun().GetQueueCounts(ctx.Request().Context(), tenant.ID)

	if err != nil {
		return nil, err
	}

	respWorkflowMap := make(map[string]gen.QueueMetrics, len(metrics.ByWorkflowId))

	for k, v := range metrics.ByWorkflowId {
		respWorkflowMap[k] = gen.QueueMetrics{
			NumPending: v.Pending,
			NumQueued:  v.PendingAssignment,
			NumRunning: v.Running,
		}
	}

	resp := gen.TenantQueueMetrics{
		Total: &gen.QueueMetrics{
			NumPending: metrics.Total.Pending,
			NumQueued:  metrics.Total.PendingAssignment,
			NumRunning: metrics.Total.Running,
		},
		Workflow: &respWorkflowMap,
		Queues:   &stepRunQueueCounts,
	}

	return gen.TenantGetQueueMetrics200JSONResponse(resp), nil
}
