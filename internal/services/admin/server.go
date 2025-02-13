package admin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (a *AdminServiceImpl) TriggerWorkflow(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	additionalMeta := ""

	if req.AdditionalMetadata != nil {
		additionalMeta = *req.AdditionalMetadata
	}

	var parentTaskId *int64

	if req.ParentStepRunId != nil && strings.HasPrefix(*req.ParentStepRunId, "id-") {
		taskIdStr := strings.TrimPrefix(*req.ParentStepRunId, "id-")
		taskId, err := strconv.ParseInt(taskIdStr, 10, 64)

		if err != nil {
			return nil, fmt.Errorf("could not parse task id: %w", err)
		}

		parentTaskId = &taskId
	}

	var childIndex *int64

	if req.ChildIndex != nil {
		i := int64(*req.ChildIndex)

		childIndex = &i
	}

	taskExternalId, err := a.ingestSingleton(
		tenantId,
		req.Name,
		[]byte(req.Input),
		[]byte(additionalMeta),
		parentTaskId,
		childIndex,
		req.ChildKey,
	)

	if err != nil {
		return nil, fmt.Errorf("could not trigger workflow: %w", err)
	}

	return &contracts.TriggerWorkflowResponse{
		WorkflowRunId: taskExternalId,
	}, nil
}

func (a *AdminServiceImpl) BulkTriggerWorkflow(ctx context.Context, req *contracts.BulkTriggerWorkflowRequest) (*contracts.BulkTriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	runIds := make([]string, len(req.Workflows))

	for _, workflow := range req.Workflows {
		additionalMeta := ""

		if workflow.AdditionalMetadata != nil {
			additionalMeta = *workflow.AdditionalMetadata
		}

		var parentTaskId *int64

		if workflow.ParentStepRunId != nil && strings.HasPrefix(*workflow.ParentStepRunId, "id-") {
			taskIdStr := strings.TrimPrefix(*workflow.ParentStepRunId, "id-")
			taskId, err := strconv.ParseInt(taskIdStr, 10, 64)

			if err != nil {
				return nil, fmt.Errorf("could not parse task id: %w", err)
			}

			parentTaskId = &taskId
		}

		var childIndex *int64

		if workflow.ChildIndex != nil {
			i := int64(*workflow.ChildIndex)

			childIndex = &i
		}

		taskExternalId, err := a.ingestSingleton(
			tenantId,
			workflow.Name,
			[]byte(workflow.Input),
			[]byte(additionalMeta),
			parentTaskId,
			childIndex,
			workflow.ChildKey,
		)

		if err != nil {
			return nil, fmt.Errorf("could not trigger workflow: %w", err)
		}

		runIds = append(runIds, taskExternalId)
	}

	return &contracts.BulkTriggerWorkflowResponse{
		WorkflowRunIds: runIds,
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

			if strings.Contains(err.Error(), "23503") {
				return nil, status.Error(
					codes.InvalidArgument,
					"invalid rate limit, are you using a static key without first creating a rate limit with the same key?",
				)
			}

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
				oldWorkflowVersion,
			)

			if err != nil {

				if strings.Contains(err.Error(), "23503") {
					return nil, status.Error(
						codes.InvalidArgument,
						"invalid rate limit, are you using a static key without first creating a rate limit with the same key?",
					)
				}

				return nil, err
			}
		} else {
			workflowVersion = oldWorkflowVersion
		}
	}

	resp := toWorkflowVersion(workflowVersion, nil)

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
			return toWorkflowVersion(currWorkflow, nil), nil
		}
	}

	dbSchedules := make([]time.Time, len(req.Schedules))

	for i, scheduledTrigger := range req.Schedules {
		dbSchedules[i] = scheduledTrigger.AsTime()
	}

	workflowVersionId := sqlchelpers.UUIDToStr(currWorkflow.WorkflowVersion.ID)

	var additionalMetadata []byte

	if req.AdditionalMetadata != nil {
		additionalMetadata = []byte(*req.AdditionalMetadata)
	}

	scheduledRef, err := a.repo.Workflow().CreateSchedules(
		ctx,
		tenantId,
		workflowVersionId,
		&repository.CreateWorkflowSchedulesOpts{
			ScheduledTriggers:  dbSchedules,
			Input:              []byte(req.Input),
			AdditionalMetadata: additionalMetadata,
		},
	)

	if err != nil {
		return nil, err
	}

	resp := toWorkflowVersion(currWorkflow, scheduledRef)

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

			if errors.Is(err, repository.ErrDagParentNotFound) {
				// Extract the additional error information
				return nil, status.Error(
					codes.InvalidArgument,
					err.Error(),
				)
			}

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

	var sticky *string

	if req.Opts.Sticky != nil {
		sticky = repository.StringPtr(req.Opts.Sticky.String())
	}

	scheduledTriggers := make([]time.Time, 0)

	for _, trigger := range req.Opts.ScheduledTriggers {
		scheduledTriggers = append(scheduledTriggers, trigger.AsTime())
	}

	var concurrency *repository.CreateWorkflowConcurrencyOpts

	if req.Opts.Concurrency != nil {
		if req.Opts.Concurrency.Action == nil && req.Opts.Concurrency.Expression == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"concurrency action or expression is required",
			)
		}

		var limitStrategy *string

		if req.Opts.Concurrency.LimitStrategy != nil && req.Opts.Concurrency.LimitStrategy.String() != "" {
			limitStrategy = repository.StringPtr(req.Opts.Concurrency.LimitStrategy.String())
		}

		concurrency = &repository.CreateWorkflowConcurrencyOpts{
			Action:        req.Opts.Concurrency.Action,
			LimitStrategy: limitStrategy,
			Expression:    req.Opts.Concurrency.Expression,
			MaxRuns:       req.Opts.Concurrency.MaxRuns,
		}
	}

	var cronInput []byte

	if req.Opts.CronInput != nil {
		cronInput = []byte(*req.Opts.CronInput)
	}

	var kind *string

	if req.Opts.Kind != nil {
		kind = repository.StringPtr(req.Opts.Kind.String())
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
		Sticky:            sticky,
		Kind:              kind,
		DefaultPriority:   req.Opts.DefaultPriority,
	}, nil
}

