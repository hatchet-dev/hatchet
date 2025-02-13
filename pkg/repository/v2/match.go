package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/google/cel-go/cel"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type CandidateEventMatch struct {
	// A UUID for the event
	ID string

	// A timestamp for the event
	EventTimestamp time.Time

	// Key for the event
	Key string

	// Data for the event
	Data []byte
}

type CreateMatchOpts struct {
	Kind sqlcv2.V2MatchKind

	Conditions []GroupMatchCondition

	TriggerDAGId *int64

	TriggerDAGInsertedAt pgtype.Timestamptz

	TriggerExternalId *string

	TriggerStepId *string

	SignalTaskId *int64

	SignalKey *string
}

type InternalEventMatchResults struct {
	// The list of tasks which were created in a queued state
	CreatedQueuedTasks []*sqlcv2.V2Task

	// A list of tasks which are created in a directly cancelled state
	CreatedCancelledTasks []*sqlcv2.V2Task

	// A list of tasks which are created in a skipped state
	CreatedSkippedTasks []*sqlcv2.V2Task
}

type GroupMatchCondition struct {
	GroupId string `validate:"required,uuid"`

	EventType sqlcv2.V2EventType

	EventKey string

	Expression string

	Action sqlcv2.V2MatchConditionAction
}

type MatchRepository interface {
	ProcessInternalEventMatches(ctx context.Context, tenantId string, events []CandidateEventMatch) (*InternalEventMatchResults, error)
}

type MatchRepositoryImpl struct {
	*sharedRepository

	env *cel.Env
}

func newMatchRepository(s *sharedRepository) (MatchRepository, error) {
	env, err := cel.NewEnv()

	if err != nil {
		return nil, err
	}

	return &MatchRepositoryImpl{
		sharedRepository: s,
		env:              env,
	}, nil
}

