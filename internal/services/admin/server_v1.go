package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (a *AdminServiceImpl) triggerWorkflowV1(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	additionalMeta := ""

	if req.AdditionalMetadata != nil {
		additionalMeta = *req.AdditionalMetadata
	}

	var parentTaskId *int64

	if req.ParentStepRunId != nil && strings.HasPrefix(*req.ParentStepRunId, "id-") {
		taskIdStr := strings.TrimPrefix(*req.ParentStepRunId, "id-")
		taskId, err := strconv.ParseInt(taskIdStr, 10, 64)

		if err != nil {
			return nil, fmt.Errorf("could not parse task id: %w", err)
		}

		parentTaskId = &taskId
	}

	var childIndex *int64

	if req.ChildIndex != nil {
		i := int64(*req.ChildIndex)

		childIndex = &i
	}

	taskExternalId, err := a.ingestSingleton(
		tenantId,
		req.Name,
		[]byte(req.Input),
		[]byte(additionalMeta),
		parentTaskId,
		childIndex,
		req.ChildKey,
	)

	if err != nil {
		return nil, fmt.Errorf("could not trigger workflow: %w", err)
	}

	return &contracts.TriggerWorkflowResponse{
		WorkflowRunId: taskExternalId,
	}, nil
}

func (a *AdminServiceImpl) bulkTriggerWorkflowV1(ctx context.Context, req *contracts.BulkTriggerWorkflowRequest) (*contracts.BulkTriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	runIds := make([]string, len(req.Workflows))

	for _, workflow := range req.Workflows {
		additionalMeta := ""

		if workflow.AdditionalMetadata != nil {
			additionalMeta = *workflow.AdditionalMetadata
		}

		var parentTaskId *int64

		if workflow.ParentStepRunId != nil && strings.HasPrefix(*workflow.ParentStepRunId, "id-") {
			taskIdStr := strings.TrimPrefix(*workflow.ParentStepRunId, "id-")
			taskId, err := strconv.ParseInt(taskIdStr, 10, 64)

			if err != nil {
				return nil, fmt.Errorf("could not parse task id: %w", err)
			}

			parentTaskId = &taskId
		}

		var childIndex *int64

		if workflow.ChildIndex != nil {
			i := int64(*workflow.ChildIndex)

			childIndex = &i
		}

		taskExternalId, err := a.ingestSingleton(
			tenantId,
			workflow.Name,
			[]byte(workflow.Input),
			[]byte(additionalMeta),
			parentTaskId,
			childIndex,
			workflow.ChildKey,
		)

		if err != nil {
			return nil, fmt.Errorf("could not trigger workflow: %w", err)
		}

		runIds = append(runIds, taskExternalId)
	}

	return &contracts.BulkTriggerWorkflowResponse{
		WorkflowRunIds: runIds,
	}, nil
}

func (i *AdminServiceImpl) ingestSingleton(tenantId, name string, data []byte, metadata []byte, parentTaskId *int64, childIndex *int64, childKey *string) (string, error) {
	taskExternalId := uuid.New().String()

	msg, err := tasktypes.TriggerTaskMessage(
		tenantId,
		taskExternalId,
		name,
		data,
		metadata,
		parentTaskId,
		childIndex,
		childKey,
	)

	if err != nil {
		return "", fmt.Errorf("could not create event task: %w", err)
	}

	var runId string

	if parentTaskId != nil {
		var k string

		if childKey != nil {
			k = *childKey
		} else {
			k = fmt.Sprintf("%d", *childIndex)
		}

		runId = fmt.Sprintf("id-%d-%s", *parentTaskId, k)
	}

	err = i.mqv1.SendMessage(context.Background(), msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return "", fmt.Errorf("could not add event to task queue: %w", err)
	}

	return runId, nil
}
