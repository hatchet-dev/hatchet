package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type AdminClient interface {
	PutWorkflow(workflow *types.Workflow, opts ...PutOptFunc) error
	ScheduleWorkflow(workflowName string, opts ...ScheduleOptFunc) error

	// RunWorkflow triggers a workflow run and returns the run id
	RunWorkflow(workflowName string, input interface{}) (string, error)
}

type adminClientImpl struct {
	client admincontracts.WorkflowServiceClient

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader
}

func newAdmin(conn *grpc.ClientConn, opts *sharedClientOpts) AdminClient {
	return &adminClientImpl{
		client: admincontracts.NewWorkflowServiceClient(conn),
		l:      opts.l,
		v:      opts.v,
		ctx:    opts.ctxLoader,
	}
}

type putOpts struct {
}

type PutOptFunc func(*putOpts)

func defaultPutOpts() *putOpts {
	return &putOpts{}
}

func (a *adminClientImpl) PutWorkflow(workflow *types.Workflow, fs ...PutOptFunc) error {
	opts := defaultPutOpts()

	for _, f := range fs {
		f(opts)
	}

	req, err := a.getPutRequest(workflow)

	if err != nil {
		return fmt.Errorf("could not get put opts: %w", err)
	}

	_, err = a.client.PutWorkflow(a.ctx.newContext(context.Background()), req)

	if err != nil {
		return fmt.Errorf("could not create workflow: %w", err)
	}

	return nil
}

type scheduleOpts struct {
	schedules []time.Time
	input     any
}

type ScheduleOptFunc func(*scheduleOpts)

func WithInput(input any) ScheduleOptFunc {
	return func(opts *scheduleOpts) {
		opts.input = input
	}
}

func WithSchedules(schedules ...time.Time) ScheduleOptFunc {
	return func(opts *scheduleOpts) {
		opts.schedules = schedules
	}
}

func defaultScheduleOpts() *scheduleOpts {
	return &scheduleOpts{}
}

func (a *adminClientImpl) ScheduleWorkflow(workflowName string, fs ...ScheduleOptFunc) error {
	opts := defaultScheduleOpts()

	for _, f := range fs {
		f(opts)
	}

	if len(opts.schedules) == 0 {
		return fmt.Errorf("ScheduleWorkflow error: schedules are required")
	}

	pbSchedules := make([]*timestamppb.Timestamp, len(opts.schedules))

	for i, scheduled := range opts.schedules {
		pbSchedules[i] = timestamppb.New(scheduled)
	}

	inputBytes, err := json.Marshal(opts.input)

	if err != nil {
		return err
	}

	_, err = a.client.ScheduleWorkflow(a.ctx.newContext(context.Background()), &admincontracts.ScheduleWorkflowRequest{
		Name:      workflowName,
		Schedules: pbSchedules,
		Input:     string(inputBytes),
	})

	if err != nil {
		return fmt.Errorf("could not schedule workflow: %w", err)
	}

	return nil
}

func (a *adminClientImpl) RunWorkflow(workflowName string, input interface{}) (string, error) {
	inputBytes, err := json.Marshal(input)

	if err != nil {
		return "", fmt.Errorf("could not marshal input: %w", err)
	}

	res, err := a.client.TriggerWorkflow(a.ctx.newContext(context.Background()), &admincontracts.TriggerWorkflowRequest{
		Name:  workflowName,
		Input: string(inputBytes),
	})

	if err != nil {
		return "", fmt.Errorf("could not trigger workflow: %w", err)
	}

	return res.WorkflowRunId, nil
}

func (a *adminClientImpl) getPutRequest(workflow *types.Workflow) (*admincontracts.PutWorkflowRequest, error) {
	opts := &admincontracts.CreateWorkflowVersionOpts{
		Name:          workflow.Name,
		Version:       workflow.Version,
		Description:   workflow.Description,
		EventTriggers: workflow.Triggers.Events,
		CronTriggers:  workflow.Triggers.Cron,
	}

	if workflow.Concurrency != nil {
		opts.Concurrency = &admincontracts.WorkflowConcurrencyOpts{
			Action: workflow.Concurrency.ActionID,
		}

		switch workflow.Concurrency.LimitStrategy {
		case types.CancelInProgress:
			opts.Concurrency.LimitStrategy = admincontracts.ConcurrencyLimitStrategy_CANCEL_IN_PROGRESS
		case types.GroupRoundRobin:
			opts.Concurrency.LimitStrategy = admincontracts.ConcurrencyLimitStrategy_GROUP_ROUND_ROBIN
		default:
			opts.Concurrency.LimitStrategy = admincontracts.ConcurrencyLimitStrategy_CANCEL_IN_PROGRESS
		}

		// TODO: should be a pointer because users might want to set maxRuns temporarily for disabling
		if workflow.Concurrency.MaxRuns != 0 {
			opts.Concurrency.MaxRuns = workflow.Concurrency.MaxRuns
		}
	}

	jobOpts := make([]*admincontracts.CreateWorkflowJobOpts, 0)

	for jobName, job := range workflow.Jobs {
		jobOpt := &admincontracts.CreateWorkflowJobOpts{
			Name:        jobName,
			Description: job.Description,
			Timeout:     job.Timeout,
		}

		stepOpts := make([]*admincontracts.CreateWorkflowStepOpts, len(job.Steps))

		for i, step := range job.Steps {
			inputBytes, err := json.Marshal(step.With)

			if err != nil {
				return nil, fmt.Errorf("could not marshal step inputs: %w", err)
			}

			stepOpt := &admincontracts.CreateWorkflowStepOpts{
				ReadableId: step.ID,
				Action:     step.ActionID,
				Timeout:    step.Timeout,
				Inputs:     string(inputBytes),
				Parents:    step.Parents,
				Retries:    int32(step.Retries),
			}

			stepOpts[i] = stepOpt
		}

		jobOpt.Steps = stepOpts

		jobOpts = append(jobOpts, jobOpt)
	}

	opts.ScheduledTriggers = make([]*timestamppb.Timestamp, len(workflow.Triggers.Schedules))

	for i, scheduled := range workflow.Triggers.Schedules {
		opts.ScheduledTriggers[i] = timestamppb.New(scheduled)
	}

	opts.Jobs = jobOpts

	return &admincontracts.PutWorkflowRequest{
		Opts: opts,
	}, nil
}
