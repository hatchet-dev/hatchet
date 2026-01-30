package repository

import (
	"context"
	"encoding/json"
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
	ExternalId string

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

	// (optional) the priority of the task
	Priority *int32 `json:"priority"`
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
	TriggerFromEvents(ctx context.Context, tenantId string, opts []EventTriggerOpts) (*TriggerFromEventsResult, error)

	TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) ([]*V1TaskWithPayload, []*DAGWithData, error)

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

type Run struct {
	Id         int64
	InsertedAt time.Time
	FilterId   *string
}

type TriggerFromEventsResult struct {
	Tasks                 []*V1TaskWithPayload
	Dags                  []*DAGWithData
	EventExternalIdToRuns map[string][]*Run
	CELEvaluationFailures []CELEvaluationFailure
}

type TriggerDecision struct {
	ShouldTrigger bool
	FilterPayload []byte
	FilterId      *string
}

func (r *TriggerRepositoryImpl) makeTriggerDecisions(ctx context.Context, filters []*sqlcv1.V1Filter, hasAnyFilters bool, opt EventTriggerOpts) ([]TriggerDecision, []CELEvaluationFailure) {
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

		filterId := filter.ID.String()

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
	ExternalId string
	FilterId   *string
}

type WorkflowAndScope struct {
	WorkflowId uuid.UUID
	Scope      string
}

