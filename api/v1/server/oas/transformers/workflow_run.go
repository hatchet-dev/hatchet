package transformers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToWorkflowRunShape(
	run *dbsqlc.GetWorkflowRunByIdRow,
	version *dbsqlc.GetWorkflowVersionByIdRow,
	jobs []*dbsqlc.ListJobRunsForWorkflowRunFullRow,
	steps []*dbsqlc.GetStepsForJobsRow,
	stepRuns []*repository.StepRunForJobRun,
) *gen.WorkflowRunShape {
	res := &gen.WorkflowRunShape{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(run.ID),
			run.CreatedAt.Time,
			run.UpdatedAt.Time,
		),
		TenantId:          sqlchelpers.UUIDToStr(run.TenantId),
		WorkflowVersionId: sqlchelpers.UUIDToStr(run.WorkflowVersionId),
		DisplayName:       &run.DisplayName.String,
		Error:             &run.Error.String,
		Status:            gen.WorkflowRunStatus(run.Status),
	}

	if run.StartedAt.Valid {
		res.StartedAt = &run.StartedAt.Time
	}

	if run.FinishedAt.Valid {
		res.FinishedAt = &run.FinishedAt.Time
	}

	if run.Duration.Valid {
		duration := int(run.Duration.Int64)
		res.Duration = &duration
	}

	if version != nil {
		// TODO concurrency
		workflowId := sqlchelpers.UUIDToStr(version.Workflow.ID)
		res.WorkflowId = &workflowId
		res.WorkflowVersion = ToWorkflowVersion(&version.WorkflowVersion, &version.Workflow, nil, nil, nil, nil)
	}

	res.TriggeredBy = *ToWorkflowRunTriggeredBy(run.ParentId, &run.WorkflowRunTriggeredBy)

	if jobs != nil {
		jobRuns := make([]gen.JobRun, 0)

		for _, jobRun := range jobs {
			jobRunCp := *jobRun
			jobRuns = append(jobRuns, *ToJobRun(&jobRunCp, steps, stepRuns))

		}

		res.JobRuns = &jobRuns
	}

	if run.AdditionalMetadata != nil {

		additionalMetadata := make(map[string]interface{})
		err := json.Unmarshal(run.AdditionalMetadata, &additionalMetadata)

		if err == nil {
			res.AdditionalMetadata = &additionalMetadata
		}
	}

	return res
}

func ToWorkflowRun(
	run *dbsqlc.GetWorkflowRunByIdRow,
	jobs []*dbsqlc.ListJobRunsForWorkflowRunFullRow,
	steps []*dbsqlc.GetStepsForJobsRow,
	stepRuns []*repository.StepRunForJobRun,
) (*gen.WorkflowRun, error) {
	res := &gen.WorkflowRun{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(run.ID),
			run.CreatedAt.Time,
			run.UpdatedAt.Time,
		),
		TenantId:          sqlchelpers.UUIDToStr(run.TenantId),
		Status:            gen.WorkflowRunStatus(run.Status),
		WorkflowVersionId: sqlchelpers.UUIDToStr(run.WorkflowVersionId),
		DisplayName:       &run.DisplayName.String,
		StartedAt:         &run.StartedAt.Time,
		FinishedAt:        &run.FinishedAt.Time,
		Error:             &run.Error.String,
	}

	res.TriggeredBy = *ToWorkflowRunTriggeredBy(run.ParentId, &run.WorkflowRunTriggeredBy)

	if run.WorkflowVersionId.Valid {
		res.WorkflowVersion = ToWorkflowVersion(&run.WorkflowVersion, &run.Workflow, nil, nil, nil, nil)
	}

	if jobs != nil {
		jobRuns := make([]gen.JobRun, 0)

		for _, jobRun := range jobs {
			jobRunCp := *jobRun
			jobRuns = append(jobRuns, *ToJobRun(&jobRunCp, steps, stepRuns))
		}

		res.JobRuns = &jobRuns
	}

	if run.AdditionalMetadata != nil {

		additionalMetadata := make(map[string]interface{})
		err := json.Unmarshal(run.AdditionalMetadata, &additionalMetadata)

		if err != nil {
			return nil, err
		}

		res.AdditionalMetadata = &additionalMetadata
	}

	return res, nil
}

