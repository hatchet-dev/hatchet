package transformers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
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

	var workflowRunIdPtr *uuid.UUID

	if scheduled.WorkflowRunId != uuid.Nil {
		workflowRunId := uuid.MustParse(sqlchelpers.UUIDToStr(scheduled.WorkflowRunId))
		workflowRunIdPtr = &workflowRunId
	}

	input := make(map[string]interface{})
	if scheduled.Input != nil {
		json.Unmarshal(scheduled.Input, &input)
	}

	res := &gen.ScheduledWorkflows{
		Metadata:             *toAPIMetadata(sqlchelpers.UUIDToStr(scheduled.ID), scheduled.CreatedAt.Time, scheduled.UpdatedAt.Time),
		WorkflowVersionId:    sqlchelpers.UUIDToStr(scheduled.WorkflowVersionId),
		WorkflowId:           sqlchelpers.UUIDToStr(scheduled.WorkflowId),
		WorkflowName:         scheduled.Name,
		TenantId:             sqlchelpers.UUIDToStr(scheduled.TenantId),
		TriggerAt:            scheduled.TriggerAt.Time,
		AdditionalMetadata:   &additionalMetadata,
		WorkflowRunCreatedAt: &scheduled.WorkflowRunCreatedAt.Time,
		WorkflowRunStatus:    workflowRunStatus,
		WorkflowRunId:        workflowRunIdPtr,
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

	res := &gen.CronWorkflows{
		Metadata:           *toAPIMetadata(sqlchelpers.UUIDToStr(cron.ID_2), cron.CreatedAt_2.Time, cron.UpdatedAt_2.Time),
		WorkflowVersionId:  sqlchelpers.UUIDToStr(cron.WorkflowVersionId),
		WorkflowId:         sqlchelpers.UUIDToStr(cron.WorkflowId),
		WorkflowName:       cron.WorkflowName,
		TenantId:           sqlchelpers.UUIDToStr(cron.TenantId),
		Cron:               cron.Cron,
		AdditionalMetadata: &additionalMetadata,
		Name:               &cron.Name.String,
		Enabled:            cron.Enabled,
		Method:             gen.CronWorkflowsMethod(cron.Method),
		Priority:           &cron.Priority,
	}

	return res
}
