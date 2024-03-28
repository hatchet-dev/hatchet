package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

func (a *AdminServiceImpl) TriggerWorkflow(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	workflow, err := a.repo.Workflow().GetWorkflowByName(
		tenantId,
		req.Name,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(
				codes.NotFound,
				"workflow not found",
			)
		}

		return nil, fmt.Errorf("could not get workflow by name: %w", err)
	}

	workflowVersion, err := a.repo.Workflow().GetLatestWorkflowVersion(
		tenantId,
		sqlchelpers.UUIDToStr(workflow.ID),
	)

	if err != nil {
		return nil, fmt.Errorf("could not get latest workflow version: %w", err)
	}

	createOpts, err := repository.GetCreateWorkflowRunOptsFromManual(workflowVersion, []byte(req.Input))

	if err != nil {
		return nil, fmt.Errorf("could not create workflow run opts: %w", err)
	}

	workflowRunId, err := a.repo.WorkflowRun().CreateNewWorkflowRun(ctx, tenantId, createOpts)

	if err != nil {
		return nil, fmt.Errorf("could not create workflow run: %w", err)
	}

	// send to workflow processing queue
	err = a.mq.AddMessage(
		context.Background(),
		msgqueue.WORKFLOW_PROCESSING_QUEUE,
		tasktypes.WorkflowRunQueuedToTask(tenantId, workflowRunId),
	)
	if err != nil {
		return nil, fmt.Errorf("could not queue workflow run: %w", err)
	}

	return &contracts.TriggerWorkflowResponse{
		WorkflowRunId: workflowRunId,
	}, nil
}

func (a *AdminServiceImpl) PutWorkflow(ctx context.Context, req *contracts.PutWorkflowRequest) (*contracts.WorkflowVersion, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	createOpts, err := getCreateWorkflowOpts(req)

	if err != nil {
		return nil, err
	}

	// determine if workflow already exists
	var workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow
	var oldWorkflowVersion *dbsqlc.GetWorkflowVersionForEngineRow

	currWorkflow, err := a.repo.Workflow().GetWorkflowByName(
		tenantId,
		req.Opts.Name,
	)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		// workflow does not exist, create it
		workflowVersion, err = a.repo.Workflow().CreateNewWorkflow(
			tenantId,
			createOpts,
		)

		if err != nil {
			return nil, err
		}
	} else {
		oldWorkflowVersion, err = a.repo.Workflow().GetLatestWorkflowVersion(
			tenantId,
			sqlchelpers.UUIDToStr(currWorkflow.ID),
		)

		if err != nil {
			return nil, err
		}

		// workflow exists, look at checksum
		newCS, err := createOpts.Checksum()

		if err != nil {
			return nil, err
		}

		if oldWorkflowVersion.WorkflowVersion.Checksum != newCS {
			workflowVersion, err = a.repo.Workflow().CreateWorkflowVersion(
				tenantId,
				createOpts,
			)

			if err != nil {
				return nil, err
			}
		} else {
			workflowVersion = oldWorkflowVersion
		}
	}

	resp := toWorkflowVersion(workflowVersion)

	return resp, nil
}

func (a *AdminServiceImpl) ScheduleWorkflow(ctx context.Context, req *contracts.ScheduleWorkflowRequest) (*contracts.WorkflowVersion, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	workflow, err := a.repo.Workflow().GetWorkflowByName(
		tenantId,
		req.Name,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(
				codes.NotFound,
				"workflow not found",
			)
		}

		return nil, fmt.Errorf("could not get workflow by name: %w", err)
	}

	workflowId := sqlchelpers.UUIDToStr(workflow.ID)

	currWorkflow, err := a.repo.Workflow().GetLatestWorkflowVersion(
		tenantId,
		workflowId,
	)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("workflow with id %s does not exist", workflowId)
		}

		return nil, err
	}

	dbSchedules := make([]time.Time, len(req.Schedules))

	for i, scheduledTrigger := range req.Schedules {
		dbSchedules[i] = scheduledTrigger.AsTime()
	}

	workflowVersionId := sqlchelpers.UUIDToStr(currWorkflow.WorkflowVersion.ID)

	_, err = a.repo.Workflow().CreateSchedules(
		tenantId,
		workflowVersionId,
		&repository.CreateWorkflowSchedulesOpts{
			ScheduledTriggers: dbSchedules,
			Input:             []byte(req.Input),
		},
	)

	if err != nil {
		return nil, err
	}

	resp := toWorkflowVersion(currWorkflow)

	return resp, nil
}