func ToJobRun(
	jobRun *dbsqlc.ListJobRunsForWorkflowRunFullRow,
	steps []*dbsqlc.GetStepsForJobsRow,
	stepRuns []*repository.StepRunForJobRun,
) *gen.JobRun {

	res := &gen.JobRun{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(jobRun.ID),
			jobRun.CreatedAt.Time,
			jobRun.UpdatedAt.Time,
		),
		Status:          gen.JobRunStatus(jobRun.Status),
		JobId:           sqlchelpers.UUIDToStr(jobRun.JobId),
		TenantId:        sqlchelpers.UUIDToStr(jobRun.TenantId),
		WorkflowRunId:   sqlchelpers.UUIDToStr(jobRun.WorkflowRunId),
		StartedAt:       &jobRun.StartedAt.Time,
		FinishedAt:      &jobRun.FinishedAt.Time,
		CancelledAt:     &jobRun.CancelledAt.Time,
		CancelledError:  &jobRun.CancelledError.String,
		CancelledReason: &jobRun.CancelledReason.String,
		TimeoutAt:       &jobRun.TimeoutAt.Time,
	}

	res.Job = ToJob(&jobRun.Job, steps)

	resStepRuns := make([]gen.StepRun, 0)

	for _, stepRun := range stepRuns {

		if stepRun.JobRunId != jobRun.ID {
			continue
		}

		stepRunCp := stepRun
		genStepRun := ToStepRun(stepRunCp)
		resStepRuns = append(resStepRuns, *genStepRun)
	}

	res.StepRuns = &resStepRuns

	return res
}

func ToStepRunFull(stepRun *repository.GetStepRunFull) *gen.StepRun {
	res := &gen.StepRun{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(stepRun.ID),
			stepRun.CreatedAt.Time,
			stepRun.UpdatedAt.Time,
		),
		Status:   gen.StepRunStatus(stepRun.Status),
		StepId:   sqlchelpers.UUIDToStr(stepRun.StepId),
		TenantId: sqlchelpers.UUIDToStr(stepRun.TenantId),
		JobRunId: sqlchelpers.UUIDToStr(stepRun.JobRunId),
	}

	if stepRun.CancelledError.Valid {
		cancelledError := stepRun.CancelledError.String
		res.CancelledError = &cancelledError
	}

	if stepRun.CancelledReason.Valid {
		cancelledReason := stepRun.CancelledReason.String
		res.CancelledReason = &cancelledReason
	}

	if stepRun.Error.Valid {
		err := stepRun.Error.String
		res.Error = &err
	}

	if stepRun.StartedAt.Valid {
		res.StartedAt = &stepRun.StartedAt.Time
		res.StartedAtEpoch = getEpochFromPgTime(stepRun.StartedAt)
	}

	if stepRun.FinishedAt.Valid {
		res.FinishedAt = &stepRun.FinishedAt.Time
		res.FinishedAtEpoch = getEpochFromPgTime(stepRun.FinishedAt)
	}

	if stepRun.CancelledAt.Valid {
		res.CancelledAt = &stepRun.CancelledAt.Time
		res.CancelledAtEpoch = getEpochFromPgTime(stepRun.CancelledAt)
	}

	if stepRun.TimeoutAt.Valid {
		res.TimeoutAt = &stepRun.TimeoutAt.Time
		res.TimeoutAtEpoch = getEpochFromPgTime(stepRun.TimeoutAt)
	}

	if stepRun.ChildWorkflowRuns != nil {
		res.ChildWorkflowRuns = &stepRun.ChildWorkflowRuns
	}

	if stepRun.WorkerId.Valid {
		workerId := sqlchelpers.UUIDToStr(stepRun.WorkerId)
		res.WorkerId = &workerId
	}

	if stepRun.Input != nil {
		input := string(stepRun.Input)
		res.Input = &input
	}

	if stepRun.Output != nil {
		output := string(stepRun.Output)
		res.Output = &output
	}

	return res
}

func ToStepRun(stepRun *repository.StepRunForJobRun) *gen.StepRun {
	res := &gen.StepRun{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(stepRun.ID),
			stepRun.CreatedAt.Time,
			stepRun.UpdatedAt.Time,
		),
		Status:              gen.StepRunStatus(stepRun.Status),
		StepId:              sqlchelpers.UUIDToStr(stepRun.StepId),
		TenantId:            sqlchelpers.UUIDToStr(stepRun.TenantId),
		JobRunId:            sqlchelpers.UUIDToStr(stepRun.JobRunId),
		ChildWorkflowsCount: &stepRun.ChildWorkflowsCount,
		Output:              byteSliceToStringPointer(stepRun.Output),
	}

	if stepRun.CancelledError.Valid {
		cancelledError := stepRun.CancelledError.String
		res.CancelledError = &cancelledError
	}

	if stepRun.CancelledReason.Valid {
		cancelledReason := stepRun.CancelledReason.String
		res.CancelledReason = &cancelledReason
	}

	if stepRun.Error.Valid {
		err := stepRun.Error.String
		res.Error = &err
	}

	if stepRun.StartedAt.Valid {
		res.StartedAt = &stepRun.StartedAt.Time
		res.StartedAtEpoch = getEpochFromPgTime(stepRun.StartedAt)
	}

	if stepRun.FinishedAt.Valid {
		res.FinishedAt = &stepRun.FinishedAt.Time
		res.FinishedAtEpoch = getEpochFromPgTime(stepRun.FinishedAt)
	}

	if stepRun.CancelledAt.Valid {
		res.CancelledAt = &stepRun.CancelledAt.Time
		res.CancelledAtEpoch = getEpochFromPgTime(stepRun.CancelledAt)
	}

	if stepRun.TimeoutAt.Valid {
		res.TimeoutAt = &stepRun.TimeoutAt.Time
		res.TimeoutAtEpoch = getEpochFromPgTime(stepRun.TimeoutAt)
	}

	if stepRun.WorkerId.Valid {
		workerId := sqlchelpers.UUIDToStr(stepRun.WorkerId)
		res.WorkerId = &workerId
	}

	return res
}

