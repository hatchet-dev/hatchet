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

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (a *AdminServiceImpl) TriggerWorkflow(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	createContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
			createContext,
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

		// if we find a matching workflow run, we return the id and stop triggering the workflow

		if err == nil && workflowRun != nil {
			return &contracts.TriggerWorkflowResponse{
				WorkflowRunId: sqlchelpers.UUIDToStr(workflowRun.ID),
			}, nil
		}
	}

	workflow, err := a.repo.Workflow().GetWorkflowByName(
		createContext,
		tenantId,
		req.Name,
	)

	if err == metered.ErrResourceExhausted {
		return nil, status.Error(
			codes.ResourceExhausted,
			"workflow run limit exceeded",
		)
	}

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
		createContext,
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
		parent, err := a.repo.WorkflowRun().GetWorkflowRunById(createContext, tenantId, sqlchelpers.UUIDToStr(sqlchelpers.UUIDFromStr(*req.ParentId)))

		if err != nil {
			return nil, fmt.Errorf("could not get parent workflow run: %w", err)
		}

		var parentAdditionalMeta map[string]interface{}

		if parent.WorkflowRun.AdditionalMetadata != nil {
			err := json.Unmarshal(parent.WorkflowRun.AdditionalMetadata, &parentAdditionalMeta)
			if err != nil {
				return nil, fmt.Errorf("could not unmarshal parent additional metadata: %w", err)
			}
		}

		createOpts, err = repository.GetCreateWorkflowRunOptsFromParent(
			workflowVersion,
			[]byte(req.Input),
			// we have already checked for nil values above
			*req.ParentId,
			*req.ParentStepRunId,
			int(*req.ChildIndex),
			req.ChildKey,
			additionalMetadata,
			parentAdditionalMeta,
		)

		if err != nil {
			return nil, fmt.Errorf("Trigger Workflow could not create workflow run opts: %w", err)
		}
	} else {
		createOpts, err = repository.GetCreateWorkflowRunOptsFromManual(workflowVersion, []byte(req.Input), additionalMetadata)
		if err != nil {
			return nil, fmt.Errorf("Trigger Workflow not after parent triggered check could not create workflow run opts: %w", err)
		}
	}

	if req.DesiredWorkerId != nil {
		if !workflowVersion.WorkflowVersion.Sticky.Valid {
			return nil, status.Errorf(codes.Canceled, "workflow version %s does not have sticky enabled", workflowVersion.WorkflowName)
		}

		createOpts.DesiredWorkerId = req.DesiredWorkerId
	}

	if workflowVersion.WorkflowVersion.DefaultPriority.Valid {
		createOpts.Priority = &workflowVersion.WorkflowVersion.DefaultPriority.Int32
	}

	if req.Priority != nil {
		createOpts.Priority = req.Priority
	}

	// this is what is really happening we are creating WorkflowRuns

	workflowRunId, err := a.repo.WorkflowRun().CreateNewWorkflowRun(createContext, tenantId, createOpts)

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
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	createContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var opts []*repository.CreateWorkflowRunOpts
	var existingWorkflows []string

	opts, existingWorkflows, err := getOpts(ctx, req.Workflows, a)
	if err != nil {

		return nil, err
	}

	if len(opts) == 0 {
		if len(existingWorkflows) == 0 {
			return &contracts.BulkTriggerWorkflowResponse{}, status.Error(codes.InvalidArgument, "no suitable new workflows provided")
		}
		return &contracts.BulkTriggerWorkflowResponse{WorkflowRunIds: existingWorkflows}, nil

	}

	workflowRunIds, err := a.repo.WorkflowRun().CreateNewWorkflowRuns(createContext, tenantId, opts)

	if err == metered.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: workflow run limit exceeded for tenant")
	}
	if err != nil {
		return nil, fmt.Errorf("could not create workflow runs: %w", err)
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
		return &contracts.BulkTriggerWorkflowResponse{}, status.Error(codes.InvalidArgument, "no workflows created")
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

func getOpts(ctx context.Context, requests []*contracts.TriggerWorkflowRequest, a *AdminServiceImpl) ([]*repository.CreateWorkflowRunOpts, []string, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	results := make([]*repository.CreateWorkflowRunOpts, 0)
	existingWorkflowRuns := make([]string, 0)

	nonParentWorkflows := make([]*contracts.TriggerWorkflowRequest, 0)

	createContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, req := range requests {
		isParentTriggered := req.ParentId != nil

		if isParentTriggered {
			if req.ParentStepRunId == nil {
				return nil, nil, status.Error(
					codes.InvalidArgument,
					"parent step run id is required when parent id is provided",
				)
			}

			if req.ChildIndex == nil {
				return nil, nil, status.Error(
					codes.InvalidArgument,
					"child index is required when parent id is provided",
				)
			}

			workflowRun, err := a.repo.WorkflowRun().GetChildWorkflowRun(
				createContext,
				*req.ParentId,
				*req.ParentStepRunId,
				int(*req.ChildIndex),
				req.ChildKey,
			)

			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					return nil, nil, fmt.Errorf("could not get child workflow run: %w", err)
				}
			}

			if err == nil && workflowRun != nil {

				existingWorkflowRuns = append(existingWorkflowRuns, sqlchelpers.UUIDToStr(workflowRun.ID))
				continue
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

	workflowVersionMap := make(map[string]*dbsqlc.GetLatestWorkflowVersionForWorkflowsRow)

	for _, w := range workflowVersions {
		workflowVersionMap[sqlchelpers.UUIDToStr(w.WorkflowId)] = w
	}

	for _, req := range nonParentWorkflows {

		workflow := workflowMap[req.Name]

		if workflow == nil {
			return nil, nil, status.Errorf(codes.NotFound, "workflow %s not found", req.Name)
		}

		workflowVersion := workflowVersionMap[sqlchelpers.UUIDToStr(workflow.ID)]

		latestVersion := dbsqlc.GetWorkflowVersionForEngineRow{
			WorkflowVersion: dbsqlc.WorkflowVersion{
				ID:              workflowVersion.ID,
				CreatedAt:       workflowVersion.CreatedAt,
				UpdatedAt:       workflowVersion.UpdatedAt,
				DeletedAt:       workflowVersion.DeletedAt,
				Version:         workflowVersion.Version,
				Order:           workflowVersion.Order,
				WorkflowId:      workflowVersion.WorkflowId,
				Checksum:        workflowVersion.Checksum,
				ScheduleTimeout: workflowVersion.ScheduleTimeout,
				OnFailureJobId:  workflowVersion.OnFailureJobId,
				Sticky:          workflowVersion.Sticky,
				Kind:            workflowVersion.Kind,
				DefaultPriority: workflowVersion.DefaultPriority,
			},
			WorkflowName:             workflow.Name,
			ConcurrencyLimitStrategy: workflowVersion.ConcurrencyLimitStrategy,
			ConcurrencyMaxRuns:       workflowVersion.ConcurrencyMaxRuns,
		}

		var createOpts *repository.CreateWorkflowRunOpts

		var additionalMetadata map[string]interface{}

		if req.AdditionalMetadata != nil {
			err := json.Unmarshal([]byte(*req.AdditionalMetadata), &additionalMetadata)
			if err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal additional metadata: %w", err)
			}
		}
		isParentTriggered := req.ParentId != nil

		if isParentTriggered {
			parent, err := a.repo.WorkflowRun().GetWorkflowRunById(createContext, tenantId, sqlchelpers.UUIDToStr(sqlchelpers.UUIDFromStr(*req.ParentId)))

			if err != nil {
				return nil, nil, fmt.Errorf("could not get parent workflow run: %w", err)
			}

			var parentAdditionalMeta map[string]interface{}

			if parent.WorkflowRun.AdditionalMetadata != nil {
				err := json.Unmarshal(parent.WorkflowRun.AdditionalMetadata, &parentAdditionalMeta)
				if err != nil {
					return nil, nil, fmt.Errorf("could not unmarshal parent additional metadata: %w", err)
				}
			}

			createOpts, err = repository.GetCreateWorkflowRunOptsFromParent(
				&latestVersion,
				[]byte(req.Input),

				*req.ParentId,
				*req.ParentStepRunId,
				int(*req.ChildIndex),
				req.ChildKey,
				additionalMetadata,
				parentAdditionalMeta,
			)

			if err != nil {
				return nil, nil, fmt.Errorf("Trigger Workflow could not create workflow run opts: %w", err)
			}
		} else {
			createOpts, err = repository.GetCreateWorkflowRunOptsFromManual(&latestVersion, []byte(req.Input), additionalMetadata)
			if err != nil {
				return nil, nil, fmt.Errorf("Trigger Workflow not after parent triggered check could not create workflow run opts: %w", err)
			}
		}

		if req.DesiredWorkerId != nil {
			if !latestVersion.WorkflowVersion.Sticky.Valid {
				return nil, nil, status.Errorf(codes.Canceled, "workflow version %s does not have sticky enabled", workflowVersion.WorkflowName)
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

	if err == metered.ErrResourceExhausted {
		return nil, status.Error(
			codes.ResourceExhausted,
			"workflow run limit exceeded",
		)
	}

	if err != nil {
		return nil, fmt.Errorf("could not get workflows by names: %w", err)
	}

	workflowMap := make(map[string]*dbsqlc.Workflow)

	for _, w := range workflows {

		workflowMap[w.Name] = w

	}

	return workflowMap, nil

}
