package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type optimisticSchedulingRepositoryImpl struct {
	*sharedRepository
}

func newOptimisticSchedulingRepository(shared *sharedRepository) *optimisticSchedulingRepositoryImpl {
	return &optimisticSchedulingRepositoryImpl{
		sharedRepository: shared,
	}
}

func (r *optimisticSchedulingRepositoryImpl) StartTx(ctx context.Context) (*OptimisticTx, error) {
	return r.PrepareOptimisticTx(ctx)
}

func (r *optimisticSchedulingRepositoryImpl) TriggerFromEvents(ctx context.Context, tx *OptimisticTx, tenantId string, opts []EventTriggerOpts) ([]*sqlcv1.V1QueueItem, *TriggerFromEventsResult, error) {
	pre, post := r.m.Meter(ctx, dbsqlc.LimitResourceEVENT, tenantId, int32(len(opts))) // nolint: gosec

	if err := pre(); err != nil {
		return nil, nil, err
	}

	eventKeysToOpts := make(map[string][]EventTriggerOpts)
	eventExternalIdToRuns := make(map[string][]*Run)

	eventKeys := make([]string, 0, len(opts))
	uniqueEventKeys := make(map[string]struct{})

	for _, opt := range opts {
		eventExternalIdToRuns[opt.ExternalId] = []*Run{}

		eventKeysToOpts[opt.Key] = append(eventKeysToOpts[opt.Key], opt)

		if _, ok := uniqueEventKeys[opt.Key]; ok {
			continue
		}

		uniqueEventKeys[opt.Key] = struct{}{}
		eventKeys = append(eventKeys, opt.Key)
	}

	workflowVersionIdsAndEventKeys, err := r.queries.ListWorkflowsForEvents(ctx, tx.tx, sqlcv1.ListWorkflowsForEventsParams{
		Eventkeys: eventKeys,
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workflows for events: %w", err)
	}

	externalIdToEventIdAndFilterId := make(map[string]EventExternalIdFilterId)

	workflowIdScopePairs := make(map[WorkflowAndScope]bool)

	for _, workflow := range workflowVersionIdsAndEventKeys {
		opts, ok := eventKeysToOpts[workflow.IncomingEventKey]

		if !ok {
			continue
		}

		for _, opt := range opts {
			if opt.Scope == nil {
				continue
			}

			workflowIdScopePairs[WorkflowAndScope{
				WorkflowId: workflow.WorkflowId,
				Scope:      *opt.Scope,
			}] = true
		}
	}

	workflowIds := make([]pgtype.UUID, 0)
	scopes := make([]string, 0)

	for pair := range workflowIdScopePairs {
		workflowIds = append(workflowIds, pair.WorkflowId)
		scopes = append(scopes, pair.Scope)
	}

	filters, err := r.queries.ListFiltersForEventTriggers(ctx, tx.tx, sqlcv1.ListFiltersForEventTriggersParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Workflowids: workflowIds,
		Scopes:      scopes,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list filters: %w", err)
	}

	workflowIdAndScopeToFilters := make(map[WorkflowAndScope][]*sqlcv1.V1Filter)

	for _, filter := range filters {
		key := WorkflowAndScope{
			WorkflowId: filter.WorkflowID,
			Scope:      filter.Scope,
		}

		workflowIdAndScopeToFilters[key] = append(workflowIdAndScopeToFilters[key], filter)
	}

	filterCounts, err := r.queries.ListFilterCountsForWorkflows(ctx, tx.tx, sqlcv1.ListFilterCountsForWorkflowsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Workflowids: workflowIds,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list filter counts: %w", err)
	}

	workflowIdToCount := make(map[string]int64)

	for _, count := range filterCounts {
		workflowIdToCount[sqlchelpers.UUIDToStr(count.WorkflowID)] = count.Count
	}

	// each (workflowVersionId, eventKey, opt) is a separate workflow that we need to create
	triggerOpts := make([]triggerTuple, 0)
	celEvaluationFailures := make([]CELEvaluationFailure, 0)

	for _, workflow := range workflowVersionIdsAndEventKeys {
		opts, ok := eventKeysToOpts[workflow.IncomingEventKey]

		if !ok {
			continue
		}

		numFilters := workflowIdToCount[sqlchelpers.UUIDToStr(workflow.WorkflowId)]

		hasAnyFilters := numFilters > 0

		for _, opt := range opts {
			var filters = []*sqlcv1.V1Filter{}

			if opt.Scope != nil {
				key := WorkflowAndScope{
					WorkflowId: workflow.WorkflowId,
					Scope:      *opt.Scope,
				}

				filters = workflowIdAndScopeToFilters[key]
			}

			triggerDecisions, evalFailures := r.makeTriggerDecisions(ctx, filters, hasAnyFilters, opt)

			celEvaluationFailures = append(celEvaluationFailures, evalFailures...)

			for _, decision := range triggerDecisions {
				if !decision.ShouldTrigger {
					continue
				}

				triggerConverter := &TriggeredByEvent{
					l:        r.l,
					eventID:  opt.ExternalId,
					eventKey: opt.Key,
				}

				additionalMetadata := triggerConverter.ToMetadata(opt.AdditionalMetadata)
				externalId := uuid.NewString()

				triggerOpts = append(triggerOpts, triggerTuple{
					workflowVersionId:  sqlchelpers.UUIDToStr(workflow.WorkflowVersionId),
					workflowId:         sqlchelpers.UUIDToStr(workflow.WorkflowId),
					workflowName:       workflow.WorkflowName,
					externalId:         externalId,
					input:              opt.Data,
					additionalMetadata: additionalMetadata,
					priority:           opt.Priority,
					filterPayload:      decision.FilterPayload,
				})

				externalIdToEventIdAndFilterId[externalId] = EventExternalIdFilterId{
					ExternalId: opt.ExternalId,
					FilterId:   decision.FilterId,
				}
			}
		}
	}

	tasks, dags, err := r.triggerWorkflows(ctx, tenantId, triggerOpts, tx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	for _, task := range tasks {
		externalId := task.ExternalID

		eventIdAndFilterId, ok := externalIdToEventIdAndFilterId[externalId.String()]

		if !ok {
			continue
		}

		eventExternalIdToRuns[eventIdAndFilterId.ExternalId] = append(eventExternalIdToRuns[eventIdAndFilterId.ExternalId], &Run{
			Id:         task.ID,
			InsertedAt: task.InsertedAt.Time,
			FilterId:   eventIdAndFilterId.FilterId,
		})
	}

	for _, dag := range dags {
		externalId := dag.ExternalID

		eventIdAndFilterId, ok := externalIdToEventIdAndFilterId[externalId.String()]

		if !ok {
			continue
		}

		eventExternalIdToRuns[eventIdAndFilterId.ExternalId] = append(eventExternalIdToRuns[eventIdAndFilterId.ExternalId], &Run{
			Id:         dag.ID,
			InsertedAt: dag.InsertedAt.Time,
			FilterId:   eventIdAndFilterId.FilterId,
		})
	}

	// get the queue items for the tasks that were created
	taskIds := make([]int64, 0, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(tasks))
	retryCounts := make([]int32, 0, len(tasks))

	for _, task := range tasks {
		taskIds = append(taskIds, task.ID)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		retryCounts = append(retryCounts, task.RetryCount)
	}

	qis, err := r.queries.ListQueueItemsForTasks(ctx, tx.tx, sqlcv1.ListQueueItemsForTasksParams{
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			qis = []*sqlcv1.V1QueueItem{}
		} else {
			return nil, nil, fmt.Errorf("failed to list queue items for tasks: %w", err)
		}
	}

	tx.AddPostCommit(post)

	return qis, &TriggerFromEventsResult{
		Tasks:                 tasks,
		Dags:                  dags,
		EventExternalIdToRuns: eventExternalIdToRuns,
		CELEvaluationFailures: celEvaluationFailures,
	}, nil
}

func (r *optimisticSchedulingRepositoryImpl) TriggerFromNames(ctx context.Context, tx *OptimisticTx, tenantId pgtype.UUID, opts []*WorkflowNameTriggerOpts) ([]*sqlcv1.V1QueueItem, []*sqlcv1.V1Task, []*DAGWithData, error) {
	workflowNames := make([]string, 0, len(opts))
	uniqueNames := make(map[string]struct{})
	namesToOpts := make(map[string][]*WorkflowNameTriggerOpts)

	for _, opt := range opts {
		namesToOpts[opt.WorkflowName] = append(namesToOpts[opt.WorkflowName], opt)

		if _, ok := uniqueNames[opt.WorkflowName]; ok {
			continue
		}

		uniqueNames[opt.WorkflowName] = struct{}{}
		workflowNames = append(workflowNames, opt.WorkflowName)
	}

	workflowVersionsByNames, err := r.listWorkflowsByNames(ctx, tx.tx, sqlchelpers.UUIDToStr(tenantId), workflowNames)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to list workflows for names: %w", err)
	}

	// each (workflowVersionId, opt) is a separate workflow that we need to create
	triggerOpts := make([]triggerTuple, 0, len(opts))

	for _, workflowVersion := range workflowVersionsByNames {
		opts, ok := namesToOpts[workflowVersion.WorkflowName]

		if !ok {
			continue
		}

		for _, opt := range opts {
			triggerOpts = append(triggerOpts, triggerTuple{
				workflowVersionId:    sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersionId),
				workflowId:           sqlchelpers.UUIDToStr(workflowVersion.WorkflowId),
				workflowName:         workflowVersion.WorkflowName,
				externalId:           opt.ExternalId,
				input:                opt.Data,
				additionalMetadata:   opt.AdditionalMetadata,
				desiredWorkerId:      opt.DesiredWorkerId,
				parentExternalId:     opt.ParentExternalId,
				parentTaskId:         opt.ParentTaskId,
				parentTaskInsertedAt: opt.ParentTaskInsertedAt,
				childIndex:           opt.ChildIndex,
				childKey:             opt.ChildKey,
				priority:             opt.Priority,
			})
		}
	}

	tasks, dags, err := r.triggerWorkflows(ctx, sqlchelpers.UUIDToStr(tenantId), triggerOpts, tx)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	// get the queue items for the tasks that were created
	taskIds := make([]int64, 0, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(tasks))
	retryCounts := make([]int32, 0, len(tasks))

	for _, task := range tasks {
		taskIds = append(taskIds, task.ID)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		retryCounts = append(retryCounts, task.RetryCount)
	}

	qis, err := r.queries.ListQueueItemsForTasks(ctx, tx.tx, sqlcv1.ListQueueItemsForTasksParams{
		Tenantid:        tenantId,
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			qis = []*sqlcv1.V1QueueItem{}
		} else {
			return nil, nil, nil, fmt.Errorf("failed to list queue items for tasks: %w", err)
		}
	}

	return qis, tasks, dags, nil
}

func (r *optimisticSchedulingRepositoryImpl) MarkQueueItemsProcessed(ctx context.Context, tx *OptimisticTx, tenantId pgtype.UUID, r2 *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error) {
	return r.markQueueItemsProcessed(ctx, tenantId, r2, tx.tx, true)
}
