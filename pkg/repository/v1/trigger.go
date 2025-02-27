package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
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

	// (optional) the external id of the parent, if this is a child workflow run
	ParentTaskId *int64

	// (optional) the index of the child workflow run, if this is a child workflow run
	ChildIndex *int64

	// (optional) the key of the child workflow run
	ChildKey *string
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

	// (required) the name of the workflow
	WorkflowName string

	// (optional) the additional metadata for the DAG
	AdditionalMetadata []byte
}

type TriggerRepository interface {
	TriggerFromEvents(ctx context.Context, tenantId string, opts []EventTriggerOpts) ([]*sqlcv1.V1Task, []*DAGWithData, error)

	TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []WorkflowNameTriggerOpts) ([]*sqlcv1.V1Task, []*DAGWithData, error)
}

type TriggerRepositoryImpl struct {
	*sharedRepository
}

func newTriggerRepository(s *sharedRepository) TriggerRepository {
	return &TriggerRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *TriggerRepositoryImpl) TriggerFromEvents(ctx context.Context, tenantId string, opts []EventTriggerOpts) ([]*sqlcv1.V1Task, []*DAGWithData, error) {
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
	workflowVersionIdsAndEventKeys, err := r.queries.ListWorkflowsForEvents(ctx, r.pool, sqlcv1.ListWorkflowsForEventsParams{
		Eventkeys: eventKeys,
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workflows for events: %w", err)
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
				workflowName:       workflow.WorkflowName,
				externalId:         uuid.NewString(),
				input:              opt.Data,
				additionalMetadata: opt.AdditionalMetadata,
			})
		}
	}

	return r.triggerWorkflows(ctx, tenantId, triggerOpts)
}

func (r *TriggerRepositoryImpl) TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []WorkflowNameTriggerOpts) ([]*sqlcv1.V1Task, []*DAGWithData, error) {
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
	workflowVersionsByNames, err := r.queries.ListWorkflowsByNames(ctx, r.pool, sqlcv1.ListWorkflowsByNamesParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflownames: workflowNames,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workflows for names: %w", err)
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
				workflowName:       workflowVersion.WorkflowName,
				externalId:         opt.ExternalId,
				input:              opt.Data,
				additionalMetadata: opt.AdditionalMetadata,
				parentTaskId:       opt.ParentTaskId,
				childIndex:         opt.ChildIndex,
				childKey:           opt.ChildKey,
			})
		}
	}

	return r.triggerWorkflows(ctx, tenantId, triggerOpts)
}

type triggerTuple struct {
	workflowVersionId string

	workflowId string

	workflowName string

	externalId string

	input []byte

	additionalMetadata []byte

	// relevant parameters for child workflows
	parentTaskId *int64
	childIndex   *int64
	childKey     *string
}

