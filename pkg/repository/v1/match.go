package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/google/cel-go/cel"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type CandidateEventMatch struct {
	// A UUID for the event
	ID string

	// A timestamp for the event
	EventTimestamp time.Time

	// Key for the event
	Key string

	// Resource hint for the event
	ResourceHint *string

	// Data for the event
	Data []byte
}

type CreateMatchOpts struct {
	Kind sqlcv1.V1MatchKind

	Conditions []GroupMatchCondition

	TriggerDAGId *int64

	TriggerDAGInsertedAt pgtype.Timestamptz

	TriggerExternalId *string

	TriggerWorkflowRunId *string

	TriggerStepId *string

	TriggerStepIndex pgtype.Int8

	TriggerExistingTaskId *int64

	TriggerExistingTaskInsertedAt pgtype.Timestamptz

	TriggerParentTaskExternalId pgtype.UUID

	TriggerParentTaskId pgtype.Int8

	TriggerParentTaskInsertedAt pgtype.Timestamptz

	TriggerChildIndex pgtype.Int8

	TriggerChildKey pgtype.Text

	SignalTaskId *int64

	SignalTaskInsertedAt pgtype.Timestamptz

	SignalExternalId *string

	SignalKey *string
}

type InternalEventMatchResults struct {
	// The list of tasks which were created from the matches
	CreatedTasks []*sqlcv1.V1Task

	// The list of tasks which were replayed from the matches
	ReplayedTasks []*sqlcv1.V1Task
}

type GroupMatchCondition struct {
	GroupId string `validate:"required,uuid"`

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
	ProcessInternalEventMatches(ctx context.Context, tenantId string, events []CandidateEventMatch) (*InternalEventMatchResults, error)
}

type MatchRepositoryImpl struct {
	*sharedRepository
}

func newMatchRepository(s *sharedRepository) (MatchRepository, error) {
	return &MatchRepositoryImpl{
		sharedRepository: s,
	}, nil
}

// ProcessInternalEventMatches processes a list of internal events
func (m *MatchRepositoryImpl) ProcessInternalEventMatches(ctx context.Context, tenantId string, events []CandidateEventMatch) (*InternalEventMatchResults, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, m.pool, m.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	res, err := m.processInternalEventMatches(ctx, tx, tenantId, events)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return res, nil
}

