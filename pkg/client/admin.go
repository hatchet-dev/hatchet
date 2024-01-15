package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type AdminClient interface {
	PutWorkflow(workflow *types.Workflow, opts ...PutOptFunc) error
	ScheduleWorkflow(workflowName string, opts ...ScheduleOptFunc) error
}

type adminClientImpl struct {
	client admincontracts.WorkflowServiceClient

	tenantId string

	l *zerolog.Logger

	v validator.Validator
}

func newAdmin(conn *grpc.ClientConn, opts *sharedClientOpts) AdminClient {
	return &adminClientImpl{
		client:   admincontracts.NewWorkflowServiceClient(conn),
		tenantId: opts.tenantId,
		l:        opts.l,
		v:        opts.v,
	}
}

type putOpts struct {
	autoVersion bool
}

type PutOptFunc func(*putOpts)

func WithAutoVersion() PutOptFunc {
	return func(opts *putOpts) {
		opts.autoVersion = true
	}
}

func defaultPutOpts() *putOpts {
	return &putOpts{}
}

func (a *adminClientImpl) PutWorkflow(workflow *types.Workflow, fs ...PutOptFunc) error {
	opts := defaultPutOpts()

	for _, f := range fs {
		f(opts)
	}

	if workflow.Version == "" && !opts.autoVersion {
		return fmt.Errorf("PutWorkflow error: workflow version is required, or use WithAutoVersion()")
	}

	req, err := a.getPutRequest(workflow)

	if err != nil {
		return fmt.Errorf("could not get put opts: %w", err)
	}

	apiWorkflow, err := a.client.GetWorkflowByName(context.Background(), &admincontracts.GetWorkflowByNameRequest{
		TenantId: a.tenantId,
		Name:     req.Opts.Name,
	})

	shouldPut := opts.autoVersion

	if err != nil {
		// if not found, create
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
			shouldPut = true
		} else {
			return fmt.Errorf("could not get workflow: %w", err)
		}

		if workflow.Version == "" && opts.autoVersion {
			req.Opts.Version = "0.1.0"
		}
	} else {
		// if there are no versions, exit
		if len(apiWorkflow.Versions) == 0 {
			return fmt.Errorf("found workflow, but it has no versions")
		}

		// get the workflow version to determine whether to update
		if apiWorkflow.Versions[0].Version != workflow.Version {
			shouldPut = true
		}

		if workflow.Version == "" && opts.autoVersion {
			req.Opts.Version, err = bumpMinorVersion(apiWorkflow.Versions[0].Version)

			if err != nil {
				return fmt.Errorf("could not bump version: %w", err)
			}
		}
	}

	if shouldPut {
		_, err = a.client.PutWorkflow(context.Background(), req)

		if err != nil {
			return fmt.Errorf("could not create workflow: %w", err)
		}
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

	// get the workflow id from the name
	workflow, err := a.client.GetWorkflowByName(context.Background(), &admincontracts.GetWorkflowByNameRequest{
		TenantId: a.tenantId,
		Name:     workflowName,
	})

	if err != nil {
		return fmt.Errorf("could not get workflow: %w", err)
	}

	pbSchedules := make([]*timestamppb.Timestamp, len(opts.schedules))

	for i, scheduled := range opts.schedules {
		pbSchedules[i] = timestamppb.New(scheduled)
	}

	inputBytes, err := json.Marshal(opts.input)

	if err != nil {
		return err
	}

	_, err = a.client.ScheduleWorkflow(context.Background(), &admincontracts.ScheduleWorkflowRequest{
		TenantId:   a.tenantId,
		WorkflowId: workflow.Id,
		Schedules:  pbSchedules,
		Input:      string(inputBytes),
	})

	if err != nil {
		return fmt.Errorf("could not schedule workflow: %w", err)
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
		TenantId: a.tenantId,
		Opts:     opts,
	}, nil
}

func bumpMinorVersion(version string) (string, error) {
	currVersion, err := semver.NewVersion(version)

	if err != nil {
		return "", fmt.Errorf("could not parse version: %w", err)
	}

	newVersion := currVersion.IncMinor()

	return newVersion.String(), nil
}
