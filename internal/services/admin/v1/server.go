package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/listutils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/statusutils"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	schedulingv1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

func (a *AdminServiceImpl) CancelTasks(ctx context.Context, req *contracts.CancelTasksRequest) (*contracts.CancelTasksResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)

	externalIds := make([]uuid.UUID, 0)

	for _, idStr := range req.ExternalIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid external id")
		}
		externalIds = append(externalIds, id)
	}

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
				parsedId, err := uuid.Parse(id)
				if err != nil {
					return nil, status.Error(codes.InvalidArgument, "invalid workflow id")
				}

				workflowIds = append(workflowIds, parsedId)
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
				kv_pairs := strings.SplitN(v, ":", 2)
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
			IncludePayloads:    false,
		}

		runs, _, err := a.repo.OLAP().ListWorkflowRuns(ctx, tenant.ID, opts)

		if err != nil {
			return nil, err
		}

		runExternalIds := make([]uuid.UUID, len(runs))

		for i, run := range runs {
			runExternalIds[i] = run.ExternalID
		}

		externalIds = append(externalIds, runExternalIds...)
	}

	tasks, err := a.repo.Tasks().FlattenExternalIds(ctx, tenant.ID, externalIds)

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
		tenant.ID,
		msgqueue.MsgIDCancelTasks,
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

	externalIdsToReturn := make([]string, len(externalIds))
	for i, id := range externalIds {
		externalIdsToReturn[i] = id.String()
	}

	return &contracts.CancelTasksResponse{
		CancelledTasks: externalIdsToReturn,
	}, nil
}

func (a *AdminServiceImpl) ReplayTasks(ctx context.Context, req *contracts.ReplayTasksRequest) (*contracts.ReplayTasksResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)

	externalIds := make([]uuid.UUID, 0)
	for _, idStr := range req.ExternalIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid external id")
		}

		externalIds = append(externalIds, id)
	}

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
				parsedId, err := uuid.Parse(id)
				if err != nil {
					return nil, status.Error(codes.InvalidArgument, "invalid workflow id")
				}
				workflowIds = append(workflowIds, parsedId)
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
				kv_pairs := strings.SplitN(v, ":", 2)
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
			IncludePayloads:    false,
		}

		runs, _, err := a.repo.OLAP().ListWorkflowRuns(ctx, tenant.ID, opts)

		if err != nil {
			return nil, err
		}

		runExternalIds := make([]uuid.UUID, len(runs))

		for i, run := range runs {
			runExternalIds[i] = run.ExternalID
		}

		externalIds = append(externalIds, runExternalIds...)
	}

	tasksToReplay := []tasktypes.TaskIdInsertedAtRetryCountWithExternalId{}

	tasks, err := a.repo.Tasks().FlattenExternalIds(ctx, tenant.ID, externalIds)

	if err != nil {
		return nil, err
	}

	// Deduplicate based on TaskIdInsertedAtRetryCountWithExternalId
	existingReplays := make(map[tasktypes.TaskIdInsertedAtRetryCountWithExternalId]bool)

	for _, task := range tasks {
		record := tasktypes.TaskIdInsertedAtRetryCountWithExternalId{
			TaskIdInsertedAtRetryCount: v1.TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			},
			WorkflowRunExternalId: task.WorkflowRunID,
			TaskExternalId:        task.ExternalID,
		}

		if _, exists := existingReplays[record]; exists {
			continue
		}

		existingReplays[record] = true
		tasksToReplay = append(tasksToReplay, record)
	}

	workflowRunIdToTasksToReplay := make(map[uuid.UUID][]tasktypes.TaskIdInsertedAtRetryCountWithExternalId)
	for _, item := range tasksToReplay {
		workflowRunIdToTasksToReplay[item.WorkflowRunExternalId] = append(
			workflowRunIdToTasksToReplay[item.WorkflowRunExternalId],
			item,
		)
	}

	var batches [][]tasktypes.TaskIdInsertedAtRetryCountWithExternalId
	var currentBatch []tasktypes.TaskIdInsertedAtRetryCountWithExternalId
	batchSize := 100

	for _, tasksForWorkflowRun := range workflowRunIdToTasksToReplay {
		if len(currentBatch) > 0 && len(currentBatch)+len(tasksForWorkflowRun) > batchSize {
			// If the current batch would exceed the batch size if we added the current workflow run's tasks,
			// we "finalize" the batch and start a new one
			batches = append(batches, currentBatch)
			currentBatch = nil
		}

		if len(tasksForWorkflowRun) > batchSize {
			// If the current workflow run's task count exceeds the batch size on its own,
			// we let it be its own batch
			batches = append(batches, tasksForWorkflowRun)
		} else {
			// Otherwise, add it to the current batch
			currentBatch = append(currentBatch, tasksForWorkflowRun...)
		}
	}

	if len(currentBatch) > 0 {
		// Last case to handle - add the last batch if it has any tasks
		batches = append(batches, currentBatch)
	}

	replayedIds := make([]string, 0)

	for _, batch := range batches {
		// send the payload to the tasks controller, and send the list of tasks back to the client
		toReplay := tasktypes.ReplayTasksPayload{
			Tasks: batch,
		}

		msg, err := msgqueue.NewTenantMessage(
			tenant.ID,
			msgqueue.MsgIDReplayTasks,
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

		for _, task := range batch {
			replayedIds = append(replayedIds, task.TaskExternalId.String())
		}

		time.Sleep(200 * time.Millisecond)
	}

	return &contracts.ReplayTasksResponse{
		ReplayedTasks: replayedIds,
	}, nil
}