func ToRecentStepRun(stepRun *dbsqlc.GetStepRunForEngineRow) (*gen.RecentStepRuns, error) {
	workflowRunId := uuid.MustParse(sqlchelpers.UUIDToStr(stepRun.WorkflowRunId))

	res := &gen.RecentStepRuns{
		Metadata:      *toAPIMetadata(sqlchelpers.UUIDToStr(stepRun.SRID), stepRun.SRCreatedAt.Time, stepRun.SRUpdatedAt.Time),
		Status:        gen.StepRunStatus(stepRun.SRStatus),
		StartedAt:     &stepRun.SRStartedAt.Time,
		FinishedAt:    &stepRun.SRFinishedAt.Time,
		CancelledAt:   &stepRun.SRCancelledAt.Time,
		ActionId:      stepRun.ActionId,
		WorkflowRunId: workflowRunId,
	}

	return res, nil
}

func ToStepRunEvent(stepRunEvent *dbsqlc.StepRunEvent) *gen.StepRunEvent {
	res := &gen.StepRunEvent{
		Id:            int(stepRunEvent.ID),
		TimeFirstSeen: stepRunEvent.TimeFirstSeen.Time,
		TimeLastSeen:  stepRunEvent.TimeLastSeen.Time,
		Severity:      gen.StepRunEventSeverity(stepRunEvent.Severity),
		Reason:        gen.StepRunEventReason(stepRunEvent.Reason),
		Message:       stepRunEvent.Message,
		Count:         int(stepRunEvent.Count),
	}

	if stepRunEvent.StepRunId.Valid {
		srId := sqlchelpers.UUIDToStr(stepRunEvent.StepRunId)
		res.StepRunId = &srId
	}

	if stepRunEvent.WorkflowRunId.Valid {
		wrId := sqlchelpers.UUIDToStr(stepRunEvent.WorkflowRunId)
		res.WorkflowRunId = &wrId
	}

	if stepRunEvent.Data != nil {
		data := make(map[string]interface{})

		json.Unmarshal(stepRunEvent.Data, &data) // nolint:errcheck

		res.Data = &data
	}

	return res
}

func ToStepRunArchive(stepRunArchive *dbsqlc.StepRunResultArchive) *gen.StepRunArchive {
	res := &gen.StepRunArchive{
		CreatedAt:  stepRunArchive.CreatedAt.Time,
		StepRunId:  sqlchelpers.UUIDToStr(stepRunArchive.StepRunId),
		RetryCount: int(stepRunArchive.RetryCount),
		Order:      int(stepRunArchive.Order),
		Input:      byteSliceToStringPointer(stepRunArchive.Input),
		Output:     byteSliceToStringPointer(stepRunArchive.Output),
	}

	if stepRunArchive.CancelledAt.Valid {
		res.CancelledAt = &stepRunArchive.CancelledAt.Time
		res.CancelledAtEpoch = getEpochFromPgTime(stepRunArchive.CancelledAt)
	}

	if stepRunArchive.Error.Valid {
		res.Error = &stepRunArchive.Error.String
	}

	if stepRunArchive.CancelledError.Valid {
		res.CancelledError = &stepRunArchive.CancelledError.String
	}

	if stepRunArchive.CancelledReason.Valid {
		res.CancelledReason = &stepRunArchive.CancelledReason.String
	}

	if stepRunArchive.FinishedAt.Valid {
		res.FinishedAt = &stepRunArchive.FinishedAt.Time
		res.FinishedAtEpoch = getEpochFromPgTime(stepRunArchive.FinishedAt)
	}

	if stepRunArchive.TimeoutAt.Valid {
		res.TimeoutAt = &stepRunArchive.TimeoutAt.Time
		res.TimeoutAtEpoch = getEpochFromPgTime(stepRunArchive.TimeoutAt)
	}

	if stepRunArchive.StartedAt.Valid {
		res.StartedAt = &stepRunArchive.StartedAt.Time
		res.StartedAtEpoch = getEpochFromPgTime(stepRunArchive.StartedAt)
	}

	return res
}

