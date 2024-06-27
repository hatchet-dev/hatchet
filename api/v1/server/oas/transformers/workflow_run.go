package transformers

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToWorkflowRun(run *db.WorkflowRunModel) (*gen.WorkflowRun, error) {
	res := &gen.WorkflowRun{
		Metadata:          *toAPIMetadata(run.ID, run.CreatedAt, run.UpdatedAt),
		TenantId:          run.TenantID,
		Status:            gen.WorkflowRunStatus(run.Status),
		WorkflowVersionId: run.WorkflowVersionID,
	}

	if displayName, ok := run.DisplayName(); ok {
		res.DisplayName = &displayName
	}

	if startedAt, ok := run.StartedAt(); ok && !startedAt.IsZero() {
		res.StartedAt = &startedAt
	}

	if finishedAt, ok := run.FinishedAt(); ok && !finishedAt.IsZero() {
		res.FinishedAt = &finishedAt
	}

	if runErr, ok := run.Error(); ok {
		res.Error = &runErr
	}

	if run.RelationsWorkflowRun.TriggeredBy != nil {
		if triggeredBy, ok := run.TriggeredBy(); ok {
			res.TriggeredBy = *ToWorkflowRunTriggeredBy(triggeredBy)
		}
	}

	if run.RelationsWorkflowRun.WorkflowVersion != nil {
		workflowVersion := run.WorkflowVersion()

		resWorkflowVersion, err := ToWorkflowVersion(nil, workflowVersion)

		if err != nil {
			return nil, err
		}

		res.WorkflowVersion = resWorkflowVersion
	}

	if run.RelationsWorkflowRun.JobRuns != nil {
		jobRuns := make([]gen.JobRun, 0)

		for _, jobRun := range run.JobRuns() {
			jobRunCp := jobRun
			genJobRun, err := ToJobRun(&jobRunCp)

			if err != nil {
				return nil, err
			}
			jobRuns = append(jobRuns, *genJobRun)
		}

		res.JobRuns = &jobRuns
	}

	return res, nil
}

func ToJobRun(jobRun *db.JobRunModel) (*gen.JobRun, error) {
	res := &gen.JobRun{
		Metadata:      *toAPIMetadata(jobRun.ID, jobRun.CreatedAt, jobRun.UpdatedAt),
		Status:        gen.JobRunStatus(jobRun.Status),
		JobId:         jobRun.JobID,
		TenantId:      jobRun.TenantID,
		WorkflowRunId: jobRun.WorkflowRunID,
	}

	if startedAt, ok := jobRun.StartedAt(); ok && !startedAt.IsZero() {
		res.StartedAt = &startedAt
	}

	if finishedAt, ok := jobRun.FinishedAt(); ok && !finishedAt.IsZero() {
		res.FinishedAt = &finishedAt
	}

	if cancelledAt, ok := jobRun.CancelledAt(); ok && !cancelledAt.IsZero() {
		res.CancelledAt = &cancelledAt
	}

	if cancelledError, ok := jobRun.CancelledError(); ok {
		res.CancelledError = &cancelledError
	}

	if cancelledReason, ok := jobRun.CancelledReason(); ok {
		res.CancelledReason = &cancelledReason
	}

	if timeoutAt, ok := jobRun.TimeoutAt(); ok && !timeoutAt.IsZero() {
		res.TimeoutAt = &timeoutAt
	}

	if jobRun.RelationsJobRun.Job != nil {
		var err error
		job := jobRun.Job()
		res.Job, err = ToJob(job)

		if err != nil {
			return nil, err
		}
	}

	if jobRun.RelationsJobRun.WorkflowRun != nil {
		var err error
		workflowRun := jobRun.WorkflowRun()
		res.WorkflowRun, err = ToWorkflowRun(workflowRun)

		if err != nil {
			return nil, err
		}
	}

	if stepRuns := jobRun.RelationsJobRun.StepRuns; stepRuns != nil {

		orderedStepRuns := stepRuns

		sort.SliceStable(orderedStepRuns, func(i, j int) bool {
			return orderedStepRuns[i].Order < orderedStepRuns[j].Order
		})

		stepRuns := make([]gen.StepRun, 0)

		for _, stepRun := range orderedStepRuns {
			stepRunCp := stepRun
			genStepRun, err := ToStepRun(&stepRunCp)

			if err != nil {
				return nil, err
			}

			stepRuns = append(stepRuns, *genStepRun)
		}

		res.StepRuns = &stepRuns
	}

	return res, nil
}

