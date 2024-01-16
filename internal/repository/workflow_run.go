package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CreateWorkflowRunOpts struct {
	// (required) the workflow version id
	WorkflowVersionId string `validate:"required,uuid"`

	// (optional) the event id that triggered the workflow run
	TriggeringEventId *string `validate:"omitnil,uuid,required_without=Cron,required_without=ScheduledWorkflowId,excluded_with=Cron,excluded_with=ScheduledWorkflowId"`

	// (optional) the cron schedule that triggered the workflow run
	Cron         *string `validate:"omitnil,cron,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`
	CronParentId *string `validate:"omitnil,uuid,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`

	// (optional) the scheduled trigger
	ScheduledWorkflowId *string `validate:"omitnil,uuid,required_without=TriggeringEventId,required_without=Cron,excluded_with=TriggeringEventId,excluded_with=Cron"`

	// (required) the workflow jobs
	JobRuns []CreateWorkflowJobRunOpts `validate:"required,min=1,dive"`
}

func GetCreateWorkflowRunOptsFromEvent(event *db.EventModel, workflowVersion *db.WorkflowVersionModel) (*CreateWorkflowRunOpts, error) {
	eventId := event.ID
	data := event.InnerEvent.Data

	var jobRunData []byte

	var err error

	if data != nil {
		jobRunData = []byte(json.RawMessage(*data))
	}

	opts := &CreateWorkflowRunOpts{
		WorkflowVersionId: workflowVersion.ID,
		TriggeringEventId: &eventId,
	}

	opts.JobRuns, err = getJobsFromWorkflowVersion(workflowVersion, datautils.TriggeredByEvent, jobRunData)

	return opts, err
}

func GetCreateWorkflowRunOptsFromCron(cron, cronParentId string, workflowVersion *db.WorkflowVersionModel) (*CreateWorkflowRunOpts, error) {
	opts := &CreateWorkflowRunOpts{
		WorkflowVersionId: workflowVersion.ID,
		Cron:              &cron,
		CronParentId:      &cronParentId,
	}

	var err error

	opts.JobRuns, err = getJobsFromWorkflowVersion(workflowVersion, datautils.TriggeredByCron, nil)

	return opts, err
}

func GetCreateWorkflowRunOptsFromSchedule(scheduledTrigger *db.WorkflowTriggerScheduledRefModel, workflowVersion *db.WorkflowVersionModel) (*CreateWorkflowRunOpts, error) {
	data := scheduledTrigger.InnerWorkflowTriggerScheduledRef.Input
	var jobRunData []byte

	var err error

	if data != nil {
		jobRunData = []byte(json.RawMessage(*data))
	}

	opts := &CreateWorkflowRunOpts{
		WorkflowVersionId:   workflowVersion.ID,
		ScheduledWorkflowId: &scheduledTrigger.ID,
	}

	opts.JobRuns, err = getJobsFromWorkflowVersion(workflowVersion, datautils.TriggeredBySchedule, jobRunData)

	return opts, err
}

func getJobsFromWorkflowVersion(workflowVersion *db.WorkflowVersionModel, triggeredBy datautils.TriggeredBy, input []byte) ([]CreateWorkflowJobRunOpts, error) {
	resJobRunOpts := []CreateWorkflowJobRunOpts{}

	for _, job := range workflowVersion.Jobs() {
		jobOpts := CreateWorkflowJobRunOpts{
			JobId:       job.ID,
			TriggeredBy: string(triggeredBy),
			InputData:   input,
		}

		for _, step := range job.Steps() {
			stepOpts := CreateWorkflowStepRunOpts{
				StepId: step.ID,
			}

			jobOpts.StepRuns = append(jobOpts.StepRuns, stepOpts)
		}

		resJobRunOpts = append(resJobRunOpts, jobOpts)
	}

	return resJobRunOpts, nil
}

type CreateWorkflowJobRunOpts struct {
	// (required) the job id
	JobId string `validate:"required,uuid"`

	// (optional) the job run input
	InputData []byte

	TriggeredBy string

	// (required) the job step runs
	StepRuns []CreateWorkflowStepRunOpts `validate:"required,min=1,dive"`

	// (optional) the job run requeue after time, if not set this defaults to 5 seconds after the
	// current time
	RequeueAfter *time.Time `validate:"omitempty"`
}

type CreateWorkflowStepRunOpts struct {
	// (required) the step id
	StepId string `validate:"required,uuid"`
}

type ListWorkflowRunsOpts struct {
	// (optional) the workflow version id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the event id that triggered the workflow run
	EventId *string `validate:"omitempty,uuid"`

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int
}

type ListWorkflowRunsResult struct {
	Rows  []*dbsqlc.ListWorkflowRunsRow
	Count int
}

type WorkflowRunRepository interface {
	// ListWorkflowRuns returns workflow runs for a given workflow version id.
	ListWorkflowRuns(tenantId string, opts *ListWorkflowRunsOpts) (*ListWorkflowRunsResult, error)

	// CreateNewWorkflowRun creates a new workflow run for a workflow version.
	CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *CreateWorkflowRunOpts) (*db.WorkflowRunModel, error)

	// GetWorkflowRunById returns a workflow run by id.
	GetWorkflowRunById(tenantId, runId string) (*db.WorkflowRunModel, error)
}
