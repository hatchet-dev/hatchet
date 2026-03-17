package tasks

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TasksService) V1TaskRestore(ctx echo.Context, request gen.V1TaskRestoreRequestObject) (gen.V1TaskRestoreResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	msg, err := tasktypes.DurableRestoreTaskMessage(tenant.ID, request.Task, "Restore via REST API")
	if err != nil {
		return nil, err
	}

	err = t.config.MessageQueueV1.SendMessage(ctx.Request().Context(), msgqueue.TASK_PROCESSING_QUEUE, msg)
	if err != nil {
		return nil, err
	}

	t.config.Analytics.Count(ctx.Request().Context(), analytics.DurableTask, analytics.Restore)

	return gen.V1TaskRestore200JSONResponse{Requeued: true}, nil
}
