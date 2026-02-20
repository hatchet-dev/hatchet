package sqlcv1

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const listMatchConditionsForEventWithHint = `-- name: ListMatchConditionsForEventWithHint :many
WITH input AS (
    SELECT
        event_key, event_resource_hint
    FROM
        (
            SELECT
                unnest($3::text[]) AS event_key,
                unnest($4::text[]) AS event_resource_hint
        ) AS subquery
)
SELECT
    v1_match_id,
    id,
    registered_at,
    event_type,
    m.event_key,
    m.event_resource_hint,
    readable_data_key,
    expression
FROM
    v1_match_condition m, input i
WHERE
    m.tenant_id = $1::uuid
    AND m.event_type = $2::v1_event_type
    AND m.event_key = i.event_key
    AND m.is_satisfied = FALSE
    AND m.event_resource_hint = i.event_resource_hint
`

type ListMatchConditionsForEventWithHintParams struct {
	Tenantid           uuid.UUID   `json:"tenantid"`
	Eventtype          V1EventType `json:"eventtype"`
	Eventkeys          []string    `json:"eventkeys"`
	Eventresourcehints []string    `json:"eventresourcehints"`
}

type ListMatchConditionsForEventRow struct {
	V1MatchID         int64              `json:"v1_match_id"`
	ID                int64              `json:"id"`
	RegisteredAt      pgtype.Timestamptz `json:"registered_at"`
	EventType         V1EventType        `json:"event_type"`
	EventKey          string             `json:"event_key"`
	EventResourceHint pgtype.Text        `json:"event_resource_hint"`
	ReadableDataKey   string             `json:"readable_data_key"`
	Expression        pgtype.Text        `json:"expression"`
}

