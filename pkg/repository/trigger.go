package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type EventTriggerOpts struct {
	ExternalId uuid.UUID

	Key string

	Data []byte

	AdditionalMetadata []byte

	Priority *int32

	Scope *string

	TriggeringWebhookName *string
}

type TriggerTaskData struct {
	// (required) the workflow name
	WorkflowName string `json:"workflow_name" validate:"required"`

	// (optional) the workflow run data
	Data []byte `json:"data"`

	// (optional) the workflow run metadata
	AdditionalMetadata []byte `json:"additional_metadata"`

	// (optional) the desired worker id
	DesiredWorkerId *uuid.UUID `json:"desired_worker_id"`

	// (optional) the parent external id
	ParentExternalId *uuid.UUID `json:"parent_external_id"`

	// (optional) the parent task id
	ParentTaskId *int64 `json:"parent_task_id"`

	// (optional) the parent inserted_at
	ParentTaskInsertedAt *time.Time `json:"parent_task_inserted_at"`

	// (optional) the child index
	ChildIndex *int64 `json:"child_index"`

	// (optional) the child key
	ChildKey *string `json:"child_key"`

	// (optional) the priority of the task
	Priority *int32 `json:"priority"`
}

type createDAGOpts struct {
	// (required) the external id
	ExternalId uuid.UUID `validate:"required"`

	// (required) the input bytes to the DAG
	Input []byte

	// (required) a list of task external ids that are part of this DAG
	TaskIds []uuid.UUID

	// (required) the workflow id for this DAG
	WorkflowId uuid.UUID

	// (required) the workflow version id for this DAG
	WorkflowVersionId uuid.UUID

	// (required) the name of the workflow
	WorkflowName string

	// (optional) the additional metadata for the DAG
	AdditionalMetadata []byte

	ParentTaskExternalID *uuid.UUID
}

type TriggerRepository interface {
	TriggerFromEvents(ctx context.Context, tenantId uuid.UUID, opts []EventTriggerOpts) (*TriggerFromEventsResult, error)

	TriggerFromWorkflowNames(ctx context.Context, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) ([]*V1TaskWithPayload, []*DAGWithData, error)

	PopulateExternalIdsForWorkflow(ctx context.Context, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) error

	PreflightVerifyWorkflowNameOpts(ctx context.Context, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) error
}

type TriggerRepositoryImpl struct {
	*sharedRepository

	enableDurableUserEventLog bool
}

func newTriggerRepository(s *sharedRepository, enableDurableUserEventLog bool) TriggerRepository {
	return &TriggerRepositoryImpl{
		sharedRepository:          s,
		enableDurableUserEventLog: enableDurableUserEventLog,
	}
}

type Run struct {
	Id         int64
	InsertedAt time.Time
	FilterId   *uuid.UUID
}

type TriggerFromEventsResult struct {
	Tasks                 []*V1TaskWithPayload
	Dags                  []*DAGWithData
	EventExternalIdToRuns map[uuid.UUID][]*Run
	CELEvaluationFailures []CELEvaluationFailure
}

type TriggerDecision struct {
	ShouldTrigger bool
	FilterPayload []byte
	FilterId      *uuid.UUID
}

func (r *sharedRepository) makeTriggerDecisions(ctx context.Context, filters []*sqlcv1.V1Filter, hasAnyFilters bool, opt EventTriggerOpts) ([]TriggerDecision, []CELEvaluationFailure) {
	celEvaluationFailures := make([]CELEvaluationFailure, 0)

	// Cases to handle:
	// 1. If there are no filters that exist for the workflow, we should trigger it.
	// 2. If there _are_ filters that exist, but the list is empty, then there were no scope matches so we should _not_ trigger.
	// 3. If there _are_ filters that exist and the list is non-empty, then we should loop through the list and evaluate each expression, and trigger if the expression evaluates to `true`.

	// Case 1 - no filters exist for the workflow
	if !hasAnyFilters {
		return []TriggerDecision{
			{
				ShouldTrigger: true,
				FilterPayload: nil,
				FilterId:      nil,
			},
		}, celEvaluationFailures
	}

	// Case 2 - no filters were found matching the provided scope,
	// so we should not trigger the workflow
	if len(filters) == 0 {
		return []TriggerDecision{
			{
				ShouldTrigger: false,
				FilterPayload: nil,
				FilterId:      nil,
			},
		}, celEvaluationFailures
	}

	// Case 3 - we have filters, so we should evaluate each expression and return a list of decisions
	decisions := make([]TriggerDecision, 0, len(filters))

	for _, filter := range filters {
		if filter == nil {
			continue
		}

		filterId := filter.ID

		if filter.Expression == "" {
			decisions = append(decisions, TriggerDecision{
				ShouldTrigger: false,
				FilterPayload: filter.Payload,
				FilterId:      &filterId,
			})
		} else {
			shouldTrigger, err := r.processWorkflowExpression(ctx, filter.Expression, opt, filter.Payload)

			if err != nil {
				r.l.Error().
					Err(err).
					Str("expression", filter.Expression).
					Msg("Failed to evaluate workflow expression")

				// If we fail to parse the expression, we should not run the workflow.
				// See: https://github.com/hatchet-dev/hatchet/pull/1676#discussion_r2073790939
				decisions = append(decisions, TriggerDecision{
					ShouldTrigger: false,
					FilterPayload: filter.Payload,
					FilterId:      &filterId,
				})

				celEvaluationFailures = append(celEvaluationFailures, CELEvaluationFailure{
					Source:       sqlcv1.V1CelEvaluationFailureSourceFILTER,
					ErrorMessage: err.Error(),
				})
			}

			decisions = append(decisions, TriggerDecision{
				ShouldTrigger: shouldTrigger,
				FilterPayload: filter.Payload,
				FilterId:      &filterId,
			})
		}
	}

	return decisions, celEvaluationFailures
}

type EventExternalIdFilterId struct {
	ExternalId uuid.UUID
	FilterId   *uuid.UUID
}

type EventIds struct {
	SeenAt pgtype.Timestamptz
	Id     int64
}

type WorkflowAndScope struct {
	WorkflowId uuid.UUID
	Scope      string
}

func (r *TriggerRepositoryImpl) TriggerFromEvents(ctx context.Context, tenantId uuid.UUID, opts []EventTriggerOpts) (*TriggerFromEventsResult, error) {
	pre, post := r.m.Meter(ctx, sqlcv1.LimitResourceEVENT, tenantId, int32(len(opts))) // nolint: gosec

	if err := pre(); err != nil {
		return nil, err
	}

	result, err := r.doTriggerFromEvents(ctx, nil, tenantId, opts)

	if err != nil {
		return nil, err
	}

	post()

	return result, nil
}

func (r *sharedRepository) doTriggerFromEvents(
	ctx context.Context,
	tx *OptimisticTx,
	tenantId uuid.UUID,
	opts []EventTriggerOpts,
) (*TriggerFromEventsResult, error) {
	var prepareTx sqlcv1.DBTX

	if tx != nil {
		prepareTx = tx.tx
	} else {
		prepareTx = r.pool
	}

	triggerOpts, createCoreEventOpts, externalIdToEventIdAndFilterId, celEvaluationFailures, err := r.prepareTriggerFromEvents(ctx, prepareTx, tenantId, opts)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare trigger from events: %w", err)
	}

	tasks, dags, err := r.triggerWorkflows(ctx, tx, tenantId, triggerOpts, createCoreEventOpts)

	if err != nil {
		return nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	eventExternalIdToRuns := getEventExternalIdToRuns(opts, externalIdToEventIdAndFilterId, tasks, dags)

	return &TriggerFromEventsResult{
		Tasks:                 tasks,
		Dags:                  dags,
		EventExternalIdToRuns: eventExternalIdToRuns,
		CELEvaluationFailures: celEvaluationFailures,
	}, nil
}

