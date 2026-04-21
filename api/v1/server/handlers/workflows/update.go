package workflows

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowUpdate(ctx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	if request.Body.IsPaused == nil {
		return gen.WorkflowUpdate400JSONResponse(gen.APIErrors{
			Errors: []gen.APIError{{Description: "isPaused is required"}},
		}), nil
	}

	type olapEvent struct {
		EventType    sqlcv1.V1EventTypeOlap
		EventMessage string
		TaskId       int64
		RetryCount   int32
	}

	var updated *sqlcv1.Workflow
	var err error
	var events []olapEvent

	if *request.Body.IsPaused {
		var movedItems []*sqlcv1.MovePausedWorkflowQueueItemsRow
		updated, movedItems, err = t.config.V1.Workflows().PauseWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID)
		if err != nil {
			return nil, err
		}

		for _, item := range movedItems {
			events = append(events, olapEvent{
				TaskId:       item.TaskID,
				RetryCount:   item.RetryCount,
				EventType:    sqlcv1.V1EventTypeOlapWORKFLOWPAUSED,
				EventMessage: "workflow paused",
			})
		}

		// cancel any running tasks for the workflow when paused
		_, err = t.proxyCancel.Do(ctx.Request().Context(), tenant, &contracts.CancelTasksRequest{
			Filter: &contracts.TasksFilter{
				WorkflowIds: []string{workflow.Workflow.ID.String()},
				Statuses:    []string{"RUNNING"},
			},
		})

		if err != nil {
			return nil, err
		}
	} else {
		var requeuedItems []*sqlcv1.RequeuePausedWorkflowQueueItemsRow
		updated, requeuedItems, err = t.config.V1.Workflows().UnpauseWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID)
		if err != nil {
			return nil, err
		}

		for _, item := range requeuedItems {
			events = append(events, olapEvent{
				TaskId:       item.TaskID,
				RetryCount:   item.RetryCount,
				EventType:    sqlcv1.V1EventTypeOlapWORKFLOWUNPAUSED,
				EventMessage: "workflow unpaused",
			})
		}
	}

	for _, event := range events {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         event.TaskId,
				RetryCount:     event.RetryCount,
				EventType:      event.EventType,
				EventMessage:   event.EventMessage,
				EventTimestamp: time.Now(),
			},
		)
		if err != nil {
			return nil, err
		}
		_ = t.config.MessageQueueV1.SendMessage(ctx.Request().Context(), msgqueue.OLAP_QUEUE, msg)
	}

	resp := transformers.ToWorkflowFromSQLC(updated)

	return gen.WorkflowUpdate200JSONResponse(*resp), nil
}
