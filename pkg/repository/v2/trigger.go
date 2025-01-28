package v2

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type WorkflowVersionWithTriggeringEvent struct {
	EventId  string
	EventKey string

	WorkflowStartData *sqlcv2.GetWorkflowStartDataRow
}

type EventIdKey struct {
	EventId string
	Key     string
}

type WorkflowVersionWithTriggeringTask struct {
	TaskId string

	WorkflowStartData *sqlcv2.GetWorkflowStartDataRow
}

type TaskIdName struct {
	TaskId string
	Name   string
}

type TriggerRepository interface {
	ListTriggeredWorkflowsForEvents(ctx context.Context, tenantId string, tuples []EventIdKey) ([]*WorkflowVersionWithTriggeringEvent, error)

	ListTriggeredWorkflowsByNames(ctx context.Context, tenantId string, names []TaskIdName) ([]*WorkflowVersionWithTriggeringTask, error)
}

type TriggerRepositoryImpl struct {
	*sharedRepository
}

func newTriggerRepository(s *sharedRepository) TriggerRepository {
	return &TriggerRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *TriggerRepositoryImpl) ListTriggeredWorkflowsForEvents(ctx context.Context, tenantId string, tuples []EventIdKey) ([]*WorkflowVersionWithTriggeringEvent, error) {
	eventIds := make([]pgtype.UUID, 0, len(tuples))
	eventKeys := make([]string, 0, len(tuples))

	for _, tuple := range tuples {
		eventIds = append(eventIds, sqlchelpers.UUIDFromStr(tuple.EventId))
		eventKeys = append(eventKeys, tuple.Key)
	}

	// we don't run this in a transaction because workflow versions won't change during the course of this operation
	workflowVersionIdsAndEvents, err := r.queries.ListWorkflowsForEvents(ctx, r.pool, sqlcv2.ListWorkflowsForEventsParams{
		Eventids:  eventIds,
		Eventkeys: eventKeys,
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workflows for events: %w", err)
	}

	uniqueWorkflowVersionIds := make(map[string]struct{})

	for _, workflow := range workflowVersionIdsAndEvents {
		versionId := sqlchelpers.UUIDToStr(workflow.WorkflowVersionId)

		uniqueWorkflowVersionIds[versionId] = struct{}{}
	}

	workflowVersionIds := make([]pgtype.UUID, 0, len(uniqueWorkflowVersionIds))

	for id := range uniqueWorkflowVersionIds {
		workflowVersionIds = append(workflowVersionIds, sqlchelpers.UUIDFromStr(id))
	}

	// populate workflow versions
	workflowVersions, err := r.queries.GetWorkflowStartData(ctx, r.pool, sqlcv2.GetWorkflowStartDataParams{
		Ids:      workflowVersionIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get workflow versions for engine: %w", err)
	}

	// join the workflow versions with the triggering event ids
	result := make([]*WorkflowVersionWithTriggeringEvent, 0, len(workflowVersionIdsAndEvents))

	workflowVersionsMap := make(map[string]*sqlcv2.GetWorkflowStartDataRow)

	for _, version := range workflowVersions {
		workflowVersionsMap[sqlchelpers.UUIDToStr(version.WorkflowVersionId)] = version
	}

	for _, workflow := range workflowVersionIdsAndEvents {
		startData, ok := workflowVersionsMap[sqlchelpers.UUIDToStr(workflow.WorkflowVersionId)]

		if !ok {
			return nil, fmt.Errorf("could not find workflow version for workflow version id: %s", sqlchelpers.UUIDToStr(workflow.WorkflowVersionId))
		}

		result = append(result, &WorkflowVersionWithTriggeringEvent{
			EventKey:          workflow.EventKey,
			EventId:           sqlchelpers.UUIDToStr(workflow.EventId),
			WorkflowStartData: startData,
		})
	}

	return result, nil
}

func (r *TriggerRepositoryImpl) ListTriggeredWorkflowsByNames(ctx context.Context, tenantId string, tuples []TaskIdName) ([]*WorkflowVersionWithTriggeringTask, error) {
	uniqueNames := make(map[string]struct{})
	namesToTaskIds := make(map[string][]string)

	for _, tuple := range tuples {
		uniqueNames[tuple.Name] = struct{}{}

		if _, ok := namesToTaskIds[tuple.Name]; !ok {
			namesToTaskIds[tuple.Name] = make([]string, 0)
		}

		namesToTaskIds[tuple.Name] = append(namesToTaskIds[tuple.Name], tuple.TaskId)
	}

	paramNames := make([]string, 0, len(uniqueNames))

	for name := range uniqueNames {
		paramNames = append(paramNames, name)
	}

	// we don't run this in a transaction because workflow versions won't change during the course of this operation
	workflowVersionsByNames, err := r.queries.ListWorkflowsByNames(ctx, r.pool, sqlcv2.ListWorkflowsByNamesParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflownames: paramNames,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workflows for events: %w", err)
	}

	uniqueWorkflowVersionIds := make(map[string]struct{})

	for _, workflowVersion := range workflowVersionsByNames {
		versionId := sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersionId)

		uniqueWorkflowVersionIds[versionId] = struct{}{}
	}

	workflowVersionIds := make([]pgtype.UUID, 0, len(uniqueWorkflowVersionIds))

	for id := range uniqueWorkflowVersionIds {
		workflowVersionIds = append(workflowVersionIds, sqlchelpers.UUIDFromStr(id))
	}

	// populate workflow versions
	workflowVersions, err := r.queries.GetWorkflowStartData(ctx, r.pool, sqlcv2.GetWorkflowStartDataParams{
		Ids:      workflowVersionIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get workflow versions for engine: %w", err)
	}

	// join the tuples
	workflowNamesMap := make(map[string]*sqlcv2.GetWorkflowStartDataRow)

	for _, version := range workflowVersions {
		workflowNamesMap[version.WorkflowName] = version
	}

	result := make([]*WorkflowVersionWithTriggeringTask, 0, len(tuples))

	for workflowName, taskIds := range namesToTaskIds {
		for _, taskId := range taskIds {
			startData, ok := workflowNamesMap[workflowName]

			if !ok {
				return nil, fmt.Errorf("could not find workflow version for workflow name: %s", workflowName)
			}

			result = append(result, &WorkflowVersionWithTriggeringTask{
				TaskId:            taskId,
				WorkflowStartData: startData,
			})
		}

	}

	return result, nil
}