func getCreateJobOpts(req *contracts.CreateWorkflowJobOpts, kind string) (*repository.CreateWorkflowJobOpts, error) {
	steps := make([]repository.CreateWorkflowStepOpts, len(req.Steps))

	stepReadableIdMap := make(map[string]bool)

	for j, step := range req.Steps {
		stepCp := step

		parsedAction, err := types.ParseActionID(step.Action)

		if err != nil {
			return nil, err
		}

		retries := int(stepCp.Retries)

		stepReadableIdMap[stepCp.ReadableId] = true

		var affinity map[string]repository.DesiredWorkerLabelOpts

		if stepCp.WorkerLabels != nil {
			affinity = map[string]repository.DesiredWorkerLabelOpts{}
			for k, v := range stepCp.WorkerLabels {

				var c *string

				if v.Comparator != nil {
					cPtr := v.Comparator.String()
					c = &cPtr
				}

				(affinity)[k] = repository.DesiredWorkerLabelOpts{
					Key:        k,
					StrValue:   v.StrValue,
					IntValue:   v.IntValue,
					Required:   v.Required,
					Weight:     v.Weight,
					Comparator: c,
				}
			}
		}

		steps[j] = repository.CreateWorkflowStepOpts{
			ReadableId:          stepCp.ReadableId,
			Action:              parsedAction.String(),
			Parents:             stepCp.Parents,
			Retries:             &retries,
			DesiredWorkerLabels: affinity,
		}

		if stepCp.BackoffFactor != nil {
			f64 := float64(*stepCp.BackoffFactor)
			steps[j].RetryBackoffFactor = &f64

			if stepCp.BackoffMaxSeconds != nil {
				maxInt := int(*stepCp.BackoffMaxSeconds)
				steps[j].RetryBackoffMaxSeconds = &maxInt
			} else {
				maxInt := 24 * 60 * 60
				steps[j].RetryBackoffMaxSeconds = &maxInt
			}
		}

		if stepCp.Timeout != "" {
			steps[j].Timeout = &stepCp.Timeout
		}

		for _, rateLimit := range stepCp.RateLimits {
			opt := repository.CreateWorkflowStepRateLimitOpts{
				Key:       rateLimit.Key,
				KeyExpr:   rateLimit.KeyExpr,
				LimitExpr: rateLimit.LimitValuesExpr,
				UnitsExpr: rateLimit.UnitsExpr,
			}

			if rateLimit.Duration != nil {
				dur := rateLimit.Duration.String()
				opt.Duration = &dur
			}

			if rateLimit.Units != nil {
				units := int(*rateLimit.Units)
				opt.Units = &units
			}

			steps[j].RateLimits = append(steps[j].RateLimits, opt)
		}

		if stepCp.UserData != "" {
			steps[j].UserData = &stepCp.UserData
		}
	}

	// Check if parents are in the map
	for _, step := range req.Steps {
		for _, parent := range step.Parents {
			if !stepReadableIdMap[parent] {
				return nil, fmt.Errorf("%w: parent step '%s' not found for step '%s'", repository.ErrDagParentNotFound, parent, step.ReadableId)
			}
		}
	}

	return &repository.CreateWorkflowJobOpts{
		Name:        req.Name,
		Description: &req.Description,
		Steps:       steps,
		Kind:        kind,
	}, nil
}