func getEventExternalIdToRuns(opts []EventTriggerOpts, externalIdToEventIdAndFilterId map[uuid.UUID]EventExternalIdFilterId, tasks []*V1TaskWithPayload, dags []*DAGWithData) map[uuid.UUID][]*Run {
	eventExternalIdToRuns := make(map[uuid.UUID][]*Run)

	for _, opt := range opts {
		eventExternalIdToRuns[opt.ExternalId] = make([]*Run, 0)
	}

	for _, task := range tasks {
		externalId := task.ExternalID

		eventIdAndFilterId, ok := externalIdToEventIdAndFilterId[externalId]

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

		eventIdAndFilterId, ok := externalIdToEventIdAndFilterId[externalId]

		if !ok {
			continue
		}

		eventExternalIdToRuns[eventIdAndFilterId.ExternalId] = append(eventExternalIdToRuns[eventIdAndFilterId.ExternalId], &Run{
			Id:         dag.ID,
			InsertedAt: dag.InsertedAt.Time,
			FilterId:   eventIdAndFilterId.FilterId,
		})
	}

	return eventExternalIdToRuns
}

func (r *TriggerRepositoryImpl) TriggerFromWorkflowNames(ctx context.Context, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) ([]*V1TaskWithPayload, []*DAGWithData, error) {
	tx, err := r.PrepareOptimisticTx(ctx)

	if err != nil {
		return nil, nil, err
	}

	rolledBack := false
	defer func() {
		if !rolledBack {
			tx.Rollback()
		}
	}()

	triggerOpts, denyUpdateKeys, duplicateKeys, err := r.prepareTriggerFromWorkflowNames(ctx, tx.tx, tenantId, opts)

	if err != nil {
		tx.Rollback()
		rolledBack = true
		if errors.Is(err, ErrIdempotencyKeyAlreadyClaimed) && len(denyUpdateKeys) > 0 {
			updateErr := r.queries.UpdateIdempotencyKeysLastDeniedAt(ctx, r.pool, sqlcv1.UpdateIdempotencyKeysLastDeniedAtParams{
				Tenantid: tenantId,
				Keys:     denyUpdateKeys,
			})
			if updateErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to update idempotency key deny timestamps: %w", updateErr))
			}
		}

		return nil, nil, fmt.Errorf("failed to prepare trigger from workflow names: %w", err)
	}

	if len(duplicateKeys) > 0 && len(triggerOpts) > 0 {
		r.l.Warn().
			Str("tenantId", tenantId.String()).
			Int("duplicateKeyCount", len(duplicateKeys)).
			Msg("partial idempotency duplicates skipped during trigger")
	}

	tasks, dags, err := r.triggerWorkflows(ctx, tx, tenantId, triggerOpts, nil)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	rolledBack = true

	if len(denyUpdateKeys) > 0 {
		updateErr := r.queries.UpdateIdempotencyKeysLastDeniedAt(ctx, r.pool, sqlcv1.UpdateIdempotencyKeysLastDeniedAtParams{
			Tenantid: tenantId,
			Keys:     denyUpdateKeys,
		})
		if updateErr != nil {
			r.l.Error().
				Err(updateErr).
				Str("tenantId", tenantId.String()).
				Int("keyCount", len(denyUpdateKeys)).
				Msg("failed to update idempotency key deny timestamps")
		}
	}

	return tasks, dags, nil
}

type ErrNamesNotFound struct {
	Names []string
}

func (e *ErrNamesNotFound) Error() string {
	return fmt.Sprintf("workflow names not found: %s", strings.Join(e.Names, ", "))
}

var ErrIdempotencyKeyAlreadyClaimed = errors.New("idempotency key already claimed")

type IdempotencyKeyAlreadyClaimedError struct {
	Keys []string
}

func (e *IdempotencyKeyAlreadyClaimedError) Error() string {
	return fmt.Sprintf("idempotency key already claimed: %s", strings.Join(e.Keys, ", "))
}

func (e *IdempotencyKeyAlreadyClaimedError) Is(target error) bool {
	return target == ErrIdempotencyKeyAlreadyClaimed
}

func isTerminalReadableStatus(status sqlcv1.V1ReadableStatusOlap) bool {
	switch status {
	case sqlcv1.V1ReadableStatusOlapCOMPLETED,
		sqlcv1.V1ReadableStatusOlapFAILED,
		sqlcv1.V1ReadableStatusOlapCANCELLED:
		return true
	default:
		return false
	}
}

func (r *TriggerRepositoryImpl) PreflightVerifyWorkflowNameOpts(ctx context.Context, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) error {
	// get a list of workflow names
	workflowNamesFound := make(map[string]bool)

	for _, opt := range opts {
		workflowNamesFound[opt.WorkflowName] = false
	}

	uniqueWorkflowNames := make([]string, 0, len(workflowNamesFound))

	for name := range workflowNamesFound {
		uniqueWorkflowNames = append(uniqueWorkflowNames, name)
	}

	rows, err := r.listWorkflowsByNames(ctx, r.pool, tenantId, uniqueWorkflowNames)

	if err != nil {
		return fmt.Errorf("failed to list workflows by names: %w", err)
	}

	for _, row := range rows {
		workflowNamesFound[row.WorkflowName] = true
	}

	workflowNamesNotFound := make([]string, 0)

	for name, found := range workflowNamesFound {
		if !found {
			workflowNamesNotFound = append(workflowNamesNotFound, name)
		}
	}

	if len(workflowNamesNotFound) > 0 {
		return &ErrNamesNotFound{
			Names: workflowNamesNotFound,
		}
	}

	return nil
}

type TriggeredBy interface {
	ToMetadata([]byte) []byte
}

type TriggeredByEvent struct {
	l        *zerolog.Logger
	eventID  uuid.UUID
	eventKey string
}

func cleanAdditionalMetadata(additionalMetadata []byte) map[string]interface{} {
	res := make(map[string]interface{})

	if len(additionalMetadata) == 0 {
		res = make(map[string]interface{})
	} else {
		err := json.Unmarshal(additionalMetadata, &res)

		if err != nil || res == nil {
			res = make(map[string]interface{})
		}
	}

	for key := range res {
		if strings.HasPrefix(key, "hatchet__") {
			delete(res, key)
		}
	}

	return res
}

func (t *TriggeredByEvent) ToMetadata(additionalMetadata []byte) []byte {
	res := cleanAdditionalMetadata(additionalMetadata)

	res[constants.EventIDKey.String()] = t.eventID
	res[constants.EventKeyKey.String()] = t.eventKey

	resBytes, err := json.Marshal(res)

	if err != nil {
		t.l.Error().Err(err).Msg("failed to marshal additional metadata")
		return nil
	}

	return resBytes
}

type triggerTuple struct {
	desiredWorkerId      *uuid.UUID
	childKey             *string
	childIndex           *int64
	parentTaskInsertedAt *time.Time
	parentTaskId         *int64
	parentExternalId     *uuid.UUID
	priority             *int32
	externalId           uuid.UUID
	workflowVersionId    uuid.UUID
	workflowName         string
	workflowId           uuid.UUID
	additionalMetadata   []byte
	filterPayload        []byte
	input                []byte
}

type createCoreUserEventOpts struct {
	externalIdToEventIdAndFilterId map[uuid.UUID]EventExternalIdFilterId
	externalIdsToPayloads          map[uuid.UUID][]byte
	params                         sqlcv1.BulkCreateEventsParams
}