func ToStepRun(stepRun *db.StepRunModel) (*gen.StepRun, error) {
	res := &gen.StepRun{
		Metadata: *toAPIMetadata(stepRun.ID, stepRun.CreatedAt, stepRun.UpdatedAt),
		Status:   gen.StepRunStatus(stepRun.Status),
		StepId:   stepRun.StepID,
		TenantId: stepRun.TenantID,
		JobRunId: stepRun.JobRunID,
	}

	if startedAt, ok := stepRun.StartedAt(); ok && !startedAt.IsZero() {
		res.StartedAt = &startedAt
		res.StartedAtEpoch = getEpochFromTime(startedAt)
	}

	if finishedAt, ok := stepRun.FinishedAt(); ok && !finishedAt.IsZero() {
		res.FinishedAt = &finishedAt
		res.FinishedAtEpoch = getEpochFromTime(finishedAt)
	}

	if cancelledAt, ok := stepRun.CancelledAt(); ok && !cancelledAt.IsZero() {
		res.CancelledAt = &cancelledAt
		res.CancelledAtEpoch = getEpochFromTime(cancelledAt)
	}

	if cancelledError, ok := stepRun.CancelledError(); ok {
		res.CancelledError = &cancelledError
	}

	if cancelledReason, ok := stepRun.CancelledReason(); ok {
		res.CancelledReason = &cancelledReason
	}

	if runErr, ok := stepRun.Error(); ok {
		res.Error = &runErr
	}

	if timeoutAt, ok := stepRun.TimeoutAt(); ok && !timeoutAt.IsZero() {
		res.TimeoutAt = &timeoutAt
		res.TimeoutAtEpoch = getEpochFromTime(timeoutAt)
	}

	if workerId, ok := stepRun.WorkerID(); ok {
		res.WorkerId = &workerId
	}

	if stepRun.RelationsStepRun.Step != nil {
		step := stepRun.Step()

		res.Step = ToStep(step)
	}

	if stepRun.RelationsStepRun.ChildWorkflowRuns != nil {
		childWorkflowRuns := make([]string, 0)

		for _, childWorkflowRun := range stepRun.ChildWorkflowRuns() {
			childWorkflowRuns = append(childWorkflowRuns, childWorkflowRun.ID)
		}

		res.ChildWorkflowRuns = &childWorkflowRuns
	}

	if inputData, ok := stepRun.Input(); ok {
		res.Input = repository.StringPtr(string(json.RawMessage(inputData)))
	}

	if outputData, ok := stepRun.Output(); ok {
		res.Output = repository.StringPtr(string(json.RawMessage(outputData)))
	}

	if jobRun := stepRun.RelationsStepRun.JobRun; jobRun != nil {
		var err error

		res.JobRun, err = ToJobRun(jobRun)

		if err != nil {
			return nil, err
		}
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

	res := &gen.WorkflowRun{
		Metadata:           *toAPIMetadata(workflowRunId, run.CreatedAt.Time, run.UpdatedAt.Time),
		DisplayName:        &run.DisplayName.String,
		TenantId:           pgUUIDToStr(run.TenantId),
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		Status:             gen.WorkflowRunStatus(run.Status),
		WorkflowVersionId:  pgUUIDToStr(run.WorkflowVersionId),
		WorkflowVersion:    workflowVersion,
		TriggeredBy:        *triggeredBy,
		AdditionalMetadata: &additionalMetadata,
	}

	return res
}
