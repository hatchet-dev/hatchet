package v2

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type EventTriggerOpts struct {
	EventId string

	Key string

	Data []byte

	AdditionalMetadata []byte
}

type WorkflowNameTriggerOpts struct {
	WorkflowName string

	ExternalId string

	Data []byte

	AdditionalMetadata []byte
}

type createDAGOpts struct {
	// (required) the external id
	ExternalId string `validate:"required,uuid"`

	// (required) the input bytes to the DAG
	Input []byte

	// (required) a list of task external ids that are part of this DAG
	TaskIds []string

	// (required) the workflow id for this DAG
	WorkflowId string

	// (required) the workflow version id for this DAG
	WorkflowVersionId string

	// (optional) the additional metadata for the DAG
	AdditionalMetadata []byte
}

type TriggerRepository interface {
	TriggerFromEvents(ctx context.Context, tenantId string, opts []EventTriggerOpts) ([]*sqlcv2.V2Task, error)

	TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []WorkflowNameTriggerOpts) ([]*sqlcv2.V2Task, error)
}

type TriggerRepositoryImpl struct {
	*sharedRepository
}

func newTriggerRepository(s *sharedRepository) TriggerRepository {
	return &TriggerRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *TriggerRepositoryImpl) TriggerFromEvents(ctx context.Context, tenantId string, opts []EventTriggerOpts) ([]*sqlcv2.V2Task, error) {
	eventKeys := make([]string, 0, len(opts))
	eventKeysToOpts := make(map[string][]EventTriggerOpts)
	uniqueEventKeys := make(map[string]struct{})

	for _, opt := range opts {
		eventKeysToOpts[opt.Key] = append(eventKeysToOpts[opt.Key], opt)

		if _, ok := uniqueEventKeys[opt.Key]; ok {
			continue
		}

		uniqueEventKeys[opt.Key] = struct{}{}
		eventKeys = append(eventKeys, opt.Key)
	}

	// we don't run this in a transaction because workflow versions won't change during the course of this operation
	workflowVersionIdsAndEventKeys, err := r.queries.ListWorkflowsForEvents(ctx, r.pool, sqlcv2.ListWorkflowsForEventsParams{
		Eventkeys: eventKeys,
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workflows for events: %w", err)
	}

	// each (workflowVersionId, eventKey, opt) is a separate workflow that we need to create
	triggerOpts := make([]triggerTuple, 0)

	for _, workflow := range workflowVersionIdsAndEventKeys {
		opts, ok := eventKeysToOpts[workflow.EventKey]

		if !ok {
			continue
		}

		for _, opt := range opts {
			triggerOpts = append(triggerOpts, triggerTuple{
				workflowVersionId:  sqlchelpers.UUIDToStr(workflow.WorkflowVersionId),
				workflowId:         sqlchelpers.UUIDToStr(workflow.WorkflowId),
				externalId:         opt.EventId,
				input:              opt.Data,
				additionalMetadata: opt.AdditionalMetadata,
			})
		}
	}

	return r.triggerWorkflows(ctx, tenantId, triggerOpts)
}

func (r *TriggerRepositoryImpl) TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []WorkflowNameTriggerOpts) ([]*sqlcv2.V2Task, error) {
	workflowNames := make([]string, 0, len(opts))
	uniqueNames := make(map[string]struct{})
	namesToOpts := make(map[string][]WorkflowNameTriggerOpts)

	for _, opt := range opts {
		namesToOpts[opt.WorkflowName] = append(namesToOpts[opt.WorkflowName], opt)

		if _, ok := uniqueNames[opt.WorkflowName]; ok {
			continue
		}

		uniqueNames[opt.WorkflowName] = struct{}{}
		workflowNames = append(workflowNames, opt.WorkflowName)
	}

	// we don't run this in a transaction because workflow versions won't change during the course of this operation
	workflowVersionsByNames, err := r.queries.ListWorkflowsByNames(ctx, r.pool, sqlcv2.ListWorkflowsByNamesParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflownames: workflowNames,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workflows for names: %w", err)
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
				workflowVersionId:  sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersionId),
				workflowId:         sqlchelpers.UUIDToStr(workflowVersion.WorkflowId),
				externalId:         opt.ExternalId,
				input:              opt.Data,
				additionalMetadata: opt.AdditionalMetadata,
			})
		}
	}

	return r.triggerWorkflows(ctx, tenantId, triggerOpts)
}

