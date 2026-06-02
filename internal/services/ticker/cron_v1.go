package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func RunCronWorkflow(ctx context.Context, mq msgqueue.MessageQueue, tenantId uuid.UUID, cron string, workflowName string, cronName *string, input []byte, additionalMetadata map[string]interface{}, priority *int32, scheduledAt time.Time) (*uuid.UUID, error) {
	if additionalMetadata == nil {
		additionalMetadata = make(map[string]interface{})
	}

	metadata := map[string]any{
		constants.CronExpressionKey.String():  cron,
		constants.CronScheduledAtKey.String(): scheduledAt.Format(time.RFC3339),
	}

	if cronName != nil {
		metadata[constants.CronNameKey.String()] = *cronName
	}

	// copy metadata into additionalMetadata as to not override hatchet_* keys
	maps.Copy(additionalMetadata, metadata)

	additionalMetaBytes, err := json.Marshal(additionalMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not marshal additional metadata: %w", err)
	}

	externalId := uuid.New()

	opt := &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: &v1.TriggerTaskData{
			WorkflowName:       workflowName,
			Data:               input,
			AdditionalMetadata: additionalMetaBytes,
			Priority:           priority,
		},
		ExternalId: externalId,
		ShouldSkip: false,
	}

	msg, err := tasktypes.TriggerTaskMessage(tenantId, opt)
	if err != nil {
		return nil, fmt.Errorf("could not create trigger task message: %w", err)
	}

	if err := mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err != nil {
		return nil, fmt.Errorf("could not send message to task queue: %w", err)
	}

	return &externalId, nil
}

func (t *TickerImpl) runCronWorkflowV1(ctx context.Context, tenantId uuid.UUID, workflowVersion *sqlcv1.GetWorkflowVersionForEngineRow, cron, cronParentId string, cronName *string, input []byte, additionalMetadata map[string]interface{}, priority *int32, scheduledAt time.Time) error {
	_, err := RunCronWorkflow(ctx, t.mqv1, tenantId, cron, workflowVersion.WorkflowName, cronName, input, additionalMetadata, priority, scheduledAt)
	return err
}