func (a *AdminServiceImpl) TriggerWorkflowRun(ctx context.Context, req *contracts.TriggerWorkflowRunRequest) (*contracts.TriggerWorkflowRunResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	canCreateTR, trLimit, err := a.repo.TenantLimit().CanCreate(
		ctx,
		sqlcv1.LimitResourceTASKRUN,
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
		ExternalId: opt.ExternalId.String(),
	}, nil
}

func (a *AdminServiceImpl) GetRunDetails(ctx context.Context, req *contracts.GetRunDetailsRequest) (*contracts.GetRunDetailsResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	externalId, err := uuid.Parse(req.ExternalId)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid external id")
	}

	details, err := a.repo.Tasks().GetWorkflowRunResultDetails(ctx, tenantId, externalId)

	if err != nil {
		return nil, fmt.Errorf("could not get workflow run result details: %w", err)
	}

	if details == nil {
		return nil, status.Error(codes.NotFound, "workflow run not found")
	}

	taskRunDetails := make(map[string]*contracts.TaskRunDetail)

	statuses := make([]statusutils.V1RunStatus, 0)

	for readableId, details := range details.ReadableIdToDetails {
		status, err := details.Status.ToProto()

		if err != nil {
			return nil, fmt.Errorf("could not convert status to proto: %w", err)
		}

		statuses = append(statuses, details.Status)

		taskRunDetails[string(readableId)] = &contracts.TaskRunDetail{
			Status:     *status,
			Error:      details.Error,
			Output:     details.OutputPayload,
			ReadableId: string(readableId),
			ExternalId: details.ExternalId.String(),
		}
	}

	done := !listutils.Any(statuses, "QUEUED") && !listutils.Any(statuses, "RUNNING")
	derivedWorkflowRunStatus, err := statusutils.DeriveWorkflowRunStatus(ctx, statuses)

	if err != nil {
		return nil, fmt.Errorf("could not derive workflow run status: %w", err)
	}

	derivedStatusPtr, err := derivedWorkflowRunStatus.ToProto()

	if err != nil {
		return nil, fmt.Errorf("could not convert derived status to proto: %w", err)
	}

	return &contracts.GetRunDetailsResponse{
		Input:              details.InputPayload,
		AdditionalMetadata: details.AdditionalMetadata,
		TaskRuns:           taskRunDetails,
		Status:             *derivedStatusPtr,
		Done:               done,
	}, nil
}

