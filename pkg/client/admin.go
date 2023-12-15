package client

import (
	"context"
	"encoding/json"
	"fmt"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AdminClient interface {
	PutWorkflow(workflow *types.Workflow) error
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

func (a *adminClientImpl) PutWorkflow(workflow *types.Workflow) error {
	opts, err := a.getPutOpts(workflow)

	if err != nil {
		return fmt.Errorf("could not get put opts: %w", err)
	}

	apiWorkflow, err := a.client.GetWorkflowByName(context.Background(), &admincontracts.GetWorkflowByNameRequest{
		TenantId: a.tenantId,
		Name:     opts.Opts.Name,
	})

	shouldPut := false

	if err != nil {
		// if not found, create
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
			shouldPut = true
		} else {
			return fmt.Errorf("could not get workflow: %w", err)
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
	}

	if shouldPut {
		_, err = a.client.PutWorkflow(context.Background(), opts)

		if err != nil {
			return fmt.Errorf("could not create workflow: %w", err)
		}
	}

	return nil
}

func (a *adminClientImpl) getPutOpts(workflow *types.Workflow) (*admincontracts.PutWorkflowRequest, error) {
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

	opts.Jobs = jobOpts

	return &admincontracts.PutWorkflowRequest{
		TenantId: a.tenantId,
		Opts:     opts,
	}, nil
}
