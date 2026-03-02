package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel/attribute"

	"github.com/google/cel-go/cel"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type CandidateEventMatch struct {
	// A UUID for the event
	ID uuid.UUID

	// A timestamp for the event
	EventTimestamp time.Time

	// Key for the event
	Key string

	// Resource hint for the event
	ResourceHint *string

	// Data for the event
	Data []byte
}

type ExternalCreateSignalMatchOpts struct {
	Conditions []CreateExternalSignalConditionOpt `validate:"required,min=1,dive"`

	SignalTaskId int64 `validate:"required,gt=0"`

	SignalTaskInsertedAt pgtype.Timestamptz

	SignalExternalId uuid.UUID `validate:"required"`

	SignalKey string `validate:"required"`
}

type CreateExternalSignalConditionKind string

const (
	CreateExternalSignalConditionKindSLEEP     CreateExternalSignalConditionKind = "SLEEP"
	CreateExternalSignalConditionKindUSEREVENT CreateExternalSignalConditionKind = "USER_EVENT"
)

type CreateExternalSignalConditionOpt struct {
	Kind CreateExternalSignalConditionKind `validate:"required, oneof=SLEEP USER_EVENT"`

	ReadableDataKey string `validate:"required"`

	OrGroupId uuid.UUID `validate:"required"`

	UserEventKey *string

	SleepFor *string `validate:"omitempty,duration"`

	Expression string
}

type CreateMatchOpts struct {
	Kind sqlcv1.V1MatchKind

	ExistingMatchData []byte

	Conditions []GroupMatchCondition

	TriggerDAGId *int64

	TriggerDAGInsertedAt pgtype.Timestamptz

	TriggerExternalId *uuid.UUID

	TriggerWorkflowRunId *uuid.UUID

	TriggerStepId *uuid.UUID

	TriggerStepIndex pgtype.Int8

	TriggerExistingTaskId *int64

	TriggerExistingTaskInsertedAt pgtype.Timestamptz

	TriggerParentTaskExternalId *uuid.UUID

	TriggerParentTaskId pgtype.Int8

	TriggerParentTaskInsertedAt pgtype.Timestamptz

	TriggerChildIndex pgtype.Int8

	TriggerChildKey pgtype.Text

	TriggerPriority pgtype.Int4

	SignalTaskId *int64

	SignalTaskInsertedAt pgtype.Timestamptz

	SignalExternalId *uuid.UUID

	SignalKey *string
}

type EventMatchResults struct {
	// The list of tasks which were created from the matches
	CreatedTasks []*V1TaskWithPayload

	// The list of tasks which were replayed from the matches
	ReplayedTasks []*V1TaskWithPayload
}

type GroupMatchCondition struct {
	GroupId uuid.UUID `validate:"required"`

	EventType sqlcv1.V1EventType

	EventKey string

	// (optional) a hint for querying the event data
	EventResourceHint *string

	// the data key which gets inserted into the returned data from a satisfied match condition
	ReadableDataKey string

	Expression string

	Action sqlcv1.V1MatchConditionAction

	// (optional) the data which was used to satisfy the condition (relevant for replays)
	Data []byte
}

type MatchRepository interface {
	RegisterSignalMatchConditions(ctx context.Context, tenantId uuid.UUID, eventMatches []ExternalCreateSignalMatchOpts) error

	ProcessUserEventMatches(ctx context.Context, tenantId uuid.UUID, events []CandidateEventMatch) (*EventMatchResults, error)
	ProcessInternalEventMatches(ctx context.Context, tenantId uuid.UUID, events []CandidateEventMatch) (*EventMatchResults, error)
}

type MatchRepositoryImpl struct {
	*sharedRepository
}

func newMatchRepository(s *sharedRepository) MatchRepository {
	return &MatchRepositoryImpl{
		sharedRepository: s,
	}
}

func (m *MatchRepositoryImpl) RegisterSignalMatchConditions(ctx context.Context, tenantId uuid.UUID, signalMatches []ExternalCreateSignalMatchOpts) error {
	// TODO: ADD BACK VALIDATION
	// if err := m.v.Validate(signalMatches); err != nil {
	// 	return err
	// }

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, m.pool, m.l)

	if err != nil {
		return err
	}

	defer rollback()

	eventMatches := make([]CreateMatchOpts, 0, len(signalMatches))

	for _, signalMatch := range signalMatches {
		conditions := make([]GroupMatchCondition, 0, len(signalMatch.Conditions))

		for _, condition := range signalMatch.Conditions {
			switch condition.Kind {
			case CreateExternalSignalConditionKindSLEEP:
				if condition.SleepFor == nil {
					return fmt.Errorf("sleep condition requires a duration")
				}

				c, err := m.durableSleepCondition(
					ctx,
					tx,
					tenantId,
					condition.OrGroupId,
					condition.ReadableDataKey,
					*condition.SleepFor,
					sqlcv1.V1MatchConditionActionCREATE,
				)

				if err != nil {
					return err
				}

				conditions = append(conditions, *c)
			case CreateExternalSignalConditionKindUSEREVENT:
				if condition.UserEventKey == nil {
					return fmt.Errorf("user event condition requires a user event key")
				}

				conditions = append(conditions, m.userEventCondition(
					condition.OrGroupId,
					condition.ReadableDataKey,
					*condition.UserEventKey,
					condition.Expression,
					sqlcv1.V1MatchConditionActionCREATE,
				))
			}
		}

		taskId := signalMatch.SignalTaskId
		externalId := signalMatch.SignalExternalId
		signalKey := signalMatch.SignalKey

		eventMatches = append(eventMatches, CreateMatchOpts{
			Kind:                 sqlcv1.V1MatchKindSIGNAL,
			Conditions:           conditions,
			SignalTaskId:         &taskId,
			SignalTaskInsertedAt: signalMatch.SignalTaskInsertedAt,
			SignalExternalId:     &externalId,
			SignalKey:            &signalKey,
		})
	}

	err = m.createEventMatches(ctx, tx, tenantId, eventMatches)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

