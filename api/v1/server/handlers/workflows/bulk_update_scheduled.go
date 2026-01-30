package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowScheduledBulkUpdate(ctx echo.Context, request gen.WorkflowScheduledBulkUpdateRequestObject) (gen.WorkflowScheduledBulkUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	if request.Body == nil {
		return gen.WorkflowScheduledBulkUpdate400JSONResponse(gen.APIErrors{
			Errors: []gen.APIError{{Description: "Request body is required."}},
		}), nil
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	updated := make([]openapi_types.UUID, 0, len(request.Body.Updates))
	errors := make([]gen.ScheduledWorkflowsBulkError, 0)

	const chunkSize = 200
	for i := 0; i < len(request.Body.Updates); i += chunkSize {
		end := i + chunkSize
		if end > len(request.Body.Updates) {
			end = len(request.Body.Updates)
		}
		chunk := request.Body.Updates[i:end]

		chunkIds := make([]string, 0, len(chunk))
		chunkUUIDByStr := make(map[string]openapi_types.UUID, len(chunk))
		for _, u := range chunk {
			idStr := u.Id.String()
			chunkIds = append(chunkIds, idStr)
			chunkUUIDByStr[idStr] = u.Id
		}

		metaById, err := t.config.V1.WorkflowSchedules().ScheduledWorkflowMetaByIds(dbCtx, tenantId, chunkIds)
		if err != nil {
			return nil, err
		}

		toUpdate := make([]v1.ScheduledWorkflowUpdate, 0, len(chunk))
		for _, u := range chunk {
			id := u.Id
			idStr := id.String()

			meta, ok := metaById[idStr]
			if !ok {
				idCp := id
				errors = append(errors, gen.ScheduledWorkflowsBulkError{Id: &idCp, Error: "Scheduled workflow not found."})
				continue
			}

			if meta.HasTriggeredRun {
				idCp := id
				errors = append(errors, gen.ScheduledWorkflowsBulkError{Id: &idCp, Error: "Scheduled run has already been triggered and cannot be rescheduled."})
				continue
			}

			toUpdate = append(toUpdate, v1.ScheduledWorkflowUpdate{
				Id:        idStr,
				TriggerAt: u.TriggerAt,
			})
		}

		updatedIds, err := t.config.V1.WorkflowSchedules().BulkUpdateScheduledWorkflows(dbCtx, tenantId, toUpdate)
		if err != nil {
			return nil, err
		}

		updatedSet := make(map[string]struct{}, len(updatedIds))
		for _, idStr := range updatedIds {
			updatedSet[idStr] = struct{}{}
		}

		for _, u := range chunk {
			if _, ok := updatedSet[u.Id.String()]; ok {
				updated = append(updated, u.Id)
			}
		}

		// Should be rare (race conditions); report per-id errors for anything we expected to update but didn't.
		for _, u := range toUpdate {
			if _, ok := updatedSet[u.Id]; ok {
				continue
			}
			idCp := chunkUUIDByStr[u.Id]
			errors = append(errors, gen.ScheduledWorkflowsBulkError{Id: &idCp, Error: "Failed to update scheduled workflow."})
		}
	}

	return gen.WorkflowScheduledBulkUpdate200JSONResponse(gen.ScheduledWorkflowsBulkUpdateResponse{
		UpdatedIds: updated,
		Errors:     errors,
	}), nil
}
