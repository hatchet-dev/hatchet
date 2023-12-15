package transformers

import (
	"sort"
	"time"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

func ToWorkflowRun(run *db.WorkflowRunModel) (*gen.WorkflowRun, error) {
	res := &gen.WorkflowRun{
		Metadata:          *toAPIMetadata(run.ID, run.CreatedAt, run.UpdatedAt),
		TenantId:          run.TenantID,
		Status:            gen.WorkflowRunStatus(run.Status),
		WorkflowVersionId: run.WorkflowVersionID,
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
		Metadata: *toAPIMetadata(jobRun.ID, jobRun.CreatedAt, jobRun.UpdatedAt),
		Status:   gen.JobRunStatus(jobRun.Status),
		JobId:    jobRun.JobID,
		TenantId: jobRun.TenantID,
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

	orderedStepRuns := jobRun.StepRuns()

	sort.SliceStable(orderedStepRuns, func(i, j int) bool {
		return orderedStepRuns[i].Order < orderedStepRuns[j].Order
	})

	stepRuns := make([]gen.StepRun, 0)

	for _, stepRun := range orderedStepRuns {
		stepRunCp := stepRun
		stepRuns = append(stepRuns, *ToStepRun(&stepRunCp))
	}

	res.StepRuns = &stepRuns

	return res, nil
}

func ToStepRun(stepRun *db.StepRunModel) *gen.StepRun {
	res := &gen.StepRun{
		Metadata: *toAPIMetadata(stepRun.ID, stepRun.CreatedAt, stepRun.UpdatedAt),
		Status:   gen.StepRunStatus(stepRun.Status),
		StepId:   stepRun.StepID,
		TenantId: stepRun.TenantID,
		JobRunId: stepRun.JobRunID,
	}

	if startedAt, ok := stepRun.StartedAt(); ok && !startedAt.IsZero() {
		res.StartedAt = &startedAt
	}

	if finishedAt, ok := stepRun.FinishedAt(); ok && !finishedAt.IsZero() {
		res.FinishedAt = &finishedAt
	}

	if cancelledAt, ok := stepRun.CancelledAt(); ok && !cancelledAt.IsZero() {
		res.CancelledAt = &cancelledAt
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
	}

	if workerId, ok := stepRun.WorkerID(); ok {
		res.WorkerId = &workerId
	}

	if stepRun.RelationsStepRun.Step != nil {
		step := stepRun.Step()

		res.Step = ToStep(step)
	}

	return res
}

func ToWorkflowRunTriggeredBy(triggeredBy *db.WorkflowRunTriggeredByModel) *gen.WorkflowRunTriggeredBy {
	res := &gen.WorkflowRunTriggeredBy{
		Metadata: *toAPIMetadata(triggeredBy.ID, triggeredBy.CreatedAt, triggeredBy.UpdatedAt),
		ParentId: triggeredBy.ParentID,
	}

	if event, ok := triggeredBy.Event(); ok {
		res.Event = ToEvent(event)
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
		ParentId: runTriggeredBy.ParentId,
	}

	if event != nil {
		triggeredBy.Event = event
	}

	res := &gen.WorkflowRun{
		Metadata:          *toAPIMetadata(run.ID, run.CreatedAt.Time, run.UpdatedAt.Time),
		TenantId:          pgUUIDToStr(run.TenantId),
		StartedAt:         startedAt,
		FinishedAt:        finishedAt,
		Status:            gen.WorkflowRunStatus(run.Status),
		WorkflowVersionId: pgUUIDToStr(run.WorkflowVersionId),
		WorkflowVersion:   workflowVersion,
		TriggeredBy:       *triggeredBy,
	}

	return res
}