func (r *TriggerRepositoryImpl) TriggerFromEvents(ctx context.Context, tenantId string, opts []EventTriggerOpts) (*TriggerFromEventsResult, error) {
	pre, post := r.m.Meter(ctx, sqlcv1.LimitResourceEVENT, tenantId, int32(len(opts))) // nolint: gosec

	if err := pre(); err != nil {
		return nil, err
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

	// we don't run this in a transaction because workflow versions won't change during the course of this operation
	workflowVersionIdsAndEventKeys, err := r.queries.ListWorkflowsForEvents(ctx, r.pool, sqlcv1.ListWorkflowsForEventsParams{
		Eventkeys: eventKeys,
		Tenantid:  uuid.MustParse(tenantId),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workflows for events: %w", err)
	}

	externalIdToEventIdAndFilterId := make(map[string]EventExternalIdFilterId)

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

	filters, err := r.queries.ListFiltersForEventTriggers(ctx, r.pool, sqlcv1.ListFiltersForEventTriggersParams{
		Tenantid:    uuid.MustParse(tenantId),
		Workflowids: workflowIds,
		Scopes:      scopes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list filters: %w", err)
	}

	workflowIdAndScopeToFilters := make(map[WorkflowAndScope][]*sqlcv1.V1Filter)

	for _, filter := range filters {
		key := WorkflowAndScope{
			WorkflowId: filter.WorkflowID,
			Scope:      filter.Scope,
		}

		workflowIdAndScopeToFilters[key] = append(workflowIdAndScopeToFilters[key], filter)
	}

	filterCounts, err := r.queries.ListFilterCountsForWorkflows(ctx, r.pool, sqlcv1.ListFilterCountsForWorkflowsParams{
		Tenantid:    uuid.MustParse(tenantId),
		Workflowids: workflowIdsForFilterCounts,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list filter counts: %w", err)
	}

	workflowIdToCount := make(map[string]int64)

	for _, count := range filterCounts {
		workflowIdToCount[count.WorkflowID.String()] = count.Count
	}

	// each (workflowVersionId, eventKey, opt) is a separate workflow that we need to create
	triggerOpts := make([]triggerTuple, 0)
	celEvaluationFailures := make([]CELEvaluationFailure, 0)

	for _, workflow := range workflowVersionIdsAndEventKeys {
		opts, ok := eventKeysToOpts[workflow.IncomingEventKey]

		if !ok {
			continue
		}

		numFilters := workflowIdToCount[workflow.WorkflowId.String()]

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
					workflowVersionId:  workflow.WorkflowVersionId.String(),
					workflowId:         workflow.WorkflowId.String(),
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

	tasks, dags, err := r.triggerWorkflows(ctx, tenantId, triggerOpts)

	if err != nil {
		return nil, fmt.Errorf("failed to trigger workflows: %w", err)
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

	post()

	return &TriggerFromEventsResult{
		Tasks:                 tasks,
		Dags:                  dags,
		EventExternalIdToRuns: eventExternalIdToRuns,
		CELEvaluationFailures: celEvaluationFailures,
	}, nil
}

func (r *TriggerRepositoryImpl) TriggerFromWorkflowNames(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) ([]*V1TaskWithPayload, []*DAGWithData, error) {
	workflowNames := make([]string, 0, len(opts))
	uniqueNames := make(map[string]struct{})
	namesToOpts := make(map[string][]*WorkflowNameTriggerOpts)
	idempotencyKeyToExternalIds := make(map[IdempotencyKey]uuid.UUID)

	for _, opt := range opts {
		if opt.IdempotencyKey != nil {
			idempotencyKeyToExternalIds[*opt.IdempotencyKey] = uuid.MustParse(opt.ExternalId)
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

	keyClaimantPairToWasClaimed, err := claimIdempotencyKeys(ctx, r.queries, r.pool, tenantId, keyClaimantPairs)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to claim idempotency keys: %w", err)
	}

	// we don't run this in a transaction because workflow versions won't change during the course of this operation
	workflowVersionsByNames, err := r.queries.ListWorkflowsByNames(ctx, r.pool, sqlcv1.ListWorkflowsByNamesParams{
		Tenantid:      uuid.MustParse(tenantId),
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
			if opt.IdempotencyKey != nil {
				keyClaimantPair := KeyClaimantPair{
					IdempotencyKey:      *opt.IdempotencyKey,
					ClaimedByExternalId: uuid.MustParse(opt.ExternalId),
				}

				wasSuccessfullyClaimed := keyClaimantPairToWasClaimed[keyClaimantPair]

				// if we did not successfully claim the idempotency key, we should not trigger the workflow
				if !wasSuccessfullyClaimed {
					continue
				}
			}

			triggerOpts = append(triggerOpts, triggerTuple{
				workflowVersionId:    workflowVersion.WorkflowVersionId.String(),
				workflowId:           workflowVersion.WorkflowId.String(),
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
		Tenantid:      uuid.MustParse(tenantId),
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

type TriggeredBy interface {
	ToMetadata([]byte) []byte
}

type TriggeredByEvent struct {
	l        *zerolog.Logger
	eventID  string
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
	workflowVersionId string

	workflowId string

	workflowName string

	externalId string

	input []byte

	filterPayload []byte

	additionalMetadata []byte

	desiredWorkerId *string

	priority *int32

	// relevant parameters for child workflows
	parentExternalId     *string
	parentTaskId         *int64
	parentTaskInsertedAt *time.Time
	childIndex           *int64
	childKey             *string
}

func (r *TriggerRepositoryImpl) triggerWorkflows(ctx context.Context, tenantId string, tuples []triggerTuple) ([]*V1TaskWithPayload, []*DAGWithData, error) {
	// get unique workflow version ids
	uniqueWorkflowVersionIds := make(map[string]struct{})

	for _, tuple := range tuples {
		uniqueWorkflowVersionIds[tuple.workflowVersionId] = struct{}{}
	}

	// get all data for triggering tasks in this workflow
	workflowVersionIds := make([]uuid.UUID, 0, len(uniqueWorkflowVersionIds))

	for id := range uniqueWorkflowVersionIds {
		workflowVersionIds = append(workflowVersionIds, uuid.MustParse(id))
	}

	// get steps for the workflow versions
	steps, err := r.queries.ListStepsByWorkflowVersionIds(ctx, r.pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      workflowVersionIds,
		Tenantid: uuid.MustParse(tenantId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get workflow versions for engine: %w", err)
	}

	// group steps by workflow version ids
	workflowVersionToSteps := make(map[string][]*sqlcv1.ListStepsByWorkflowVersionIdsRow)
	stepIdsToReadableIds := make(map[string]string)

	for _, step := range steps {
		workflowVersionId := step.WorkflowVersionId.String()

		workflowVersionToSteps[workflowVersionId] = append(workflowVersionToSteps[workflowVersionId], step)

		stepIdsToReadableIds[step.ID.String()] = step.ReadableId.String
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

	// if any steps have additional match conditions, query for the additional matches
	stepsWithAdditionalMatchConditions := make([]uuid.UUID, 0)

	for _, step := range steps {
		if step.MatchConditionCount > 0 {
			stepsWithAdditionalMatchConditions = append(stepsWithAdditionalMatchConditions, step.ID)
		}
	}

	stepsToAdditionalMatches := make(map[string][]*sqlcv1.V1StepMatchCondition)

	if len(stepsWithAdditionalMatchConditions) > 0 {
		additionalMatches, err := r.queries.ListStepMatchConditions(ctx, r.pool, sqlcv1.ListStepMatchConditionsParams{
			Stepids:  stepsWithAdditionalMatchConditions,
			Tenantid: uuid.MustParse(tenantId),
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to list step match conditions: %w", err)
		}

		for _, match := range additionalMatches {
			stepId := match.StepID.String()

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
				stepsToExternalIds[i][step.ID.String()] = tuple.externalId
			} else {
				externalId := uuid.NewString()
				stepsToExternalIds[i][step.ID.String()] = externalId
				dagToTaskIds[tuple.externalId] = append(dagToTaskIds[tuple.externalId], externalId)
			}
		}
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

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
			stepId := step.ID.String()
			taskExternalId := stepsToExternalIds[i][stepId]

			// if this is an on failure step, create match conditions for every other step in the DAG
			switch {
			case step.JobKind == sqlcv1.JobKindONFAILURE:
				conditions := make([]GroupMatchCondition, 0)
				groupId := uuid.NewString()

				for _, otherStep := range steps {
					if otherStep.ID.String() == stepId {
						continue
					}

					otherExternalId := stepsToExternalIds[i][otherStep.ID.String()]
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
					parsed := uuid.MustParse(*tuple.parentExternalId)
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
								condition.OrGroupID.String(),
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
								condition.OrGroupID.String(),
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
						parsed := uuid.MustParse(*tuple.parentExternalId)
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
						StepId:               step.ID.String(),
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

				cancelGroupId := uuid.NewString()

				additionalMatches, ok := stepsToAdditionalMatches[stepId]

				if !ok {
					additionalMatches = make([]*sqlcv1.V1StepMatchCondition, 0)
				}

				for _, parent := range step.Parents {
					parentExternalId := stepsToExternalIds[i][parent.String()]
					readableId := stepIdsToReadableIds[parent.String()]

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
					parsed := uuid.MustParse(*tuple.parentExternalId)
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
		opts, ok := dagTaskOpts[dag.ExternalID.String()]

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
		opts := eventMatches[dag.ExternalID.String()]

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

	err = r.payloadStore.Store(ctx, tx, storePayloadOpts...)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to store payloads: %w", err)
	}

	// commit
	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	postTask()

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

func (r *TriggerRepositoryImpl) createDAGs(ctx context.Context, tx sqlcv1.DBTX, tenantId string, opts []createDAGOpts) ([]*DAGWithData, error) {
	if len(opts) == 0 {
		return nil, nil
	}

	tenantIds := make([]uuid.UUID, 0, len(opts))
	externalIds := make([]uuid.UUID, 0, len(opts))
	displayNames := make([]string, 0, len(opts))
	workflowIds := make([]uuid.UUID, 0, len(opts))
	workflowVersionIds := make([]uuid.UUID, 0, len(opts))
	parentTaskExternalIds := make([]uuid.UUID, 0, len(opts))
	dagIdToOpt := make(map[string]createDAGOpts, 0)

	unix := time.Now().UnixMilli()

	for _, opt := range opts {
		tenantIds = append(tenantIds, uuid.MustParse(tenantId))
		externalIds = append(externalIds, uuid.MustParse(opt.ExternalId))
		displayNames = append(displayNames, fmt.Sprintf("%s-%d", opt.WorkflowName, unix))
		workflowIds = append(workflowIds, uuid.MustParse(opt.WorkflowId))
		workflowVersionIds = append(workflowVersionIds, uuid.MustParse(opt.WorkflowVersionId))

		if opt.ParentTaskExternalID == nil {
			parentTaskExternalIds = append(parentTaskExternalIds, uuid.UUID{})
		} else {
			parentTaskExternalIds = append(parentTaskExternalIds, uuid.MustParse(*opt.ParentTaskExternalID))
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
		externalId := dag.ExternalID.String()
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
			parentTaskExternalID = uuid.MustParse(*opt.ParentTaskExternalID)
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
			stepId := step.ID.String()
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
			Tenantid:        uuid.MustParse(tenantId),
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
			TenantId:   uuid.MustParse(tenantId),
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
			TenantId:   uuid.MustParse(tenantId),
		}]

		if !ok {
			payload = event.Data
		}

		c, err := newChildWorkflowSignalCreatedDataFromBytes(payload)

		if err != nil {
			r.l.Error().Msgf("failed to unmarshal child workflow signal created data: %s", err)
			continue
		}

		if c.ChildExternalId != "" {
			rootExternalIdsToLookup = append(rootExternalIdsToLookup, uuid.MustParse(c.ChildExternalId))
		}
	}

	// get the child external IDs that have already been written
	existingExternalIds, err := r.queries.LookupExternalIds(ctx, tx, sqlcv1.LookupExternalIdsParams{
		Tenantid:    uuid.MustParse(tenantId),
		Externalids: rootExternalIdsToLookup,
	})

	if err != nil {
		return nil, err
	}

	tuplesToSkip = make(map[string]struct{})

	for _, dbExternalId := range existingExternalIds {
		tuplesToSkip[dbExternalId.ExternalID.String()] = struct{}{}
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
				stepId := step.ID.String()
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
			res = append(res, GroupMatchCondition{
				GroupId:           override.OrGroupID.String(),
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
				GroupId:           override.OrGroupID.String(),
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
				ReadableDataKey:   parentReadableId,
				EventResourceHint: &parentExternalId,
				Expression:        override.Expression.String,
				Action:            sqlcv1.V1MatchConditionActionSKIP,
			})
		}
	} else if !hasAnySkippingParentOverrides {
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
			res = append(res,
				GroupMatchCondition{
					GroupId:   override.OrGroupID.String(),
					EventType: sqlcv1.V1EventTypeINTERNAL,
					// The custom cancel condition matches on the completed event
					EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
					ReadableDataKey:   parentReadableId,
					EventResourceHint: &parentExternalId,
					Expression:        override.Expression.String,
					Action:            sqlcv1.V1MatchConditionActionCANCEL,
				},
				// always add the original cancel group match conditions. these can't be modified otherwise DAGs risk
				// getting stuck in a concurrency queue.
				GroupMatchCondition{
					GroupId:           override.OrGroupID.String(),
					EventType:         sqlcv1.V1EventTypeINTERNAL,
					EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
					ReadableDataKey:   parentReadableId,
					EventResourceHint: &parentExternalId,
					Expression:        "true",
					Action:            sqlcv1.V1MatchConditionActionCANCEL,
				}, GroupMatchCondition{
					GroupId:           override.OrGroupID.String(),
					EventType:         sqlcv1.V1EventTypeINTERNAL,
					EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
					ReadableDataKey:   parentReadableId,
					EventResourceHint: &parentExternalId,
					Expression:        "true",
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