func (r *sharedRepository) triggerWorkflows(
	ctx context.Context,
	existingTx *OptimisticTx,
	tenantId uuid.UUID,
	tuples []triggerTuple,
	coreEvents *createCoreUserEventOpts,
) ([]*V1TaskWithPayload, []*DAGWithData, error) {
	// get unique workflow version ids
	uniqueWorkflowVersionIds := make(map[uuid.UUID]struct{})

	for _, tuple := range tuples {
		uniqueWorkflowVersionIds[tuple.workflowVersionId] = struct{}{}
	}

	// get all data for triggering tasks in this workflow
	workflowVersionIds := make([]uuid.UUID, 0, len(uniqueWorkflowVersionIds))

	for id := range uniqueWorkflowVersionIds {
		workflowVersionIds = append(workflowVersionIds, id)
	}

	var listStepsTx sqlcv1.DBTX = r.pool

	if existingTx != nil {
		listStepsTx = existingTx.tx
	}

	workflowVersionToSteps, err := r.listStepsByWorkflowVersionIds(ctx, listStepsTx, tenantId, workflowVersionIds)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get workflow versions for engine: %w", err)
	}

	// group steps by workflow version ids
	stepIdsToReadableIds := make(map[uuid.UUID]string)
	stepsWithAdditionalMatchConditions := make([]uuid.UUID, 0)

	for _, steps := range workflowVersionToSteps {
		for _, step := range steps {
			stepIdsToReadableIds[step.ID] = step.ReadableId.String

			if step.MatchConditionCount > 0 {
				stepsWithAdditionalMatchConditions = append(stepsWithAdditionalMatchConditions, step.ID)
			}
		}
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

	preTask, postTask := r.m.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, int32(countTasks)) // nolint: gosec

	if err := preTask(); err != nil {
		return nil, nil, err
	}

	stepsToAdditionalMatches := make(map[uuid.UUID][]*sqlcv1.V1StepMatchCondition)

	if len(stepsWithAdditionalMatchConditions) > 0 {
		additionalMatches, err := r.queries.ListStepMatchConditions(ctx, r.pool, sqlcv1.ListStepMatchConditionsParams{
			Stepids:  stepsWithAdditionalMatchConditions,
			Tenantid: tenantId,
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to list step match conditions: %w", err)
		}

		for _, match := range additionalMatches {
			stepId := match.StepID

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
	dagTaskOpts := make(map[uuid.UUID][]CreateTaskOpts)
	nonDagTaskOpts := make([]CreateTaskOpts, 0)

	// map of task external IDs to matches
	eventMatches := make(map[uuid.UUID][]CreateMatchOpts)
	createMatchOpts := make([]CreateMatchOpts, 0)

	// a map of trigger tuples to step external IDs
	stepsToExternalIds := make([]map[uuid.UUID]uuid.UUID, len(tuples))
	dagToTaskIds := make(map[uuid.UUID][]uuid.UUID)

	// generate UUIDs for each step
	for i, tuple := range tuples {
		stepsToExternalIds[i] = make(map[uuid.UUID]uuid.UUID)

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
				stepsToExternalIds[i][step.ID] = tuple.externalId
			} else {
				externalId := uuid.New()
				stepsToExternalIds[i][step.ID] = externalId
				dagToTaskIds[tuple.externalId] = append(dagToTaskIds[tuple.externalId], externalId)
			}
		}
	}

	var commit func(context.Context) error
	var rollback func()
	var tx sqlcv1.DBTX

	if existingTx == nil {
		tx, commit, rollback, err = sqlchelpers.PrepareTx(ctx, r.pool, r.l)

		if err != nil {
			return nil, nil, err
		}

		defer rollback()
	} else {
		tx = existingTx.tx
	}

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
			stepId := step.ID
			taskExternalId := stepsToExternalIds[i][stepId]

			// if this is an on failure step, create match conditions for every other step in the DAG
			switch {
			case step.JobKind == sqlcv1.JobKindONFAILURE:
				conditions := make([]GroupMatchCondition, 0)
				groupId := uuid.New()

				for _, otherStep := range steps {
					if otherStep.ID == stepId {
						continue
					}

					otherExternalId := stepsToExternalIds[i][otherStep.ID]
					readableId := otherStep.ReadableId.String

					conditions = append(conditions, getParentOnFailureGroupMatches(groupId, otherExternalId, readableId)...)
				}

				var (
					parentTaskExternalId *uuid.UUID
					parentTaskId         pgtype.Int8
					parentTaskInsertedAt pgtype.Timestamptz
					childIndex           pgtype.Int8
					childKey             pgtype.Text
					priority             pgtype.Int4
				)

				if tuple.parentExternalId != nil {
					parsed := *tuple.parentExternalId
					parentTaskExternalId = &parsed
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

				if tuple.priority != nil {
					priority = pgtype.Int4{
						Int32: *tuple.priority,
						Valid: true,
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
					TriggerPriority:             priority,
				})
			case len(step.Parents) == 0:
				// if we have additional match conditions, create a match instead of triggering a workflow for this step
				additionalMatches := stepsToAdditionalMatches[stepId]

				if len(additionalMatches) > 0 {
					// create an event match
					groupConditions := make([]GroupMatchCondition, 0)

					for _, condition := range additionalMatches {
						switch condition.Kind {
						case sqlcv1.V1StepMatchConditionKindSLEEP:
							c, err := r.durableSleepCondition(
								ctx,
								tx,
								tenantId,
								condition.OrGroupID,
								condition.ReadableDataKey,
								condition.SleepDuration.String,
								condition.Action,
							)

							if err != nil {
								return nil, nil, fmt.Errorf("failed to create sleep condition: %w", err)
							}

							groupConditions = append(groupConditions, *c)
						case sqlcv1.V1StepMatchConditionKindUSEREVENT:
							groupConditions = append(groupConditions, r.userEventCondition(
								condition.OrGroupID,
								condition.ReadableDataKey,
								condition.EventKey.String,
								condition.Expression.String,
								condition.Action,
							))
						default:
							// PARENT_OVERRIDE is another kind, but it isn't processed here
							continue
						}
					}

					var (
						parentTaskExternalId *uuid.UUID
						parentTaskId         pgtype.Int8
						parentTaskInsertedAt pgtype.Timestamptz
						childIndex           pgtype.Int8
						childKey             pgtype.Text
						priority             pgtype.Int4
					)

					if tuple.parentExternalId != nil {
						parsed := *tuple.parentExternalId
						parentTaskExternalId = &parsed
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

					if tuple.priority != nil {
						priority = pgtype.Int4{
							Int32: *tuple.priority,
							Valid: true,
						}
					}

					eventMatches[tuple.externalId] = append(eventMatches[tuple.externalId], CreateMatchOpts{
						Kind:                 sqlcv1.V1MatchKindTRIGGER,
						Conditions:           groupConditions,
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
						TriggerPriority:             priority,
					})
				} else {
					opt := CreateTaskOpts{
						ExternalId:           taskExternalId,
						WorkflowRunId:        tuple.externalId,
						StepId:               step.ID,
						Input:                r.newTaskInput(tuple.input, nil, tuple.filterPayload),
						AdditionalMetadata:   tuple.additionalMetadata,
						InitialState:         sqlcv1.V1TaskInitialStateQUEUED,
						DesiredWorkerId:      tuple.desiredWorkerId,
						ParentTaskExternalId: tuple.parentExternalId,
						ParentTaskId:         tuple.parentTaskId,
						ParentTaskInsertedAt: tuple.parentTaskInsertedAt,
						StepIndex:            stepIndex,
						ChildIndex:           tuple.childIndex,
						ChildKey:             tuple.childKey,
						Priority:             tuple.priority,
					}

					if isDag {
						dagTaskOpts[tuple.externalId] = append(dagTaskOpts[tuple.externalId], opt)
					} else {
						nonDagTaskOpts = append(nonDagTaskOpts, opt)
					}
				}
			default:
				conditions := make([]GroupMatchCondition, 0)

				cancelGroupId := uuid.New()

				additionalMatches, ok := stepsToAdditionalMatches[stepId]

				if !ok {
					additionalMatches = make([]*sqlcv1.V1StepMatchCondition, 0)
				}

				for _, parent := range step.Parents {
					parentExternalId := stepsToExternalIds[i][parent]
					readableId := stepIdsToReadableIds[parent]

					hasUserEventOrSleepMatches := false
					hasAnySkippingParentOverrides := false

					parentOverrideMatches := make([]*sqlcv1.V1StepMatchCondition, 0)

					for _, match := range additionalMatches {
						if match.Kind == sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE {
							if match.ParentReadableID.String == readableId {
								parentOverrideMatches = append(parentOverrideMatches, match)
							}

							if match.Action == sqlcv1.V1MatchConditionActionSKIP {
								hasAnySkippingParentOverrides = true
							}
						} else {
							hasUserEventOrSleepMatches = true
						}
					}

					conditions = append(conditions, getParentInDAGGroupMatch(cancelGroupId, parentExternalId, readableId, parentOverrideMatches, hasUserEventOrSleepMatches, hasAnySkippingParentOverrides)...)
				}

				var (
					parentTaskExternalId *uuid.UUID
					parentTaskId         pgtype.Int8
					parentTaskInsertedAt pgtype.Timestamptz
					childIndex           pgtype.Int8
					childKey             pgtype.Text
					priority             pgtype.Int4
				)

				if tuple.parentExternalId != nil {
					parentTaskExternalId = tuple.parentExternalId
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

				if tuple.priority != nil {
					priority = pgtype.Int4{
						Int32: *tuple.priority,
						Valid: true,
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
					TriggerPriority:             priority,
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
		opts, ok := dagTaskOpts[dag.ExternalID]

		if !ok {
			r.l.Error().Msgf("could not find task opts for DAG with external id: %s", dag.ExternalID.String())
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
		opts := eventMatches[dag.ExternalID]

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

	storePayloadOpts := make([]StorePayloadOpts, 0, len(tasks))

	for _, task := range tasks {
		storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			ExternalId: task.ExternalID,
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			Payload:    task.Payload,
			TenantId:   tenantId,
		})
	}

	for _, dag := range dags {
		storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
			Id:         dag.ID,
			InsertedAt: dag.InsertedAt,
			ExternalId: dag.ExternalID,
			Type:       sqlcv1.V1PayloadTypeDAGINPUT,
			Payload:    dag.Input,
			TenantId:   tenantId,
		})
	}

	if coreEvents != nil {
		eventExternalIdsToIds := make(map[uuid.UUID]EventIds)

		createdEvents, err := r.queries.BulkCreateEvents(ctx, tx, coreEvents.params)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to create core events: %w", err)
		}

		for _, createdEvent := range createdEvents {
			eventExternalIdsToIds[createdEvent.ExternalID] = EventIds{
				Id:     createdEvent.ID,
				SeenAt: createdEvent.SeenAt,
			}
		}

		eventToRunExternalIds := []uuid.UUID{}
		eventToRunEventIds := []int64{}
		eventToRunEventSeenAts := []pgtype.Timestamptz{}
		eventToRunRunFilterIds := []uuid.UUID{}

		for _, task := range tasks {
			externalId := task.ExternalID

			eventIdAndFilterId, ok := coreEvents.externalIdToEventIdAndFilterId[externalId]

			if !ok {
				continue
			}

			eventIds, ok := eventExternalIdsToIds[eventIdAndFilterId.ExternalId]

			if !ok {
				continue
			}

			eventToRunExternalIds = append(eventToRunExternalIds, task.ExternalID)
			eventToRunEventIds = append(eventToRunEventIds, eventIds.Id)
			eventToRunEventSeenAts = append(eventToRunEventSeenAts, eventIds.SeenAt)

			if eventIdAndFilterId.FilterId != nil {
				eventToRunRunFilterIds = append(eventToRunRunFilterIds, *eventIdAndFilterId.FilterId)
			} else {
				// fixme: this will write a bunch of nil ids into the filter id column (which is nullable)
				eventToRunRunFilterIds = append(eventToRunRunFilterIds, uuid.Nil)
			}
		}

		for _, dag := range dags {
			externalId := dag.ExternalID

			eventIdAndFilterId, ok := coreEvents.externalIdToEventIdAndFilterId[externalId]

			if !ok {
				continue
			}

			eventIds, ok := eventExternalIdsToIds[eventIdAndFilterId.ExternalId]

			if !ok {
				continue
			}

			eventToRunExternalIds = append(eventToRunExternalIds, dag.ExternalID)
			eventToRunEventIds = append(eventToRunEventIds, eventIds.Id)
			eventToRunEventSeenAts = append(eventToRunEventSeenAts, eventIds.SeenAt)

			if eventIdAndFilterId.FilterId != nil {
				eventToRunRunFilterIds = append(eventToRunRunFilterIds, *eventIdAndFilterId.FilterId)
			} else {
				eventToRunRunFilterIds = append(eventToRunRunFilterIds, uuid.Nil)
			}
		}

		_, err = r.queries.CreateEventToRuns(ctx, tx, sqlcv1.CreateEventToRunsParams{
			Runexternalids: eventToRunExternalIds,
			Eventids:       eventToRunEventIds,
			Eventseenats:   eventToRunEventSeenAts,
			Filterids:      eventToRunRunFilterIds,
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to create event to runs: %w", err)
		}

		for _, e := range createdEvents {
			payload, ok := coreEvents.externalIdsToPayloads[e.ExternalID]

			if !ok {
				continue
			}

			storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
				Id:         e.ID,
				InsertedAt: e.SeenAt,
				ExternalId: e.ExternalID,
				Type:       sqlcv1.V1PayloadTypeUSEREVENTINPUT,
				Payload:    payload,
				TenantId:   tenantId,
			})
		}
	}

	err = r.payloadStore.Store(ctx, tx, storePayloadOpts...)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to store payloads: %w", err)
	}

	// commit if we started the transaction
	if existingTx == nil {
		if err := commit(ctx); err != nil {
			return nil, nil, err
		}

		postTask()

	} else {
		existingTx.AddPostCommit(postTask)
	}

	return tasks, dags, nil
}

