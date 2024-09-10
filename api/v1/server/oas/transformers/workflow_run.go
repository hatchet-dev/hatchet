package transformers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToWorkflowRunShape(
	run *dbsqlc.GetWorkflowRunByIdRow,
	version *dbsqlc.GetWorkflowVersionByIdRow,
	jobs []*dbsqlc.ListJobRunsForWorkflowRunFullRow,
	steps []*dbsqlc.GetStepsForJobsRow,
	stepRuns []*dbsqlc.GetStepRunsForJobRunsRow,
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
		StartedAt:         &run.StartedAt.Time,
		FinishedAt:        &run.FinishedAt.Time,
		Error:             &run.Error.String,
		Status:            gen.WorkflowRunStatus(run.Status),
	}

	if version != nil {
		// TODO concurrency
		res.WorkflowVersion = ToWorkflowVersion(&version.WorkflowVersion, &version.Workflow, nil, nil, nil, nil)
	}

	res.TriggeredBy = *ToWorkflowRunTriggeredBy(&run.WorkflowRunTriggeredBy)

	if jobs != nil {
		jobRuns := make([]gen.JobRun, 0)

		for _, jobRun := range jobs {
			jobRunCp := *jobRun
			jobRuns = append(jobRuns, *ToJobRun(&jobRunCp, steps, stepRuns))

		}

		res.JobRuns = &jobRuns
	}

	return res
}

func ToWorkflowRun(
	run *dbsqlc.GetWorkflowRunByIdRow,
	jobs []*dbsqlc.ListJobRunsForWorkflowRunFullRow,
	steps []*dbsqlc.GetStepsForJobsRow,
	stepRuns []*dbsqlc.GetStepRunsForJobRunsRow,
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

	res.TriggeredBy = *ToWorkflowRunTriggeredBy(&run.WorkflowRunTriggeredBy)

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

	return res, nil
}

func ToJobRun(
	jobRun *dbsqlc.ListJobRunsForWorkflowRunFullRow,
	steps []*dbsqlc.GetStepsForJobsRow,
	stepRuns []*dbsqlc.GetStepRunsForJobRunsRow,
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

func ToStepRunFull(stepRun *dbsqlc.StepRun) *gen.StepRun {
	res := &gen.StepRun{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(stepRun.ID),
			stepRun.CreatedAt.Time,
			stepRun.UpdatedAt.Time,
		),
		Status:          gen.StepRunStatus(stepRun.Status),
		StepId:          sqlchelpers.UUIDToStr(stepRun.StepId),
		TenantId:        sqlchelpers.UUIDToStr(stepRun.TenantId),
		JobRunId:        sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StartedAt:       &stepRun.StartedAt.Time,
		FinishedAt:      &stepRun.FinishedAt.Time,
		CancelledAt:     &stepRun.CancelledAt.Time,
		CancelledError:  &stepRun.CancelledError.String,
		CancelledReason: &stepRun.CancelledReason.String,
		TimeoutAt:       &stepRun.TimeoutAt.Time,
		Error:           &stepRun.Error.String,
	}

	if stepRun.WorkerId.Valid {
		workerId := sqlchelpers.UUIDToStr(stepRun.WorkerId)
		res.WorkerId = &workerId
	}

	// TODO input/output

	return res
}

func ToStepRun(stepRun *dbsqlc.GetStepRunsForJobRunsRow) *gen.StepRun {
	res := &gen.StepRun{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(stepRun.ID),
			stepRun.CreatedAt.Time,
			stepRun.UpdatedAt.Time,
		),
		Status:          gen.StepRunStatus(stepRun.Status),
		StepId:          sqlchelpers.UUIDToStr(stepRun.StepId),
		TenantId:        sqlchelpers.UUIDToStr(stepRun.TenantId),
		JobRunId:        sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StartedAt:       &stepRun.StartedAt.Time,
		FinishedAt:      &stepRun.FinishedAt.Time,
		CancelledAt:     &stepRun.CancelledAt.Time,
		CancelledError:  &stepRun.CancelledError.String,
		CancelledReason: &stepRun.CancelledReason.String,
		TimeoutAt:       &stepRun.TimeoutAt.Time,
		Error:           &stepRun.Error.String,
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
		StepRunId:     sqlchelpers.UUIDToStr(stepRunEvent.StepRunId),
		Severity:      gen.StepRunEventSeverity(stepRunEvent.Severity),
		Reason:        gen.StepRunEventReason(stepRunEvent.Reason),
		Message:       stepRunEvent.Message,
		Count:         int(stepRunEvent.Count),
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
		CreatedAt:        stepRunArchive.CreatedAt.Time,
		StepRunId:        sqlchelpers.UUIDToStr(stepRunArchive.StepRunId),
		Order:            int(stepRunArchive.Order),
		Input:            byteSliceToStringPointer(stepRunArchive.Input),
		Output:           byteSliceToStringPointer(stepRunArchive.Output),
		Error:            &stepRunArchive.Error.String,
		CancelledAt:      &stepRunArchive.CancelledAt.Time,
		CancelledAtEpoch: getEpochFromPgTime(stepRunArchive.CancelledAt),
		CancelledError:   &stepRunArchive.CancelledError.String,
		CancelledReason:  &stepRunArchive.CancelledReason.String,
		FinishedAt:       &stepRunArchive.FinishedAt.Time,
		FinishedAtEpoch:  getEpochFromPgTime(stepRunArchive.FinishedAt),
		StartedAt:        &stepRunArchive.StartedAt.Time,
		StartedAtEpoch:   getEpochFromPgTime(stepRunArchive.StartedAt),
		TimeoutAt:        &stepRunArchive.TimeoutAt.Time,
		TimeoutAtEpoch:   getEpochFromPgTime(stepRunArchive.TimeoutAt),
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

func ToWorkflowRunTriggeredBy(triggeredBy *dbsqlc.WorkflowRunTriggeredBy) *gen.WorkflowRunTriggeredBy {
	res := &gen.WorkflowRunTriggeredBy{
		Metadata: *toAPIMetadata(sqlchelpers.UUIDToStr(triggeredBy.ID), triggeredBy.CreatedAt.Time, triggeredBy.UpdatedAt.Time),
		ParentId: sqlchelpers.UUIDToStr(triggeredBy.ParentId),
	}

	// TODO
	// if triggeredBy.EventId.Valid {
	// 	res.EventId = &event.ID
	// 	res.Event = ToEvent(event)
	// }

	// if triggeredBy.CronSchedule.Valid {
	// 	res.CronSchedule = &cron.Cron
	// 	res.CronSchedule = &triggeredBy.CronSchedule.String
	// }

	// if triggeredBy.ScheduledId.Valid {
	// 	res.ScheduledId = &scheduled.ID
	// 	res.Scheduled = ToScheduled(scheduled)
	// }

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

	var event *gen.Event

	if row.ID.Valid && row.Key.Valid {
		event = &gen.Event{
			Key:      row.Key.String,
			Metadata: *toAPIMetadata(pgUUIDToStr(row.ID), row.CreatedAt.Time, row.UpdatedAt.Time),
		}
	}

	triggeredBy := &gen.WorkflowRunTriggeredBy{
		Metadata: *toAPIMetadata(pgUUIDToStr(runTriggeredBy.ID), runTriggeredBy.CreatedAt.Time, runTriggeredBy.UpdatedAt.Time),
		ParentId: sqlchelpers.UUIDToStr(runTriggeredBy.ParentId),
	}

	if event != nil {
		triggeredBy.Event = event
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
		duration = int(run.Duration.Int32)
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
