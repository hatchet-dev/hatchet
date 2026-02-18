package workflowruns

import (
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1WorkflowRunsService) V1DurableTaskReset(ctx echo.Context, request gen.V1DurableTaskResetRequestObject) (gen.V1DurableTaskResetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	grpcReq := &contracts.ResetDurableTaskRequest{
		TaskExternalId: request.Body.TaskExternalId.String(),
		NodeId:         request.Body.NodeId,
	}

	resp, err := t.proxyResetDurableTask.Do(
		ctx.Request().Context(),
		tenant,
		grpcReq,
	)

	if err != nil {
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.InvalidArgument:
				return gen.V1DurableTaskReset400JSONResponse(
					apierrors.NewAPIErrors(e.Message()),
				), nil
			}
		}

		return nil, err
	}

	return gen.V1DurableTaskReset200JSONResponse{
		TaskExternalId: request.Body.TaskExternalId,
		NodeId:         resp.NodeId,
		BranchId:       resp.BranchId,
	}, nil
}