type DAGWithData struct {
	*sqlcv1.V1Dag

	Input []byte

	AdditionalMetadata []byte

	ParentTaskExternalID *uuid.UUID

	TotalTasks int
}

type V1TaskWithPayload struct {
	*sqlcv1.V1Task
	Payload []byte `json:"payload"`
}

type V1TaskEventWithPayload struct {
	*sqlcv1.V1TaskEvent
	Payload []byte `json:"payload"`
}

func (r *sharedRepository) createDAGs(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, opts []createDAGOpts) ([]*DAGWithData, error) {
	if len(opts) == 0 {
		return nil, nil
	}

	tenantIds := make([]uuid.UUID, 0, len(opts))
	externalIds := make([]uuid.UUID, 0, len(opts))
	displayNames := make([]string, 0, len(opts))
	workflowIds := make([]uuid.UUID, 0, len(opts))
	workflowVersionIds := make([]uuid.UUID, 0, len(opts))
	parentTaskExternalIds := make([]uuid.UUID, 0, len(opts))
	dagIdToOpt := make(map[uuid.UUID]createDAGOpts, 0)

	unix := time.Now().UnixMilli()

	for _, opt := range opts {
		tenantIds = append(tenantIds, tenantId)
		externalIds = append(externalIds, opt.ExternalId)
		displayNames = append(displayNames, fmt.Sprintf("%s-%d", opt.WorkflowName, unix))
		workflowIds = append(workflowIds, opt.WorkflowId)
		workflowVersionIds = append(workflowVersionIds, opt.WorkflowVersionId)

		if opt.ParentTaskExternalID == nil {
			parentTaskExternalIds = append(parentTaskExternalIds, uuid.UUID{})
		} else {
			parentTaskExternalIds = append(parentTaskExternalIds, *opt.ParentTaskExternalID)
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
		externalId := dag.ExternalID
		opt, ok := dagIdToOpt[externalId]

		if !ok {
			r.l.Error().Msgf("could not find DAG opt for DAG with external id: %s", externalId)
			continue
		}

		input := opt.Input

		if len(input) == 0 {
			input = []byte("{}")
		}

		// todo: remove this logic when we remove the need for dual writes
		// in the meantime, basically just passes the dag data through this function
		// back to the caller without writing it
		inputToWrite := input
		if !r.payloadStore.DagDataDualWritesEnabled() {
			inputToWrite = []byte("{}")
		}

		additionalMeta := opt.AdditionalMetadata

		if len(additionalMeta) == 0 {
			additionalMeta = []byte("{}")
		}

		dagDataParams = append(dagDataParams, sqlcv1.CreateDAGDataParams{
			DagID:              dag.ID,
			DagInsertedAt:      dag.InsertedAt,
			Input:              inputToWrite,
			AdditionalMetadata: additionalMeta,
		})

		parentTaskExternalID := uuid.UUID{}

		if opt.ParentTaskExternalID != nil {
			parentTaskExternalID = *opt.ParentTaskExternalID
		}

		res = append(res, &DAGWithData{
			V1Dag:                dag,
			Input:                input,
			AdditionalMetadata:   additionalMeta,
			ParentTaskExternalID: &parentTaskExternalID,
			TotalTasks:           len(opt.TaskIds),
		})
	}

	_, err = r.queries.CreateDAGData(ctx, tx, dagDataParams)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *sharedRepository) registerChildWorkflows(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	tuples []triggerTuple,
	stepsToExternalIds []map[uuid.UUID]uuid.UUID,
	workflowVersionToSteps map[uuid.UUID][]*sqlcv1.ListStepsByWorkflowVersionIdsRow,
) (tuplesToSkip map[uuid.UUID]struct{}, err error) {
	potentialMatchKeys := make([]string, 0, len(tuples))
	potentialMatchTaskIds := make([]int64, 0, len(tuples))
	potentialMatchTaskInsertedAts := make([]pgtype.Timestamptz, 0, len(tuples))
	externalIdsToKeys := make(map[uuid.UUID]string)

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
			stepId := step.ID
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
			Tenantid:        tenantId,
			Taskids:         potentialMatchTaskIds,
			Taskinsertedats: potentialMatchTaskInsertedAts,
			Eventkeys:       potentialMatchKeys,
		},
	)

	if err != nil {
		return nil, err
	}

	retrievePayloadOpts := make([]RetrievePayloadOpts, len(matchingEvents))

	for i, event := range matchingEvents {
		retrievePayloadOpts[i] = RetrievePayloadOpts{
			Id:         event.ID,
			InsertedAt: event.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKEVENTDATA,
			TenantId:   tenantId,
		}
	}

	payloads, err := r.payloadStore.Retrieve(ctx, tx, retrievePayloadOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payloads for signal created events: %w", err)
	}

	// parse the event match data, and determine whether the child external ID has already been written
	// we're safe to do this read since we've acquired a lock on the relevant rows
	rootExternalIdsToLookup := make([]uuid.UUID, 0, len(matchingEvents))

	for _, event := range matchingEvents {
		payload, ok := payloads[RetrievePayloadOpts{
			Id:         event.ID,
			InsertedAt: event.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKEVENTDATA,
			TenantId:   tenantId,
		}]

		if !ok {
			payload = event.Data
		}

		c, err := newChildWorkflowSignalCreatedDataFromBytes(payload)

		if err != nil {
			r.l.Error().Msgf("failed to unmarshal child workflow signal created data: %s", err)
			continue
		}

		if c.ChildExternalId != uuid.Nil {
			rootExternalIdsToLookup = append(rootExternalIdsToLookup, c.ChildExternalId)
		}
	}

	// get the child external IDs that have already been written
	existingExternalIds, err := r.queries.LookupExternalIds(ctx, tx, sqlcv1.LookupExternalIdsParams{
		Tenantid:    tenantId,
		Externalids: rootExternalIdsToLookup,
	})

	if err != nil {
		return nil, err
	}

	tuplesToSkip = make(map[uuid.UUID]struct{})

	for _, dbExternalId := range existingExternalIds {
		tuplesToSkip[dbExternalId.ExternalID] = struct{}{}
	}

	createMatchOpts := make([]CreateMatchOpts, 0)
	tuplesToSkip = make(map[uuid.UUID]struct{})

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
				stepId := step.ID
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
	cancelGroupId, parentExternalId uuid.UUID, parentReadableId string,
	parentOverrideMatches []*sqlcv1.V1StepMatchCondition,
	hasUserEventOrSleepMatches, hasAnySkippingParentOverrides bool,
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
			hint := parentExternalId.String()
			res = append(res, GroupMatchCondition{
				GroupId:           override.OrGroupID,
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
				ReadableDataKey:   parentReadableId,
				EventResourceHint: &hint,
				Expression:        override.Expression.String,
				Action:            completeAction,
			})
		}
	} else {
		hint := parentExternalId.String()
		res = append(res, GroupMatchCondition{
			GroupId:           uuid.New(),
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &hint,
			// NOTE: complete match on skip takes precedence over queue, so we might meet all QUEUE conditions with a skipped
			// parent but end up skipping anyway
			Expression: "true",
			Action:     completeAction,
		})
	}

	if len(actionsToOverrides[sqlcv1.V1MatchConditionActionSKIP]) > 0 {
		hint := parentExternalId.String()
		for _, override := range actionsToOverrides[sqlcv1.V1MatchConditionActionSKIP] {
			res = append(res, GroupMatchCondition{
				GroupId:           override.OrGroupID,
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
				ReadableDataKey:   parentReadableId,
				EventResourceHint: &hint,
				Expression:        override.Expression.String,
				Action:            sqlcv1.V1MatchConditionActionSKIP,
			})
		}
	} else if !hasAnySkippingParentOverrides {
		hint := parentExternalId.String()
		res = append(res, GroupMatchCondition{
			GroupId:           uuid.New(),
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &hint,
			Expression:        "has(output.skipped) && output.skipped",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		})
	}

	if len(actionsToOverrides[sqlcv1.V1MatchConditionActionCANCEL]) > 0 {
		for _, override := range actionsToOverrides[sqlcv1.V1MatchConditionActionCANCEL] {
			hint := parentExternalId.String()
			res = append(res,
				GroupMatchCondition{
					GroupId:   override.OrGroupID,
					EventType: sqlcv1.V1EventTypeINTERNAL,
					// The custom cancel condition matches on the completed event
					EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
					ReadableDataKey:   parentReadableId,
					EventResourceHint: &hint,
					Expression:        override.Expression.String,
					Action:            sqlcv1.V1MatchConditionActionCANCEL,
				},
				// always add the original cancel group match conditions. these can't be modified otherwise DAGs risk
				// getting stuck in a concurrency queue.
				GroupMatchCondition{
					GroupId:           override.OrGroupID,
					EventType:         sqlcv1.V1EventTypeINTERNAL,
					EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
					ReadableDataKey:   parentReadableId,
					EventResourceHint: &hint,
					Expression:        "true",
					Action:            sqlcv1.V1MatchConditionActionCANCEL,
				}, GroupMatchCondition{
					GroupId:           override.OrGroupID,
					EventType:         sqlcv1.V1EventTypeINTERNAL,
					EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
					ReadableDataKey:   parentReadableId,
					EventResourceHint: &hint,
					Expression:        "true",
					Action:            sqlcv1.V1MatchConditionActionCANCEL,
				})
		}
	} else {
		hint := parentExternalId.String()
		res = append(res, GroupMatchCondition{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCANCEL,
		}, GroupMatchCondition{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCANCEL,
		})
	}

	return res
}