func (m *sharedRepository) processInternalEventMatches(ctx context.Context, tx sqlcv1.DBTX, tenantId string, events []CandidateEventMatch) (*InternalEventMatchResults, error) {
	start := time.Now()

	res := &InternalEventMatchResults{}

	eventKeys := make([]string, 0, len(events))
	resourceHints := make([]pgtype.Text, 0, len(events))
	uniqueEventKeys := make(map[string]struct{})
	idsToEvents := make(map[string]CandidateEventMatch)

	for _, event := range events {
		idsToEvents[event.ID] = event

		if event.ResourceHint == nil {
			if _, ok := uniqueEventKeys[event.Key]; ok {
				continue
			}
		}

		eventKeys = append(eventKeys, event.Key)
		uniqueEventKeys[event.Key] = struct{}{}

		if event.ResourceHint != nil {
			resourceHints = append(resourceHints, pgtype.Text{String: *event.ResourceHint, Valid: true})
		} else {
			resourceHints = append(resourceHints, pgtype.Text{})
		}
	}

	// list all match conditions
	matchConditions, err := m.queries.ListMatchConditionsForEvent(
		ctx,
		tx,
		sqlcv1.ListMatchConditionsForEventParams{
			Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
			Eventtype:          sqlcv1.V1EventTypeINTERNAL,
			Eventkeys:          eventKeys,
			Eventresourcehints: resourceHints,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to list match conditions for event: %w", err)
	}

	// pass match conditions through CEL expressions parser
	matches, err := m.processCELExpressions(ctx, events, matchConditions)

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

	tasks := make([]*sqlcv1.V1Task, 0)

	if len(dagIds) > 0 {
		dagInputDatas, err := m.queries.GetDAGData(ctx, tx, sqlcv1.GetDAGDataParams{
			Dagids:         dagIds,
			Daginsertedats: dagInsertedAts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get DAG data: %w", err)
		}

		dagIdsToInput := make(map[int64][]byte)
		dagIdsToMetadata := make(map[int64][]byte)

		for _, dagData := range dagInputDatas {
			dagIdsToInput[dagData.DagID] = dagData.Input
			dagIdsToMetadata[dagData.DagID] = dagData.AdditionalMetadata
		}

		// determine which tasks to create based on step ids
		createTaskOpts := make([]CreateTaskOpts, 0, len(satisfiedMatches))
		replayTaskOpts := make([]ReplayTaskOpts, 0, len(satisfiedMatches))

		for _, match := range satisfiedMatches {
			if match.TriggerStepID.Valid && match.TriggerExternalID.Valid {
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
						ExternalId:         sqlchelpers.UUIDToStr(match.TriggerExternalID),
						StepId:             sqlchelpers.UUIDToStr(match.TriggerStepID),
						AdditionalMetadata: additionalMetadata,
						InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
					}

					switch matchData.Action() {
					case sqlcv1.V1MatchConditionActionQUEUE:
						opt.Input = m.newTaskInput(input, matchData)
						opt.InitialState = sqlcv1.V1TaskInitialStateQUEUED
					case sqlcv1.V1MatchConditionActionCANCEL:
						opt.InitialState = sqlcv1.V1TaskInitialStateCANCELLED
					case sqlcv1.V1MatchConditionActionSKIP:
						opt.InitialState = sqlcv1.V1TaskInitialStateSKIPPED
					}

					replayTaskOpts = append(replayTaskOpts, opt)
				} else {
					opt := CreateTaskOpts{
						ExternalId:         sqlchelpers.UUIDToStr(match.TriggerExternalID),
						WorkflowRunId:      sqlchelpers.UUIDToStr(match.TriggerWorkflowRunID),
						StepId:             sqlchelpers.UUIDToStr(match.TriggerStepID),
						StepIndex:          int(match.TriggerStepIndex.Int64),
						AdditionalMetadata: additionalMetadata,
					}

					switch matchData.Action() {
					case sqlcv1.V1MatchConditionActionQUEUE:
						opt.Input = m.newTaskInput(input, matchData)
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

					if match.TriggerParentTaskExternalID.Valid {
						externalId := sqlchelpers.UUIDToStr(match.TriggerParentTaskExternalID)
						opt.ParentTaskExternalId = &externalId
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

					createTaskOpts = append(createTaskOpts, opt)
				}
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
		externalIds := make([]string, 0, len(satisfiedMatches))

		for _, match := range satisfiedMatches {
			if match.SignalTaskID.Valid && match.SignalTaskInsertedAt.Valid {
				taskIds = append(taskIds, TaskIdInsertedAtRetryCount{
					Id:         match.SignalTaskID.Int64,
					InsertedAt: match.SignalTaskInsertedAt,
					RetryCount: -1,
				})
				externalIds = append(externalIds, sqlchelpers.UUIDToStr(match.SignalExternalID))
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

func (m *sharedRepository) processCELExpressions(ctx context.Context, events []CandidateEventMatch, conditions []*sqlcv1.ListMatchConditionsForEventRow) (map[string][]*sqlcv1.ListMatchConditionsForEventRow, error) {
	// parse CEL expressions
	programs := make(map[int64]cel.Program)
	conditionIdsToConditions := make(map[int64]*sqlcv1.ListMatchConditionsForEventRow)

	for _, condition := range conditions {
		ast, issues := m.env.Compile(condition.Expression.String)

		if issues != nil {
			return nil, issues.Err()
		}

		program, err := m.env.Program(ast)

		if err != nil {
			return nil, err
		}

		programs[condition.ID] = program
		conditionIdsToConditions[condition.ID] = condition
	}

	// map of event ids to matched conditions
	matches := make(map[string][]*sqlcv1.ListMatchConditionsForEventRow)

	for _, event := range events {
		inputData := map[string]interface{}{}

		if len(event.Data) > 0 {
			err := json.Unmarshal(event.Data, &inputData)

			if err != nil {
				m.l.Error().Err(err).Msgf("failed to unmarshal event data %s", string(event.Data))
				return nil, err
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
				"input": inputData,
			})

			if err != nil {
				return nil, err
			}

			if out.Value().(bool) {
				matches[event.ID] = append(matches[event.ID], conditionIdsToConditions[conditionId])
			}
		}
	}

	return matches, nil
}

func (m *sharedRepository) createEventMatches(ctx context.Context, tx sqlcv1.DBTX, tenantId string, eventMatches []CreateMatchOpts) error {
	// create the event matches first
	dagTenantIds := make([]pgtype.UUID, 0, len(eventMatches))
	dagKinds := make([]string, 0, len(eventMatches))
	triggerDagIds := make([]int64, 0, len(eventMatches))
	triggerDagInsertedAts := make([]pgtype.Timestamptz, 0, len(eventMatches))
	triggerStepIds := make([]pgtype.UUID, 0, len(eventMatches))
	triggerStepIndices := make([]int64, 0, len(eventMatches))
	triggerExternalIds := make([]pgtype.UUID, 0, len(eventMatches))
	triggerWorkflowRunIds := make([]pgtype.UUID, 0, len(eventMatches))
	triggerExistingTaskIds := make([]pgtype.Int8, 0, len(eventMatches))
	triggerExistingTaskInsertedAts := make([]pgtype.Timestamptz, 0, len(eventMatches))
	triggerParentExternalIds := make([]pgtype.UUID, 0, len(eventMatches))
	triggerParentTaskIds := make([]pgtype.Int8, 0, len(eventMatches))
	triggerParentTaskInsertedAts := make([]pgtype.Timestamptz, 0, len(eventMatches))
	triggerChildIndices := make([]pgtype.Int8, 0, len(eventMatches))
	triggerChildKeys := make([]pgtype.Text, 0, len(eventMatches))

	signalTenantIds := make([]pgtype.UUID, 0, len(eventMatches))
	signalKinds := make([]string, 0, len(eventMatches))
	signalTaskIds := make([]int64, 0, len(eventMatches))
	signalTaskInsertedAts := make([]pgtype.Timestamptz, 0, len(eventMatches))
	signalKeys := make([]string, 0, len(eventMatches))

	for _, match := range eventMatches {
		// at the moment, we skip creating matches for things that don't have all fields set
		if match.TriggerDAGId != nil && match.TriggerDAGInsertedAt.Valid && match.TriggerStepId != nil && match.TriggerExternalId != nil {
			dagTenantIds = append(dagTenantIds, sqlchelpers.UUIDFromStr(tenantId))
			dagKinds = append(dagKinds, string(match.Kind))
			triggerDagIds = append(triggerDagIds, *match.TriggerDAGId)
			triggerDagInsertedAts = append(triggerDagInsertedAts, match.TriggerDAGInsertedAt)
			triggerStepIds = append(triggerStepIds, sqlchelpers.UUIDFromStr(*match.TriggerStepId))
			triggerStepIndices = append(triggerStepIndices, match.TriggerStepIndex.Int64)
			triggerExternalIds = append(triggerExternalIds, sqlchelpers.UUIDFromStr(*match.TriggerExternalId))
			triggerParentExternalIds = append(triggerParentExternalIds, match.TriggerParentTaskExternalId)
			triggerParentTaskIds = append(triggerParentTaskIds, match.TriggerParentTaskId)
			triggerParentTaskInsertedAts = append(triggerParentTaskInsertedAts, match.TriggerParentTaskInsertedAt)
			triggerChildIndices = append(triggerChildIndices, match.TriggerChildIndex)
			triggerChildKeys = append(triggerChildKeys, match.TriggerChildKey)

			if match.TriggerExistingTaskId != nil {
				triggerExistingTaskIds = append(triggerExistingTaskIds, pgtype.Int8{Int64: *match.TriggerExistingTaskId, Valid: true})
			} else {
				triggerExistingTaskIds = append(triggerExistingTaskIds, pgtype.Int8{})
			}

			if match.TriggerWorkflowRunId != nil {
				triggerWorkflowRunIds = append(triggerWorkflowRunIds, sqlchelpers.UUIDFromStr(*match.TriggerWorkflowRunId))
			} else {
				triggerWorkflowRunIds = append(triggerWorkflowRunIds, pgtype.UUID{})
			}

			triggerExistingTaskInsertedAts = append(triggerExistingTaskInsertedAts, match.TriggerExistingTaskInsertedAt)
		} else if match.SignalTaskId != nil && match.SignalKey != nil && match.SignalTaskInsertedAt.Valid {
			signalTenantIds = append(signalTenantIds, sqlchelpers.UUIDFromStr(tenantId))
			signalKinds = append(signalKinds, string(match.Kind))
			signalTaskIds = append(signalTaskIds, *match.SignalTaskId)
			signalTaskInsertedAts = append(signalTaskInsertedAts, match.SignalTaskInsertedAt)
			signalKeys = append(signalKeys, *match.SignalKey)
		}
	}

	var createdMatches []*sqlcv1.V1Match

	if len(dagTenantIds) > 0 {
		dagCreatedMatches, err := m.queries.CreateMatchesForDAGTriggers(
			ctx,
			tx,
			sqlcv1.CreateMatchesForDAGTriggersParams{
				Tenantids:                     dagTenantIds,
				Kinds:                         dagKinds,
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
			},
		)

		if err != nil {
			return err
		}

		createdMatches = append(createdMatches, dagCreatedMatches...)
	}

	if len(signalTenantIds) > 0 {
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

		createdMatches = append(createdMatches, signalCreatedMatches...)
	}

	if len(createdMatches) != len(eventMatches) {
		return fmt.Errorf("expected %d matches to be created, but only %d were created", len(eventMatches), len(createdMatches))
	}

	// next, create the match conditions
	params := make([]sqlcv1.CreateMatchConditionsParams, 0, len(eventMatches))

	for i, match := range eventMatches {
		createdMatch := createdMatches[i]

		for _, condition := range match.Conditions {
			param := sqlcv1.CreateMatchConditionsParams{
				V1MatchID:       createdMatch.ID,
				TenantID:        sqlchelpers.UUIDFromStr(tenantId),
				EventType:       condition.EventType,
				EventKey:        condition.EventKey,
				ReadableDataKey: condition.ReadableDataKey,
				OrGroupID:       sqlchelpers.UUIDFromStr(condition.GroupId),
				Expression:      sqlchelpers.TextFromStr(condition.Expression),
				Action:          condition.Action,
				IsSatisfied:     false,
				Data:            condition.Data,
			}

			if condition.EventResourceHint != nil {
				param.EventResourceHint = sqlchelpers.TextFromStr(*condition.EventResourceHint)
			}

			params = append(params, param)
		}
	}

	_, err := m.queries.CreateMatchConditions(ctx, tx, params)

	if err != nil {
		return err
	}

	return nil
}