func getCreateWorkflowOpts(req *contracts.PutWorkflowRequest) (*repository.CreateWorkflowVersionOpts, error) {
	jobs := make([]repository.CreateWorkflowJobOpts, len(req.Opts.Jobs))

	for i, job := range req.Opts.Jobs {
		jobCp := job

		steps := make([]repository.CreateWorkflowStepOpts, len(job.Steps))

		for j, step := range job.Steps {
			stepCp := step

			parsedAction, err := types.ParseActionID(step.Action)

			if err != nil {
				return nil, err
			}

			retries := int(stepCp.Retries)

			steps[j] = repository.CreateWorkflowStepOpts{
				ReadableId: stepCp.ReadableId,
				Action:     parsedAction.String(),
				Timeout:    &stepCp.Timeout,
				Parents:    stepCp.Parents,
				Retries:    &retries,
			}

			if stepCp.UserData != "" {
				steps[j].UserData = &stepCp.UserData
			}
		}

		jobs[i] = repository.CreateWorkflowJobOpts{
			Name:        jobCp.Name,
			Description: &jobCp.Description,
			Timeout:     &jobCp.Timeout,
			Steps:       steps,
		}
	}

	scheduledTriggers := make([]time.Time, 0)

	for _, trigger := range req.Opts.ScheduledTriggers {
		scheduledTriggers = append(scheduledTriggers, trigger.AsTime())
	}

	var concurrency *repository.CreateWorkflowConcurrencyOpts

	if req.Opts.Concurrency != nil {
		var limitStrategy *string

		if req.Opts.Concurrency.LimitStrategy.String() != "" {
			limitStrategy = repository.StringPtr(req.Opts.Concurrency.LimitStrategy.String())
		}

		concurrency = &repository.CreateWorkflowConcurrencyOpts{
			Action:        req.Opts.Concurrency.Action,
			LimitStrategy: limitStrategy,
		}

		if req.Opts.Concurrency.MaxRuns != 0 {
			concurrency.MaxRuns = &req.Opts.Concurrency.MaxRuns
		}
	}

	return &repository.CreateWorkflowVersionOpts{
		Name:              req.Opts.Name,
		Concurrency:       concurrency,
		Description:       &req.Opts.Description,
		Version:           &req.Opts.Version,
		EventTriggers:     req.Opts.EventTriggers,
		CronTriggers:      req.Opts.CronTriggers,
		ScheduledTriggers: scheduledTriggers,
		Jobs:              jobs,
		ScheduleTimeout:   req.Opts.ScheduleTimeout,
	}, nil
}

func toWorkflowVersion(workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow) *contracts.WorkflowVersion {
	version := &contracts.WorkflowVersion{
		Id:         sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		CreatedAt:  timestamppb.New(workflowVersion.WorkflowVersion.CreatedAt.Time),
		UpdatedAt:  timestamppb.New(workflowVersion.WorkflowVersion.UpdatedAt.Time),
		Order:      int32(workflowVersion.WorkflowVersion.Order),
		WorkflowId: sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.WorkflowId),
	}

	if workflowVersion.WorkflowVersion.Version.String != "" {
		version.Version = workflowVersion.WorkflowVersion.Version.String
	}

	return version
}
