package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	contracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts/workflows"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (a *AdminServiceImpl) TriggerWorkflow(ctx context.Context, req *v1contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	return a.triggerWorkflowV1(ctx, req)
}

func (a *AdminServiceImpl) BulkTriggerWorkflow(ctx context.Context, req *contracts.BulkTriggerWorkflowRequest) (*contracts.BulkTriggerWorkflowResponse, error) {
	return a.bulkTriggerWorkflowV1(ctx, req)
}

func (a *AdminServiceImpl) PutWorkflow(ctx context.Context, req *contracts.PutWorkflowRequest) (*contracts.WorkflowVersion, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

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

	currWorkflow, err := a.repov1.Workflows().PutWorkflowVersion(
		ctx,
		tenantId,
		createOpts,
	)

	if err != nil {
		return nil, err
	}

	// notify that a new set of queues have been created
	// important: this assumes that actions correspond 1:1 with queues, which they do at the moment
	// but might not in the future
	actions, err := getActionsForTasks(createOpts)

	if tenant.SchedulerPartitionId.Valid && err == nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for _, action := range actions {
				a.l.Debug().Msgf("notifying new queue for tenant %s and action %s", tenantId, action)

				msg, err := tasktypes.NotifyNewQueue(tenantId, action)

				if err != nil {
					a.l.Err(err).Msg("could not create message for notifying new queue")
				} else {
					err = a.mqv1.SendMessage(
						notifyCtx,
						msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler),
						msg,
					)

					if err != nil {
						a.l.Err(err).Msg("could not add message to scheduler partition queue")
					}

					a.l.Debug().Msgf("notified new queue for tenant %s and action %s", tenantId, action)
				}
			}
		}()
	} else if err != nil {
		a.l.Warn().Err(err).Msgf("could not get actions for tasks for workflow version %s, skipping notifying new queues for tenant %s", currWorkflow.WorkflowVersion.ID.String(), tenantId)
	} else if !tenant.SchedulerPartitionId.Valid {
		a.l.Debug().Msgf("tenant %s does not have a valid scheduler partition id, skipping notifying new queues for workflow version %s", tenantId, currWorkflow.WorkflowVersion.ID.String())
	}

	resp := toWorkflowVersion(currWorkflow)

	return resp, nil
}

func (a *AdminServiceImpl) ScheduleWorkflow(ctx context.Context, req *contracts.ScheduleWorkflowRequest) (*contracts.WorkflowVersion, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	workflow, err := a.repov1.Workflows().GetWorkflowByName(
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

	workflowId := workflow.ID

	currWorkflow, err := a.repov1.Workflows().GetLatestWorkflowVersion(
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
		if req.ParentTaskRunExternalId == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"parent task run id is required when parent id is provided",
			)
		}

		if req.ChildIndex == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"child index is required when parent id is provided",
			)
		}

		// FIXME: should check whether the scheduled workflow already exists for this parent/child index combo
	}

	scheduleTimes := make([]time.Time, len(req.Schedules))

	for i, scheduledTrigger := range req.Schedules {
		scheduleTimes[i] = scheduledTrigger.AsTime()
	}

	var additionalMetadata []byte

	if req.AdditionalMetadata != nil {
		additionalMetadata = []byte(*req.AdditionalMetadata)
	}

	if err := v1.ValidateJSONB(additionalMetadata, "additionalMetadata"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	payloadBytes := []byte(req.Input)

	if err := v1.ValidateJSONB(payloadBytes, "payload"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	if req.Priority != nil && (*req.Priority < 1 || *req.Priority > 3) {
		return nil, status.Errorf(codes.InvalidArgument, "priority must be between 1 and 3, got %d", *req.Priority)
	}

	dbSchedules := make([]*sqlcv1.ListScheduledWorkflowsRow, 0)

	for _, scheduleTime := range scheduleTimes {
		scheduledRef, err := a.repov1.WorkflowSchedules().CreateScheduledWorkflow(
			ctx,
			tenantId,
			&v1.CreateScheduledWorkflowRunForWorkflowOpts{
				WorkflowId:         workflowId,
				ScheduledTrigger:   scheduleTime,
				Input:              payloadBytes,
				AdditionalMetadata: additionalMetadata,
				Priority:           req.Priority,
			},
		)

		if err != nil {
			return nil, err
		}

		dbSchedules = append(dbSchedules, scheduledRef)
	}

	resp := toWorkflowVersionLegacy(currWorkflow, dbSchedules)

	return resp, nil
}

func (a *AdminServiceImpl) PutRateLimit(ctx context.Context, req *contracts.PutRateLimitRequest) (*contracts.PutRateLimitResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	if req.Key == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"key is required",
		)
	}

	limit := int(req.Limit)
	duration := req.Duration.String()

	createOpts := &v1.UpsertRateLimitOpts{
		Limit:    limit,
		Duration: &duration,
	}

	_, err := a.repov1.RateLimit().UpsertRateLimit(ctx, tenantId, req.Key, createOpts)

	if err != nil {
		return nil, err
	}

	return &contracts.PutRateLimitResponse{}, nil
}

