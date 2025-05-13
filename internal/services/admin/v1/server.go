package v1

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (a *AdminServiceImpl) CancelTasks(ctx context.Context, req *contracts.CancelTasksRequest) (*contracts.CancelTasksResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	externalIds := req.ExternalIds

	if len(externalIds) != 0 && req.Filter != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot provide both external ids and filter")
	}

	if len(externalIds) == 0 && req.Filter != nil {
		var (
			statuses = []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapFAILED,
				sqlcv1.V1ReadableStatusOlapCOMPLETED,
				sqlcv1.V1ReadableStatusOlapCANCELLED,
			}
			since       = req.Filter.Since.AsTime()
			until       *time.Time
			workflowIds       = []uuid.UUID{}
			limit       int64 = 20000
			offset      int64
		)

		if len(req.Filter.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}

			for _, status := range req.Filter.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
		}

		if len(req.Filter.WorkflowIds) > 0 {
			for _, id := range req.Filter.WorkflowIds {
				workflowIds = append(workflowIds, uuid.MustParse(id))
			}
		}

		if req.Filter.Until != nil {
			t := req.Filter.Until.AsTime()
			until = &t
		}

		var additionalMetadataFilters map[string]interface{}

		if len(req.Filter.AdditionalMetadata) > 0 {
			additionalMetadataFilters = make(map[string]interface{})
			for _, v := range req.Filter.AdditionalMetadata {
				kv_pairs := strings.Split(v, ":")
				if len(kv_pairs) == 2 {
					additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
				} else {
					return nil, status.Errorf(codes.InvalidArgument, "invalid additional metadata filter: %s", v)
				}
			}
		}

		opts := v1.ListWorkflowRunOpts{
			CreatedAfter:       since,
			FinishedBefore:     until,
			Statuses:           statuses,
			WorkflowIds:        workflowIds,
			Limit:              limit,
			Offset:             offset,
			AdditionalMetadata: additionalMetadataFilters,
		}

		runs, _, err := a.repo.OLAP().ListWorkflowRuns(ctx, sqlchelpers.UUIDToStr(tenant.ID), opts)

		if err != nil {
			return nil, err
		}

		runExternalIds := make([]string, len(runs))

		for i, run := range runs {
			runExternalIds[i] = sqlchelpers.UUIDToStr(run.ExternalID)
		}

		externalIds = append(externalIds, runExternalIds...)
	}

	tasks, err := a.repo.Tasks().FlattenExternalIds(ctx, sqlchelpers.UUIDToStr(tenant.ID), externalIds)

	if err != nil {
		return nil, err
	}

	tasksToCancel := []v1.TaskIdInsertedAtRetryCount{}

	for _, task := range tasks {
		tasksToCancel = append(tasksToCancel, v1.TaskIdInsertedAtRetryCount{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		})
	}

	// send the payload to the tasks controller, and send the list of tasks back to the client
	toCancel := tasktypes.CancelTasksPayload{
		Tasks: tasksToCancel,
	}

	msg, err := msgqueue.NewTenantMessage(
		sqlchelpers.UUIDToStr(tenant.ID),
		"cancel-tasks",
		false,
		true,
		toCancel,
	)

	if err != nil {
		return nil, err
	}

	err = a.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	return &contracts.CancelTasksResponse{
		CancelledTasks: externalIds,
	}, nil
}

