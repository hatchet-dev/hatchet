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
}

type GroupMatchCondition struct {
	GroupId string `validate:"required,uuid"`

	EventType sqlcv2.V2EventType

	EventKey string

	Expression string
}

type MatchRepository interface {
	ProcessInternalEventMatches(ctx context.Context, tenantId string, events []CandidateEventMatch) ([]*sqlcv2.V2Task, error)
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
func (m *MatchRepositoryImpl) ProcessInternalEventMatches(ctx context.Context, tenantId string, events []CandidateEventMatch) ([]*sqlcv2.V2Task, error) {
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
		return []*sqlcv2.V2Task{}, nil
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

	for _, match := range satisfiedMatches {
		if match.TriggerDagID.Valid && match.TriggerDagInsertedAt.Valid {
			dagIds = append(dagIds, match.TriggerDagID.Int64)
			dagInsertedAts = append(dagInsertedAts, match.TriggerDagInsertedAt)
		}
	}

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
				Input:              m.newTaskInput(input, match.McAggregatedData),
				AdditionalMetadata: additionalMetadata,
			}

			if match.TriggerDagID.Valid && match.TriggerDagInsertedAt.Valid {
				opt.DagId = &match.TriggerDagID.Int64
				opt.DagInsertedAt = match.TriggerDagInsertedAt
			}

			taskOpts = append(taskOpts, opt)
		}
	}

	// create tasks
	tasks, err := m.createTasks(ctx, tx, tenantId, taskOpts)

	if err != nil {
		return nil, err
	}

	// commit
	if err := commit(ctx); err != nil {
		return nil, err
	}

	return tasks, nil
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

		err := json.Unmarshal(event.Data, &inputData)

		if err != nil {
			return nil, err
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
	tenantIds := make([]pgtype.UUID, 0, len(eventMatches))
	kinds := make([]string, 0, len(eventMatches))
	triggerDagIds := make([]int64, 0, len(eventMatches))
	triggerDagInsertedAts := make([]pgtype.Timestamptz, 0, len(eventMatches))
	triggerStepIds := make([]pgtype.UUID, 0, len(eventMatches))
	triggerExternalIds := make([]pgtype.UUID, 0, len(eventMatches))

	for _, match := range eventMatches {
		// at the moment, we skip creating matches for things that don't have all fields set
		if match.TriggerDAGId == nil || !match.TriggerDAGInsertedAt.Valid || match.TriggerStepId == nil || match.TriggerExternalId == nil {
			continue
		}

		tenantIds = append(tenantIds, sqlchelpers.UUIDFromStr(tenantId))
		kinds = append(kinds, string(match.Kind))
		triggerDagIds = append(triggerDagIds, *match.TriggerDAGId)
		triggerDagInsertedAts = append(triggerDagInsertedAts, match.TriggerDAGInsertedAt)
		triggerStepIds = append(triggerStepIds, sqlchelpers.UUIDFromStr(*match.TriggerStepId))
		triggerExternalIds = append(triggerExternalIds, sqlchelpers.UUIDFromStr(*match.TriggerExternalId))
	}

	createdMatches, err := m.queries.CreateMatchesForDAGTriggers(
		ctx,
		tx,
		sqlcv2.CreateMatchesForDAGTriggersParams{
			Tenantids:             tenantIds,
			Kinds:                 kinds,
			Triggerdagids:         triggerDagIds,
			Triggerdaginsertedats: triggerDagInsertedAts,
			Triggerstepids:        triggerStepIds,
			Triggerexternalids:    triggerExternalIds,
		},
	)

	if err != nil {
		return err
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
			})
		}
	}

	_, err = m.queries.CreateMatchConditions(ctx, tx, params)

	if err != nil {
		return err
	}

	return nil
}
