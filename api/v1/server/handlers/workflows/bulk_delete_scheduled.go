package workflows

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowScheduledBulkDelete(ctx echo.Context, request gen.WorkflowScheduledBulkDeleteRequestObject) (gen.WorkflowScheduledBulkDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	if request.Body == nil {
		return gen.WorkflowScheduledBulkDelete400JSONResponse(gen.APIErrors{
			Errors: []gen.APIError{{Description: "Request body is required."}},
		}), nil
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	var ids []uuid.UUID
	if request.Body.ScheduledWorkflowRunIds != nil {
		ids = *request.Body.ScheduledWorkflowRunIds
	}

	errors := make([]gen.ScheduledWorkflowsBulkError, 0)

	// Mode validation: must provide either ids or filter (but not both).
	if len(ids) > 0 && request.Body.Filter != nil {
		return gen.WorkflowScheduledBulkDelete400JSONResponse(apierrors.NewAPIErrors("Provide either scheduledWorkflowRunIds or filter, not both.")), nil
	}

	// If filter mode, expand filter to ids (across pages) first.
	if len(ids) == 0 && request.Body.Filter != nil {
		filter := request.Body.Filter

		limit := 500
		offset := 0
		orderBy := "triggerAt"
		orderDirection := "DESC"

		opts := &v1.ListScheduledWorkflowsOpts{
			Limit:          &limit,
			Offset:         &offset,
			OrderBy:        &orderBy,
			OrderDirection: &orderDirection,
		}

		if filter.WorkflowId != nil {
			wid := filter.WorkflowId.String()
			opts.WorkflowId = &wid
		}
		if filter.ParentWorkflowRunId != nil {
			pid := filter.ParentWorkflowRunId.String()
			opts.ParentWorkflowRunId = &pid
		}
		if filter.ParentStepRunId != nil {
			psid := filter.ParentStepRunId.String()
			opts.ParentStepRunId = &psid
		}
		if filter.AdditionalMetadata != nil {
			additionalMetadata := make(map[string]interface{}, len(*filter.AdditionalMetadata))
			for _, v := range *filter.AdditionalMetadata {
				splitValue := strings.Split(fmt.Sprintf("%v", v), ":")
				if len(splitValue) == 2 {
					additionalMetadata[splitValue[0]] = splitValue[1]
				} else {
					return gen.WorkflowScheduledBulkDelete400JSONResponse(apierrors.NewAPIErrors("Additional metadata filters must be in the format key:value.")), nil
				}
			}
			opts.AdditionalMetadata = additionalMetadata
		}

		all := make([]*sqlcv1.ListScheduledWorkflowsRow, 0)
		for {
			rows, count, err := t.config.V1.WorkflowSchedules().ListScheduledWorkflows(dbCtx, tenantId, opts)
			if err != nil {
				return nil, err
			}
			all = append(all, rows...)

			if len(all) >= int(count) || len(rows) == 0 {
				break
			}

			offset += limit
			opts.Offset = &offset
		}

		// Convert list results into ids + pre-fill errors for non-API items.
		ids = make([]uuid.UUID, 0, len(all))
		for _, row := range all {
			idStr := row.ID.String()
			idUUID, err := uuid.Parse(idStr)
			if err != nil {
				// fall back to skip with generic error (should never happen)
				errors = append(errors, gen.ScheduledWorkflowsBulkError{Error: "Invalid scheduled id."})
				continue
			}

			if row.Method != sqlcv1.WorkflowTriggerScheduledRefMethodsAPI {
				idCp := idUUID
				errors = append(errors, gen.ScheduledWorkflowsBulkError{Id: &idCp, Error: "Cannot delete scheduled run created via code definition."})
				continue
			}

			ids = append(ids, idUUID)
		}
	}

	if len(ids) == 0 {
		return gen.WorkflowScheduledBulkDelete400JSONResponse(apierrors.NewAPIErrors("Provide scheduledWorkflowRunIds or filter.")), nil
	}

	deleted := make([]uuid.UUID, 0, len(ids))

	// Chunk to keep queries/params reasonable even if clients send large requests.
	const chunkSize = 200
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[i:end]

		chunkStr := make([]string, 0, len(chunk))
		chunkUUIDByStr := make(map[string]uuid.UUID, len(chunk))
		for _, id := range chunk {
			idStr := id.String()
			chunkStr = append(chunkStr, idStr)
			chunkUUIDByStr[idStr] = id
		}

		deletedIds, err := t.config.V1.WorkflowSchedules().BulkDeleteScheduledWorkflows(dbCtx, tenantId, chunkStr)
		if err != nil {
			return nil, err
		}

		deletedSet := make(map[string]struct{}, len(deletedIds))
		for _, idStr := range deletedIds {
			deletedSet[idStr] = struct{}{}
		}

		for _, id := range chunk {
			if _, ok := deletedSet[id.String()]; ok {
				deleted = append(deleted, id)
			}
		}

		// Should be rare (race conditions); report per-id errors for anything we expected to delete but didn't.
		for _, idStr := range chunkStr {
			if _, ok := deletedSet[idStr]; ok {
				continue
			}
			idCp := chunkUUIDByStr[idStr]
			errors = append(errors, gen.ScheduledWorkflowsBulkError{Id: &idCp, Error: "Failed to delete scheduled workflow."})
		}
	}

	return gen.WorkflowScheduledBulkDelete200JSONResponse(gen.ScheduledWorkflowsBulkDeleteResponse{
		DeletedIds: deleted,
		Errors:     errors,
	}), nil
}
