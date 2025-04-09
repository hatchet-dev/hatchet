package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/dagutils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (a *AdminServiceImpl) TriggerWorkflow(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return a.triggerWorkflowV0(ctx, req)
	case dbsqlc.TenantMajorEngineVersionV1:
		return a.triggerWorkflowV1(ctx, req)
	default:
		return nil, status.Errorf(codes.Unimplemented, "TriggerWorkflow is not supported on major engine version %s", tenant.Version)
	}
}

func (a *AdminServiceImpl) triggerWorkflowV0(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	createContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createOpts, existingWorkflows, err := getOpts(ctx, []*contracts.TriggerWorkflowRequest{req}, a)
	if err != nil {
		return nil, err
	}

	if len(existingWorkflows) > 0 {
		return &contracts.TriggerWorkflowResponse{
			WorkflowRunId: existingWorkflows[0],
		}, nil
	}

	if len(createOpts) > 1 {
		return nil, status.Error(
			codes.Internal,
			"multiple workflow options created from single request",
		)
	}
	createOpt := createOpts[0]

	workflowRun, err := a.repo.WorkflowRun().CreateNewWorkflowRun(createContext, tenantId, createOpt)

	dedupeTarget := repository.ErrDedupeValueExists{}

	if errors.As(err, &dedupeTarget) {
		return nil, status.Error(
			codes.AlreadyExists,
			fmt.Sprintf("workflow run with deduplication value %s already exists", dedupeTarget.DedupeValue),
		)
	}

	if err == metered.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: workflow run limit exceeded for tenant")
	}

	if err != nil {
		return nil, fmt.Errorf("Trigger Workflow - could not create workflow run: %w", err)
	}

	workflowRunId := sqlchelpers.UUIDToStr(workflowRun.ID)

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

func (a *AdminServiceImpl) BulkTriggerWorkflow(ctx context.Context, req *contracts.BulkTriggerWorkflowRequest) (*contracts.BulkTriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return a.bulkTriggerWorkflowV0(ctx, req)
	case dbsqlc.TenantMajorEngineVersionV1:
		return a.bulkTriggerWorkflowV1(ctx, req)
	default:
		return nil, status.Errorf(codes.Unimplemented, "TriggerWorkflow is not supported on major engine version %s", tenant.Version)
	}
}

