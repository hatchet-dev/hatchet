package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TickerImpl) runCronWorkflowV1(ctx context.Context, tenantId string, workflowVersion *sqlcv1.GetWorkflowVersionForEngineRow, cron, cronParentId string, cronName *string, input []byte, additionalMetadata map[string]interface{}, priority *int32) error {
	if additionalMetadata == nil {
		additionalMetadata = make(map[string]interface{})
	}

	metadata := map[string]any{
		constants.TriggeredByKey.String():    "cron",
		constants.CronExpressionKey.String(): cron,
	}

	if cronName != nil {
		metadata[constants.CronNameKey.String()] = *cronName
	}

	// copy metadata into additionalMetadata as to not override hatchet_* keys
	maps.Copy(additionalMetadata, metadata)

	additionalMetaBytes, err := json.Marshal(additionalMetadata)
	if err != nil {
		return fmt.Errorf("could not marshal additional metadata: %w", err)
	}

	// send workflow run to task controller
	opt := &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: &v1.TriggerTaskData{
			WorkflowName:       workflowVersion.WorkflowName,
			Data:               input,
			AdditionalMetadata: additionalMetaBytes,
			Priority:           priority,
		},
		ExternalId: uuid.NewString(),
		ShouldSkip: false,
	}

	msg, err := tasktypes.TriggerTaskMessage(
		tenantId,
		opt,
	)

	if err != nil {
		return fmt.Errorf("could not create trigger task message: %w", err)
	}

	err = t.mqv1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return fmt.Errorf("could not send message to task queue: %w", err)
	}

	return nil
}