func byteSliceToStringPointer(b []byte) *string {
	if b == nil {
		return nil
	}
	s := string(b)
	return &s
}

func getEpochFromPgTime(t pgtype.Timestamp) *int {
	return getEpochFromTime(t.Time)
}

func getEpochFromTime(t time.Time) *int {
	epoch := int(t.UnixMilli())
	return &epoch
}

func ToWorkflowRunTriggeredBy(parentWorkflowRunId pgtype.UUID, triggeredBy *dbsqlc.WorkflowRunTriggeredBy) *gen.WorkflowRunTriggeredBy {
	res := &gen.WorkflowRunTriggeredBy{
		Metadata: *toAPIMetadata(sqlchelpers.UUIDToStr(triggeredBy.ID), triggeredBy.CreatedAt.Time, triggeredBy.UpdatedAt.Time),
	}

	if parentWorkflowRunId.Valid {
		parent := sqlchelpers.UUIDToStr(parentWorkflowRunId)
		res.ParentWorkflowRunId = &parent
	}

	if triggeredBy.EventId.Valid {
		eventId := sqlchelpers.UUIDToStr(triggeredBy.EventId)
		res.EventId = &eventId
	}

	if triggeredBy.CronParentId.Valid {
		cronParentId := sqlchelpers.UUIDToStr(triggeredBy.CronParentId)
		res.CronParentId = &cronParentId
	}

	if triggeredBy.CronSchedule.Valid {
		res.CronSchedule = &triggeredBy.CronSchedule.String
	}

	return res
}

func ToWorkflowRunFromSQLC(row *dbsqlc.ListWorkflowRunsRow) *gen.WorkflowRun {
	run := row.WorkflowRun
	runTriggeredBy := row.WorkflowRunTriggeredBy

	workflow := ToWorkflowFromSQLC(&row.Workflow)
	workflowVersion := ToWorkflowVersionFromSQLC(&row.WorkflowVersion, workflow)
	var startedAt *time.Time

	if !run.StartedAt.Time.IsZero() {
		startedAt = &run.StartedAt.Time
	}

	var finishedAt *time.Time

	if !run.FinishedAt.Time.IsZero() {
		finishedAt = &run.FinishedAt.Time
	}

	triggeredBy := &gen.WorkflowRunTriggeredBy{
		Metadata: *toAPIMetadata(pgUUIDToStr(runTriggeredBy.ID), runTriggeredBy.CreatedAt.Time, runTriggeredBy.UpdatedAt.Time),
	}

	if run.ParentId.Valid {
		parentId := sqlchelpers.UUIDToStr(run.ParentId)
		triggeredBy.ParentWorkflowRunId = &parentId
	}

	workflowRunId := sqlchelpers.UUIDToStr(run.ID)

	var additionalMetadata map[string]interface{}

	if run.AdditionalMetadata != nil {
		err := json.Unmarshal(run.AdditionalMetadata, &additionalMetadata)

		if err != nil {
			return nil
		}

	}

	var duration int

	if run.Duration.Valid {
		duration = int(run.Duration.Int64)
	}

	res := &gen.WorkflowRun{
		Metadata:           *toAPIMetadata(workflowRunId, run.CreatedAt.Time, run.UpdatedAt.Time),
		DisplayName:        &run.DisplayName.String,
		TenantId:           pgUUIDToStr(run.TenantId),
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		Duration:           &duration,
		Status:             gen.WorkflowRunStatus(run.Status),
		WorkflowVersionId:  pgUUIDToStr(run.WorkflowVersionId),
		WorkflowVersion:    workflowVersion,
		TriggeredBy:        *triggeredBy,
		AdditionalMetadata: &additionalMetadata,
	}

	return res
}

func ToScheduledWorkflowsFromSQLC(scheduled *dbsqlc.ListScheduledWorkflowsRow) *gen.ScheduledWorkflows {

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

	if scheduled.WorkflowRunId.Valid {
		workflowRunId := uuid.MustParse(sqlchelpers.UUIDToStr(scheduled.WorkflowRunId))
		workflowRunIdPtr = &workflowRunId
	}

	input := make(map[string]interface{})
	if scheduled.Input != nil {
		err := json.Unmarshal(scheduled.Input, &input)
		if err != nil {
			return nil
		}
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

func ToCronWorkflowsFromSQLC(cron *dbsqlc.ListCronWorkflowsRow) *gen.CronWorkflows {
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