func (a *AdminServiceImpl) bulkTriggerWorkflowV0(ctx context.Context, req *contracts.BulkTriggerWorkflowRequest) (*contracts.BulkTriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	createContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var opts []*repository.CreateWorkflowRunOpts
	var existingWorkflows []string

	if len(req.Workflows) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no workflows provided")
	}

	if len(req.Workflows) > 1000 {
		return nil, status.Error(codes.InvalidArgument, "maximum of 1000 workflows can be triggered at once")
	}

	opts, existingWorkflows, err := getOpts(ctx, req.Workflows, a)
	if err != nil {
		return nil, err
	}

	if len(opts) == 0 {
		if len(existingWorkflows) == 0 {
			return nil, status.Error(codes.InvalidArgument, "no suitable new workflows provided")
		}
		return &contracts.BulkTriggerWorkflowResponse{WorkflowRunIds: existingWorkflows}, nil

	}

	workflowRuns, err := a.repo.WorkflowRun().CreateNewWorkflowRuns(createContext, tenantId, opts)

	if err == metered.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: workflow run limit exceeded for tenant")
	}
	if err != nil {
		return nil, fmt.Errorf("could not create workflow runs: %w", err)
	}

	var workflowRunIds []string
	for _, workflowRun := range workflowRuns {
		workflowRunIds = append(workflowRunIds, sqlchelpers.UUIDToStr(workflowRun.ID))
	}

	for _, workflowRunId := range workflowRunIds {
		err = a.mq.AddMessage(
			context.Background(),
			msgqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunQueuedToTask(tenantId, workflowRunId),
		)

		if err != nil {
			return nil, fmt.Errorf("could not queue workflow run: %w", err)
		}
	}

	// adding in the pre-existing workflows to the response.

	workflowRunIds = append(workflowRunIds, existingWorkflows...)

	if len(workflowRunIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no workflows created")
	}

	return &contracts.BulkTriggerWorkflowResponse{WorkflowRunIds: workflowRunIds}, nil

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
		newCS, err := dagutils.Checksum(createOpts)

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
		// Wire up priority to here
		&repository.CreateWorkflowSchedulesOpts{
			ScheduledTriggers:  dbSchedules,
			Input:              []byte(req.Input),
			AdditionalMetadata: additionalMetadata,
			Priority:           &req.Priority,
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

func getOpts(ctx context.Context, requests []*contracts.TriggerWorkflowRequest, a *AdminServiceImpl) ([]*repository.CreateWorkflowRunOpts, []string, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	results := make([]*repository.CreateWorkflowRunOpts, 0)
	existingWorkflowRuns := make([]string, 0)

	nonParentWorkflows := make([]*contracts.TriggerWorkflowRequest, 0)

	createContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var childWorkflowRunChecks []repository.ChildWorkflowRun

	childWorkflowMap := make(map[string]*dbsqlc.WorkflowRun)

	for _, req := range requests {
		isParentTriggered := req.ParentId != nil

		if isParentTriggered {
			if req.ParentStepRunId == nil {
				return nil, nil, status.Error(
					codes.InvalidArgument,
					fmt.Sprintf("parent step run id is required when parent id is provided. name: %v parent id: %v, parent step run id: %v", req.Name, req.ParentId, req.ParentStepRunId),
				)
			}

			if req.ChildIndex == nil {
				return nil, nil, status.Error(
					codes.InvalidArgument,
					fmt.Sprintf("child index is required when parent id is provided. name: %v , child key: %v child index: %v", req.Name, req.ChildKey, req.ChildIndex),
				)
			}
			childWorkflowRunChecks = append(childWorkflowRunChecks, repository.ChildWorkflowRun{
				ParentId:        *req.ParentId,
				ParentStepRunId: *req.ParentStepRunId,
				ChildIndex:      int(*req.ChildIndex),
				Childkey:        req.ChildKey,
			})

		}
	}

	if len(childWorkflowRunChecks) > 0 {
		workflowRuns, err := a.repo.WorkflowRun().GetChildWorkflowRuns(
			createContext,
			childWorkflowRunChecks,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("could not get child workflow runs: %w", err)
		}

		for _, wfr := range workflowRuns {
			var childKey *string

			if wfr.ChildKey.Valid {
				childKey = &wfr.ChildKey.String
			}

			key := getChildKey(sqlchelpers.UUIDToStr(wfr.ParentStepRunId), int(wfr.ChildIndex.Int32), childKey)

			childWorkflowMap[key] = wfr
		}
	}

	for _, req := range requests {
		isParentTriggered := req.ParentId != nil

		if isParentTriggered {
			key := getChildKey(*req.ParentStepRunId, int(*req.ChildIndex), req.ChildKey)

			workflowRun := childWorkflowMap[key]

			if workflowRun != nil {
				existingWorkflowRuns = append(existingWorkflowRuns, sqlchelpers.UUIDToStr(workflowRun.ID))
			} else {
				// can't find the child workflow run, so we need to trigger it
				nonParentWorkflows = append(nonParentWorkflows, req)
			}
		} else {
			nonParentWorkflows = append(nonParentWorkflows, req)
		}
	}

	workflowMap, err := getWorkflowsForWorkflowNames(createContext, tenantId, nonParentWorkflows, a)

	if err != nil {
		return nil, nil, err
	}

	var workflowIds []string

	for _, w := range workflowMap {
		workflowIds = append(workflowIds, sqlchelpers.UUIDToStr(w.ID))
	}

	workflowVersions, err := a.repo.Workflow().GetLatestWorkflowVersions(createContext, tenantId, workflowIds)

	if err != nil {
		return nil, nil, fmt.Errorf("could not get latest workflow versions: %w", err)
	}

	workflowVersionMap := make(map[string]*dbsqlc.GetWorkflowVersionForEngineRow)

	for _, w := range workflowVersions {
		workflowVersionMap[sqlchelpers.UUIDToStr(w.WorkflowVersion.WorkflowId)] = w
	}

	parentTriggeredWorkflowRuns := make(map[string]*dbsqlc.GetWorkflowRunRow)
	var parentIds []string

	for _, req := range nonParentWorkflows {
		isParentTriggered := req.ParentId != nil

		if isParentTriggered {
			parentIds = append(parentIds, sqlchelpers.UUIDToStr(sqlchelpers.UUIDFromStr(*req.ParentId)))
		}
	}

	parentWorkflowRuns, err := a.repo.WorkflowRun().GetWorkflowRunByIds(createContext, tenantId, parentIds)

	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "could not get parent workflow runs: %s -  %v", strings.Join(parentIds, ","), err)
	}

	for _, wfr := range parentWorkflowRuns {
		parentTriggeredWorkflowRuns[sqlchelpers.UUIDToStr(wfr.WorkflowRun.ID)] = wfr
	}

	for _, req := range nonParentWorkflows {

		workflow := workflowMap[req.Name]

		if workflow == nil {
			return nil, nil, status.Errorf(codes.NotFound, "workflow %s not found", req.Name)
		}

		latestVersion := workflowVersionMap[sqlchelpers.UUIDToStr(workflow.ID)]

		var createOpts *repository.CreateWorkflowRunOpts

		var additionalMetadata map[string]interface{}

		if req.AdditionalMetadata != nil {
			err := json.Unmarshal([]byte(*req.AdditionalMetadata), &additionalMetadata)
			if err != nil {
				return nil, nil, status.Errorf(codes.InvalidArgument, "could not unmarshal additional metadata received metadata %s but %v", *req.AdditionalMetadata, err)
			}
		}
		isParentTriggered := req.ParentId != nil

		if isParentTriggered {

			parent := parentTriggeredWorkflowRuns[sqlchelpers.UUIDToStr(sqlchelpers.UUIDFromStr(*req.ParentId))]

			if parent == nil {
				return nil, nil, status.Errorf(codes.NotFound, "parent workflow run %s not found", *req.ParentId)
			}

			var parentAdditionalMeta map[string]interface{}

			if parent.WorkflowRun.AdditionalMetadata != nil {
				err := json.Unmarshal(parent.WorkflowRun.AdditionalMetadata, &parentAdditionalMeta)
				if err != nil {
					return nil, nil, fmt.Errorf("could not unmarshal parent additional metadata: %w", err)
				}
			}

			createOpts, err = repository.GetCreateWorkflowRunOptsFromParent(
				latestVersion,
				[]byte(req.Input),
				*req.ParentId,
				*req.ParentStepRunId,
				int(*req.ChildIndex),
				req.ChildKey,
				additionalMetadata,
				parentAdditionalMeta,
				req.Priority,
			)

			if err != nil {
				return nil, nil, fmt.Errorf("Trigger Workflow could not create workflow run opts: %w", err)
			}
		} else {
			createOpts, err = repository.GetCreateWorkflowRunOptsFromManual(latestVersion, []byte(req.Input), additionalMetadata)
			if err != nil {
				return nil, nil, fmt.Errorf("Trigger Workflow not after parent triggered check could not create workflow run opts: %w", err)
			}
		}

		if req.DesiredWorkerId != nil {
			if !latestVersion.WorkflowVersion.Sticky.Valid {
				return nil, nil, status.Errorf(codes.Canceled, "workflow version %s does not have sticky enabled", latestVersion.WorkflowName)
			}

			createOpts.DesiredWorkerId = req.DesiredWorkerId
		}

		if latestVersion.WorkflowVersion.DefaultPriority.Valid {
			createOpts.Priority = &latestVersion.WorkflowVersion.DefaultPriority.Int32
		}

		if req.Priority != nil {
			createOpts.Priority = req.Priority
		}

		results = append(results, createOpts)

	}

	return results, existingWorkflowRuns, nil
}

// lets grab all of the workflows by name

func getWorkflowsForWorkflowNames(ctx context.Context, tenantId string, reqs []*contracts.TriggerWorkflowRequest, a *AdminServiceImpl) (map[string]*dbsqlc.Workflow, error) {

	workflowNames := make([]string, len(reqs))

	for i, req := range reqs {
		workflowNames[i] = req.Name
	}

	workflows, err := a.repo.Workflow().GetWorkflowsByNames(
		ctx,
		tenantId,
		workflowNames,
	)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not get workflows by names: %v", err)
	}

	workflowMap := make(map[string]*dbsqlc.Workflow)

	for _, w := range workflows {

		workflowMap[w.Name] = w

	}

	return workflowMap, nil

}

func getChildKey(parentStepRunId string, childIndex int, childKey *string) string {
	if childKey != nil {
		return fmt.Sprintf("%s-%s", parentStepRunId, *childKey)
	}

	return fmt.Sprintf("%s-%d", parentStepRunId, childIndex)
}