func getChildWorkflowGroupMatches(taskExternalId uuid.UUID, stepReadableId string) []GroupMatchCondition {
	groupId := uuid.New()
	hint := taskExternalId.String()
	return []GroupMatchCondition{
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCREATE,
		},
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCREATE,
		},
		{
			GroupId:           groupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   stepReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionCREATE,
		},
	}
}

func getParentOnFailureGroupMatches(createGroupId, parentExternalId uuid.UUID, parentReadableId string) []GroupMatchCondition {
	cancelGroupId := uuid.New()
	hint := parentExternalId.String()
	return []GroupMatchCondition{
		{
			GroupId:           createGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionQUEUE,
		},
		{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		},
		{
			GroupId:           cancelGroupId,
			EventType:         sqlcv1.V1EventTypeINTERNAL,
			EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
			ReadableDataKey:   parentReadableId,
			EventResourceHint: &hint,
			Expression:        "true",
			Action:            sqlcv1.V1MatchConditionActionSKIP,
		},
	}
}

func orderSteps(steps []*sqlcv1.ListStepsByWorkflowVersionIdsRow) []*sqlcv1.ListStepsByWorkflowVersionIdsRow {
	slices.SortStableFunc(steps, func(i, j *sqlcv1.ListStepsByWorkflowVersionIdsRow) int {
		idA := i.ID.String()
		idB := j.ID.String()
		return strings.Compare(idA, idB)
	})

	return steps
}