func (a *AdminServiceImpl) ReplayTasks(ctx context.Context, req *contracts.ReplayTasksRequest) (*contracts.ReplayTasksResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	externalIds := req.ExternalIds

	if len(externalIds) != 0 && req.Filter != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot provide both external ids and filter")
	}

	if len(externalIds) == 0 && req.Filter != nil {
		var (
			statuses = []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapFAILED,
				sqlcv1.V1ReadableStatusOlapCOMPLETED,
				sqlcv1.V1ReadableStatusOlapCANCELLED,
			}
			since       = req.Filter.Since.AsTime()
			until       *time.Time
			workflowIds       = []uuid.UUID{}
			limit       int64 = 1000
			offset      int64
		)

		if len(req.Filter.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}

			for _, status := range req.Filter.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
		}

		if len(req.Filter.WorkflowIds) > 0 {
			for _, id := range req.Filter.WorkflowIds {
				workflowIds = append(workflowIds, uuid.MustParse(id))
			}
		}

		if req.Filter.Until != nil {
			t := req.Filter.Until.AsTime()
			until = &t
		}

		var additionalMetadataFilters map[string]interface{}

		if len(req.Filter.AdditionalMetadata) > 0 {
			additionalMetadataFilters = make(map[string]interface{})
			for _, v := range req.Filter.AdditionalMetadata {
				kv_pairs := strings.Split(v, ":")
				if len(kv_pairs) == 2 {
					additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
				} else {
					return nil, status.Errorf(codes.InvalidArgument, "invalid additional metadata filter: %s", v)
				}
			}
		}

		opts := v1.ListWorkflowRunOpts{
			CreatedAfter:       since,
			FinishedBefore:     until,
			Statuses:           statuses,
			WorkflowIds:        workflowIds,
			Limit:              limit,
			Offset:             offset,
			AdditionalMetadata: additionalMetadataFilters,
		}

		runs, _, err := a.repo.OLAP().ListWorkflowRuns(ctx, sqlchelpers.UUIDToStr(tenant.ID), opts)

		if err != nil {
			return nil, err
		}

		runExternalIds := make([]string, len(runs))

		for i, run := range runs {
			runExternalIds[i] = sqlchelpers.UUIDToStr(run.ExternalID)
		}

		externalIds = append(externalIds, runExternalIds...)
	}

	tasksToReplay := []v1.TaskIdInsertedAtRetryCount{}

	tasks, err := a.repo.Tasks().FlattenExternalIds(ctx, sqlchelpers.UUIDToStr(tenant.ID), externalIds)

	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		tasksToReplay = append(tasksToReplay, v1.TaskIdInsertedAtRetryCount{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		})
	}

	// FIXME: group tasks by their workflow run id, and send in batches of 50 workflow run ids...

	// send the payload to the tasks controller, and send the list of tasks back to the client
	toReplay := tasktypes.ReplayTasksPayload{
		Tasks: tasksToReplay,
	}

	msg, err := msgqueue.NewTenantMessage(
		sqlchelpers.UUIDToStr(tenant.ID),
		"replay-tasks",
		false,
		true,
		toReplay,
	)

	if err != nil {
		return nil, err
	}

	err = a.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	return &contracts.ReplayTasksResponse{
		ReplayedTasks: externalIds,
	}, nil
}

func (a *AdminServiceImpl) TriggerWorkflowRun(ctx context.Context, req *contracts.TriggerWorkflowRunRequest) (*contracts.TriggerWorkflowRunResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	canCreateWR, wrLimit, err := a.entitlements.TenantLimit().CanCreate(
		ctx,
		dbsqlc.LimitResourceWORKFLOWRUN,
		tenantId,
		1,
	)

	if err != nil {
		return nil, fmt.Errorf("could not check tenant limit: %w", err)
	}

	if !canCreateWR {
		return nil, status.Error(
			codes.ResourceExhausted,
			fmt.Sprintf("tenant has reached reached %d%% of its workflow runs limit", wrLimit),
		)
	}

	canCreateTR, trLimit, err := a.entitlements.TenantLimit().CanCreate(
		ctx,
		dbsqlc.LimitResourceTASKRUN,
		tenantId,
		// NOTE: this isn't actually the number of tasks per workflow run, but we're just checking to see
		// if we've exceeded the limit
		1,
	)

	if err != nil {
		return nil, fmt.Errorf("could not check tenant limit: %w", err)
	}

	if !canCreateTR {
		return nil, status.Error(
			codes.ResourceExhausted,
			fmt.Sprintf("tenant has reached %d%% of its task runs limit", trLimit),
		)
	}

	opt, err := a.newTriggerOpt(ctx, tenantId, req)

	if err != nil {
		return nil, fmt.Errorf("could not create trigger opt: %w", err)
	}

	err = a.generateExternalIds(ctx, tenantId, []*v1.WorkflowNameTriggerOpts{opt})

	if err != nil {
		return nil, fmt.Errorf("could not generate external ids: %w", err)
	}

	err = a.ingest(
		ctx,
		tenantId,
		opt,
	)

	if err != nil {
		return nil, err
	}

	return &contracts.TriggerWorkflowRunResponse{
		ExternalId: opt.ExternalId,
	}, nil
}