// ProcessInternalEventMatches processes a list of internal events
func (m *MatchRepositoryImpl) ProcessInternalEventMatches(ctx context.Context, tenantId string, events []CandidateEventMatch) (*InternalEventMatchResults, error) {
	start := time.Now()

	res := &InternalEventMatchResults{}

	eventKeys := make([]string, 0, len(events))
	uniqueEventKeys := make(map[string]struct{})
	idsToEvents := make(map[string]CandidateEventMatch)

	for _, event := range events {
		idsToEvents[event.ID] = event

		if _, ok := uniqueEventKeys[event.Key]; ok {
			continue
		}

		eventKeys = append(eventKeys, event.Key)
		uniqueEventKeys[event.Key] = struct{}{}
	}

	// list all match conditions
	matchConditions, err := m.queries.ListMatchConditionsForEvent(
		ctx,
		m.pool,
		sqlcv2.ListMatchConditionsForEventParams{
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
			Eventtype: sqlcv2.V2EventTypeINTERNAL,
			Eventkeys: eventKeys,
		},
	)

	if err != nil {
		return nil, err
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

			matchIds = append(matchIds, condition.V2MatchID)
			conditionIds = append(conditionIds, condition.ID)
			datas = append(datas, event.Data)
		}
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, m.pool, m.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// update condition rows in the database
	satisfiedMatchIds, err := m.queries.GetSatisfiedMatchConditions(
		ctx,
		tx,
		sqlcv2.GetSatisfiedMatchConditionsParams{
			Matchids:     matchIds,
			Conditionids: conditionIds,
			Datas:        datas,
		},
	)

	if err != nil {
		return nil, err
	}

	satisfiedMatches := make([]*sqlcv2.SaveSatisfiedMatchConditionsRow, 0)

	if len(satisfiedMatchIds) > 0 {
		satisfiedMatches, err = m.queries.SaveSatisfiedMatchConditions(
			ctx,
			tx,
			satisfiedMatchIds,
		)

		if err != nil {
			return nil, err
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

		if match.SignalTargetID.Valid {
			signalIds = append(signalIds, match.SignalTargetID.Int64)
		}
	}

	tasks := make([]*sqlcv2.V2Task, 0)

	if len(dagIds) > 0 {
		dagInputDatas, err := m.queries.GetDAGData(ctx, tx, sqlcv2.GetDAGDataParams{
			Dagids:         dagIds,
			Daginsertedats: dagInsertedAts,
		})

		if err != nil {
			return nil, err
		}

		dagIdsToInput := make(map[int64][]byte)
		dagIdsToMetadata := make(map[int64][]byte)

		for _, dagData := range dagInputDatas {
			dagIdsToInput[dagData.DagID] = dagData.Input
			dagIdsToMetadata[dagData.DagID] = dagData.AdditionalMetadata
		}

		// determine which tasks to create based on step ids
		taskOpts := make([]CreateTaskOpts, 0, len(satisfiedMatches))

		for _, match := range satisfiedMatches {
			if match.TriggerStepID.Valid && match.TriggerExternalID.Valid {
				var input, additionalMetadata []byte

				if match.TriggerDagID.Valid {
					input = dagIdsToInput[match.TriggerDagID.Int64]
					additionalMetadata = dagIdsToMetadata[match.TriggerDagID.Int64]
				}

				opt := CreateTaskOpts{
					ExternalId:         sqlchelpers.UUIDToStr(match.TriggerExternalID),
					StepId:             sqlchelpers.UUIDToStr(match.TriggerStepID),
					AdditionalMetadata: additionalMetadata,
				}

				action, data, err := m.parseTriggerData(match.McAggregatedData)

				if err != nil {
					return nil, err
				}

				switch *action {
				case sqlcv2.V2MatchConditionActionCREATE:
					opt.Input = m.newTaskInput(input, data)
					opt.InitialState = sqlcv2.V2TaskInitialStateQUEUED
				case sqlcv2.V2MatchConditionActionCANCEL:
					opt.InitialState = sqlcv2.V2TaskInitialStateCANCELLED
				case sqlcv2.V2MatchConditionActionSKIP:
					opt.InitialState = sqlcv2.V2TaskInitialStateSKIPPED
				}

				if match.TriggerDagID.Valid && match.TriggerDagInsertedAt.Valid {
					opt.DagId = &match.TriggerDagID.Int64
					opt.DagInsertedAt = match.TriggerDagInsertedAt
				}

				taskOpts = append(taskOpts, opt)
			}
		}

		// create tasks
		tasks, err = m.createTasks(ctx, tx, tenantId, taskOpts)

		if err != nil {
			return nil, err
		}

		for _, task := range tasks {
			if task.InitialState == sqlcv2.V2TaskInitialStateQUEUED {
				res.CreatedQueuedTasks = append(res.CreatedQueuedTasks, task)
			} else if task.InitialState == sqlcv2.V2TaskInitialStateCANCELLED {
				res.CreatedCancelledTasks = append(res.CreatedCancelledTasks, task)
			} else if task.InitialState == sqlcv2.V2TaskInitialStateSKIPPED {
				res.CreatedSkippedTasks = append(res.CreatedSkippedTasks, task)
			}
		}
	}

	if len(signalIds) > 0 {
		// create a SIGNAL_COMPLETED event for any signal
		taskIds := make([]TaskIdRetryCount, 0, len(satisfiedMatches))
		datas := make([][]byte, 0, len(satisfiedMatches))
		eventKeys := make([]string, 0, len(satisfiedMatches))

		for _, match := range satisfiedMatches {
			if match.SignalTargetID.Valid {
				taskIds = append(taskIds, TaskIdRetryCount{
					Id:         match.SignalTargetID.Int64,
					RetryCount: -1,
				})
				datas = append(datas, match.McAggregatedData)
				eventKeys = append(eventKeys, match.SignalKey.String)
			}
		}

		err = m.createTaskEvents(ctx, tx, tenantId, taskIds, datas, sqlcv2.V2TaskEventTypeSIGNALCOMPLETED, eventKeys)

		if err != nil {
			return nil, err
		}
	}

	// commit
	if err := commit(ctx); err != nil {
		return nil, err
	}

	end := time.Now()

	if end.Sub(start) > 100*time.Millisecond {
		m.l.Warn().Msgf("processing internal event matches took %s", end.Sub(start))
	}

	return res, nil
}

func (m *MatchRepositoryImpl) processCELExpressions(ctx context.Context, events []CandidateEventMatch, conditions []*sqlcv2.ListMatchConditionsForEventRow) (map[string][]*sqlcv2.ListMatchConditionsForEventRow, error) {
	// parse CEL expressions
	programs := make(map[int64]cel.Program)
	conditionIdsToConditions := make(map[int64]*sqlcv2.ListMatchConditionsForEventRow)

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
	matches := make(map[string][]*sqlcv2.ListMatchConditionsForEventRow)

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
			out, _, err := program.Eval(map[string]interface{}{
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

func (m *sharedRepository) createEventMatches(ctx context.Context, tx sqlcv2.DBTX, tenantId string, eventMatches []CreateMatchOpts) error {
	// create the event matches first
	dagTenantIds := make([]pgtype.UUID, 0, len(eventMatches))
	dagKinds := make([]string, 0, len(eventMatches))
	triggerDagIds := make([]int64, 0, len(eventMatches))
	triggerDagInsertedAts := make([]pgtype.Timestamptz, 0, len(eventMatches))
	triggerStepIds := make([]pgtype.UUID, 0, len(eventMatches))
	triggerExternalIds := make([]pgtype.UUID, 0, len(eventMatches))

	signalTenantIds := make([]pgtype.UUID, 0, len(eventMatches))
	signalKinds := make([]string, 0, len(eventMatches))
	signalTargetIds := make([]int64, 0, len(eventMatches))
	signalKeys := make([]string, 0, len(eventMatches))

	for _, match := range eventMatches {
		// at the moment, we skip creating matches for things that don't have all fields set
		if match.TriggerDAGId != nil && match.TriggerDAGInsertedAt.Valid && match.TriggerStepId != nil && match.TriggerExternalId != nil {
			dagTenantIds = append(dagTenantIds, sqlchelpers.UUIDFromStr(tenantId))
			dagKinds = append(dagKinds, string(match.Kind))
			triggerDagIds = append(triggerDagIds, *match.TriggerDAGId)
			triggerDagInsertedAts = append(triggerDagInsertedAts, match.TriggerDAGInsertedAt)
			triggerStepIds = append(triggerStepIds, sqlchelpers.UUIDFromStr(*match.TriggerStepId))
			triggerExternalIds = append(triggerExternalIds, sqlchelpers.UUIDFromStr(*match.TriggerExternalId))
		} else if match.SignalTaskId != nil && match.SignalKey != nil {
			signalTenantIds = append(signalTenantIds, sqlchelpers.UUIDFromStr(tenantId))
			signalKinds = append(signalKinds, string(match.Kind))
			signalTargetIds = append(signalTargetIds, *match.SignalTaskId)
			signalKeys = append(signalKeys, *match.SignalKey)
		}
	}

	var createdMatches []*sqlcv2.V2Match

	if len(dagTenantIds) > 0 {
		dagCreatedMatches, err := m.queries.CreateMatchesForDAGTriggers(
			ctx,
			tx,
			sqlcv2.CreateMatchesForDAGTriggersParams{
				Tenantids:             dagTenantIds,
				Kinds:                 dagKinds,
				Triggerdagids:         triggerDagIds,
				Triggerdaginsertedats: triggerDagInsertedAts,
				Triggerstepids:        triggerStepIds,
				Triggerexternalids:    triggerExternalIds,
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
			sqlcv2.CreateMatchesForSignalTriggersParams{
				Tenantids:       signalTenantIds,
				Kinds:           signalKinds,
				Signaltargetids: signalTargetIds,
				Signalkeys:      signalKeys,
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
	params := make([]sqlcv2.CreateMatchConditionsParams, 0, len(eventMatches))

	for i, match := range eventMatches {
		createdMatch := createdMatches[i]

		for _, condition := range match.Conditions {
			params = append(params, sqlcv2.CreateMatchConditionsParams{
				V2MatchID:  createdMatch.ID,
				TenantID:   sqlchelpers.UUIDFromStr(tenantId),
				EventType:  condition.EventType,
				EventKey:   condition.EventKey,
				OrGroupID:  sqlchelpers.UUIDFromStr(condition.GroupId),
				Expression: sqlchelpers.TextFromStr(condition.Expression),
				Action:     condition.Action,
			})
		}
	}

	_, err := m.queries.CreateMatchConditions(ctx, tx, params)

	if err != nil {
		return err
	}

	return nil
}