func (r *sharedRepository) processWorkflowExpression(ctx context.Context, expression string, opt EventTriggerOpts, filterPayload []byte) (bool, error) {
	var inputData map[string]interface{}
	if opt.Data != nil {
		err := json.Unmarshal(opt.Data, &inputData)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal input data: %w", err)
		}
	} else {
		inputData = make(map[string]interface{})
	}

	var additionalMetadata map[string]interface{}
	if opt.AdditionalMetadata != nil {
		err := json.Unmarshal(opt.AdditionalMetadata, &additionalMetadata)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal additional metadata: %w", err)
		}
	} else {
		additionalMetadata = make(map[string]interface{})
	}

	payload := make(map[string]interface{})
	if filterPayload != nil {
		err := json.Unmarshal(filterPayload, &payload)

		if err != nil {
			return false, fmt.Errorf("failed to unmarshal filter payload: %w", err)
		}
	}

	match, err := r.celParser.EvaluateEventExpression(
		expression,
		cel.NewInput(
			cel.WithInput(inputData),
			cel.WithAdditionalMetadata(additionalMetadata),
			cel.WithPayload(payload),
			cel.WithEventID(opt.ExternalId),
			cel.WithEventKey(opt.Key),
		),
	)

	if err != nil {
		r.l.Warn().
			Err(err).
			Str("expression", expression).
			Msg("Failed to evaluate event expression")

		return false, err
	}

	return match, nil
}