func (i *AdminServiceImpl) newTriggerOpt(
	ctx context.Context,
	tenantId string,
	req *contracts.TriggerWorkflowRunRequest,
) (*v1.WorkflowNameTriggerOpts, error) {
	t := &v1.TriggerTaskData{
		WorkflowName:       req.WorkflowName,
		Data:               req.Input,
		AdditionalMetadata: req.AdditionalMetadata,
	}

	return &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: t,
	}, nil
}

func (i *AdminServiceImpl) generateExternalIds(ctx context.Context, tenantId string, opts []*v1.WorkflowNameTriggerOpts) error {
	return i.repo.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
}

func (i *AdminServiceImpl) ingest(ctx context.Context, tenantId string, opts ...*v1.WorkflowNameTriggerOpts) error {
	optsToSend := make([]*v1.WorkflowNameTriggerOpts, 0)

	for _, opt := range opts {
		if opt.ShouldSkip {
			continue
		}

		optsToSend = append(optsToSend, opt)
	}

	if len(optsToSend) > 0 {
		verifyErr := i.repo.Triggers().PreflightVerifyWorkflowNameOpts(ctx, tenantId, optsToSend)

		if verifyErr != nil {
			namesNotFound := &v1.ErrNamesNotFound{}

			if errors.As(verifyErr, &namesNotFound) {
				return status.Error(
					codes.InvalidArgument,
					verifyErr.Error(),
				)
			}

			return fmt.Errorf("could not verify workflow name opts: %w", verifyErr)
		}

		msg, err := tasktypes.TriggerTaskMessage(
			tenantId,
			optsToSend...,
		)

		if err != nil {
			return fmt.Errorf("could not create event task: %w", err)
		}

		err = i.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

		if err != nil {
			return fmt.Errorf("could not add event to task queue: %w", err)
		}
	}

	return nil
}

func (a *AdminServiceImpl) PutWorkflow(ctx context.Context, req *contracts.CreateWorkflowVersionRequest) (*contracts.CreateWorkflowVersionResponse, error) {
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

	currWorkflow, err := a.repo.Workflows().PutWorkflowVersion(
		ctx,
		tenantId,
		createOpts,
	)

	if err != nil {
		return nil, err
	}

	return &contracts.CreateWorkflowVersionResponse{
		Id:         sqlchelpers.UUIDToStr(currWorkflow.WorkflowVersion.ID),
		WorkflowId: sqlchelpers.UUIDToStr(currWorkflow.WorkflowVersion.WorkflowId),
	}, nil
}

func getCreateWorkflowOpts(req *contracts.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionOpts, error) {
	tasks, err := getCreateTaskOpts(req.Tasks, "DEFAULT")

	if err != nil {
		if errors.Is(err, v1.ErrDagParentNotFound) {
			// Extract the additional error information
			return nil, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
		}

		return nil, err
	}

	var onFailureTask *v1.CreateStepOpts

	if req.OnFailureTask != nil {
		onFailureTasks, err := getCreateTaskOpts([]*contracts.CreateTaskOpts{req.OnFailureTask}, "ON_FAILURE")

		if err != nil {
			return nil, err
		}

		if len(onFailureTasks) != 1 {
			return nil, fmt.Errorf("expected 1 on failure job, got %d", len(onFailureTasks))
		}

		onFailureTask = &onFailureTasks[0]
	}

	var sticky *string

	if req.Sticky != nil {
		s := req.Sticky.String()
		sticky = &s
	}

	var concurrency []v1.CreateConcurrencyOpts

	if req.Concurrency != nil {
		if req.Concurrency.Expression == "" {
			return nil, status.Error(
				codes.InvalidArgument,
				"CEL expression is required for concurrency",
			)
		}

		var limitStrategy *string

		if req.Concurrency.LimitStrategy != nil && req.Concurrency.LimitStrategy.String() != "" {
			s := req.Concurrency.LimitStrategy.String()
			limitStrategy = &s
		}

		concurrency = append(concurrency, v1.CreateConcurrencyOpts{
			LimitStrategy: limitStrategy,
			Expression:    req.Concurrency.Expression,
			MaxRuns:       req.Concurrency.MaxRuns,
		})
	}

	for _, c := range req.ConcurrencyArr {
		if c.Expression == "" {
			return nil, status.Error(
				codes.InvalidArgument,
				"CEL expression is required for concurrency",
			)
		}

		var limitStrategy *string

		if c.LimitStrategy != nil && c.LimitStrategy.String() != "" {
			s := c.LimitStrategy.String()
			limitStrategy = &s
		}

		concurrency = append(concurrency, v1.CreateConcurrencyOpts{
			LimitStrategy: limitStrategy,
			Expression:    c.Expression,
			MaxRuns:       c.MaxRuns,
		})
	}

	var cronInput []byte

	if req.CronInput != nil {
		cronInput = []byte(*req.CronInput)
	}

	return &v1.CreateWorkflowVersionOpts{
		Name:            req.Name,
		Concurrency:     concurrency,
		Description:     &req.Description,
		EventTriggers:   req.EventTriggers,
		CronTriggers:    req.CronTriggers,
		CronInput:       cronInput,
		Tasks:           tasks,
		OnFailure:       onFailureTask,
		Sticky:          sticky,
		DefaultPriority: req.DefaultPriority,
	}, nil
}