// ProcessInternalEventMatches processes a list of internal events
func (m *MatchRepositoryImpl) ProcessInternalEventMatches(ctx context.Context, tenantId uuid.UUID, events []CandidateEventMatch) (*EventMatchResults, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, m.pool, m.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	res, err := m.processEventMatches(ctx, tx, tenantId, events, sqlcv1.V1EventTypeINTERNAL)

	if err != nil {
		return nil, err
	}

	storePayloadOpts := make([]StorePayloadOpts, len(res.CreatedTasks))

	for i, task := range res.CreatedTasks {
		storePayloadOpts[i] = StorePayloadOpts{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			ExternalId: task.ExternalID,
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			Payload:    task.Payload,
			TenantId:   task.TenantID,
		}
	}

	if len(storePayloadOpts) > 0 {
		err = m.payloadStore.Store(ctx, tx, storePayloadOpts...)

		if err != nil {
			return nil, fmt.Errorf("failed to store payloads for created tasks for internal event matches: %w", err)
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return res, nil
}

// ProcessUserEventMatches processes a list of user events
func (m *MatchRepositoryImpl) ProcessUserEventMatches(ctx context.Context, tenantId uuid.UUID, events []CandidateEventMatch) (*EventMatchResults, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, m.pool, m.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	res, err := m.processEventMatches(ctx, tx, tenantId, events, sqlcv1.V1EventTypeUSER)

	if err != nil {
		return nil, err
	}

	storePayloadOpts := make([]StorePayloadOpts, len(res.CreatedTasks))
	for i, task := range res.CreatedTasks {
		storePayloadOpts[i] = StorePayloadOpts{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			ExternalId: task.ExternalID,
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			Payload:    task.Payload,
			TenantId:   task.TenantID,
		}
	}

	if len(storePayloadOpts) > 0 {
		err = m.payloadStore.Store(ctx, tx, storePayloadOpts...)

		if err != nil {
			return nil, fmt.Errorf("failed to store payloads for created tasks for user event matches: %w", err)
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return res, nil
}

func (m *sharedRepository) processEventMatches(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, events []CandidateEventMatch, eventType sqlcv1.V1EventType) (*EventMatchResults, error) {
	start := time.Now()

	res := &EventMatchResults{}

	eventKeysWithHints := make([]string, 0, len(events))
	eventKeysWithoutHints := make([]string, 0, len(events))
	resourceHints := make([]string, 0, len(events))
	uniqueEventKeys := make(map[string]struct{})
	idsToEvents := make(map[uuid.UUID]CandidateEventMatch)

	for _, event := range events {
		idsToEvents[event.ID] = event

		if event.ResourceHint == nil {
			if _, ok := uniqueEventKeys[event.Key]; ok {
				continue
			}
		}

		uniqueEventKeys[event.Key] = struct{}{}

		if event.ResourceHint != nil {
			eventKeysWithHints = append(eventKeysWithHints, event.Key)
			resourceHints = append(resourceHints, *event.ResourceHint)
		} else {
			eventKeysWithoutHints = append(eventKeysWithoutHints, event.Key)
		}
	}

	var matchConditions []*sqlcv1.ListMatchConditionsForEventRow

	if len(eventKeysWithHints) > 0 {
		matchConditionsWithHints, err := m.queries.ListMatchConditionsForEventWithHint(
			ctx,
			tx,
			sqlcv1.ListMatchConditionsForEventWithHintParams{
				Tenantid:           tenantId,
				Eventtype:          eventType,
				Eventkeys:          eventKeysWithHints,
				Eventresourcehints: resourceHints,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("failed to list match conditions with hints for event: %w", err)
		}

		matchConditions = append(matchConditions, matchConditionsWithHints...)
	}

	if len(eventKeysWithoutHints) > 0 {
		matchConditionsWithoutHints, err := m.queries.ListMatchConditionsForEventWithoutHint(
			ctx,
			tx,
			sqlcv1.ListMatchConditionsForEventWithoutHintParams{
				Tenantid:  tenantId,
				Eventtype: eventType,
				Eventkeys: eventKeysWithoutHints,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("failed to list match conditions without hints for event: %w", err)
		}

		matchConditions = append(matchConditions, matchConditionsWithoutHints...)
	}

	// pass match conditions through CEL expressions parser
	matches, err := m.processCELExpressions(ctx, events, matchConditions, eventType)

	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return res, nil
	}

	matchIds := make([]int64, 0, len(matches))
	conditionIds := make([]int64, 0, len(matches))
	datas := make([][]byte, 0, len(matches))

	for eventId, conditions := range matches {
		for _, condition := range conditions {
			event, ok := idsToEvents[eventId]

			if !ok {
				m.l.Error().Msgf("event with id %s not found", eventId)
				continue
			}

			matchIds = append(matchIds, condition.V1MatchID)
			conditionIds = append(conditionIds, condition.ID)

			datas = append(datas, event.Data)
		}
	}

	// update condition rows in the database
	satisfiedMatchIds, err := m.queries.GetSatisfiedMatchConditions(
		ctx,
		tx,
		sqlcv1.GetSatisfiedMatchConditionsParams{
			Matchids:     matchIds,
			Conditionids: conditionIds,
			Datas:        datas,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get satisfied match conditions: %w", err)
	}

	satisfiedMatches := make([]*sqlcv1.SaveSatisfiedMatchConditionsRow, 0)

	if len(satisfiedMatchIds) > 0 {
		satisfiedMatches, err = m.queries.SaveSatisfiedMatchConditions(
			ctx,
			tx,
			satisfiedMatchIds,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to save satisfied match conditions: %w", err)
		}
	}

	// get any DAG input and additional metadata for the tasks
	dagIds := make([]int64, 0, len(satisfiedMatches))
	dagInsertedAts := make([]pgtype.Timestamptz, 0, len(satisfiedMatches))

	signalIds := make([]int64, 0, len(satisfiedMatches))

	for _, match := range satisfiedMatches {
		if match.TriggerDagID.Valid && match.TriggerDagInsertedAt.Valid {
			dagIds = append(dagIds, match.TriggerDagID.Int64)
			dagInsertedAts = append(dagInsertedAts, match.TriggerDagInsertedAt)
		}

		if match.SignalTaskID.Valid {
			signalIds = append(signalIds, match.SignalTaskID.Int64)
		}
	}

	tasks := make([]*V1TaskWithPayload, 0)

	if len(dagIds) > 0 {
		dagInputDatas, err := m.queries.GetDAGData(ctx, tx, sqlcv1.GetDAGDataParams{
			Dagids:         dagIds,
			Daginsertedats: dagInsertedAts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get DAG data: %w", err)
		}

		retrievePayloadOpts := make([]RetrievePayloadOpts, len(dagInputDatas))
		for i, dagData := range dagInputDatas {
			retrievePayloadOpts[i] = RetrievePayloadOpts{
				Id:         dagData.DagID,
				InsertedAt: dagData.DagInsertedAt,
				Type:       sqlcv1.V1PayloadTypeDAGINPUT,
				TenantId:   tenantId,
			}
		}

		payloads, err := m.payloadStore.Retrieve(ctx, tx, retrievePayloadOpts...)

		if err != nil {
			return nil, fmt.Errorf("failed to retrieve dag input payloads: %w", err)
		}

		dagIdsToInput := make(map[int64][]byte)
		dagIdsToMetadata := make(map[int64][]byte)

		for _, dagData := range dagInputDatas {
			retrieveOpts := RetrievePayloadOpts{
				Id:         dagData.DagID,
				InsertedAt: dagData.DagInsertedAt,
				Type:       sqlcv1.V1PayloadTypeDAGINPUT,
				TenantId:   tenantId,
			}

			payload, ok := payloads[retrieveOpts]

			if !ok {
				payload = dagData.Input
			}

			dagIdsToInput[dagData.DagID] = payload
			dagIdsToMetadata[dagData.DagID] = dagData.AdditionalMetadata
		}

		// determine which tasks to create based on step ids
		createTaskOpts := make([]CreateTaskOpts, 0, len(satisfiedMatches))
		replayTaskOpts := make([]ReplayTaskOpts, 0, len(satisfiedMatches))

		dependentMatches := make([]*sqlcv1.SaveSatisfiedMatchConditionsRow, 0)

		for _, match := range satisfiedMatches {
			if match.TriggerStepID != nil && match.TriggerExternalID != nil {
				if match.Action == sqlcv1.V1MatchConditionActionCREATEMATCH {
					dependentMatches = append(dependentMatches, match)
					continue
				}

				var input, additionalMetadata []byte

				if match.TriggerDagID.Valid {
					input = dagIdsToInput[match.TriggerDagID.Int64]
					additionalMetadata = dagIdsToMetadata[match.TriggerDagID.Int64]
				}

				matchData, err := NewMatchData(match.McAggregatedData)

				if err != nil {
					return nil, err
				}

				if match.TriggerExistingTaskID.Valid {
					opt := ReplayTaskOpts{
						TaskId:             match.TriggerExistingTaskID.Int64,
						InsertedAt:         match.TriggerExistingTaskInsertedAt,
						ExternalId:         *match.TriggerExternalID,
						StepId:             *match.TriggerStepID,
						AdditionalMetadata: additionalMetadata,
						InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
					}

					switch matchData.Action() {
					case sqlcv1.V1MatchConditionActionQUEUE:
						opt.Input = m.newTaskInput(input, matchData, nil)
						opt.InitialState = sqlcv1.V1TaskInitialStateQUEUED
					case sqlcv1.V1MatchConditionActionCANCEL:
						opt.InitialState = sqlcv1.V1TaskInitialStateCANCELLED
					case sqlcv1.V1MatchConditionActionSKIP:
						opt.InitialState = sqlcv1.V1TaskInitialStateSKIPPED
					}

					replayTaskOpts = append(replayTaskOpts, opt)
				} else {
					opt := CreateTaskOpts{
						ExternalId:         *match.TriggerExternalID,
						WorkflowRunId:      *match.TriggerWorkflowRunID,
						StepId:             *match.TriggerStepID,
						StepIndex:          int(match.TriggerStepIndex.Int64),
						AdditionalMetadata: additionalMetadata,
						InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
					}

					switch matchData.Action() {
					case sqlcv1.V1MatchConditionActionQUEUE:
						opt.Input = m.newTaskInput(input, matchData, nil)
						opt.DesiredWorkerId = m.DesiredWorkerId(opt.Input)
						opt.InitialState = sqlcv1.V1TaskInitialStateQUEUED
					case sqlcv1.V1MatchConditionActionCANCEL:
						opt.InitialState = sqlcv1.V1TaskInitialStateCANCELLED
					case sqlcv1.V1MatchConditionActionSKIP:
						opt.InitialState = sqlcv1.V1TaskInitialStateSKIPPED
					}

					if match.TriggerDagID.Valid && match.TriggerDagInsertedAt.Valid {
						opt.DagId = &match.TriggerDagID.Int64
						opt.DagInsertedAt = match.TriggerDagInsertedAt
					}

					if match.TriggerParentTaskExternalID != nil {
						opt.ParentTaskExternalId = match.TriggerParentTaskExternalID
					}

					if match.TriggerParentTaskID.Valid {
						opt.ParentTaskId = &match.TriggerParentTaskID.Int64
					}

					if match.TriggerParentTaskInsertedAt.Valid {
						opt.ParentTaskInsertedAt = &match.TriggerParentTaskInsertedAt.Time
					}

					if match.TriggerChildIndex.Valid {
						opt.ChildIndex = &match.TriggerChildIndex.Int64
					}

					if match.TriggerChildKey.Valid {
						opt.ChildKey = &match.TriggerChildKey.String
					}

					if match.TriggerPriority.Valid {
						opt.Priority = &match.TriggerPriority.Int32
					}

					createTaskOpts = append(createTaskOpts, opt)
				}
			}
		}

		// create dependent matches
		if len(dependentMatches) > 0 {
			err = m.createAdditionalMatches(ctx, tx, tenantId, dependentMatches)

			if err != nil {
				return nil, fmt.Errorf("failed to create additional matches: %w", err)
			}
		}

		// create tasks
		tasks, err = m.createTasks(ctx, tx, tenantId, createTaskOpts)

		if err != nil {
			return nil, fmt.Errorf("failed to create tasks: %w", err)
		}

		if len(replayTaskOpts) > 0 {
			replayedTasks, err := m.replayTasks(ctx, tx, tenantId, replayTaskOpts)

			if err != nil {
				return nil, fmt.Errorf("failed to replay %d tasks: %w", len(replayTaskOpts), err)
			}

			res.ReplayedTasks = replayedTasks
		}
	}

	res.CreatedTasks = tasks

	if len(signalIds) > 0 {
		// create a SIGNAL_COMPLETED event for any signal
		taskIds := make([]TaskIdInsertedAtRetryCount, 0, len(satisfiedMatches))
		datas := make([][]byte, 0, len(satisfiedMatches))
		eventKeys := make([]string, 0, len(satisfiedMatches))
		externalIds := make([]uuid.UUID, 0, len(satisfiedMatches))

		for _, match := range satisfiedMatches {
			if match.SignalTaskID.Valid && match.SignalTaskInsertedAt.Valid {
				taskIds = append(taskIds, TaskIdInsertedAtRetryCount{
					Id:         match.SignalTaskID.Int64,
					InsertedAt: match.SignalTaskInsertedAt,
					// signals are durable, meaning they persist between retries, so a retryCount of -1 is used
					RetryCount: -1,
				})
				if match.SignalExternalID != nil {
					externalIds = append(externalIds, *match.SignalExternalID)
				} else {
					externalIds = append(externalIds, uuid.Nil)
				}
				datas = append(datas, match.McAggregatedData)
				eventKeys = append(eventKeys, match.SignalKey.String)
			}
		}

		_, err = m.createTaskEvents(
			ctx,
			tx,
			tenantId,
			taskIds,
			externalIds,
			datas,
			makeEventTypeArr(sqlcv1.V1TaskEventTypeSIGNALCOMPLETED, len(taskIds)),
			eventKeys,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to create signal completed events: %w", err)
		}
	}

	end := time.Now()

	if end.Sub(start) > 100*time.Millisecond {
		m.l.Warn().Msgf("processing internal event matches took %s", end.Sub(start))
	}

	return res, nil
}

func (m *sharedRepository) processCELExpressions(ctx context.Context, events []CandidateEventMatch, conditions []*sqlcv1.ListMatchConditionsForEventRow, eventType sqlcv1.V1EventType) (map[uuid.UUID][]*sqlcv1.ListMatchConditionsForEventRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "MatchRepositoryImpl.processCELExpressions")
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "match_repository.process_cel_expressions.events_count",
		Value: attribute.IntValue(len(events)),
	})

	span.SetAttributes(attribute.KeyValue{
		Key:   "match_repository.process_cel_expressions.conditions_count",
		Value: attribute.IntValue(len(conditions)),
	})

	// parse CEL expressions
	programs := make(map[int64]cel.Program)
	conditionIdsToConditions := make(map[int64]*sqlcv1.ListMatchConditionsForEventRow)

	for _, condition := range conditions {
		expr := condition.Expression.String

		if expr == "" {
			expr = "true"
		}

		ast, issues := m.env.Compile(expr)

		if issues != nil {
			m.l.Error().Msgf("failed to compile CEL expression: %s", issues.String())
			continue
		}

		program, err := m.env.Program(ast)

		if err != nil {
			m.l.Error().Err(err).Msgf("failed to create CEL program: %s", expr)
			continue
		}

		programs[condition.ID] = program
		conditionIdsToConditions[condition.ID] = condition
	}

	// map of event ids to matched conditions
	matches := make(map[uuid.UUID][]*sqlcv1.ListMatchConditionsForEventRow)

	for _, event := range events {
		inputData := map[string]interface{}{}
		outputData := map[string]interface{}{}

		if len(event.Data) > 0 {
			switch eventType {
			case sqlcv1.V1EventTypeINTERNAL:
				// first unmarshal to event data, then parse the output data
				outputEventData := &TaskOutputEvent{}

				err := json.Unmarshal(event.Data, &outputEventData)

				if err != nil {
					m.l.Warn().Err(err).Msgf("[0] failed to unmarshal output event data. id: %s, key: %s", event.ID, event.Key)
					continue
				}

				if len(outputEventData.Output) > 0 {
					err = json.Unmarshal(outputEventData.Output, &outputData)

					if err != nil {
						m.l.Warn().Err(err).Msgf("failed to unmarshal output event data, output subfield for task %d", outputEventData.TaskId)
						continue
					}
				} else {
					err = json.Unmarshal(event.Data, &inputData)

					if err != nil {
						m.l.Warn().Err(err).Msgf("[1] failed to unmarshal output event data. id: %s, key: %s", event.ID, event.Key)
						continue
					}
				}
			case sqlcv1.V1EventTypeUSER:
				err := json.Unmarshal(event.Data, &inputData)

				if err != nil {
					m.l.Warn().Err(err).Msgf("failed to unmarshal user event data %s", string(event.Data))
					continue
				}
			}
		}

		for conditionId, program := range programs {
			condition := conditionIdsToConditions[conditionId]

			// if we don't match the event key and resource hint, skip
			if condition.EventKey != event.Key {
				continue
			}

			if condition.EventResourceHint.Valid && condition.EventResourceHint.String != *event.ResourceHint {
				continue
			}

			out, _, err := program.ContextEval(ctx, map[string]interface{}{
				"input":  inputData,
				"output": outputData,
			})

			if err != nil {
				// FIXME: we'd like to display this error to the user somehow, which is difficult as the
				// task hasn't necessarily been created yet. Additionally, we might have other conditions
				// which are valid, so we don't necessarily want to fail the entire match process. At the
				// same time, we need to remove it from the database, so we'll want to mark the condition as
				// satisfied and write an error to it. If the relevant conditions have errors, the task
				// should be created in a failed state.
				// How should we handle signals?
				m.l.Warn().Err(err).Msgf("failed to eval CEL program")
			}

			if b, ok := out.Value().(bool); ok && b {
				matches[event.ID] = append(matches[event.ID], conditionIdsToConditions[conditionId])
			}
		}
	}

	span.SetAttributes(attribute.KeyValue{
		Key:   "match_repository.process_cel_expressions.match_conditions_count",
		Value: attribute.IntValue(len(matches)),
	})

	return matches, nil
}

func (m *sharedRepository) createEventMatches(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, eventMatches []CreateMatchOpts) error {
	// Create maps to store match details by key
	matchByKey := make(map[string]CreateMatchOpts)

	// Separate DAG and signal matches
	dagMatches := make([]CreateMatchOpts, 0)
	signalMatches := make([]CreateMatchOpts, 0)

	// Group matches and create lookup map
	for _, match := range eventMatches {
		key := getMatchKey(match)
		matchByKey[key] = match

		if match.TriggerDAGId != nil && match.TriggerDAGInsertedAt.Valid && match.TriggerStepId != nil && match.TriggerExternalId != nil {
			dagMatches = append(dagMatches, match)
		} else if match.SignalTaskId != nil && match.SignalKey != nil && match.SignalTaskInsertedAt.Valid {
			signalMatches = append(signalMatches, match)
		}
	}

	// Create match conditions
	var matchConditionParams []sqlcv1.CreateMatchConditionsParams

	// Create DAG trigger matches
	if len(dagMatches) > 0 {
		// Prepare data for DAG trigger matches
		dagTenantIds := make([]uuid.UUID, len(dagMatches))
		dagKinds := make([]string, len(dagMatches))
		dagExistingDatas := make([][]byte, len(dagMatches))
		triggerDagIds := make([]int64, len(dagMatches))
		triggerDagInsertedAts := make([]pgtype.Timestamptz, len(dagMatches))
		triggerStepIds := make([]uuid.UUID, len(dagMatches))
		triggerStepIndices := make([]int64, len(dagMatches))
		triggerExternalIds := make([]uuid.UUID, len(dagMatches))
		triggerWorkflowRunIds := make([]*uuid.UUID, len(dagMatches))
		triggerExistingTaskIds := make([]pgtype.Int8, len(dagMatches))
		triggerExistingTaskInsertedAts := make([]pgtype.Timestamptz, len(dagMatches))
		triggerParentExternalIds := make([]*uuid.UUID, len(dagMatches))
		triggerParentTaskIds := make([]pgtype.Int8, len(dagMatches))
		triggerParentTaskInsertedAts := make([]pgtype.Timestamptz, len(dagMatches))
		triggerChildIndices := make([]pgtype.Int8, len(dagMatches))
		triggerChildKeys := make([]pgtype.Text, len(dagMatches))
		triggerPriorities := make([]pgtype.Int4, len(dagMatches))

		for i, match := range dagMatches {
			dagTenantIds[i] = tenantId
			dagKinds[i] = string(match.Kind)
			dagExistingDatas[i] = match.ExistingMatchData
			triggerDagIds[i] = *match.TriggerDAGId
			triggerDagInsertedAts[i] = match.TriggerDAGInsertedAt
			triggerStepIds[i] = *match.TriggerStepId
			triggerStepIndices[i] = match.TriggerStepIndex.Int64
			triggerExternalIds[i] = *match.TriggerExternalId
			triggerParentExternalIds[i] = match.TriggerParentTaskExternalId
			triggerParentTaskIds[i] = match.TriggerParentTaskId
			triggerParentTaskInsertedAts[i] = match.TriggerParentTaskInsertedAt
			triggerChildIndices[i] = match.TriggerChildIndex
			triggerChildKeys[i] = match.TriggerChildKey
			triggerPriorities[i] = match.TriggerPriority

			if match.TriggerExistingTaskId != nil {
				triggerExistingTaskIds[i] = pgtype.Int8{Int64: *match.TriggerExistingTaskId, Valid: true}
			} else {
				triggerExistingTaskIds[i] = pgtype.Int8{}
			}

			triggerWorkflowRunIds[i] = match.TriggerWorkflowRunId

			triggerExistingTaskInsertedAts[i] = match.TriggerExistingTaskInsertedAt
		}

		// Create matches in the database
		dagCreatedMatches, err := m.queries.CreateMatchesForDAGTriggers(
			ctx,
			tx,
			sqlcv1.CreateMatchesForDAGTriggersParams{
				Tenantids:                     dagTenantIds,
				Kinds:                         dagKinds,
				ExistingDatas:                 dagExistingDatas,
				Triggerdagids:                 triggerDagIds,
				Triggerdaginsertedats:         triggerDagInsertedAts,
				Triggerstepids:                triggerStepIds,
				Triggerstepindex:              triggerStepIndices,
				Triggerexternalids:            triggerExternalIds,
				Triggerworkflowrunids:         triggerWorkflowRunIds,
				Triggerexistingtaskids:        triggerExistingTaskIds,
				Triggerexistingtaskinsertedat: triggerExistingTaskInsertedAts,
				TriggerParentTaskExternalIds:  triggerParentExternalIds,
				TriggerParentTaskIds:          triggerParentTaskIds,
				TriggerParentTaskInsertedAt:   triggerParentTaskInsertedAts,
				TriggerChildIndex:             triggerChildIndices,
				TriggerChildKey:               triggerChildKeys,
				TriggerPriorities:             triggerPriorities,
			},
		)

		if err != nil {
			return err
		}

		// For each created match, generate a key from its properties and map it to its ID
		for _, createdMatch := range dagCreatedMatches {
			// Get existingTaskId pointer if valid
			var existingTaskId *int64
			if createdMatch.TriggerExistingTaskID.Valid {
				taskId := createdMatch.TriggerExistingTaskID.Int64
				existingTaskId = &taskId
			}

			// Generate key using the specific function for DAG matches
			key := getDagMatchKey(
				string(createdMatch.Kind),
				createdMatch.TriggerDagID.Int64,
				*createdMatch.TriggerExternalID,
				*createdMatch.TriggerStepID,
				existingTaskId,
				createdMatch.TriggerParentTaskID,
			)

			// Get the original match from the map
			match, exists := matchByKey[key]

			if !exists {
				return fmt.Errorf("match not found for key %s", key)
			}

			for _, condition := range match.Conditions {
				matchConditionParams = append(matchConditionParams, getConditionParam(tenantId, createdMatch.ID, condition))
			}
		}
	}

	// Create signal trigger matches
	if len(signalMatches) > 0 {
		// Prepare data for signal trigger matches
		signalTenantIds := make([]uuid.UUID, len(signalMatches))
		signalKinds := make([]string, len(signalMatches))
		signalTaskIds := make([]int64, len(signalMatches))
		signalTaskInsertedAts := make([]pgtype.Timestamptz, len(signalMatches))
		signalKeys := make([]string, len(signalMatches))

		for i, match := range signalMatches {
			signalTenantIds[i] = tenantId
			signalKinds[i] = string(match.Kind)
			signalTaskIds[i] = *match.SignalTaskId
			signalTaskInsertedAts[i] = match.SignalTaskInsertedAt
			signalKeys[i] = *match.SignalKey
		}

		// Create matches in the database
		signalCreatedMatches, err := m.queries.CreateMatchesForSignalTriggers(
			ctx,
			tx,
			sqlcv1.CreateMatchesForSignalTriggersParams{
				Tenantids:             signalTenantIds,
				Kinds:                 signalKinds,
				Signaltaskids:         signalTaskIds,
				Signaltaskinsertedats: signalTaskInsertedAts,
				Signalkeys:            signalKeys,
			},
		)

		if err != nil {
			return err
		}

		// For each created match, generate a key from its properties and map it to its ID
		for _, createdMatch := range signalCreatedMatches {
			// Generate key using the specific function for signal matches
			key := getSignalMatchKey(
				string(createdMatch.Kind),
				createdMatch.SignalTaskID.Int64,
				createdMatch.SignalKey.String,
			)

			// Get the original match from the map
			match, exists := matchByKey[key]

			if !exists {
				return fmt.Errorf("match not found for key %s", key)
			}

			for _, condition := range match.Conditions {
				matchConditionParams = append(matchConditionParams, getConditionParam(tenantId, createdMatch.ID, condition))
			}
		}
	}

	_, err := m.queries.CreateMatchConditions(ctx, tx, matchConditionParams)

	if err != nil {
		return err
	}

	return nil
}

func getConditionParam(tenantId uuid.UUID, createdMatchId int64, condition GroupMatchCondition) sqlcv1.CreateMatchConditionsParams {
	param := sqlcv1.CreateMatchConditionsParams{
		V1MatchID:       createdMatchId,
		TenantID:        tenantId,
		EventType:       condition.EventType,
		EventKey:        condition.EventKey,
		ReadableDataKey: condition.ReadableDataKey,
		OrGroupID:       condition.GroupId,
		Expression:      sqlchelpers.TextFromStr(condition.Expression),
		Action:          condition.Action,
		IsSatisfied:     false,
		Data:            condition.Data,
	}

	if condition.EventResourceHint != nil {
		param.EventResourceHint = sqlchelpers.TextFromStr(*condition.EventResourceHint)
	}

	return param
}

func getDagMatchKey(kind string, dagId int64, externalId uuid.UUID, stepId uuid.UUID, existingTaskId *int64, parentTaskId pgtype.Int8) string {
	existingTaskIdStr := ""
	if existingTaskId != nil {
		existingTaskIdStr = fmt.Sprintf("%d", *existingTaskId)
	}

	parentTaskIdStr := ""
	if parentTaskId.Valid {
		parentTaskIdStr = fmt.Sprintf("%d", parentTaskId.Int64)
	}

	return fmt.Sprintf("dag:%s:%d:%s:%s:%s:%s",
		kind,
		dagId,
		externalId,
		stepId,
		existingTaskIdStr,
		parentTaskIdStr)
}

func getSignalMatchKey(kind string, signalTaskId int64, signalKey string) string {
	return fmt.Sprintf("signal:%s:%d:%s", kind, signalTaskId, signalKey)
}

func getMatchKey(match CreateMatchOpts) string {
	// For DAG triggers
	if match.TriggerDAGId != nil && match.TriggerStepId != nil && match.TriggerExternalId != nil {
		return getDagMatchKey(
			string(sqlcv1.V1MatchKindTRIGGER),
			*match.TriggerDAGId,
			*match.TriggerExternalId,
			*match.TriggerStepId,
			match.TriggerExistingTaskId,
			match.TriggerParentTaskId)
	}

	// For signal triggers
	if match.SignalTaskId != nil && match.SignalKey != nil {
		return getSignalMatchKey(
			string(sqlcv1.V1MatchKindSIGNAL),
			*match.SignalTaskId,
			*match.SignalKey)
	}

	// Fallback for incomplete match data
	return uuid.New().String()
}

func (m *sharedRepository) createAdditionalMatches(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, satisfiedMatches []*sqlcv1.SaveSatisfiedMatchConditionsRow) error { // nolint: unused
	additionalMatchStepIds := make([]uuid.UUID, 0, len(satisfiedMatches))

	for _, match := range satisfiedMatches {
		if match.Action == sqlcv1.V1MatchConditionActionCREATEMATCH && match.TriggerStepID != nil {
			additionalMatchStepIds = append(additionalMatchStepIds, *match.TriggerStepID)
		}
	}

	// get the configs for the additional matches
	stepMatchConditions, err := m.queries.ListStepMatchConditions(
		ctx,
		tx,
		sqlcv1.ListStepMatchConditionsParams{
			Tenantid: tenantId,
			Stepids:  additionalMatchStepIds,
		},
	)

	if err != nil {
		return err
	}

	stepIdsToConditions := make(map[string][]*sqlcv1.V1StepMatchCondition)

	for _, condition := range stepMatchConditions {
		stepId := condition.StepID.String()
		if _, ok := stepIdsToConditions[stepId]; !ok {
			stepIdsToConditions[stepId] = make([]*sqlcv1.V1StepMatchCondition, 0)
		}

		stepIdsToConditions[stepId] = append(stepIdsToConditions[stepId], condition)
	}

	additionalMatches := make([]CreateMatchOpts, 0, len(satisfiedMatches))

	for _, match := range satisfiedMatches {
		if match.TriggerStepID != nil && match.Action == sqlcv1.V1MatchConditionActionCREATEMATCH {
			conditions, ok := stepIdsToConditions[match.TriggerStepID.String()]

			if !ok {
				continue
			}

			triggerExternalId := match.TriggerExternalID
			triggerWorkflowRunId := match.TriggerWorkflowRunID
			triggerStepId := match.TriggerStepID
			var triggerExistingTaskId *int64

			if match.TriggerExistingTaskID.Valid {
				triggerExistingTaskId = &match.TriggerExistingTaskID.Int64
			}

			// copy over the match data
			opt := CreateMatchOpts{
				Kind:                          sqlcv1.V1MatchKindTRIGGER,
				ExistingMatchData:             match.McAggregatedData,
				Conditions:                    make([]GroupMatchCondition, 0),
				TriggerDAGId:                  &match.TriggerDagID.Int64,
				TriggerDAGInsertedAt:          match.TriggerDagInsertedAt,
				TriggerExternalId:             triggerExternalId,
				TriggerWorkflowRunId:          triggerWorkflowRunId,
				TriggerStepId:                 triggerStepId,
				TriggerStepIndex:              match.TriggerStepIndex,
				TriggerExistingTaskId:         triggerExistingTaskId,
				TriggerExistingTaskInsertedAt: match.TriggerExistingTaskInsertedAt,
				TriggerParentTaskExternalId:   match.TriggerParentTaskExternalID,
				TriggerParentTaskId:           match.TriggerParentTaskID,
				TriggerParentTaskInsertedAt:   match.TriggerParentTaskInsertedAt,
				TriggerChildIndex:             match.TriggerChildIndex,
				TriggerChildKey:               match.TriggerChildKey,
				TriggerPriority:               match.TriggerPriority,
			}

			for _, condition := range conditions {
				switch condition.Kind {
				case sqlcv1.V1StepMatchConditionKindSLEEP:
					c, err := m.durableSleepCondition(
						ctx,
						tx,
						tenantId,
						condition.OrGroupID,
						condition.ReadableDataKey,
						condition.SleepDuration.String,
						condition.Action,
					)

					if err != nil {
						return err
					}

					opt.Conditions = append(opt.Conditions, *c)
				case sqlcv1.V1StepMatchConditionKindUSEREVENT:
					opt.Conditions = append(opt.Conditions, m.userEventCondition(
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

			additionalMatches = append(additionalMatches, opt)
		}
	}

	if len(additionalMatches) > 0 {
		err := m.createEventMatches(ctx, tx, tenantId, additionalMatches)

		if err != nil {
			return err
		}
	}

	return nil
}

func (m *sharedRepository) durableSleepCondition(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, orGroupId uuid.UUID, readableDataKey, sleepDuration string, action sqlcv1.V1MatchConditionAction) (*GroupMatchCondition, error) {
	// FIXME: make this a proper bulk write
	sleep, err := m.queries.CreateDurableSleep(ctx, tx, sqlcv1.CreateDurableSleepParams{
		TenantID:       tenantId,
		SleepDurations: []string{sleepDuration},
	})

	if err != nil {
		return nil, err
	}

	if len(sleep) != 1 {
		return nil, fmt.Errorf("expected 1 sleep to be created, but got %d", len(sleep))
	}

	eventKey := getDurableSleepEventKey(sleep[0].ID)
	eventType := sqlcv1.V1EventTypeINTERNAL
	expression := "true"

	return &GroupMatchCondition{
		GroupId:         orGroupId,
		EventType:       eventType,
		EventKey:        eventKey,
		ReadableDataKey: readableDataKey,
		Expression:      expression,
		Action:          action,
	}, nil
}

func (m *sharedRepository) userEventCondition(orGroupId uuid.UUID, readableDataKey, eventKey, expression string, action sqlcv1.V1MatchConditionAction) GroupMatchCondition {
	return GroupMatchCondition{
		GroupId:         orGroupId,
		EventType:       sqlcv1.V1EventTypeUSER,
		EventKey:        eventKey,
		ReadableDataKey: readableDataKey,
		Expression:      expression,
		Action:          action,
	}
}

func getDurableSleepEventKey(sleepId int64) string {
	return fmt.Sprintf("sleep-%d", sleepId)
}