func (i *AdminServiceImpl) newTriggerOpt(
	ctx context.Context,
	tenantId uuid.UUID,
	req *contracts.TriggerWorkflowRunRequest,
) (*v1.WorkflowNameTriggerOpts, error) {
	t := &v1.TriggerTaskData{
		WorkflowName:       req.WorkflowName,
		Data:               req.Input,
		AdditionalMetadata: req.AdditionalMetadata,
	}

	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 3 {
			return nil, status.Errorf(codes.InvalidArgument, "priority must be between 1 and 3, got %d", *req.Priority)
		}
		t.Priority = req.Priority
	}

	return &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: t,
	}, nil
}

func (i *AdminServiceImpl) generateExternalIds(ctx context.Context, tenantId uuid.UUID, opts []*v1.WorkflowNameTriggerOpts) error {
	return i.repo.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
}

func (i *AdminServiceImpl) ingest(ctx context.Context, tenantId uuid.UUID, opts ...*v1.WorkflowNameTriggerOpts) error {
	optsToSend := make([]*v1.WorkflowNameTriggerOpts, 0)

	for _, opt := range opts {
		if opt.ShouldSkip {
			continue
		}

		optsToSend = append(optsToSend, opt)
	}

	if len(optsToSend) == 0 {
		return nil
	}

	if i.localScheduler != nil {
		localWorkerIds := map[uuid.UUID]struct{}{}

		if i.localDispatcher != nil {
			localWorkerIds = i.localDispatcher.GetLocalWorkerIds()
		}

		localAssigned, schedulingErr := i.localScheduler.RunOptimisticScheduling(ctx, tenantId, opts, localWorkerIds)

		// if we have a scheduling error, we'll fall back to normal ingestion
		if schedulingErr != nil {
			if !errors.Is(schedulingErr, schedulingv1.ErrTenantNotFound) && !errors.Is(schedulingErr, schedulingv1.ErrNoOptimisticSlots) {
				i.l.Error().Err(schedulingErr).Msg("could not run optimistic scheduling")
			}
		}

		if i.localDispatcher != nil && len(localAssigned) > 0 {
			eg := errgroup.Group{}

			for workerId, assignedItems := range localAssigned {
				eg.Go(func() error {
					err := i.localDispatcher.HandleLocalAssignments(ctx, tenantId, workerId, assignedItems)

					if err != nil {
						return fmt.Errorf("could not dispatch assigned items: %w", err)
					}

					return nil
				})
			}

			dispatcherErr := eg.Wait()

			if dispatcherErr != nil {
				i.l.Error().Err(dispatcherErr).Msg("could not handle local assignments")
			}

			// we return nil because the failed assignments would have been requeued by the local dispatcher,
			// and we have already written the tasks to the database
			return nil
		}

		// if there's no scheduling error, we return here because the tasks have been scheduled optimistically
		if schedulingErr == nil {
			return nil
		}
	} else if i.tw != nil {
		triggerErr := i.tw.TriggerFromWorkflowNames(ctx, tenantId, optsToSend)

		// if we fail to trigger via gRPC, we fall back to normal ingestion
		if triggerErr != nil && !errors.Is(triggerErr, trigger.ErrNoTriggerSlots) {
			i.l.Error().Err(triggerErr).Msg("could not trigger workflow runs via gRPC")
		} else if triggerErr == nil {
			return nil
		}
	}

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

	return nil
}