type triggerTuple struct {
	workflowVersionId string

	workflowId string

	externalId string

	input []byte

	additionalMetadata []byte
}

func (r *TriggerRepositoryImpl) triggerWorkflows(ctx context.Context, tenantId string, tuples []triggerTuple) ([]*sqlcv2.V2Task, error) {
	// get unique workflow version ids
	uniqueWorkflowVersionIds := make(map[string]struct{})

	for _, tuple := range tuples {
		uniqueWorkflowVersionIds[tuple.workflowVersionId] = struct{}{}
	}

	// get all data for triggering tasks in this workflow
	workflowVersionIds := make([]pgtype.UUID, 0, len(uniqueWorkflowVersionIds))

	for id := range uniqueWorkflowVersionIds {
		workflowVersionIds = append(workflowVersionIds, sqlchelpers.UUIDFromStr(id))
	}

	// get steps for the workflow versions
	steps, err := r.queries.ListStepsByWorkflowVersionIds(ctx, r.pool, sqlcv2.ListStepsByWorkflowVersionIdsParams{
		Ids:      workflowVersionIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get workflow versions for engine: %w", err)
	}

	// group steps by workflow version ids
	workflowVersionToSteps := make(map[string][]*sqlcv2.ListStepsByWorkflowVersionIdsRow)

	for _, step := range steps {
		workflowVersionId := sqlchelpers.UUIDToStr(step.WorkflowVersionId)

		workflowVersionToSteps[workflowVersionId] = append(workflowVersionToSteps[workflowVersionId], step)
	}

	// start constructing options for creating tasks, DAGs, and triggers. logic is as follows:
	//
	// 1. if a step does not have any parent steps, create a task
	// 2. if a workflow version has multiple steps, create a DAG and a task
	//
	// FIXME: this logic will change slightly once we add arbitrary event triggers for tasks. The
	// new logic will be as follows:
	//
	// 1. if a step does not have any parent steps, create a task directly
	// 2. if a workflow version has a single step and has additional step triggers, create a task and a trigger
	// 3. if a workflow version has multiple steps, create a DAG and tasks for any non-parent

	dagOpts := make([]createDAGOpts, 0)

	// map of task external IDs to task options
	dagTaskOpts := make(map[string][]CreateTaskOpts)
	nonDagTaskOpts := make([]CreateTaskOpts, 0)

	// map of task external IDs to matches
	eventMatches := make(map[string][]CreateMatchOpts)

	for _, tuple := range tuples {
		steps, ok := workflowVersionToSteps[tuple.workflowVersionId]

		if !ok {
			// TODO: properly handle this error
			r.l.Error().Msgf("could not find steps for workflow version id: %s", tuple.workflowVersionId)
			continue
		}

		if len(steps) == 0 {
			// TODO: properly handle this error
			r.l.Error().Msgf("no steps found for workflow version id: %s", tuple.workflowVersionId)
			continue
		}

		isDag := false

		if len(steps) > 1 {
			isDag = true
		}

		// generate UUIDs for each step
		stepsToExternalIds := make(map[string]string)
		taskIds := make([]string, 0, len(steps))

		for _, step := range steps {
			if !isDag {
				stepsToExternalIds[sqlchelpers.UUIDToStr(step.ID)] = tuple.externalId
			} else {
				externalId := uuid.NewString()
				stepsToExternalIds[sqlchelpers.UUIDToStr(step.ID)] = externalId
				taskIds = append(taskIds, externalId)
			}
		}

		for _, step := range steps {
			stepId := sqlchelpers.UUIDToStr(step.ID)
			taskExternalId := stepsToExternalIds[stepId]

			// if this is an on failure step, create match conditions for every other step in the DAG
			switch {
			case step.JobKind == sqlcv2.JobKindONFAILURE:
				conditions := make([]GroupMatchCondition, 0)
				groupId := uuid.NewString()

				for _, otherStep := range steps {
					if sqlchelpers.UUIDToStr(otherStep.ID) == stepId {
						continue
					}

					otherExternalId := stepsToExternalIds[sqlchelpers.UUIDToStr(otherStep.ID)]

					conditions = append(conditions, getParentFailedGroupMatch(groupId, otherExternalId))
				}

				eventMatches[tuple.externalId] = append(eventMatches[tuple.externalId], CreateMatchOpts{
					Kind:              sqlcv2.V2MatchKindTRIGGER,
					Conditions:        conditions,
					TriggerExternalId: &taskExternalId,
					TriggerStepId:     &stepId,
				})
			case len(step.Parents) == 0:
				opt := CreateTaskOpts{
					ExternalId:         taskExternalId,
					StepId:             sqlchelpers.UUIDToStr(step.ID),
					Input:              r.newTaskInput(tuple.input, nil),
					AdditionalMetadata: tuple.additionalMetadata,
				}

				if isDag {
					dagTaskOpts[tuple.externalId] = append(dagTaskOpts[tuple.externalId], opt)
				} else {
					nonDagTaskOpts = append(nonDagTaskOpts, opt)
				}
			default:
				conditions := make([]GroupMatchCondition, 0)

				for _, parent := range step.Parents {
					parentExternalId := stepsToExternalIds[sqlchelpers.UUIDToStr(parent)]

					conditions = append(conditions, getParentCompletedGroupMatch(parentExternalId))
				}

				// create an event match
				eventMatches[tuple.externalId] = append(eventMatches[tuple.externalId], CreateMatchOpts{
					Kind:              sqlcv2.V2MatchKindTRIGGER,
					Conditions:        conditions,
					TriggerExternalId: &taskExternalId,
					TriggerStepId:     &stepId,
				})
			}
		}

		if isDag {
			dagOpts = append(dagOpts, createDAGOpts{
				ExternalId:         tuple.externalId,
				Input:              tuple.input,
				TaskIds:            taskIds,
				WorkflowId:         tuple.workflowId,
				WorkflowVersionId:  tuple.workflowVersionId,
				AdditionalMetadata: tuple.additionalMetadata,
			})
		}
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// create DAGs
	dags, err := r.createDAGs(ctx, tx, tenantId, dagOpts)

	if err != nil {
		return nil, err
	}

	// populate taskOpts with inserted DAG data
	createTaskOpts := nonDagTaskOpts

	for _, dag := range dags {
		opts, ok := dagTaskOpts[sqlchelpers.UUIDToStr(dag.ExternalID)]

		if !ok {
			r.l.Error().Msgf("could not find task opts for DAG with external id: %s", sqlchelpers.UUIDToStr(dag.ExternalID))
			continue
		}

		for _, opt := range opts {
			opt.DagId = &dag.ID
			opt.DagInsertedAt = dag.InsertedAt
			createTaskOpts = append(createTaskOpts, opt)
		}
	}

	// create tasks
	tasks, err := r.createTasks(ctx, tx, tenantId, createTaskOpts)

	if err != nil {
		return nil, err
	}

	// populate event matches with inserted DAG data
	createMatchOpts := make([]CreateMatchOpts, 0)

	for _, dag := range dags {
		opts, ok := eventMatches[sqlchelpers.UUIDToStr(dag.ExternalID)]

		if !ok {
			r.l.Error().Msgf("could not find event matches for DAG with external id: %s", sqlchelpers.UUIDToStr(dag.ExternalID))
			continue
		}

		for _, opt := range opts {
			opt.TriggerDAGId = &dag.ID
			opt.TriggerDAGInsertedAt = dag.InsertedAt

			createMatchOpts = append(createMatchOpts, opt)
		}
	}

	err = r.createEventMatches(ctx, tx, tenantId, createMatchOpts)

	if err != nil {
		return nil, err
	}

	// commit
	if err := commit(ctx); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TriggerRepositoryImpl) createDAGs(ctx context.Context, tx sqlcv2.DBTX, tenantId string, opts []createDAGOpts) ([]*sqlcv2.V2Dag, error) {
	// Tenantids          []pgtype.UUID `json:"tenantids"`
	// Externalids        []pgtype.UUID `json:"externalids"`
	// Displaynames       []string      `json:"displaynames"`
	// Workflowids        []pgtype.UUID `json:"workflowids"`
	// Workflowversionids []pgtype.UUID `json:"workflowversionids"`
	tenantIds := make([]pgtype.UUID, 0, len(opts))
	externalIds := make([]pgtype.UUID, 0, len(opts))
	displayNames := make([]string, 0, len(opts))
	workflowIds := make([]pgtype.UUID, 0, len(opts))
	workflowVersionIds := make([]pgtype.UUID, 0, len(opts))
	dagIdToOpt := make(map[string]createDAGOpts, 0)

	for _, opt := range opts {
		tenantIds = append(tenantIds, sqlchelpers.UUIDFromStr(tenantId))
		externalIds = append(externalIds, sqlchelpers.UUIDFromStr(opt.ExternalId))
		displayNames = append(displayNames, opt.ExternalId)
		workflowIds = append(workflowIds, sqlchelpers.UUIDFromStr(opt.WorkflowId))
		workflowVersionIds = append(workflowVersionIds, sqlchelpers.UUIDFromStr(opt.WorkflowVersionId))
		dagIdToOpt[opt.ExternalId] = opt
	}

	createdDAGs, err := r.queries.CreateDAGs(ctx, tx, sqlcv2.CreateDAGsParams{
		Tenantids:          tenantIds,
		Externalids:        externalIds,
		Displaynames:       displayNames,
		Workflowids:        workflowIds,
		Workflowversionids: workflowVersionIds,
	})

	if err != nil {
		return nil, err
	}

	dagDataParams := make([]sqlcv2.CreateDAGDataParams, 0, len(createdDAGs))

	for _, dag := range createdDAGs {
		externalId := sqlchelpers.UUIDToStr(dag.ExternalID)
		opt, ok := dagIdToOpt[externalId]

		if !ok {
			r.l.Error().Msgf("could not find DAG opt for DAG with external id: %s", externalId)
			continue
		}

		dagDataParams = append(dagDataParams, sqlcv2.CreateDAGDataParams{
			DagID:              dag.ID,
			DagInsertedAt:      dag.InsertedAt,
			Input:              opt.Input,
			AdditionalMetadata: opt.AdditionalMetadata,
		})
	}

	_, err = r.queries.CreateDAGData(ctx, tx, dagDataParams)

	if err != nil {
		return nil, err
	}

	return createdDAGs, nil
}

func getParentCompletedGroupMatch(parentExternalId string) GroupMatchCondition {
	return GroupMatchCondition{
		GroupId:    uuid.NewString(),
		EventType:  sqlcv2.V2EventTypeINTERNAL,
		EventKey:   GetTaskCompletedEventKey(parentExternalId),
		Expression: "true",
	}
}

func getParentFailedGroupMatch(groupId, parentExternalId string) GroupMatchCondition {
	return GroupMatchCondition{
		GroupId:    groupId,
		EventType:  sqlcv2.V2EventTypeINTERNAL,
		EventKey:   GetTaskFailedEventKey(parentExternalId),
		Expression: "true",
	}
}

func GetTaskCompletedEventKey(externalId string) string {
	return fmt.Sprintf("task.completed.%s", externalId)
}

func GetTaskFailedEventKey(externalId string) string {
	return fmt.Sprintf("task.failed.%s", externalId)
}
