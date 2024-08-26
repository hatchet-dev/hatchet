package transformers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToWorkflowRun(run *dbsqlc.WorkflowRun) (*gen.WorkflowRun, error) {
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

	//
	// if run.RelationsWorkflowRun.TriggeredBy != nil {
	// 	if triggeredBy, ok := run.TriggeredBy(); ok {
	// 		res.TriggeredBy = *ToWorkflowRunTriggeredBy(triggeredBy)
	// 	}
	// }

	// if run.RelationsWorkflowRun.WorkflowVersion != nil {
	// 	workflowVersion := run.WorkflowVersion()

	// 	resWorkflowVersion, err := ToWorkflowVersion(nil, workflowVersion)

	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	res.WorkflowVersion = resWorkflowVersion
	// }

	// if run.RelationsWorkflowRun.JobRuns != nil {
	// 	jobRuns := make([]gen.JobRun, 0)

	// 	for _, jobRun := range run.JobRuns() {
	// 		jobRunCp := jobRun
	// 		genJobRun, err := ToJobRun(&jobRunCp)

	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		jobRuns = append(jobRuns, *genJobRun)
	// 	}

	// 	res.JobRuns = &jobRuns
	// }

	return res, nil
}

func ToJobRun(jobRun *dbsqlc.JobRun) (*gen.JobRun, error) {
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

	// if jobRun.RelationsJobRun.Job != nil {
	// 	var err error
	// 	job := jobRun.Job()
	// 	res.Job, err = ToJob(job)

	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// if jobRun.RelationsJobRun.WorkflowRun != nil {
	// 	var err error
	// 	workflowRun := jobRun.WorkflowRun()
	// 	res.WorkflowRun, err = ToWorkflowRun(workflowRun)

	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// if stepRuns := jobRun.StepRuns; stepRuns != nil {

	// 	orderedStepRuns := stepRuns

	// 	sort.SliceStable(orderedStepRuns, func(i, j int) bool {
	// 		return orderedStepRuns[i].Order < orderedStepRuns[j].Order
	// 	})

	// 	stepRuns := make([]gen.StepRun, 0)

	// 	for _, stepRun := range orderedStepRuns {
	// 		stepRunCp := stepRun
	// 		genStepRun, err := ToStepRun(&stepRunCp)

	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		stepRuns = append(stepRuns, *genStepRun)
	// 	}

	// 	res.StepRuns = &stepRuns
	// }

	return res, nil
}

func ToStepRun(stepRun *dbsqlc.StepRun) (*gen.StepRun, error) {
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

	// if stepRun.RelationsStepRun.Step != nil {
	// 	step := stepRun.Step()

	// 	res.Step = ToStep(step)
	// }

	// if stepRun.RelationsStepRun.ChildWorkflowRuns != nil {
	// 	childWorkflowRuns := make([]string, 0)

	// 	for _, childWorkflowRun := range stepRun.ChildWorkflowRuns() {
	// 		childWorkflowRuns = append(childWorkflowRuns, childWorkflowRun.ID)
	// 	}

	// 	res.ChildWorkflowRuns = &childWorkflowRuns
	// }

	// if inputData, ok := stepRun.Input(); ok {
	// 	res.Input = repository.StringPtr(string(json.RawMessage(inputData)))
	// }

	// if outputData, ok := stepRun.Output(); ok {
	// 	res.Output = repository.StringPtr(string(json.RawMessage(outputData)))
	// }

	// if jobRun := stepRun.RelationsStepRun.JobRun; jobRun != nil {
	// 	var err error

	// 	res.JobRun, err = ToJobRun(jobRun)

	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	return res, nil
}

func ToRecentStepRun(stepRun *dbsqlc.ListRecentStepRunsForWorkerRow) (*gen.RecentStepRuns, error) {

	workflowRunId := uuid.MustParse(sqlchelpers.UUIDToStr(stepRun.WorkflowRunId))

	res := &gen.RecentStepRuns{
		Metadata:      *toAPIMetadata(sqlchelpers.UUIDToStr(stepRun.ID), stepRun.CreatedAt.Time, stepRun.UpdatedAt.Time),
		Status:        gen.StepRunStatus(stepRun.Status),
		StartedAt:     &stepRun.StartedAt.Time,
		FinishedAt:    &stepRun.FinishedAt.Time,
		CancelledAt:   &stepRun.CancelledAt.Time,
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

func ToWorkflowRunTriggeredBy(triggeredBy *db.WorkflowRunTriggeredByModel) *gen.WorkflowRunTriggeredBy {
	res := &gen.WorkflowRunTriggeredBy{
		Metadata: *toAPIMetadata(triggeredBy.ID, triggeredBy.CreatedAt, triggeredBy.UpdatedAt),
		ParentId: triggeredBy.ParentID,
	}

	if event, ok := triggeredBy.Event(); ok {
		res.EventId = &event.ID
		res.Event = ToEvent(event)
	}

	if cron, ok := triggeredBy.Cron(); ok {
		res.CronSchedule = &cron.Cron
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