func (q *Queries) ListMatchConditionsForEventWithHint(ctx context.Context, db DBTX, arg ListMatchConditionsForEventWithHintParams) ([]*ListMatchConditionsForEventRow, error) {
	rows, err := db.Query(ctx, listMatchConditionsForEventWithHint,
		arg.Tenantid,
		arg.Eventtype,
		arg.Eventkeys,
		arg.Eventresourcehints,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListMatchConditionsForEventRow
	for rows.Next() {
		var i ListMatchConditionsForEventRow
		if err := rows.Scan(
			&i.V1MatchID,
			&i.ID,
			&i.RegisteredAt,
			&i.EventType,
			&i.EventKey,
			&i.EventResourceHint,
			&i.ReadableDataKey,
			&i.Expression,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMatchConditionsForEventWithoutHint = `-- name: ListMatchConditionsForEventWithoutHint :many
WITH input AS (
    SELECT
        event_key
    FROM
        (
            SELECT
                unnest($3::text[]) AS event_key
        ) AS subquery
)
SELECT
    v1_match_id,
    id,
    registered_at,
    event_type,
    m.event_key,
    m.event_resource_hint,
    readable_data_key,
    expression
FROM
    v1_match_condition m, input i
WHERE
    m.tenant_id = $1::uuid
    AND m.event_type = $2::v1_event_type
    AND m.event_key = i.event_key
    AND m.is_satisfied = FALSE
    AND m.event_resource_hint IS NULL
`

type ListMatchConditionsForEventWithoutHintParams struct {
	Tenantid  uuid.UUID   `json:"tenantid"`
	Eventtype V1EventType `json:"eventtype"`
	Eventkeys []string    `json:"eventkeys"`
}

func (q *Queries) ListMatchConditionsForEventWithoutHint(ctx context.Context, db DBTX, arg ListMatchConditionsForEventWithoutHintParams) ([]*ListMatchConditionsForEventRow, error) {
	rows, err := db.Query(ctx, listMatchConditionsForEventWithoutHint,
		arg.Tenantid,
		arg.Eventtype,
		arg.Eventkeys,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListMatchConditionsForEventRow
	for rows.Next() {
		var i ListMatchConditionsForEventRow
		if err := rows.Scan(
			&i.V1MatchID,
			&i.ID,
			&i.RegisteredAt,
			&i.EventType,
			&i.EventKey,
			&i.EventResourceHint,
			&i.ReadableDataKey,
			&i.Expression,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const createMatchesForDAGTriggers = `-- name: CreateMatchesForDAGTriggers :many
WITH input AS (
    SELECT
        tenant_id, kind, existing_data, trigger_dag_id, trigger_dag_inserted_at, trigger_step_id, trigger_step_index, trigger_external_id, trigger_workflow_run_id, trigger_existing_task_id, trigger_existing_task_inserted_at, trigger_parent_task_external_id, trigger_parent_task_id, trigger_parent_task_inserted_at, trigger_child_index, trigger_child_key, trigger_priority
    FROM
        (
            SELECT
                unnest($1::uuid[]) AS tenant_id,
                unnest(cast($2::text[] as v1_match_kind[])) AS kind,
                unnest($3::bigint[]) AS trigger_dag_id,
                unnest($4::timestamptz[]) AS trigger_dag_inserted_at,
                unnest($5::uuid[]) AS trigger_step_id,
				unnest($6::bigint[]) AS trigger_step_index,
                unnest($7::uuid[]) AS trigger_external_id,
				unnest($8::uuid[]) AS trigger_workflow_run_id,
                unnest($9::bigint[]) AS trigger_existing_task_id,
				unnest($10::timestamptz[]) AS trigger_existing_task_inserted_at,
				unnest($11::uuid[]) AS trigger_parent_task_external_id,
				unnest($12::bigint[]) AS trigger_parent_task_id,
				unnest($13::timestamptz[]) AS trigger_parent_task_inserted_at,
				unnest($14::bigint[]) AS trigger_child_index,
				unnest($15::text[]) AS trigger_child_key,
				unnest($16::jsonb[]) AS existing_data,
				unnest($17::integer[]) AS trigger_priority
        ) AS subquery
)
INSERT INTO v1_match (
    tenant_id,
    kind,
	existing_data,
    trigger_dag_id,
    trigger_dag_inserted_at,
    trigger_step_id,
	trigger_step_index,
    trigger_external_id,
	trigger_workflow_run_id,
    trigger_existing_task_id,
	trigger_existing_task_inserted_at,
	trigger_parent_task_external_id,
	trigger_parent_task_id,
	trigger_parent_task_inserted_at,
    trigger_child_index,
    trigger_child_key,
	trigger_priority
)
SELECT
    i.tenant_id,
    i.kind,
	i.existing_data,
    i.trigger_dag_id,
    i.trigger_dag_inserted_at,
    i.trigger_step_id,
	i.trigger_step_index,
    i.trigger_external_id,
	i.trigger_workflow_run_id,
    i.trigger_existing_task_id,
	i.trigger_existing_task_inserted_at,
	i.trigger_parent_task_external_id,
	i.trigger_parent_task_id,
	i.trigger_parent_task_inserted_at,
	i.trigger_child_index,
	i.trigger_child_key,
	i.trigger_priority
FROM
    input i
RETURNING
    id, tenant_id, kind, existing_data, is_satisfied, signal_task_id, signal_task_inserted_at, signal_external_id, signal_key, trigger_dag_id, trigger_dag_inserted_at, trigger_step_id, trigger_step_index, trigger_external_id, trigger_workflow_run_id, trigger_existing_task_id, trigger_existing_task_inserted_at, trigger_parent_task_external_id, trigger_parent_task_id, trigger_parent_task_inserted_at, trigger_child_index, trigger_child_key, trigger_priority
`

type CreateMatchesForDAGTriggersParams struct {
	Tenantids                     []uuid.UUID          `json:"tenantids"`
	Kinds                         []string             `json:"kinds"`
	ExistingDatas                 [][]byte             `json:"existingDatas"`
	Triggerdagids                 []int64              `json:"triggerdagids"`
	Triggerdaginsertedats         []pgtype.Timestamptz `json:"triggerdaginsertedats"`
	Triggerstepids                []uuid.UUID          `json:"triggerstepids"`
	Triggerstepindex              []int64              `json:"triggerstepindex"`
	Triggerexternalids            []uuid.UUID          `json:"triggerexternalids"`
	Triggerworkflowrunids         []*uuid.UUID         `json:"triggerworkflowrunids"`
	Triggerexistingtaskids        []pgtype.Int8        `json:"triggerexistingtaskids"`
	Triggerexistingtaskinsertedat []pgtype.Timestamptz `json:"triggerexistingtaskinsertedat"`
	TriggerParentTaskExternalIds  []*uuid.UUID         `json:"triggerparentTaskExternalIds"`
	TriggerParentTaskIds          []pgtype.Int8        `json:"triggerparentTaskIds"`
	TriggerParentTaskInsertedAt   []pgtype.Timestamptz `json:"triggerparentTaskInsertedAt"`
	TriggerChildIndex             []pgtype.Int8        `json:"triggerchildIndex"`
	TriggerChildKey               []pgtype.Text        `json:"triggerchildKey"`
	TriggerPriorities             []pgtype.Int4        `json:"triggerPriorities"`
}

func (q *Queries) CreateMatchesForDAGTriggers(ctx context.Context, db DBTX, arg CreateMatchesForDAGTriggersParams) ([]*V1Match, error) {
	rows, err := db.Query(ctx, createMatchesForDAGTriggers,
		arg.Tenantids,
		arg.Kinds,
		arg.Triggerdagids,
		arg.Triggerdaginsertedats,
		arg.Triggerstepids,
		arg.Triggerstepindex,
		arg.Triggerexternalids,
		arg.Triggerworkflowrunids,
		arg.Triggerexistingtaskids,
		arg.Triggerexistingtaskinsertedat,
		arg.TriggerParentTaskExternalIds,
		arg.TriggerParentTaskIds,
		arg.TriggerParentTaskInsertedAt,
		arg.TriggerChildIndex,
		arg.TriggerChildKey,
		arg.ExistingDatas,
		arg.TriggerPriorities,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*V1Match
	for rows.Next() {
		var i V1Match
		if err := rows.Scan(
			&i.ID,
			&i.TenantID,
			&i.Kind,
			&i.ExistingData,
			&i.IsSatisfied,
			&i.SignalTaskID,
			&i.SignalTaskInsertedAt,
			&i.SignalExternalID,
			&i.SignalKey,
			&i.TriggerDagID,
			&i.TriggerDagInsertedAt,
			&i.TriggerStepID,
			&i.TriggerStepIndex,
			&i.TriggerExternalID,
			&i.TriggerWorkflowRunID,
			&i.TriggerExistingTaskID,
			&i.TriggerExistingTaskInsertedAt,
			&i.TriggerParentTaskExternalID,
			&i.TriggerParentTaskID,
			&i.TriggerParentTaskInsertedAt,
			&i.TriggerChildIndex,
			&i.TriggerChildKey,
			&i.TriggerPriority,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
