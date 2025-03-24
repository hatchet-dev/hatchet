package v1

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type EventTriggerOpts struct {
	EventId string

	Key string

	Data []byte

	AdditionalMetadata []byte
}

type TriggerTaskData struct {
	// (required) the workflow name
	WorkflowName string `json:"workflow_name" validate:"required"`

	// (optional) the workflow run data
	Data []byte `json:"data"`

	// (optional) the workflow run metadata
	AdditionalMetadata []byte `json:"additional_metadata"`

	// (optional) the desired worker id
	DesiredWorkerId *string `json:"desired_worker_id"`

	// (optional) the parent external id
	ParentExternalId *string `json:"parent_external_id"`

	// (optional) the parent task id
	ParentTaskId *int64 `json:"parent_task_id"`

	// (optional) the parent inserted_at
	ParentTaskInsertedAt *time.Time `json:"parent_task_inserted_at"`

	// (optional) the child index
	ChildIndex *int64 `json:"child_index"`

	// (optional) the child key
	ChildKey *string `json:"child_key"`
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

	ParentTaskExternalID *string
}

type TriggerRepository interface {
	TriggerFromEvents(ctx context.Context, tenantId string, opts []EventTriggerOpts) ([]*sqlcv1.V1Task, []*DAGWithData, error)

	TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) ([]*sqlcv1.V1Task, []*DAGWithData, error)

	PopulateExternalIdsForWorkflow(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) error

	PreflightVerifyWorkflowNameOpts(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) error
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
	pre, post := r.m.Meter(ctx, dbsqlc.LimitResourceEVENT, tenantId, int32(len(opts))) // nolint: gosec

	if err := pre(); err != nil {
		return nil, nil, err
	}

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

	tasks, dags, err := r.triggerWorkflows(ctx, tenantId, triggerOpts)

	if err != nil {
		return nil, nil, err
	}

	post()

	return tasks, dags, nil
}

func (r *TriggerRepositoryImpl) TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) ([]*sqlcv1.V1Task, []*DAGWithData, error) {
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
			})
		}
	}

	return r.triggerWorkflows(ctx, tenantId, triggerOpts)
}

type ErrNamesNotFound struct {
	Names []string
}

func (e *ErrNamesNotFound) Error() string {
	return fmt.Sprintf("workflow names not found: %s", strings.Join(e.Names, ", "))
}

func (r *TriggerRepositoryImpl) PreflightVerifyWorkflowNameOpts(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) error {
	// get a list of workflow names
	workflowNames := make(map[string]bool)

	for _, opt := range opts {
		workflowNames[opt.WorkflowName] = true
	}

	uniqueWorkflowNames := make([]string, 0, len(workflowNames))

	for name := range workflowNames {
		uniqueWorkflowNames = append(uniqueWorkflowNames, name)
	}

	// lookup names in the cache
	workflowNamesToLookup := make([]string, 0)

	for _, name := range uniqueWorkflowNames {
		k := fmt.Sprintf("%s:%s", tenantId, name)
		if _, ok := r.tenantIdWorkflowNameCache.Get(k); ok {
			delete(workflowNames, name)
			continue
		}

		workflowNamesToLookup = append(workflowNamesToLookup, name)
	}

	// look up the workflow versions for the workflow names
	workflowVersions, err := r.queries.ListWorkflowsByNames(ctx, r.pool, sqlcv1.ListWorkflowsByNamesParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflownames: workflowNamesToLookup,
	})

	if err != nil {
		return fmt.Errorf("failed to list workflows by names: %w", err)
	}

	for _, workflowVersion := range workflowVersions {
		// store in the cache
		k := fmt.Sprintf("%s:%s", tenantId, workflowVersion.WorkflowName)

		r.tenantIdWorkflowNameCache.Set(k, true)

		delete(workflowNames, workflowVersion.WorkflowName)
	}

	workflowNamesNotFound := make([]string, 0)

	for name := range workflowNames {
		workflowNamesNotFound = append(workflowNamesNotFound, name)
	}

	if len(workflowNamesNotFound) > 0 {
		return &ErrNamesNotFound{
			Names: workflowNamesNotFound,
		}
	}

	return nil
}