func (a *AdminServiceImpl) PutWorkflow(ctx context.Context, req *contracts.CreateWorkflowVersionRequest) (*contracts.CreateWorkflowVersionResponse, error) {
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

	currWorkflow, err := a.repo.Workflows().PutWorkflowVersion(
		ctx,
		tenantId,
		createOpts,
	)

	if err != nil {
		return nil, err
	}

	a.analytics.Enqueue(
		"workflow:create",
		"grpc",
		&tenantId,
		nil,
		map[string]interface{}{
			"workflow_id": currWorkflow.WorkflowVersion.WorkflowId.String(),
		},
	)

	return &contracts.CreateWorkflowVersionResponse{
		Id:         currWorkflow.WorkflowVersion.ID.String(),
		WorkflowId: currWorkflow.WorkflowVersion.WorkflowId.String(),
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

	defaultFilters := make([]types.DefaultFilter, 0)

	for _, f := range req.DefaultFilters {
		if f.Payload != nil && !json.Valid(f.Payload) {
			return nil, fmt.Errorf("default filter payload is not valid JSON")
		}

		payload := make(map[string]interface{})

		if f.Payload != nil {
			if err := json.Unmarshal(f.Payload, &payload); err != nil {
				return nil, fmt.Errorf("default filter payload is not valid JSON: %w", err)
			}
		}

		defaultFilters = append(defaultFilters, types.DefaultFilter{
			Expression: f.Expression,
			Scope:      f.Scope,
			Payload:    payload,
		})
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
		DefaultFilters:  defaultFilters,
		InputJsonSchema: req.InputJsonSchema,
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

					orGroupIdStr := userEventCondition.Base.OrGroupId

					if orGroupIdStr == "" || orGroupIdStr == uuid.Nil.String() {
						orGroupIdStr = uuid.New().String()
					}

					orGroupId, err := uuid.Parse(orGroupIdStr)
					if err != nil {
						return nil, fmt.Errorf("invalid OrGroupId in UserEventCondition for step %s: %w", stepCp.ReadableId, err)
					}

					eventKey := userEventCondition.UserEventKey

					steps[j].TriggerConditions = append(steps[j].TriggerConditions, v1.CreateStepMatchConditionOpt{
						MatchConditionKind: "USER_EVENT",
						ReadableDataKey:    userEventCondition.Base.ReadableDataKey,
						Action:             userEventCondition.Base.Action.String(),
						OrGroupId:          orGroupId,
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

					orGroupIdStr := sleepCondition.Base.OrGroupId

					if orGroupIdStr == "" || orGroupIdStr == uuid.Nil.String() {
						orGroupIdStr = uuid.New().String()
					}

					orGroupId, err := uuid.Parse(orGroupIdStr)
					if err != nil {
						return nil, fmt.Errorf("invalid OrGroupId in SleepCondition for step %s: %w", stepCp.ReadableId, err)
					}

					steps[j].TriggerConditions = append(steps[j].TriggerConditions, v1.CreateStepMatchConditionOpt{
						MatchConditionKind: "SLEEP",
						ReadableDataKey:    sleepCondition.Base.ReadableDataKey,
						Action:             sleepCondition.Base.Action.String(),
						OrGroupId:          orGroupId,
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

					orGroupIdStr := parentOverrideCondition.Base.OrGroupId

					if orGroupIdStr == "" || orGroupIdStr == uuid.Nil.String() {
						orGroupIdStr = uuid.New().String()
					}

					orGroupId, err := uuid.Parse(orGroupIdStr)

					if err != nil {
						return nil, fmt.Errorf("invalid OrGroupId in ParentOverrideCondition for step %s: %w", stepCp.ReadableId, err)
					}

					steps[j].TriggerConditions = append(steps[j].TriggerConditions, v1.CreateStepMatchConditionOpt{
						MatchConditionKind: "PARENT_OVERRIDE",
						ReadableDataKey:    parentReadableId,
						Action:             parentOverrideCondition.Base.Action.String(),
						Expression:         parentOverrideCondition.Base.Expression,
						OrGroupId:          orGroupId,
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
