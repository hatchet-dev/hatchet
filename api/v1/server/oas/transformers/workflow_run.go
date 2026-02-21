package transformers

import (
	"encoding/json"
	"time"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func getEpochFromTime(t time.Time) *int {
	epoch := int(t.UnixMilli())
	return &epoch
}

func ToScheduledWorkflowsFromSQLC(scheduled *sqlcv1.ListScheduledWorkflowsRow) *gen.ScheduledWorkflows {

	var additionalMetadata map[string]interface{}

	if scheduled.AdditionalMetadata != nil {
		err := json.Unmarshal(scheduled.AdditionalMetadata, &additionalMetadata)
		if err != nil {
			return nil
		}
	}

	var workflowRunStatus *gen.WorkflowRunStatus

	if scheduled.WorkflowRunStatus.Valid {
		status := gen.WorkflowRunStatus(scheduled.WorkflowRunStatus.WorkflowRunStatus)
		workflowRunStatus = &status
	}

	input := make(map[string]interface{})
	if scheduled.Input != nil {
		json.Unmarshal(scheduled.Input, &input)
	}

	res := &gen.ScheduledWorkflows{
		Metadata:             *toAPIMetadata(scheduled.ID, scheduled.CreatedAt.Time, scheduled.UpdatedAt.Time),
		WorkflowVersionId:    scheduled.WorkflowVersionId.String(),
		WorkflowId:           scheduled.WorkflowId.String(),
		WorkflowName:         scheduled.Name,
		TenantId:             scheduled.TenantId.String(),
		TriggerAt:            scheduled.TriggerAt.Time,
		AdditionalMetadata:   &additionalMetadata,
		WorkflowRunCreatedAt: &scheduled.WorkflowRunCreatedAt.Time,
		WorkflowRunStatus:    workflowRunStatus,
		WorkflowRunId:        scheduled.WorkflowRunId,
		WorkflowRunName:      &scheduled.WorkflowRunName.String,
		Method:               gen.ScheduledWorkflowsMethod(scheduled.Method),
		Priority:             &scheduled.Priority,
		Input:                &input,
	}

	return res
}

func ToCronWorkflowsFromSQLC(cron *sqlcv1.ListCronWorkflowsRow) *gen.CronWorkflows {
	var additionalMetadata map[string]interface{}

	if cron.AdditionalMetadata != nil {
		err := json.Unmarshal(cron.AdditionalMetadata, &additionalMetadata)
		if err != nil {
			return nil
		}
	}

	input := make(map[string]interface{})
	if cron.Input != nil {
		json.Unmarshal(cron.Input, &input) //nolint:errcheck
	}

	res := &gen.CronWorkflows{
		Metadata:           *toAPIMetadata(cron.ID_2, cron.CreatedAt_2.Time, cron.UpdatedAt_2.Time),
		WorkflowVersionId:  cron.WorkflowVersionId.String(),
		WorkflowId:         cron.WorkflowId.String(),
		WorkflowName:       cron.WorkflowName,
		TenantId:           cron.TenantId.String(),
		Cron:               cron.Cron,
		AdditionalMetadata: &additionalMetadata,
		Name:               &cron.Name.String,
		Enabled:            cron.Enabled,
		Method:             gen.CronWorkflowsMethod(cron.Method),
		Priority:           &cron.Priority,
		Input:              &input,
	}

	return res
}