func toWorkflowVersion(workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow, scheduledRefs []*dbsqlc.WorkflowTriggerScheduledRef) *contracts.WorkflowVersion {
	scheduledWorkflows := make([]*contracts.ScheduledWorkflow, len(scheduledRefs))

	for i, ref := range scheduledRefs {
		scheduledWorkflows[i] = &contracts.ScheduledWorkflow{
			Id:        sqlchelpers.UUIDToStr(ref.ID),
			TriggerAt: timestamppb.New(ref.TriggerAt.Time),
		}
	}

	version := &contracts.WorkflowVersion{
		Id:                 sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		CreatedAt:          timestamppb.New(workflowVersion.WorkflowVersion.CreatedAt.Time),
		UpdatedAt:          timestamppb.New(workflowVersion.WorkflowVersion.UpdatedAt.Time),
		Order:              workflowVersion.WorkflowVersion.Order,
		WorkflowId:         sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.WorkflowId),
		ScheduledWorkflows: scheduledWorkflows,
	}

	if workflowVersion.WorkflowVersion.Version.String != "" {
		version.Version = workflowVersion.WorkflowVersion.Version.String
	}

	return version
}

func (i *AdminServiceImpl) ingestSingleton(tenantId, name string, data []byte, metadata []byte, parentTaskId *int64, childIndex *int64, childKey *string) (string, error) {
	taskExternalId := uuid.New().String()

	msg, err := tasktypes.TriggerTaskMessage(
		tenantId,
		taskExternalId,
		name,
		data,
		metadata,
		parentTaskId,
		childIndex,
		childKey,
	)

	if err != nil {
		return "", fmt.Errorf("could not create event task: %w", err)
	}

	var runId string

	if parentTaskId != nil {
		var k string

		if childKey != nil {
			k = *childKey
		} else {
			k = fmt.Sprintf("%d", *childIndex)
		}

		runId = fmt.Sprintf("id-%d-%s", *parentTaskId, k)
	}

	err = i.mq.SendMessage(context.Background(), msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return "", fmt.Errorf("could not add event to task queue: %w", err)
	}

	return runId, nil
}