func getCreateTaskOpts(tasks []*contracts.CreateTaskOpts, kind string) ([]v1.CreateStepOpts, error) {
	// First, check if tasks is nil
	if tasks == nil {
		return nil, fmt.Errorf("tasks list cannot be nil")
	}

	steps := make([]v1.CreateStepOpts, len(tasks))
	stepReadableIdMap := make(map[string]bool)

	for j, step := range tasks {
		// Check if this specific task is nil
		if step == nil {
			return nil, fmt.Errorf("task at index %d is nil", j)
		}

		stepCp := step

		// Verify required fields exist
		if stepCp.Action == "" {
			return nil, fmt.Errorf("task at index %d is missing required field 'Action'", j)
		}

		if stepCp.ReadableId == "" {
			return nil, fmt.Errorf("task at index %d is missing required field 'ReadableId'", j)
		}

		parsedAction, err := types.ParseActionID(step.Action)
		if err != nil {
			return nil, err
		}

		retries := int(stepCp.Retries)
		stepReadableIdMap[stepCp.ReadableId] = true

		var affinity map[string]v1.DesiredWorkerLabelOpts

		if stepCp.WorkerLabels != nil {
			affinity = map[string]v1.DesiredWorkerLabelOpts{}
			for k, v := range stepCp.WorkerLabels {
				// Check if v is nil
				if v == nil {
					continue
				}

				var c *string
				if v.Comparator != nil {
					cPtr := v.Comparator.String()
					c = &cPtr
				}

				(affinity)[k] = v1.DesiredWorkerLabelOpts{
					Key:        k,
					StrValue:   v.StrValue,
					IntValue:   v.IntValue,
					Required:   v.Required,
					Weight:     v.Weight,
					Comparator: c,
				}
			}
		}

		// Create the step with minimal required fields
		steps[j] = v1.CreateStepOpts{
			ReadableId:          stepCp.ReadableId,
			Action:              parsedAction.String(),
			Parents:             nil, // Will be set safely below
			Retries:             &retries,
			DesiredWorkerLabels: affinity,
			TriggerConditions:   make([]v1.CreateStepMatchConditionOpt, 0),
			RateLimits:          make([]v1.CreateWorkflowStepRateLimitOpts, 0), // Initialize to avoid nil
			ScheduleTimeout:     stepCp.ScheduleTimeout,
		}

		// Safely set Parents
		if stepCp.Parents != nil {
			steps[j].Parents = stepCp.Parents
		} else {
			steps[j].Parents = []string{} // Use empty array instead of nil
		}

		// Safely handle optional fields
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

		// Safely handle rate limits
		if stepCp.RateLimits != nil {
			for _, rateLimit := range stepCp.RateLimits {
				// Skip nil rate limits
				if rateLimit == nil {
					continue
				}

				opt := v1.CreateWorkflowStepRateLimitOpts{
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
		}

		// Safely handle conditions
		if stepCp.Conditions != nil {
			// Check UserEventConditions
			if stepCp.Conditions.UserEventConditions != nil {
				for _, userEventCondition := range stepCp.Conditions.UserEventConditions {
					// Skip nil conditions
					if userEventCondition == nil || userEventCondition.Base == nil {
						continue
					}

					eventKey := userEventCondition.UserEventKey

					steps[j].TriggerConditions = append(steps[j].TriggerConditions, v1.CreateStepMatchConditionOpt{
						MatchConditionKind: "USER_EVENT",
						ReadableDataKey:    userEventCondition.Base.ReadableDataKey,
						Action:             userEventCondition.Base.Action.String(),
						OrGroupId:          userEventCondition.Base.OrGroupId,
						Expression:         userEventCondition.Base.Expression,
						EventKey:           &eventKey,
					})
				}
			}

			// Check SleepConditions
			if stepCp.Conditions.SleepConditions != nil {
				for _, sleepCondition := range stepCp.Conditions.SleepConditions {
					// Skip nil conditions
					if sleepCondition == nil || sleepCondition.Base == nil {
						continue
					}

					duration := sleepCondition.SleepFor

					steps[j].TriggerConditions = append(steps[j].TriggerConditions, v1.CreateStepMatchConditionOpt{
						MatchConditionKind: "SLEEP",
						ReadableDataKey:    sleepCondition.Base.ReadableDataKey,
						Action:             sleepCondition.Base.Action.String(),
						OrGroupId:          sleepCondition.Base.OrGroupId,
						SleepDuration:      &duration,
					})
				}
			}

			// Check ParentOverrideConditions
			if stepCp.Conditions.ParentOverrideConditions != nil {
				for _, parentOverrideCondition := range stepCp.Conditions.ParentOverrideConditions {
					// Skip nil conditions
					if parentOverrideCondition == nil || parentOverrideCondition.Base == nil {
						continue
					}

					parentReadableId := parentOverrideCondition.ParentReadableId

					steps[j].TriggerConditions = append(steps[j].TriggerConditions, v1.CreateStepMatchConditionOpt{
						MatchConditionKind: "PARENT_OVERRIDE",
						ReadableDataKey:    parentReadableId,
						Action:             parentOverrideCondition.Base.Action.String(),
						Expression:         parentOverrideCondition.Base.Expression,
						OrGroupId:          parentOverrideCondition.Base.OrGroupId,
						ParentReadableId:   &parentReadableId,
					})
				}
			}

		}

		if stepCp.Concurrency != nil {
			for _, concurrency := range stepCp.Concurrency {
				// Skip nil concurrency
				if concurrency == nil {
					continue
				}

				if concurrency.Expression == "" {
					return nil, status.Error(
						codes.InvalidArgument,
						fmt.Sprintf("CEL expression is required for concurrency (step %s)", stepCp.ReadableId),
					)
				}

				var limitStrategy *string

				if concurrency.LimitStrategy != nil && concurrency.LimitStrategy.String() != "" {
					s := concurrency.LimitStrategy.String()
					limitStrategy = &s
				}

				steps[j].Concurrency = append(steps[j].Concurrency, v1.CreateConcurrencyOpts{
					Expression:    concurrency.Expression,
					MaxRuns:       concurrency.MaxRuns,
					LimitStrategy: limitStrategy,
				})
			}
		}
	}

	// Check if parents are in the map
	for _, step := range steps {
		for _, parent := range step.Parents {
			if !stepReadableIdMap[parent] {
				return nil, fmt.Errorf("%w: parent step '%s' not found for step '%s'", v1.ErrDagParentNotFound, parent, step.ReadableId)
			}
		}
	}

	return steps, nil
}

func (a *AdminServiceImpl) CreateFilter(ctx context.Context, req *contracts.CreateFilterRequest) (*contracts.CreateFilterResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	var payload []byte
	if req.Payload != nil {
		payload = []byte(*req.Payload)
	}

	params := sqlcv1.CreateFilterParams{
		Tenantid:     tenant.ID,
		Workflowid:   sqlchelpers.UUIDFromStr(req.WorkflowId),
		Resourcehint: req.ResourceHint,
		Expression:   req.Expression,
		Payload:      payload,
	}

	filter, err := a.repo.Filters().CreateFilter(
		ctx,
		params,
	)

	if err != nil {
		return nil, status.Error(codes.Internal, "could not create filter")
	}

	return &contracts.CreateFilterResponse{
		Id: sqlchelpers.UUIDToStr(filter.ID),
	}, nil
}

func (a *AdminServiceImpl) DeleteFilter(ctx context.Context, req *contracts.DeleteFilterRequest) (*contracts.DeleteFilterResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	params := sqlcv1.DeleteFilterParams{
		Tenantid: tenant.ID,
		ID:       sqlchelpers.UUIDFromStr(req.Id),
	}

	filter, err := a.repo.Filters().DeleteFilter(
		ctx,
		params,
	)

	if err != nil {
		return nil, status.Error(codes.Internal, "could not create filter")
	}

	return &contracts.DeleteFilterResponse{
		Id: sqlchelpers.UUIDToStr(filter.ID),
	}, nil
}