func getCreateWorkflowOpts(req *contracts.PutWorkflowRequest) (*v1.CreateWorkflowVersionOpts, error) {
	// Flatten Jobs[].Steps[] into a single list of tasks
	allSteps := make([]*contracts.CreateWorkflowStepOpts, 0)

	for _, job := range req.Opts.Jobs {
		if job == nil {
			continue
		}
		allSteps = append(allSteps, job.Steps...)
	}

	tasks, err := getCreateTaskOpts(allSteps, req.Opts.ScheduleTimeout)
	if err != nil {
		if errors.Is(err, v1.ErrDagParentNotFound) {
			return nil, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
		}
		return nil, err
	}

	var onFailureTask *v1.CreateStepOpts

	if req.Opts.OnFailureJob != nil {
		onFailureTasks, err := getCreateTaskOpts(req.Opts.OnFailureJob.Steps, req.Opts.ScheduleTimeout)
		if err != nil {
			return nil, err
		}
		if len(onFailureTasks) > 0 {
			onFailureTask = &onFailureTasks[0]
		}
	}

	var sticky *string

	if req.Opts.Sticky != nil {
		s := req.Opts.Sticky.String()
		sticky = &s
	}

	var concurrency []v1.CreateConcurrencyOpts

	// we just skip setting concurrency if action is not nil, because this is deprecated in v1 and matches the
	// behavior of the v1 PutWorkflow endpoint
	if req.Opts.Concurrency != nil && req.Opts.Concurrency.Action == nil {
		if req.Opts.Concurrency.Expression == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				"CEL expression is required for concurrency",
			)
		}

		var limitStrategy *string

		if req.Opts.Concurrency.LimitStrategy != nil && req.Opts.Concurrency.LimitStrategy.String() != "" {
			s := req.Opts.Concurrency.LimitStrategy.String()
			limitStrategy = &s
		}

		concurrency = append(concurrency, v1.CreateConcurrencyOpts{
			LimitStrategy: limitStrategy,
			Expression:    *req.Opts.Concurrency.Expression,
			MaxRuns:       req.Opts.Concurrency.MaxRuns,
		})
	}

	var cronInput []byte

	if req.Opts.CronInput != nil {
		cronInput = []byte(*req.Opts.CronInput)
	}

	return &v1.CreateWorkflowVersionOpts{
		Name:            req.Opts.Name,
		Concurrency:     concurrency,
		Description:     &req.Opts.Description,
		EventTriggers:   req.Opts.EventTriggers,
		CronTriggers:    req.Opts.CronTriggers,
		CronInput:       cronInput,
		Tasks:           tasks,
		OnFailure:       onFailureTask,
		Sticky:          sticky,
		DefaultPriority: req.Opts.DefaultPriority,
	}, nil
}

