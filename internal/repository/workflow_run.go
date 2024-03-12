package repository

import (
	"context"
	"encoding/json"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
)

type CreateWorkflowRunOpts struct {
	// (optional) the workflow run display name
	DisplayName *string

	// (required) the workflow version id
	WorkflowVersionId string `validate:"required,uuid"`

	ManualTriggerInput *string `validate:"omitnil,required_without=TriggeringEventId,required_without=Cron,required_without=ScheduledWorkflowId,excluded_with=TriggeringEventId,excluded_with=Cron,excluded_with=ScheduledWorkflowId"`

	// (optional) the event id that triggered the workflow run
	TriggeringEventId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=Cron,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=Cron,excluded_with=ScheduledWorkflowId"`

	// (optional) the cron schedule that triggered the workflow run
	Cron         *string `validate:"omitnil,cron,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`
	CronParentId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`

	// (optional) the scheduled trigger
	ScheduledWorkflowId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=Cron,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=Cron"`

	InputData []byte

	TriggeredBy string

	GetGroupKeyRun *CreateGroupKeyRunOpts `validate:"omitempty"`
}

type CreateGroupKeyRunOpts struct {
	// (optional) the input data
	Input []byte
}

func GetCreateWorkflowRunOptsFromManual(workflowVersion *db.WorkflowVersionModel, input []byte) (*CreateWorkflowRunOpts, error) {
	opts := &CreateWorkflowRunOpts{
		DisplayName:        StringPtr(getWorkflowRunDisplayName(workflowVersion.Workflow().Name)),
		WorkflowVersionId:  workflowVersion.ID,
		ManualTriggerInput: StringPtr(string(input)),
		TriggeredBy:        string(datautils.TriggeredByManual),
		InputData:          input,
	}

	if input != nil {
		if _, hasConcurrency := workflowVersion.Concurrency(); hasConcurrency {
			opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
				Input: input,
			}
		}
	}

	return opts, nil
}

func GetCreateWorkflowRunOptsFromEvent(event *dbsqlc.GetEventForEngineRow, workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow) (*CreateWorkflowRunOpts, error) {
	eventId := sqlchelpers.UUIDToStr(event.ID)

	opts := &CreateWorkflowRunOpts{
		DisplayName:       StringPtr(getWorkflowRunDisplayName(workflowVersion.WorkflowName)),
		WorkflowVersionId: sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		TriggeringEventId: &eventId,
		TriggeredBy:       string(datautils.TriggeredByEvent),
		InputData:         event.Data,
	}

	if event.Data != nil {
		if workflowVersion.HasWorkflowConcurrency {
			opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
				Input: event.Data,
			}
		}
	}

	var err error

	return opts, err
}

func GetCreateWorkflowRunOptsFromCron(cron, cronParentId string, workflowVersion *db.WorkflowVersionModel) (*CreateWorkflowRunOpts, error) {
	opts := &CreateWorkflowRunOpts{
		DisplayName:       StringPtr(getWorkflowRunDisplayName(workflowVersion.Workflow().Name)),
		WorkflowVersionId: workflowVersion.ID,
		Cron:              &cron,
		CronParentId:      &cronParentId,
		TriggeredBy:       string(datautils.TriggeredByCron),
	}

	var err error

	return opts, err
}

func GetCreateWorkflowRunOptsFromSchedule(scheduledTrigger *db.WorkflowTriggerScheduledRefModel, workflowVersion *db.WorkflowVersionModel) (*CreateWorkflowRunOpts, error) {
	opts := &CreateWorkflowRunOpts{
		DisplayName:         StringPtr(getWorkflowRunDisplayName(workflowVersion.Workflow().Name)),
		WorkflowVersionId:   workflowVersion.ID,
		ScheduledWorkflowId: &scheduledTrigger.ID,
		TriggeredBy:         string(datautils.TriggeredBySchedule),
	}

	data := scheduledTrigger.InnerWorkflowTriggerScheduledRef.Input
	var jobRunData []byte

	var err error

	if data != nil {
		jobRunData = []byte(json.RawMessage(*data))

		if _, hasConcurrency := workflowVersion.Concurrency(); hasConcurrency {
			opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
				Input: jobRunData,
			}
		}
	}

	opts.InputData = jobRunData

	return opts, err
}

func getWorkflowRunDisplayName(workflowName string) string {
	workflowSuffix, _ := encryption.GenerateRandomBytes(3)

	return workflowName + "-" + workflowSuffix
}

type ListWorkflowRunsOpts struct {
	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the workflow version id
	WorkflowVersionId *string `validate:"omitempty,uuid"`

	// (optional) the event id that triggered the workflow run
	EventId *string `validate:"omitempty,uuid"`

	// (optional) the group key for the workflow run
	GroupKey *string

	// (optional) the status of the workflow run
	Status *db.WorkflowRunStatus

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type ListWorkflowRunsResult struct {
	Rows  []*dbsqlc.ListWorkflowRunsRow
	Count int
}

type CreateWorkflowRunPullRequestOpts struct {
	RepositoryOwner       string
	RepositoryName        string
	PullRequestID         int
	PullRequestTitle      string
	PullRequestNumber     int
	PullRequestHeadBranch string
	PullRequestBaseBranch string
	PullRequestState      string
}

type ListPullRequestsForWorkflowRunOpts struct {
	State *string
}

type ListWorkflowRunRoundRobinsOpts struct {
	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the workflow version id
	WorkflowVersionId *string `validate:"omitempty,uuid"`

	// (optional) the status of the workflow run
	Status *db.WorkflowRunStatus

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int
}

type WorkflowRunRepository interface {
	// ListWorkflowRuns returns workflow runs for a given workflow version id.
	ListWorkflowRuns(tenantId string, opts *ListWorkflowRunsOpts) (*ListWorkflowRunsResult, error)

	PopWorkflowRunsRoundRobin(tenantId, workflowVersionId string, maxRuns int) ([]*dbsqlc.WorkflowRun, error)

	// CreateNewWorkflowRun creates a new workflow run for a workflow version.
	CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *CreateWorkflowRunOpts) (*db.WorkflowRunModel, error)

	// GetWorkflowRunById returns a workflow run by id.
	GetWorkflowRunById(tenantId, runId string) (*db.WorkflowRunModel, error)

	CreateWorkflowRunPullRequest(tenantId, workflowRunId string, opts *CreateWorkflowRunPullRequestOpts) (*db.GithubPullRequestModel, error)

	ListPullRequestsForWorkflowRun(tenantId, workflowRunId string, opts *ListPullRequestsForWorkflowRunOpts) ([]db.GithubPullRequestModel, error)
}
