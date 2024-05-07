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

type ChildWorkflowOpts struct {
	ParentId        string
	ParentStepRunId string
	ChildIndex      int
	ChildKey        *string
}

type AdminClient interface {
	PutWorkflow(workflow *types.Workflow, opts ...PutOptFunc) error
	ScheduleWorkflow(workflowName string, opts ...ScheduleOptFunc) error

	// RunWorkflow triggers a workflow run and returns the run id
	RunWorkflow(workflowName string, input interface{}, opts ...RunOptFunc) (string, error)

	RunChildWorkflow(workflowName string, input interface{}, opts *ChildWorkflowOpts) (string, error)

	PutRateLimit(key string, opts *types.RateLimitOpts) error
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
		return fmt.Errorf("could not create workflow %s: %w", workflow.Name, err)
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

type RunOptFunc func(*admincontracts.TriggerWorkflowRequest) error

func WithRunMetadata(metadata interface{}) RunOptFunc {
	return func(r *admincontracts.TriggerWorkflowRequest) error {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return err
		}

		metadataString := string(metadataBytes)

		r.AdditionalMetadata = &metadataString

		return nil
	}
}

func (a *adminClientImpl) RunWorkflow(workflowName string, input interface{}, options ...RunOptFunc) (string, error) {
	inputBytes, err := json.Marshal(input)

	if err != nil {
		return "", fmt.Errorf("could not marshal input: %w", err)
	}

	request := admincontracts.TriggerWorkflowRequest{
		Name:  workflowName,
		Input: string(inputBytes),
	}

	for _, optionFunc := range options {
		err = optionFunc(&request)
		if err != nil {
			return "", fmt.Errorf("could not apply run option: %w", err)
		}
	}

	res, err := a.client.TriggerWorkflow(a.ctx.newContext(context.Background()), &request)

	if err != nil {
		return "", fmt.Errorf("could not trigger workflow: %w", err)
	}

	return res.WorkflowRunId, nil
}

func (a *adminClientImpl) RunChildWorkflow(workflowName string, input interface{}, opts *ChildWorkflowOpts) (string, error) {
	inputBytes, err := json.Marshal(input)

	if err != nil {
		return "", fmt.Errorf("could not marshal input: %w", err)
	}

	childIndex := int32(opts.ChildIndex)

	res, err := a.client.TriggerWorkflow(a.ctx.newContext(context.Background()), &admincontracts.TriggerWorkflowRequest{
		Name:            workflowName,
		Input:           string(inputBytes),
		ParentId:        &opts.ParentId,
		ParentStepRunId: &opts.ParentStepRunId,
		ChildIndex:      &childIndex,
		ChildKey:        opts.ChildKey,
	})

	if err != nil {
		return "", fmt.Errorf("could not trigger child workflow: %w", err)
	}

	return res.WorkflowRunId, nil
}

func (a *adminClientImpl) PutRateLimit(key string, opts *types.RateLimitOpts) error {
	if err := a.v.Validate(opts); err != nil {
		return fmt.Errorf("could not validate rate limit opts: %w", err)
	}

	putParams := &admincontracts.PutRateLimitRequest{
		Key:   key,
		Limit: int32(opts.Max),
	}

	switch opts.Duration {
	case types.Second:
		putParams.Duration = admincontracts.RateLimitDuration_SECOND
	case types.Minute:
		putParams.Duration = admincontracts.RateLimitDuration_MINUTE
	case types.Hour:
		putParams.Duration = admincontracts.RateLimitDuration_HOUR
	default:
		putParams.Duration = admincontracts.RateLimitDuration_SECOND
	}

	_, err := a.client.PutRateLimit(a.ctx.newContext(context.Background()), putParams)

	if err != nil {
		return fmt.Errorf("could not upsert rate limit: %w", err)
	}

	return nil
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

	if workflow.OnFailureJob != nil {
		onFailureJob, err := a.getJobOpts("on-failure", workflow.OnFailureJob)

		if err != nil {
			return nil, fmt.Errorf("could not get on failure job opts: %w", err)
		}

		opts.OnFailureJob = onFailureJob
	}

	jobOpts := make([]*admincontracts.CreateWorkflowJobOpts, 0)

	for jobName, job := range workflow.Jobs {
		jobCp := job

		res, err := a.getJobOpts(jobName, &jobCp)

		if err != nil {
			return nil, fmt.Errorf("could not get job opts: %w", err)
		}

		jobOpts = append(jobOpts, res)
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

func (a *adminClientImpl) getJobOpts(jobName string, job *types.WorkflowJob) (*admincontracts.CreateWorkflowJobOpts, error) {
	jobOpt := &admincontracts.CreateWorkflowJobOpts{
		Name:        jobName,
		Description: job.Description,
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

		for _, rateLimit := range step.RateLimits {
			stepOpt.RateLimits = append(stepOpt.RateLimits, &admincontracts.CreateStepRateLimit{
				Key:   rateLimit.Key,
				Units: int32(rateLimit.Units),
			})
		}

		stepOpts[i] = stepOpt
	}

	jobOpt.Steps = stepOpts

	return jobOpt, nil
}