func getCreateTaskOpts(steps []*contracts.CreateWorkflowStepOpts, scheduleTimeout *string) ([]v1.CreateStepOpts, error) {
	if steps == nil {
		return nil, fmt.Errorf("steps list cannot be nil")
	}

	tasks := make([]v1.CreateStepOpts, len(steps))
	stepReadableIdMap := make(map[string]bool)

	for j, step := range steps {
		if step == nil {
			return nil, fmt.Errorf("step at index %d is nil", j)
		}

		if step.Action == "" {
			return nil, fmt.Errorf("step at index %d is missing required field 'Action'", j)
		}

		if step.ReadableId == "" {
			return nil, fmt.Errorf("step at index %d is missing required field 'ReadableId'", j)
		}

		parsedAction, err := types.ParseActionID(step.Action)
		if err != nil {
			return nil, err
		}

		retries := int(step.Retries)
		stepReadableIdMap[step.ReadableId] = true

		var affinity map[string]v1.DesiredWorkerLabelOpts

		if step.WorkerLabels != nil {
			affinity = map[string]v1.DesiredWorkerLabelOpts{}
			for k, v := range step.WorkerLabels {
				if v == nil {
					continue
				}

				var c *string
				if v.Comparator != nil {
					cPtr := v.Comparator.String()
					c = &cPtr
				}

				affinity[k] = v1.DesiredWorkerLabelOpts{
					Key:        k,
					StrValue:   v.StrValue,
					IntValue:   v.IntValue,
					Required:   v.Required,
					Weight:     v.Weight,
					Comparator: c,
				}
			}
		}

		tasks[j] = v1.CreateStepOpts{
			ReadableId:          step.ReadableId,
			Action:              parsedAction.String(),
			Parents:             step.Parents,
			Retries:             &retries,
			DesiredWorkerLabels: affinity,
			TriggerConditions:   make([]v1.CreateStepMatchConditionOpt, 0),
			RateLimits:          make([]v1.CreateWorkflowStepRateLimitOpts, 0),
			ScheduleTimeout:     scheduleTimeout,
		}

		if step.Parents == nil {
			tasks[j].Parents = []string{}
		}

		if step.BackoffFactor != nil {
			f64 := float64(*step.BackoffFactor)
			tasks[j].RetryBackoffFactor = &f64

			if step.BackoffMaxSeconds != nil {
				maxInt := int(*step.BackoffMaxSeconds)
				tasks[j].RetryBackoffMaxSeconds = &maxInt
			} else {
				maxInt := 24 * 60 * 60
				tasks[j].RetryBackoffMaxSeconds = &maxInt
			}
		}

		if step.Timeout != "" {
			tasks[j].Timeout = &step.Timeout
		}

		if step.RateLimits != nil {
			for _, rateLimit := range step.RateLimits {
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

				tasks[j].RateLimits = append(tasks[j].RateLimits, opt)
			}
		}
	}

	// Check if parents are in the map
	for _, task := range tasks {
		for _, parent := range task.Parents {
			if !stepReadableIdMap[parent] {
				return nil, fmt.Errorf("%w: parent step '%s' not found for step '%s'", v1.ErrDagParentNotFound, parent, task.ReadableId)
			}
		}
	}

	return tasks, nil
}

func toWorkflowVersion(workflowVersion *sqlcv1.GetWorkflowVersionForEngineRow) *contracts.WorkflowVersion {
	version := &contracts.WorkflowVersion{
		Id:         workflowVersion.WorkflowVersion.ID.String(),
		CreatedAt:  timestamppb.New(workflowVersion.WorkflowVersion.CreatedAt.Time),
		UpdatedAt:  timestamppb.New(workflowVersion.WorkflowVersion.UpdatedAt.Time),
		Order:      workflowVersion.WorkflowVersion.Order,
		WorkflowId: workflowVersion.WorkflowVersion.WorkflowId.String(),
	}

	if workflowVersion.WorkflowVersion.Version.String != "" {
		version.Version = workflowVersion.WorkflowVersion.Version.String
	}

	return version
}

func toWorkflowVersionLegacy(workflowVersion *sqlcv1.GetWorkflowVersionForEngineRow, scheduledRefs []*sqlcv1.ListScheduledWorkflowsRow) *contracts.WorkflowVersion {
	scheduledWorkflows := make([]*contracts.ScheduledWorkflow, len(scheduledRefs))

	for i, ref := range scheduledRefs {
		scheduledWorkflows[i] = &contracts.ScheduledWorkflow{
			Id:        ref.ID.String(),
			TriggerAt: timestamppb.New(ref.TriggerAt.Time),
		}
	}

	version := &contracts.WorkflowVersion{
		Id:                 workflowVersion.WorkflowVersion.ID.String(),
		CreatedAt:          timestamppb.New(workflowVersion.WorkflowVersion.CreatedAt.Time),
		UpdatedAt:          timestamppb.New(workflowVersion.WorkflowVersion.UpdatedAt.Time),
		Order:              workflowVersion.WorkflowVersion.Order,
		WorkflowId:         workflowVersion.WorkflowVersion.WorkflowId.String(),
		ScheduledWorkflows: scheduledWorkflows,
	}

	if workflowVersion.WorkflowVersion.Version.String != "" {
		version.Version = workflowVersion.WorkflowVersion.Version.String
	}

	return version
}

func getActionsForTasks(createOpts *v1.CreateWorkflowVersionOpts) ([]string, error) {
	actions := make([]string, len(createOpts.Tasks))

	for i, task := range createOpts.Tasks {
		parsedAction, err := types.ParseActionID(task.Action)

		if err != nil {
			return nil, err
		}

		actions[i] = parsedAction.String()
	}

	return actions, nil
}