func (r *sharedRepository) listWorkflowsByNames(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, names []string) ([]*sqlcv1.ListWorkflowsByNamesRow, error) {
	// lookup names in the cache
	workflowNamesToLookup := make([]string, 0)
	res := make([]*sqlcv1.ListWorkflowsByNamesRow, 0, len(names))

	for _, name := range names {
		k := fmt.Sprintf("%s:%s", tenantId, name)
		if value, ok := r.tenantIdWorkflowNameCache.Get(k); ok {
			res = append(res, value)
			continue
		}

		workflowNamesToLookup = append(workflowNamesToLookup, name)
	}

	// look up the workflow versions for the workflow names
	workflowVersions, err := r.queries.ListWorkflowsByNames(ctx, tx, sqlcv1.ListWorkflowsByNamesParams{
		Tenantid:      tenantId,
		Workflownames: workflowNamesToLookup,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workflows by names: %w", err)
	}

	for _, workflowVersion := range workflowVersions {
		// store in the cache
		k := fmt.Sprintf("%s:%s", tenantId, workflowVersion.WorkflowName)

		r.tenantIdWorkflowNameCache.Add(k, workflowVersion)

		res = append(res, workflowVersion)
	}

	return res, nil
}

func (r *sharedRepository) listStepsByWorkflowVersionIds(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, workflowVersionIds []uuid.UUID) (map[uuid.UUID][]*sqlcv1.ListStepsByWorkflowVersionIdsRow, error) {
	if len(workflowVersionIds) == 0 {
		return make(map[uuid.UUID][]*sqlcv1.ListStepsByWorkflowVersionIdsRow), nil
	}

	workflowVersionsToLookup := make([]uuid.UUID, 0, len(workflowVersionIds))
	res := make(map[uuid.UUID][]*sqlcv1.ListStepsByWorkflowVersionIdsRow)

	for _, id := range workflowVersionIds {
		if steps, found := r.stepsInWorkflowVersionCache.Get(id); found {
			res[id] = steps
			continue
		}

		workflowVersionsToLookup = append(workflowVersionsToLookup, id)
	}

	steps, err := r.queries.ListStepsByWorkflowVersionIds(ctx, tx, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Tenantid: tenantId,
		Ids:      workflowVersionsToLookup,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list steps by workflow version ids: %w", err)
	}

	for _, step := range steps {
		k := step.WorkflowVersionId
		res[k] = append(res[k], step)
	}

	// update the cache with all entries we looked up
	for _, id := range workflowVersionsToLookup {
		k := id

		if steps, ok := res[k]; ok {
			r.stepsInWorkflowVersionCache.Add(k, steps)
		}
	}

	return res, nil
}

func (r *sharedRepository) prepareTriggerFromEvents(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, opts []EventTriggerOpts) (
	[]triggerTuple,
	*createCoreUserEventOpts,
	map[uuid.UUID]EventExternalIdFilterId,
	[]CELEvaluationFailure,
	error,
) {
	eventKeysToOpts := make(map[string][]EventTriggerOpts)

	var createCoreEventOpts *createCoreUserEventOpts

	createCoreEventsTenantIds := []uuid.UUID{}
	createCoreEventsExternalIds := []uuid.UUID{}
	createCoreEventsSeenAts := []pgtype.Timestamptz{}
	createCoreEventsKeys := []string{}
	createCoreEventsAdditionalMetadatas := [][]byte{}
	createCoreEventsScopes := []pgtype.Text{}
	createCoreEventsTriggeringWebhookNames := []pgtype.Text{}

	eventExternalIdsToPayloads := make(map[uuid.UUID][]byte)

	seenAt := time.Now().UTC() // TODO: propagate this to caller, and figure out how we should be setting this

	eventKeys := make([]string, 0, len(opts))
	uniqueEventKeys := make(map[string]struct{})

	for _, opt := range opts {
		if r.enableDurableUserEventLog {
			createCoreEventsTenantIds = append(createCoreEventsTenantIds, tenantId)
			createCoreEventsExternalIds = append(createCoreEventsExternalIds, opt.ExternalId)
			createCoreEventsSeenAts = append(createCoreEventsSeenAts, sqlchelpers.TimestamptzFromTime(seenAt))
			createCoreEventsKeys = append(createCoreEventsKeys, opt.Key)
			eventExternalIdsToPayloads[opt.ExternalId] = opt.Data
			createCoreEventsAdditionalMetadatas = append(createCoreEventsAdditionalMetadatas, opt.AdditionalMetadata)
			if opt.Scope != nil {
				createCoreEventsScopes = append(createCoreEventsScopes, pgtype.Text{String: *opt.Scope, Valid: true})
			} else {
				createCoreEventsScopes = append(createCoreEventsScopes, pgtype.Text{Valid: false})
			}

			if opt.TriggeringWebhookName != nil {
				createCoreEventsTriggeringWebhookNames = append(createCoreEventsTriggeringWebhookNames, pgtype.Text{String: *opt.TriggeringWebhookName, Valid: true})
			} else {
				createCoreEventsTriggeringWebhookNames = append(createCoreEventsTriggeringWebhookNames, pgtype.Text{Valid: false})
			}
		}

		eventKeysToOpts[opt.Key] = append(eventKeysToOpts[opt.Key], opt)

		if _, ok := uniqueEventKeys[opt.Key]; ok {
			continue
		}

		uniqueEventKeys[opt.Key] = struct{}{}
		eventKeys = append(eventKeys, opt.Key)
	}

	workflowVersionIdsAndEventKeys, err := r.queries.ListWorkflowsForEvents(ctx, tx, sqlcv1.ListWorkflowsForEventsParams{
		Eventkeys: eventKeys,
		Tenantid:  tenantId,
	})

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to list workflows for events: %w", err)
	}

	externalIdToEventIdAndFilterId := make(map[uuid.UUID]EventExternalIdFilterId)

	workflowIdScopePairs := make(map[WorkflowAndScope]bool)

	// important: need to include all workflow ids here, regardless of whether or
	// not the corresponding event was pushed with a scope, so we can correctly
	// tell if there are any filters for the workflows with these events registered
	workflowIdsForFilterCounts := make([]uuid.UUID, 0, len(workflowVersionIdsAndEventKeys))

	for _, workflow := range workflowVersionIdsAndEventKeys {
		opts, ok := eventKeysToOpts[workflow.IncomingEventKey]

		if !ok {
			continue
		}

		workflowIdsForFilterCounts = append(workflowIdsForFilterCounts, workflow.WorkflowId)

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

	workflowIds := make([]uuid.UUID, 0, len(workflowIdScopePairs))
	scopes := make([]string, 0, len(workflowIdScopePairs))

	for pair := range workflowIdScopePairs {
		workflowIds = append(workflowIds, pair.WorkflowId)
		scopes = append(scopes, pair.Scope)
	}

	filters, err := r.queries.ListFiltersForEventTriggers(ctx, tx, sqlcv1.ListFiltersForEventTriggersParams{
		Tenantid:    tenantId,
		Workflowids: workflowIds,
		Scopes:      scopes,
	})

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to list filters: %w", err)
	}

	workflowIdAndScopeToFilters := make(map[WorkflowAndScope][]*sqlcv1.V1Filter)

	for _, filter := range filters {
		key := WorkflowAndScope{
			WorkflowId: filter.WorkflowID,
			Scope:      filter.Scope,
		}

		workflowIdAndScopeToFilters[key] = append(workflowIdAndScopeToFilters[key], filter)
	}

	filterCounts, err := r.queries.ListFilterCountsForWorkflows(ctx, tx, sqlcv1.ListFilterCountsForWorkflowsParams{
		Tenantid:    tenantId,
		Workflowids: workflowIdsForFilterCounts,
	})

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to list filter counts: %w", err)
	}

	workflowIdToCount := make(map[uuid.UUID]int64)

	for _, count := range filterCounts {
		workflowIdToCount[count.WorkflowID] = count.Count
	}

	// each (workflowVersionId, eventKey, opt) is a separate workflow that we need to create
	triggerOpts := make([]triggerTuple, 0)
	celEvaluationFailures := make([]CELEvaluationFailure, 0)

	for _, workflow := range workflowVersionIdsAndEventKeys {
		opts, ok := eventKeysToOpts[workflow.IncomingEventKey]

		if !ok {
			continue
		}

		numFilters := workflowIdToCount[workflow.WorkflowId]

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
				externalId := uuid.New()

				triggerOpts = append(triggerOpts, triggerTuple{
					workflowVersionId:  workflow.WorkflowVersionId,
					workflowId:         workflow.WorkflowId,
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

	if r.enableDurableUserEventLog {
		createCoreEventOpts = &createCoreUserEventOpts{
			params: sqlcv1.BulkCreateEventsParams{
				Tenantids:              createCoreEventsTenantIds,
				Externalids:            createCoreEventsExternalIds,
				Seenats:                createCoreEventsSeenAts,
				Keys:                   createCoreEventsKeys,
				Additionalmetadatas:    createCoreEventsAdditionalMetadatas,
				Scopes:                 createCoreEventsScopes,
				TriggeringWebhookNames: createCoreEventsTriggeringWebhookNames,
			},
			externalIdToEventIdAndFilterId: externalIdToEventIdAndFilterId,
			externalIdsToPayloads:          eventExternalIdsToPayloads,
		}
	}

	return triggerOpts, createCoreEventOpts, externalIdToEventIdAndFilterId, celEvaluationFailures, nil
}

func (r *sharedRepository) prepareTriggerFromWorkflowNames(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) (
	[]triggerTuple,
	[]string,
	[]string,
	error,
) {
	workflowNames := make([]string, 0, len(opts))
	uniqueNames := make(map[string]struct{})
	namesToOpts := make(map[string][]*WorkflowNameTriggerOpts)
	idempotencyKeyToExternalIds := make(map[IdempotencyKey]uuid.UUID)
	var err error
	var denyUpdateKeys []string
	var duplicateKeys []string

	for _, opt := range opts {
		if opt.IdempotencyKey != nil {
			idempotencyKeyToExternalIds[*opt.IdempotencyKey] = opt.ExternalId
		}

		namesToOpts[opt.WorkflowName] = append(namesToOpts[opt.WorkflowName], opt)

		if _, ok := uniqueNames[opt.WorkflowName]; ok {
			continue
		}

		uniqueNames[opt.WorkflowName] = struct{}{}
		workflowNames = append(workflowNames, opt.WorkflowName)
	}

	keyClaimantPairs := make([]KeyClaimantPair, 0, len(idempotencyKeyToExternalIds))

	for idempotencyKey, runExternalId := range idempotencyKeyToExternalIds {
		keyClaimantPairs = append(keyClaimantPairs, KeyClaimantPair{
			IdempotencyKey:      idempotencyKey,
			ClaimedByExternalId: runExternalId,
		})
	}

	keyClaimantPairToWasClaimed := make(map[KeyClaimantPair]WasSuccessfullyClaimed)

	if len(keyClaimantPairs) > 0 {
		keys := make([]string, 0, len(keyClaimantPairs))

		for _, pair := range keyClaimantPairs {
			keys = append(keys, string(pair.IdempotencyKey))
		}

		ttl := r.idempotencyKeyTTL
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		expiresAt := sqlchelpers.TimestamptzFromTime(time.Now().Add(ttl))

		err = r.queries.CreateIdempotencyKeys(ctx, tx, sqlcv1.CreateIdempotencyKeysParams{
			Tenantid:  tenantId,
			Keys:      keys,
			Expiresat: expiresAt,
		})

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create idempotency keys: %w", err)
		}

		keyClaimantPairToWasClaimed, err = claimIdempotencyKeys(ctx, r.queries, tx, tenantId, keyClaimantPairs)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to claim idempotency keys: %w", err)
		}
	}

	if len(keyClaimantPairs) > 0 {
		unclaimedKeys := make([]string, 0)

		for _, pair := range keyClaimantPairs {
			if !keyClaimantPairToWasClaimed[pair] {
				unclaimedKeys = append(unclaimedKeys, string(pair.IdempotencyKey))
			}
		}

		if len(unclaimedKeys) > 0 {
			denyUpdateKeys, err = r.tryReclaimIdempotencyKeys(ctx, tx, tenantId, keyClaimantPairs, keyClaimantPairToWasClaimed, unclaimedKeys)

			if err != nil {
				return nil, nil, nil, err
			}

			unclaimedKeys = unclaimedKeys[:0]

			for _, pair := range keyClaimantPairs {
				if !keyClaimantPairToWasClaimed[pair] {
					unclaimedKeys = append(unclaimedKeys, string(pair.IdempotencyKey))
				}
			}

			if len(unclaimedKeys) > 0 {
				duplicateKeys = append(duplicateKeys, unclaimedKeys...)
			}
		}
	}

	workflowVersionsByNames, err := r.listWorkflowsByNames(ctx, tx, tenantId, workflowNames)

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
			if opt.IdempotencyKey != nil {
				keyClaimantPair := KeyClaimantPair{
					IdempotencyKey:      *opt.IdempotencyKey,
					ClaimedByExternalId: opt.ExternalId,
				}

				wasSuccessfullyClaimed := keyClaimantPairToWasClaimed[keyClaimantPair]

				// if we did not successfully claim the idempotency key, we should not trigger the workflow
				if !wasSuccessfullyClaimed {
					continue
				}
			}

			triggerOpts = append(triggerOpts, triggerTuple{
				workflowVersionId:    workflowVersion.WorkflowVersionId,
				workflowId:           workflowVersion.WorkflowId,
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

	if len(triggerOpts) == 0 && len(duplicateKeys) > 0 {
		return nil, denyUpdateKeys, duplicateKeys, &IdempotencyKeyAlreadyClaimedError{Keys: duplicateKeys}
	}

	return triggerOpts, denyUpdateKeys, duplicateKeys, nil
}

func (r *sharedRepository) tryReclaimIdempotencyKeys(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	keyClaimantPairs []KeyClaimantPair,
	keyClaimantPairToWasClaimed map[KeyClaimantPair]WasSuccessfullyClaimed,
	unclaimedKeys []string,
) ([]string, error) {
	if len(unclaimedKeys) == 0 {
		return nil, nil
	}

	if r.idempotencyKeyDenyRecheckInterval <= 0 {
		return nil, nil
	}

	rows, err := r.queries.ListIdempotencyKeysByKeys(ctx, r.pool, sqlcv1.ListIdempotencyKeysByKeysParams{
		Tenantid: tenantId,
		Keys:     unclaimedKeys,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list idempotency keys for recheck: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	now := time.Now()
	throttledCount := 0
	keysToCheck := make([]string, 0, len(rows))
	keysToReclaim := make([]string, 0)
	keysToReclaimSet := make(map[string]struct{})
	keyToExternalId := make(map[string]uuid.UUID)
	externalIdsToCheck := make(map[uuid.UUID]struct{})

	for _, row := range rows {
		if row.ClaimedByExternalID == nil {
			continue
		}

		if row.LastDeniedAt.Valid {
			if now.Sub(row.LastDeniedAt.Time) < r.idempotencyKeyDenyRecheckInterval {
				throttledCount++
				continue
			}
		}

		keysToCheck = append(keysToCheck, row.Key)
		keyToExternalId[row.Key] = *row.ClaimedByExternalID
		externalIdsToCheck[*row.ClaimedByExternalID] = struct{}{}
	}

	if len(keysToCheck) == 0 {
		if throttledCount > 0 {
			r.l.Debug().
				Str("tenantId", tenantId.String()).
				Int("throttledCount", throttledCount).
				Msg("idempotency recheck throttled")
		}
		return nil, nil
	}

	if throttledCount > 0 {
		r.l.Debug().
			Str("tenantId", tenantId.String()).
			Int("throttledCount", throttledCount).
			Msg("idempotency recheck throttled")
	}

	externalIdToStatus := make(map[uuid.UUID]sqlcv1.V1ReadableStatusOlap)

	externalIds := make([]uuid.UUID, 0, len(externalIdsToCheck))
	for externalId := range externalIdsToCheck {
		externalIds = append(externalIds, externalId)
	}

	statusRows, statusErr := r.queries.ReadWorkflowRunStatusesByExternalIds(ctx, r.pool, sqlcv1.ReadWorkflowRunStatusesByExternalIdsParams{
		Workflowrunexternalids: externalIds,
		Tenantid:               tenantId,
	})
	if statusErr != nil {
		r.l.Error().
			Err(statusErr).
			Str("tenantId", tenantId.String()).
			Int("externalIdCount", len(externalIds)).
			Msg("failed to read workflow run status for idempotency recheck")
		return nil, fmt.Errorf("failed to read workflow run status for idempotency recheck: %w", statusErr)
	}

	for _, row := range statusRows {
		externalIdToStatus[row.ExternalID] = row.ReadableStatus
	}

	missingExternalIds := make([]uuid.UUID, 0)
	for externalId := range externalIdsToCheck {
		if _, ok := externalIdToStatus[externalId]; !ok {
			missingExternalIds = append(missingExternalIds, externalId)
		}
	}

	terminalFallback := make(map[uuid.UUID]struct{})
	if len(missingExternalIds) > 0 {
		fallbackRows, fallbackErr := r.queries.ReadWorkflowRunTerminalStatesByExternalIds(ctx, r.pool, sqlcv1.ReadWorkflowRunTerminalStatesByExternalIdsParams{
			Externalids: missingExternalIds,
			Tenantid:    tenantId,
		})
		if fallbackErr != nil {
			r.l.Error().
				Err(fallbackErr).
				Str("tenantId", tenantId.String()).
				Int("missingOlapCount", len(missingExternalIds)).
				Msg("failed to read workflow run status from core tables for idempotency recheck")
		} else {
			for _, row := range fallbackRows {
				if row.TaskCount > 0 && row.AllTerminal {
					terminalFallback[row.ExternalID] = struct{}{}
				}
			}
		}
	}

	missingDeniedCount := 0

	for _, key := range keysToCheck {
		externalId, ok := keyToExternalId[key]
		if !ok {
			continue
		}

		status, ok := externalIdToStatus[externalId]
		terminal := ok && isTerminalReadableStatus(status)
		if !ok {
			if _, ok := terminalFallback[externalId]; ok {
				terminal = true
			} else {
				missingDeniedCount++
			}
		}

		if terminal {
			deleteErr := r.queries.DeleteIdempotencyKeysByExternalId(ctx, tx, sqlcv1.DeleteIdempotencyKeysByExternalIdParams{
				Tenantid:   tenantId,
				Externalid: externalId,
			})
			if deleteErr != nil {
				return nil, fmt.Errorf("failed to delete idempotency keys for completed workflow run: %w", deleteErr)
			}

			keysToReclaim = append(keysToReclaim, key)
			keysToReclaimSet[key] = struct{}{}
		}
	}

	if missingDeniedCount > 0 {
		r.l.Warn().
			Str("tenantId", tenantId.String()).
			Int("missingStatusDeniedCount", missingDeniedCount).
			Msg("idempotency recheck denied due to missing workflow run status")
	}

	keysToUpdate := keysToCheck
	if len(keysToReclaimSet) > 0 {
		keysToUpdate = make([]string, 0, len(keysToCheck))
		for _, key := range keysToCheck {
			if _, ok := keysToReclaimSet[key]; !ok {
				keysToUpdate = append(keysToUpdate, key)
			}
		}
	}

	if len(keysToReclaim) == 0 {
		return keysToUpdate, nil
	}

	ttl := r.idempotencyKeyTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	expiresAt := sqlchelpers.TimestamptzFromTime(time.Now().Add(ttl))

	createErr := r.queries.CreateIdempotencyKeys(ctx, tx, sqlcv1.CreateIdempotencyKeysParams{
		Tenantid:  tenantId,
		Keys:      keysToReclaim,
		Expiresat: expiresAt,
	})
	if createErr != nil {
		return nil, fmt.Errorf("failed to recreate idempotency keys after reclaim: %w", createErr)
	}

	pairsToReclaim := make([]KeyClaimantPair, 0, len(keysToReclaim))
	for _, pair := range keyClaimantPairs {
		if _, ok := keysToReclaimSet[string(pair.IdempotencyKey)]; ok {
			pairsToReclaim = append(pairsToReclaim, pair)
		}
	}

	if len(pairsToReclaim) == 0 {
		return keysToUpdate, nil
	}

	claimResults, err := claimIdempotencyKeys(ctx, r.queries, tx, tenantId, pairsToReclaim)
	if err != nil {
		return nil, fmt.Errorf("failed to reclaim idempotency keys: %w", err)
	}

	for pair, claimed := range claimResults {
		keyClaimantPairToWasClaimed[pair] = claimed
	}

	return keysToUpdate, nil
}