type triggerTuple struct {
	workflowVersionId string

	workflowId string

	workflowName string

	externalId string

	input []byte

	additionalMetadata []byte

	desiredWorkerId *string

	// relevant parameters for child workflows
	parentExternalId     *string
	parentTaskId         *int64
	parentTaskInsertedAt *time.Time
	childIndex           *int64
	childKey             *string
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

	countWorkflowRuns := 0
	countTasks := 0

	for _, tuple := range tuples {
		countWorkflowRuns++

		steps, ok := workflowVersionToSteps[tuple.workflowVersionId]

		if !ok {
			continue
		}

		countTasks += len(steps)
	}

	preWR, postWR := r.m.Meter(ctx, dbsqlc.LimitResourceWORKFLOWRUN, tenantId, int32(countWorkflowRuns)) // nolint: gosec

	if err := preWR(); err != nil {
		return nil, nil, err
	}

	preTask, postTask := r.m.Meter(ctx, dbsqlc.LimitResourceTASKRUN, tenantId, int32(countTasks)) // nolint: gosec

	if err := preTask(); err != nil {
		return nil, nil, err
	}

	// if any steps have additional match conditions, query for the additional matches
	stepsWithAdditionalMatchConditions := make([]pgtype.UUID, 0)

	for _, step := range steps {
		if step.MatchConditionCount > 0 {
			stepsWithAdditionalMatchConditions = append(stepsWithAdditionalMatchConditions, step.ID)
		}
	}

	stepsToAdditionalMatches := make(map[string][]*sqlcv1.V1StepMatchCondition)

	if len(stepsWithAdditionalMatchConditions) > 0 {
		additionalMatches, err := r.queries.ListStepMatchConditions(ctx, r.pool, sqlcv1.ListStepMatchConditionsParams{
			Stepids:  stepsWithAdditionalMatchConditions,
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to list step match conditions: %w", err)
		}

		for _, match := range additionalMatches {
			stepId := sqlchelpers.UUIDToStr(match.StepID)

			stepsToAdditionalMatches[stepId] = append(stepsToAdditionalMatches[stepId], match)
		}
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

		tupleExternalId := tuple.externalId

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

		for stepIndex, step := range orderSteps(steps) {
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

				var (
					parentTaskExternalId pgtype.UUID
					parentTaskId         pgtype.Int8
					parentTaskInsertedAt pgtype.Timestamptz
					childIndex           pgtype.Int8
					childKey             pgtype.Text
				)

				if tuple.parentExternalId != nil {
					parentTaskExternalId = sqlchelpers.UUIDFromStr(*tuple.parentExternalId)
				}

				if tuple.parentTaskId != nil {
					parentTaskId = pgtype.Int8{
						Int64: *tuple.parentTaskId,
						Valid: true,
					}
				}

				if tuple.parentTaskInsertedAt != nil {
					parentTaskInsertedAt = sqlchelpers.TimestamptzFromTime(*tuple.parentTaskInsertedAt)
				}

				if tuple.childIndex != nil {
					childIndex = pgtype.Int8{
						Int64: *tuple.childIndex,
						Valid: true,
					}
				}

				if tuple.childKey != nil {
					childKey = pgtype.Text{
						String: *tuple.childKey,
						Valid:  true,
					}
				}

				eventMatches[tuple.externalId] = append(eventMatches[tuple.externalId], CreateMatchOpts{
					Kind:                 sqlcv1.V1MatchKindTRIGGER,
					Conditions:           conditions,
					TriggerExternalId:    &taskExternalId,
					TriggerWorkflowRunId: &tupleExternalId,
					TriggerStepId:        &stepId,
					TriggerStepIndex: pgtype.Int8{
						Int64: int64(stepIndex),
						Valid: true,
					},
					TriggerParentTaskExternalId: parentTaskExternalId,
					TriggerParentTaskId:         parentTaskId,
					TriggerParentTaskInsertedAt: parentTaskInsertedAt,
					TriggerChildIndex:           childIndex,
					TriggerChildKey:             childKey,
				})
			case len(step.Parents) == 0:
				opt := CreateTaskOpts{
					ExternalId:           taskExternalId,
					WorkflowRunId:        tuple.externalId,
					StepId:               sqlchelpers.UUIDToStr(step.ID),
					Input:                r.newTaskInput(tuple.input, nil),
					AdditionalMetadata:   tuple.additionalMetadata,
					InitialState:         sqlcv1.V1TaskInitialStateQUEUED,
					DesiredWorkerId:      tuple.desiredWorkerId,
					ParentTaskExternalId: tuple.parentExternalId,
					ParentTaskId:         tuple.parentTaskId,
					ParentTaskInsertedAt: tuple.parentTaskInsertedAt,
					StepIndex:            stepIndex,
					ChildIndex:           tuple.childIndex,
					ChildKey:             tuple.childKey,
				}

				if isDag {
					dagTaskOpts[tuple.externalId] = append(dagTaskOpts[tuple.externalId], opt)
				} else {
					nonDagTaskOpts = append(nonDagTaskOpts, opt)
				}
			default:
				conditions := make([]GroupMatchCondition, 0)

				cancelGroupId := uuid.NewString()

				additionalMatches, ok := stepsToAdditionalMatches[stepId]

				if !ok {
					additionalMatches = make([]*sqlcv1.V1StepMatchCondition, 0)
				}

				for _, parent := range step.Parents {
					parentExternalId := stepsToExternalIds[i][sqlchelpers.UUIDToStr(parent)]
					readableId := stepIdsToReadableIds[sqlchelpers.UUIDToStr(parent)]

					hasUserEventOrSleepMatches := false

					parentOverrideMatches := make([]*sqlcv1.V1StepMatchCondition, 0)

					for _, match := range additionalMatches {
						if match.Kind == sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE {
							if match.ParentReadableID.String == readableId {
								parentOverrideMatches = append(parentOverrideMatches, match)
							}
						} else {
							hasUserEventOrSleepMatches = true
						}
					}

					conditions = append(conditions, getParentInDAGGroupMatch(cancelGroupId, parentExternalId, readableId, parentOverrideMatches, hasUserEventOrSleepMatches)...)
				}

				var (
					parentTaskExternalId pgtype.UUID
					parentTaskId         pgtype.Int8
					parentTaskInsertedAt pgtype.Timestamptz
					childIndex           pgtype.Int8
					childKey             pgtype.Text
				)

				if tuple.parentExternalId != nil {
					parentTaskExternalId = sqlchelpers.UUIDFromStr(*tuple.parentExternalId)
				}

				if tuple.parentTaskId != nil {
					parentTaskId = pgtype.Int8{
						Int64: *tuple.parentTaskId,
						Valid: true,
					}
				}

				if tuple.parentTaskInsertedAt != nil {
					parentTaskInsertedAt = sqlchelpers.TimestamptzFromTime(*tuple.parentTaskInsertedAt)
				}

				if tuple.childIndex != nil {
					childIndex = pgtype.Int8{
						Int64: *tuple.childIndex,
						Valid: true,
					}
				}

				if tuple.childKey != nil {
					childKey = pgtype.Text{
						String: *tuple.childKey,
						Valid:  true,
					}
				}

				// create an event match
				eventMatches[tuple.externalId] = append(eventMatches[tuple.externalId], CreateMatchOpts{
					Kind:                 sqlcv1.V1MatchKindTRIGGER,
					Conditions:           conditions,
					TriggerExternalId:    &taskExternalId,
					TriggerWorkflowRunId: &tupleExternalId,
					TriggerStepId:        &stepId,
					TriggerStepIndex: pgtype.Int8{
						Int64: int64(stepIndex),
						Valid: true,
					},
					TriggerParentTaskExternalId: parentTaskExternalId,
					TriggerParentTaskId:         parentTaskId,
					TriggerParentTaskInsertedAt: parentTaskInsertedAt,
					TriggerChildIndex:           childIndex,
					TriggerChildKey:             childKey,
				})
			}
		}

		if isDag {
			dagOpts = append(dagOpts, createDAGOpts{
				ExternalId:           tuple.externalId,
				Input:                tuple.input,
				TaskIds:              dagToTaskIds[tuple.externalId],
				WorkflowId:           tuple.workflowId,
				WorkflowVersionId:    tuple.workflowVersionId,
				WorkflowName:         tuple.workflowName,
				AdditionalMetadata:   tuple.additionalMetadata,
				ParentTaskExternalID: tuple.parentExternalId,
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
		opts := eventMatches[sqlchelpers.UUIDToStr(dag.ExternalID)]

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

	postWR()
	postTask()

	return tasks, dags, nil
}

type DAGWithData struct {
	*sqlcv1.V1Dag

	Input []byte

	AdditionalMetadata []byte

	ParentTaskExternalID *pgtype.UUID
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
	parentTaskExternalIds := make([]pgtype.UUID, 0, len(opts))
	dagIdToOpt := make(map[string]createDAGOpts, 0)

	unix := time.Now().UnixMilli()

	for _, opt := range opts {
		tenantIds = append(tenantIds, sqlchelpers.UUIDFromStr(tenantId))
		externalIds = append(externalIds, sqlchelpers.UUIDFromStr(opt.ExternalId))
		displayNames = append(displayNames, fmt.Sprintf("%s-%d", opt.WorkflowName, unix))
		workflowIds = append(workflowIds, sqlchelpers.UUIDFromStr(opt.WorkflowId))
		workflowVersionIds = append(workflowVersionIds, sqlchelpers.UUIDFromStr(opt.WorkflowVersionId))

		if opt.ParentTaskExternalID == nil {
			parentTaskExternalIds = append(parentTaskExternalIds, pgtype.UUID{})
		} else {
			parentTaskExternalIds = append(parentTaskExternalIds, sqlchelpers.UUIDFromStr(*opt.ParentTaskExternalID))
		}

		dagIdToOpt[opt.ExternalId] = opt
	}

	createdDAGs, err := r.queries.CreateDAGs(ctx, tx, sqlcv1.CreateDAGsParams{
		Tenantids:             tenantIds,
		Externalids:           externalIds,
		Displaynames:          displayNames,
		Workflowids:           workflowIds,
		Workflowversionids:    workflowVersionIds,
		Parenttaskexternalids: parentTaskExternalIds,
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

		parentTaskExternalID := pgtype.UUID{}

		if opt.ParentTaskExternalID != nil {
			parentTaskExternalID = sqlchelpers.UUIDFromStr(*opt.ParentTaskExternalID)
		}

		res = append(res, &DAGWithData{
			V1Dag:                dag,
			Input:                input,
			AdditionalMetadata:   additionalMeta,
			ParentTaskExternalID: &parentTaskExternalID,
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
	potentialMatchTaskInsertedAts := make([]pgtype.Timestamptz, 0, len(tuples))
	externalIdsToKeys := make(map[string]string)

	for i, tuple := range tuples {
		if tuple.parentTaskId == nil {
			continue
		}

		if tuple.parentExternalId == nil {
			r.l.Error().Msg("could not find parent external id")
			continue
		}

		if tuple.parentTaskInsertedAt == nil {
			r.l.Error().Msg("could not find parent task inserted at")
			continue
		}

		if tuple.childIndex == nil {
			r.l.Error().Msg("could not find child index for child workflow")
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

		for stepIndex, step := range orderSteps(steps) {
			stepId := sqlchelpers.UUIDToStr(step.ID)
			stepExternalId := stepsToExternalIds[i][stepId]

			k := getChildSignalEventKey(*tuple.parentExternalId, int64(stepIndex), *tuple.childIndex, tuple.childKey)

			potentialMatchKeys = append(potentialMatchKeys, k)
			potentialMatchTaskIds = append(potentialMatchTaskIds, *tuple.parentTaskId)
			potentialMatchTaskInsertedAts = append(potentialMatchTaskInsertedAts, sqlchelpers.TimestamptzFromTime(*tuple.parentTaskInsertedAt))
			externalIdsToKeys[stepExternalId] = k
		}
	}

	// if we have no potential matches, return early
	if len(potentialMatchKeys) == 0 {
		return nil, nil
	}

	matchingEvents, err := r.queries.LockSignalCreatedEvents(
		ctx,
		tx,
		sqlcv1.LockSignalCreatedEventsParams{
			Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
			Taskids:         potentialMatchTaskIds,
			Taskinsertedats: potentialMatchTaskInsertedAts,
			Eventkeys:       potentialMatchKeys,
		},
	)

	if err != nil {
		return nil, err
	}

	// parse the event match data, and determine whether the child external ID has already been written
	// we're safe to do this read since we've acquired a lock on the relevant rows
	rootExternalIdsToLookup := make([]pgtype.UUID, 0, len(matchingEvents))

	for _, event := range matchingEvents {
		c, err := newChildWorkflowSignalCreatedDataFromBytes(event.Data)

		if err != nil {
			r.l.Error().Msgf("failed to unmarshal child workflow signal created data: %s", err)
			continue
		}

		if c.ChildExternalId != "" {
			rootExternalIdsToLookup = append(rootExternalIdsToLookup, sqlchelpers.UUIDFromStr(c.ChildExternalId))
		}
	}

	// get the child external IDs that have already been written
	existingExternalIds, err := r.queries.LookupExternalIds(ctx, tx, sqlcv1.LookupExternalIdsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Externalids: rootExternalIdsToLookup,
	})

	if err != nil {
		return nil, err
	}

	tuplesToSkip = make(map[string]struct{})

	for _, dbExternalId := range existingExternalIds {
		tuplesToSkip[sqlchelpers.UUIDToStr(dbExternalId.ExternalID)] = struct{}{}
	}

	createMatchOpts := make([]CreateMatchOpts, 0)
	tuplesToSkip = make(map[string]struct{})

	for i, tuple := range tuples {
		if _, ok := tuplesToSkip[tuple.externalId]; ok {
			continue
		}

		// if this is a child workflow run, create a match condition for the parent for each
		// step in the DAG
		if tuple.parentTaskId != nil && tuple.parentExternalId != nil {
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

			for _, step := range orderSteps(steps) {
				stepId := sqlchelpers.UUIDToStr(step.ID)
				stepReadableId := step.ReadableId.String
				stepExternalId := stepsToExternalIds[i][stepId]

				key := externalIdsToKeys[stepExternalId]

				createMatchOpts = append(createMatchOpts, CreateMatchOpts{
					Kind:                 sqlcv1.V1MatchKindSIGNAL,
					Conditions:           getChildWorkflowGroupMatches(stepExternalId, stepReadableId),
					SignalExternalId:     tuple.parentExternalId,
					SignalTaskId:         tuple.parentTaskId,
					SignalTaskInsertedAt: sqlchelpers.TimestamptzFromTime(*tuple.parentTaskInsertedAt),
					SignalKey:            &key,
				})
			}
		}
	}

	// create the relevant matches
	err = r.createEventMatches(ctx, tx, tenantId, createMatchOpts)

	if err != nil {
		return nil, err
	}

	return tuplesToSkip, nil
}

// getParentInDAGGroupMatch encodes the following default behavior:
// - If all parents complete, the child task is created
// - If all parents are skipped, the child task is skipped
// - If parents are both created and skipped, the child is created
// - If any parent is cancelled, the child is cancelled
// - If any parent fails, the child is cancelled
//
// Users can override this behavior by setting their own skip and creation conditions.
func getParentInDAGGroupMatch(
	cancelGroupId, parentExternalId, parentReadableId string,
	parentOverrideMatches []*sqlcv1.V1StepMatchCondition,
	hasUserEventOrSleepMatches bool,
) []GroupMatchCondition {
	completeAction := sqlcv1.V1MatchConditionActionQUEUE

	if hasUserEventOrSleepMatches {
		completeAction = sqlcv1.V1MatchConditionActionCREATEMATCH
	}

	actionsToOverrides := make(map[sqlcv1.V1MatchConditionAction][]*sqlcv1.V1StepMatchCondition)

	for _, match := range parentOverrideMatches {
		actionsToOverrides[match.Action] = append(actionsToOverrides[match.Action], match)
	}

	res := []GroupMatchCondition{}

	if len(actionsToOverrides[sqlcv1.V1MatchConditionActionQUEUE]) > 0 {
		for _, override := range actionsToOverrides[sqlcv1.V1MatchConditionActionQUEUE] {
			res = append(res, GroupMatchCondition{
				GroupId:           sqlchelpers.UUIDToStr(override.OrGroupID),
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
				ReadableDataKey:   parentReadableId,
				EventResourceHint: &parentExternalId,
				Expression:        override.Expression.String,
				Action:            completeAction,
			})
		}
	} else {
		res = append(res, GroupMatchCondition{
			GroupId:           uuid.NewString(),
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			// NOTE: complete match on skip takes precedence over queue, so we might meet all QUEUE conditions with a skipped
			// parent but end up skipping anyway
			Expression: "true",
			Action:     completeAction,
		})
	}

	if len(actionsToOverrides[sqlcv1.V1MatchConditionActionSKIP]) > 0 {
		for _, override := range actionsToOverrides[sqlcv1.V1MatchConditionActionSKIP] {
			res = append(res, GroupMatchCondition{
				GroupId:           sqlchelpers.UUIDToStr(override.OrGroupID),
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
				ReadableDataKey:   parentReadableId,
				EventResourceHint: &parentExternalId,
				Expression:        override.Expression.String,
				Action:            sqlcv1.V1MatchConditionActionSKIP,
			})
		}
	} else {
		res = append(res, GroupMatchCondition{
			GroupId:           uuid.NewString(),
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "has(output.skipped) && output.skipped",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		})
	}

	if len(actionsToOverrides[sqlcv1.V1MatchConditionActionCANCEL]) > 0 {
		for _, override := range actionsToOverrides[sqlcv1.V1MatchConditionActionCANCEL] {
			res = append(res, GroupMatchCondition{
				GroupId:           sqlchelpers.UUIDToStr(override.OrGroupID),
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
				ReadableDataKey:   parentReadableId,
				EventResourceHint: &parentExternalId,
				Expression:        override.Expression.String,
				Action:            sqlcv1.V1MatchConditionActionCANCEL,
			})
		}
	} else {
		res = append(res, GroupMatchCondition{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCANCEL,
		}, GroupMatchCondition{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCANCEL,
		})
	}

	return res
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
			Action:            sqlcv1.V1MatchConditionActionCREATE,
		},
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &taskExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCREATE,
		},
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &taskExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCREATE,
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
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &parentExternalId,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		},
	}
}

func orderSteps(steps []*sqlcv1.ListStepsByWorkflowVersionIdsRow) []*sqlcv1.ListStepsByWorkflowVersionIdsRow {
	slices.SortStableFunc(steps, func(i, j *sqlcv1.ListStepsByWorkflowVersionIdsRow) int {
		idA := sqlchelpers.UUIDToStr(i.ID)
		idB := sqlchelpers.UUIDToStr(j.ID)
		return strings.Compare(idA, idB)
	})

	return steps
}
