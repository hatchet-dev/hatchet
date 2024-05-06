package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	isParentTriggered := req.ParentId != nil

	// if there's a parent id passed in, we query for an existing workflow run which matches these params
	if isParentTriggered {
		if req.ParentStepRunId == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"parent step run id is required when parent id is provided",
			)
		}

		if req.ChildIndex == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"child index is required when parent id is provided",
			)
		}

		workflowRun, err := a.repo.WorkflowRun().GetChildWorkflowRun(
			ctx,
			*req.ParentId,
			*req.ParentStepRunId,
			int(*req.ChildIndex),
			req.ChildKey,
		)

		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("could not get child workflow run: %w", err)
			}
		}

		if err == nil && workflowRun != nil {
			return &contracts.TriggerWorkflowResponse{
				WorkflowRunId: sqlchelpers.UUIDToStr(workflowRun.ID),
			}, nil
		}
	}

	workflow, err := a.repo.Workflow().GetWorkflowByName(
		ctx,
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
		ctx,
		tenantId,
		sqlchelpers.UUIDToStr(workflow.ID),
	)

	if err != nil {
		return nil, fmt.Errorf("could not get latest workflow version: %w", err)
	}

	var createOpts *repository.CreateWorkflowRunOpts

	var additionalMetadata map[string]interface{}

	if req.AdditionalMetadata != nil {
		err := json.Unmarshal([]byte(*req.AdditionalMetadata), &additionalMetadata)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal additional metadata: %w", err)
		}
	}

	if isParentTriggered {
		createOpts, err = repository.GetCreateWorkflowRunOptsFromParent(
			workflowVersion,
			[]byte(req.Input),
			// we have already checked for nil values above
			*req.ParentId,
			*req.ParentStepRunId,
			int(*req.ChildIndex),
			req.ChildKey,
			additionalMetadata,
		)
	} else {
		createOpts, err = repository.GetCreateWorkflowRunOptsFromManual(workflowVersion, []byte(req.Input), additionalMetadata)
	}

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

	// validate createOpts
	if apiErrors, err := a.v.ValidateAPI(createOpts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			apiErrors.String(),
		)
	}

	// determine if workflow already exists
	var workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow
	var oldWorkflowVersion *dbsqlc.GetWorkflowVersionForEngineRow

	currWorkflow, err := a.repo.Workflow().GetWorkflowByName(
		ctx,
		tenantId,
		req.Opts.Name,
	)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		// workflow does not exist, create it
		workflowVersion, err = a.repo.Workflow().CreateNewWorkflow(
			ctx,
			tenantId,
			createOpts,
		)

		if err != nil {
			return nil, err
		}
	} else {
		oldWorkflowVersion, err = a.repo.Workflow().GetLatestWorkflowVersion(
			ctx,
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
				ctx,
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
		ctx,
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
		ctx,
		tenantId,
		workflowId,
	)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("workflow with id %s does not exist", workflowId)
		}

		return nil, err
	}

	isParentTriggered := req.ParentId != nil

	if isParentTriggered {
		if req.ParentStepRunId == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"parent step run id is required when parent id is provided",
			)
		}

		if req.ChildIndex == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"child index is required when parent id is provided",
			)
		}

		existing, err := a.repo.WorkflowRun().GetScheduledChildWorkflowRun(
			ctx,
			*req.ParentId,
			*req.ParentStepRunId,
			int(*req.ChildIndex),
			req.ChildKey,
		)

		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("could not get scheduled child workflow run: %w", err)
			}
		}

		if err == nil && existing != nil {
			return toWorkflowVersion(currWorkflow), nil
		}
	}

	dbSchedules := make([]time.Time, len(req.Schedules))

	for i, scheduledTrigger := range req.Schedules {
		dbSchedules[i] = scheduledTrigger.AsTime()
	}

	workflowVersionId := sqlchelpers.UUIDToStr(currWorkflow.WorkflowVersion.ID)

	// FIXME add additional metadata?

	_, err = a.repo.Workflow().CreateSchedules(
		ctx,
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

func (a *AdminServiceImpl) PutRateLimit(ctx context.Context, req *contracts.PutRateLimitRequest) (*contracts.PutRateLimitResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	if req.Key == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"key is required",
		)
	}

	limit := int(req.Limit)
	duration := req.Duration.String()

	createOpts := &repository.UpsertRateLimitOpts{
		Limit:    limit,
		Duration: &duration,
	}

	_, err := a.repo.RateLimit().UpsertRateLimit(ctx, tenantId, req.Key, createOpts)

	if err != nil {
		return nil, err
	}

	return &contracts.PutRateLimitResponse{}, nil
}

func getCreateWorkflowOpts(req *contracts.PutWorkflowRequest) (*repository.CreateWorkflowVersionOpts, error) {
	jobs := make([]repository.CreateWorkflowJobOpts, len(req.Opts.Jobs))

	for i, job := range req.Opts.Jobs {
		jobCp := job
		res, err := getCreateJobOpts(jobCp, "DEFAULT")

		if err != nil {
			return nil, err
		}

		jobs[i] = *res
	}

	var onFailureJob *repository.CreateWorkflowJobOpts

	if req.Opts.OnFailureJob != nil {
		onFailureJobCp, err := getCreateJobOpts(req.Opts.OnFailureJob, "ON_FAILURE")

		if err != nil {
			return nil, err
		}

		onFailureJob = onFailureJobCp
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

	var cronInput []byte

	if req.Opts.CronInput != nil {
		cronInput = []byte(*req.Opts.CronInput)
	}

	return &repository.CreateWorkflowVersionOpts{
		Name:              req.Opts.Name,
		Concurrency:       concurrency,
		Description:       &req.Opts.Description,
		Version:           &req.Opts.Version,
		EventTriggers:     req.Opts.EventTriggers,
		CronTriggers:      req.Opts.CronTriggers,
		CronInput:         cronInput,
		ScheduledTriggers: scheduledTriggers,
		Jobs:              jobs,
		OnFailureJob:      onFailureJob,
		ScheduleTimeout:   req.Opts.ScheduleTimeout,
	}, nil
}

func getCreateJobOpts(req *contracts.CreateWorkflowJobOpts, kind string) (*repository.CreateWorkflowJobOpts, error) {
	steps := make([]repository.CreateWorkflowStepOpts, len(req.Steps))

	for j, step := range req.Steps {
		stepCp := step

		parsedAction, err := types.ParseActionID(step.Action)

		if err != nil {
			return nil, err
		}

		retries := int(stepCp.Retries)

		steps[j] = repository.CreateWorkflowStepOpts{
			ReadableId: stepCp.ReadableId,
			Action:     parsedAction.String(),
			Parents:    stepCp.Parents,
			Retries:    &retries,
		}

		if stepCp.Timeout != "" {
			steps[j].Timeout = &stepCp.Timeout
		}

		for _, rateLimit := range stepCp.RateLimits {
			steps[j].RateLimits = append(steps[j].RateLimits, repository.CreateWorkflowStepRateLimitOpts{
				Key:   rateLimit.Key,
				Units: int(rateLimit.Units),
			})
		}

		if stepCp.UserData != "" {
			steps[j].UserData = &stepCp.UserData
		}
	}

	return &repository.CreateWorkflowJobOpts{
		Name:        req.Name,
		Description: &req.Description,
		Steps:       steps,
		Kind:        kind,
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