func (r *TriggerRepositoryImpl) triggerWorkflows(ctx context.Context, tenantId string, tuples []triggerTuple) ([]*sqlcv1.V1Task, []*DAGWithData, error) {
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
	steps, err := r.queries.ListStepsByWorkflowVersionIds(ctx, r.pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      workflowVersionIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get workflow versions for engine: %w", err)
	}

	// group steps by workflow version ids
	workflowVersionToSteps := make(map[string][]*sqlcv1.ListStepsByWorkflowVersionIdsRow)
	stepIdsToReadableIds := make(map[string]string)

	for _, step := range steps {
		workflowVersionId := sqlchelpers.UUIDToStr(step.WorkflowVersionId)

		workflowVersionToSteps[workflowVersionId] = append(workflowVersionToSteps[workflowVersionId], step)

		stepIdsToReadableIds[sqlchelpers.UUIDToStr(step.ID)] = step.ReadableId.String
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
	createMatchOpts := make([]CreateMatchOpts, 0)

	// a map of trigger tuples to step external IDs
	stepsToExternalIds := make([]map[string]string, len(tuples))
	dagToTaskIds := make(map[string][]string)

	// generate UUIDs for each step
	for i, tuple := range tuples {
		stepsToExternalIds[i] = make(map[string]string)

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

		for _, step := range steps {
			if !isDag {
				stepsToExternalIds[i][sqlchelpers.UUIDToStr(step.ID)] = tuple.externalId
			} else {
				externalId := uuid.NewString()
				stepsToExternalIds[i][sqlchelpers.UUIDToStr(step.ID)] = externalId
				dagToTaskIds[tuple.externalId] = append(dagToTaskIds[tuple.externalId], externalId)
			}
		}
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	// check if we should skip the creation of any workflows if they're child workflows which
	// already have a signal registered
	tuplesToSkip, err := r.registerChildWorkflows(ctx, tx, tenantId, tuples, stepsToExternalIds, workflowVersionToSteps)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to register child workflows: %w", err)
	}

	for i, tuple := range tuples {
		if _, ok := tuplesToSkip[tuple.externalId]; ok {
			continue
		}

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

		for _, step := range steps {
			stepId := sqlchelpers.UUIDToStr(step.ID)
			taskExternalId := stepsToExternalIds[i][stepId]

			// if this is an on failure step, create match conditions for every other step in the DAG
			switch {
			case step.JobKind == sqlcv1.JobKindONFAILURE:
				conditions := make([]GroupMatchCondition, 0)
				groupId := uuid.NewString()

				for _, otherStep := range steps {
					if sqlchelpers.UUIDToStr(otherStep.ID) == stepId {
						continue
					}

					otherExternalId := stepsToExternalIds[i][sqlchelpers.UUIDToStr(otherStep.ID)]
					readableId := otherStep.ReadableId.String

					conditions = append(conditions, getParentOnFailureGroupMatches(groupId, otherExternalId, readableId)...)
				}

				eventMatches[tuple.externalId] = append(eventMatches[tuple.externalId], CreateMatchOpts{
					Kind:              sqlcv1.V1MatchKindTRIGGER,
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
					InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
				}

				if isDag {
					dagTaskOpts[tuple.externalId] = append(dagTaskOpts[tuple.externalId], opt)
				} else {
					nonDagTaskOpts = append(nonDagTaskOpts, opt)
				}
			default:
				conditions := make([]GroupMatchCondition, 0)

				createGroupId := uuid.NewString()

				for _, parent := range step.Parents {
					parentExternalId := stepsToExternalIds[i][sqlchelpers.UUIDToStr(parent)]
					readableId := stepIdsToReadableIds[sqlchelpers.UUIDToStr(parent)]

					conditions = append(conditions, getParentInDAGGroupMatch(createGroupId, parentExternalId, readableId)...)
				}

				// create an event match
				eventMatches[tuple.externalId] = append(eventMatches[tuple.externalId], CreateMatchOpts{
					Kind:              sqlcv1.V1MatchKindTRIGGER,
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
				TaskIds:            dagToTaskIds[tuple.externalId],
				WorkflowId:         tuple.workflowId,
				WorkflowVersionId:  tuple.workflowVersionId,
				WorkflowName:       tuple.workflowName,
				AdditionalMetadata: tuple.additionalMetadata,
			})
		}
	}

	// create DAGs
	dags, err := r.createDAGs(ctx, tx, tenantId, dagOpts)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create DAGs: %w", err)
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
		return nil, nil, fmt.Errorf("failed to create tasks: %w", err)
	}

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
		return nil, nil, fmt.Errorf("failed to create event matches: %w", err)
	}

	// commit
	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	return tasks, dags, nil
}

type DAGWithData struct {
	*sqlcv1.V1Dag

	Input []byte

	AdditionalMetadata []byte
}

func (r *TriggerRepositoryImpl) createDAGs(ctx context.Context, tx sqlcv1.DBTX, tenantId string, opts []createDAGOpts) ([]*DAGWithData, error) {
	if len(opts) == 0 {
		return nil, nil
	}

	tenantIds := make([]pgtype.UUID, 0, len(opts))
	externalIds := make([]pgtype.UUID, 0, len(opts))
	displayNames := make([]string, 0, len(opts))
	workflowIds := make([]pgtype.UUID, 0, len(opts))
	workflowVersionIds := make([]pgtype.UUID, 0, len(opts))
	dagIdToOpt := make(map[string]createDAGOpts, 0)

	unix := time.Now().UnixMilli()

	for _, opt := range opts {
		tenantIds = append(tenantIds, sqlchelpers.UUIDFromStr(tenantId))
		externalIds = append(externalIds, sqlchelpers.UUIDFromStr(opt.ExternalId))
		displayNames = append(displayNames, fmt.Sprintf("%s-%d", opt.WorkflowName, unix))
		workflowIds = append(workflowIds, sqlchelpers.UUIDFromStr(opt.WorkflowId))
		workflowVersionIds = append(workflowVersionIds, sqlchelpers.UUIDFromStr(opt.WorkflowVersionId))
		dagIdToOpt[opt.ExternalId] = opt
	}

	createdDAGs, err := r.queries.CreateDAGs(ctx, tx, sqlcv1.CreateDAGsParams{
		Tenantids:          tenantIds,
		Externalids:        externalIds,
		Displaynames:       displayNames,
		Workflowids:        workflowIds,
		Workflowversionids: workflowVersionIds,
	})

	if err != nil {
		return nil, err
	}

	dagDataParams := make([]sqlcv1.CreateDAGDataParams, 0, len(createdDAGs))
	res := make([]*DAGWithData, 0, len(createdDAGs))

	for _, dag := range createdDAGs {
		externalId := sqlchelpers.UUIDToStr(dag.ExternalID)
		opt, ok := dagIdToOpt[externalId]

		if !ok {
			r.l.Error().Msgf("could not find DAG opt for DAG with external id: %s", externalId)
			continue
		}

		input := opt.Input

		if len(input) == 0 {
			input = []byte("{}")
		}

		additionalMeta := opt.AdditionalMetadata

		if len(additionalMeta) == 0 {
			additionalMeta = []byte("{}")
		}

		dagDataParams = append(dagDataParams, sqlcv1.CreateDAGDataParams{
			DagID:              dag.ID,
			DagInsertedAt:      dag.InsertedAt,
			Input:              input,
			AdditionalMetadata: additionalMeta,
		})

		res = append(res, &DAGWithData{
			V1Dag:              dag,
			Input:              input,
			AdditionalMetadata: additionalMeta,
		})
	}

	_, err = r.queries.CreateDAGData(ctx, tx, dagDataParams)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *TriggerRepositoryImpl) registerChildWorkflows(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId string,
	tuples []triggerTuple,
	stepsToExternalIds []map[string]string,
	workflowVersionToSteps map[string][]*sqlcv1.ListStepsByWorkflowVersionIdsRow,
) (tuplesToSkip map[string]struct{}, err error) {
	potentialMatchKeys := make([]string, 0, len(tuples))
	potentialMatchTaskIds := make([]int64, 0, len(tuples))
	externalIdsToKeys := make(map[string]string)

	for _, tuple := range tuples {
		if tuple.parentTaskId == nil {
			continue
		}

		var childKey string

		if tuple.childKey != nil {
			childKey = *tuple.childKey
		} else if tuple.childIndex != nil {
			childKey = fmt.Sprintf("%d", *tuple.childIndex)
		} else {
			// TODO: handle error better/check with validation that this won't happen
			r.l.Error().Msg("could not find child key or index for child workflow")
			continue
		}

		k := getChildSignalEventKey(*tuple.parentTaskId, childKey)

		potentialMatchKeys = append(potentialMatchKeys, k)
		potentialMatchTaskIds = append(potentialMatchTaskIds, *tuple.parentTaskId)
		externalIdsToKeys[tuple.externalId] = k
	}

	// if we have no potential matches, return early
	if len(potentialMatchKeys) == 0 {
		return nil, nil
	}

	matchingEvents, err := r.queries.ListMatchingSignalEvents(
		ctx,
		tx,
		sqlcv1.ListMatchingSignalEventsParams{
			Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
			Taskids:    potentialMatchTaskIds,
			Signalkeys: potentialMatchKeys,
			Eventtype:  sqlcv1.V1TaskEventTypeSIGNALCREATED,
		},
	)

	if err != nil {
		return nil, err
	}

	toSkip := make(map[string]struct{})

	for _, event := range matchingEvents {
		toSkip[fmt.Sprintf("%d.%s", event.TaskID, event.EventKey.String)] = struct{}{}
	}

	taskIds := make([]TaskIdRetryCount, 0, len(tuples))
	eventKeys := make([]string, 0, len(tuples))
	datas := make([][]byte, 0, len(tuples))

	createMatchOpts := make([]CreateMatchOpts, 0)
	tuplesToSkip = make(map[string]struct{})

	for i, tuple := range tuples {
		key := externalIdsToKeys[tuple.externalId]

		// if the child workflow event has already been registered, skip it
		if key != "" {
			if _, ok := toSkip[fmt.Sprintf("%d.%s", *tuple.parentTaskId, key)]; ok {
				tuplesToSkip[tuple.externalId] = struct{}{}
				continue
			}
		}

		// if this is a child workflow run, create a match condition for the parent for each
		// step in the DAG
		if tuple.parentTaskId != nil {
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

			conditions := make([]GroupMatchCondition, 0)

			for _, step := range steps {
				stepId := sqlchelpers.UUIDToStr(step.ID)
				stepReadableId := step.ReadableId.String
				stepExternalId := stepsToExternalIds[i][stepId]

				conditions = append(conditions, getChildWorkflowGroupMatches(stepExternalId, stepReadableId)...)
			}

			createMatchOpts = append(createMatchOpts, CreateMatchOpts{
				Kind:         sqlcv1.V1MatchKindSIGNAL,
				Conditions:   conditions,
				SignalTaskId: tuple.parentTaskId,
				SignalKey:    &key,
			})

			taskIds = append(taskIds, TaskIdRetryCount{
				Id:         *tuple.parentTaskId,
				RetryCount: -1,
			})

			eventKeys = append(eventKeys, key)
			datas = append(datas, []byte{})
		}
	}

	// create the relevant matches
	err = r.createEventMatches(ctx, tx, tenantId, createMatchOpts)

	if err != nil {
		return nil, err
	}

	// create the relevant events
	err = r.createTaskEvents(ctx, tx, tenantId, taskIds, datas, sqlcv1.V1TaskEventTypeSIGNALCREATED, eventKeys)

	if err != nil {
		return nil, err
	}

	return tuplesToSkip, nil
}

func getParentInDAGGroupMatch(cancelGroupId, parentExternalId, parentReadableId string) []GroupMatchCondition {
	return []GroupMatchCondition{
		{
			GroupId:           uuid.NewString(),
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "!has(input.skipped) || (has(input.skipped) && !input.skipped)",
			Action:            sqlcv1.V1MatchConditionActionQUEUE,
		},
		{
			GroupId:           uuid.NewString(),
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "has(input.skipped) && input.skipped",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		},
		{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCANCEL,
		},
		{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCANCEL,
		},
	}
}

func getChildWorkflowGroupMatches(taskExternalId, stepReadableId string) []GroupMatchCondition {
	groupId := uuid.NewString()

	return []GroupMatchCondition{
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &taskExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionQUEUE,
		},
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &taskExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionQUEUE,
		},
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &taskExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionQUEUE,
		},
	}
}

func getParentOnFailureGroupMatches(createGroupId, parentExternalId, parentReadableId string) []GroupMatchCondition {
	cancelGroupId := uuid.NewString()

	return []GroupMatchCondition{
		{
			GroupId:           createGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionQUEUE,
		},
		{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		},
		{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		},
	}
}

func getChildSignalEventKey(parentTaskId int64, childKey string) string {
	return fmt.Sprintf("%d.%s", parentTaskId, childKey)
}
