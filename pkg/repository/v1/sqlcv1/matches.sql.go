// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: matches.sql

package sqlcv1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateMatchConditionsParams struct {
	V1MatchID         int64                  `json:"v1_match_id"`
	TenantID          pgtype.UUID            `json:"tenant_id"`
	EventType         V1EventType            `json:"event_type"`
	EventKey          string                 `json:"event_key"`
	EventResourceHint pgtype.Text            `json:"event_resource_hint"`
	ReadableDataKey   string                 `json:"readable_data_key"`
	OrGroupID         pgtype.UUID            `json:"or_group_id"`
	Expression        pgtype.Text            `json:"expression"`
	Action            V1MatchConditionAction `json:"action"`
	IsSatisfied       bool                   `json:"is_satisfied"`
	Data              []byte                 `json:"data"`
}

const createMatchesForSignalTriggers = `-- name: CreateMatchesForSignalTriggers :many
WITH input AS (
    SELECT
        tenant_id, kind, signal_task_id, signal_task_inserted_at, signal_external_id, signal_key
    FROM
        (
            SELECT
                unnest($1::uuid[]) AS tenant_id,
                unnest(cast($2::text[] as v1_match_kind[])) AS kind,
                unnest($3::bigint[]) AS signal_task_id,
                unnest($4::timestamptz[]) AS signal_task_inserted_at,
                unnest($5::uuid[]) AS signal_external_id,
                unnest($6::text[]) AS signal_key
        ) AS subquery
)
INSERT INTO v1_match (
    tenant_id,
    kind,
    signal_task_id,
    signal_task_inserted_at,
    signal_external_id,
    signal_key
)
SELECT
    i.tenant_id,
    i.kind,
    i.signal_task_id,
    i.signal_task_inserted_at,
    i.signal_external_id,
    i.signal_key
FROM
    input i
RETURNING
    id, tenant_id, kind, is_satisfied, existing_data, signal_task_id, signal_task_inserted_at, signal_external_id, signal_key, trigger_dag_id, trigger_dag_inserted_at, trigger_step_id, trigger_step_index, trigger_external_id, trigger_workflow_run_id, trigger_parent_task_external_id, trigger_parent_task_id, trigger_parent_task_inserted_at, trigger_child_index, trigger_child_key, trigger_existing_task_id, trigger_existing_task_inserted_at
`

type CreateMatchesForSignalTriggersParams struct {
	Tenantids             []pgtype.UUID        `json:"tenantids"`
	Kinds                 []string             `json:"kinds"`
	Signaltaskids         []int64              `json:"signaltaskids"`
	Signaltaskinsertedats []pgtype.Timestamptz `json:"signaltaskinsertedats"`
	Signalexternalids     []pgtype.UUID        `json:"signalexternalids"`
	Signalkeys            []string             `json:"signalkeys"`
}

func (q *Queries) CreateMatchesForSignalTriggers(ctx context.Context, db DBTX, arg CreateMatchesForSignalTriggersParams) ([]*V1Match, error) {
	rows, err := db.Query(ctx, createMatchesForSignalTriggers,
		arg.Tenantids,
		arg.Kinds,
		arg.Signaltaskids,
		arg.Signaltaskinsertedats,
		arg.Signalexternalids,
		arg.Signalkeys,
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
			&i.IsSatisfied,
			&i.ExistingData,
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
			&i.TriggerParentTaskExternalID,
			&i.TriggerParentTaskID,
			&i.TriggerParentTaskInsertedAt,
			&i.TriggerChildIndex,
			&i.TriggerChildKey,
			&i.TriggerExistingTaskID,
			&i.TriggerExistingTaskInsertedAt,
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

const getSatisfiedMatchConditions = `-- name: GetSatisfiedMatchConditions :many
WITH input AS (
    SELECT
        match_id, condition_id, data
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS match_id,
                unnest($2::bigint[]) AS condition_id,
                unnest($3::jsonb[]) AS data
        ) AS subquery
), locked_conditions AS (
    SELECT
        m.v1_match_id,
        m.id,
        i.data
    FROM
        v1_match_condition m
    JOIN
        input i ON i.match_id = m.v1_match_id AND i.condition_id = m.id
    ORDER BY
        m.id
    FOR UPDATE
), updated_conditions AS (
    UPDATE
        v1_match_condition
    SET
        is_satisfied = TRUE,
        data = c.data
    FROM
        locked_conditions c
    WHERE
        (v1_match_condition.v1_match_id, v1_match_condition.id) = (c.v1_match_id, c.id)
    RETURNING
        v1_match_condition.v1_match_id, v1_match_condition.id
), distinct_match_ids AS (
    SELECT
        DISTINCT v1_match_id
    FROM
        updated_conditions
)
SELECT
    m.id
FROM
    v1_match m
JOIN
    distinct_match_ids dm ON dm.v1_match_id = m.id
ORDER BY
    m.id
FOR UPDATE
`

type GetSatisfiedMatchConditionsParams struct {
	Matchids     []int64  `json:"matchids"`
	Conditionids []int64  `json:"conditionids"`
	Datas        [][]byte `json:"datas"`
}

// NOTE: we have to break this into a separate query because CTEs can't see modified rows
// on the same target table without using RETURNING.
func (q *Queries) GetSatisfiedMatchConditions(ctx context.Context, db DBTX, arg GetSatisfiedMatchConditionsParams) ([]int64, error) {
	rows, err := db.Query(ctx, getSatisfiedMatchConditions, arg.Matchids, arg.Conditionids, arg.Datas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const resetMatchConditions = `-- name: ResetMatchConditions :many
WITH input AS (
    SELECT
        match_id, condition_id, data
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS match_id,
                unnest($2::bigint[]) AS condition_id,
                unnest($3::jsonb[]) AS data
        ) AS subquery
), locked_conditions AS (
    SELECT
        m.v1_match_id,
        m.id,
        i.data
    FROM
        v1_match_condition m
    JOIN
        input i ON i.match_id = m.v1_match_id AND i.condition_id = m.id
    ORDER BY
        m.id
    -- We can afford a SKIP LOCKED because a match condition can only be satisfied by 1 event
    -- at a time
    FOR UPDATE SKIP LOCKED
), updated_conditions AS (
    UPDATE
        v1_match_condition
    SET
        is_satisfied = TRUE,
        data = c.data
    FROM
        locked_conditions c
    WHERE
        (v1_match_condition.v1_match_id, v1_match_condition.id) = (c.v1_match_id, c.id)
    RETURNING
        v1_match_condition.v1_match_id, v1_match_condition.id
), distinct_match_ids AS (
    SELECT
        DISTINCT v1_match_id
    FROM
        updated_conditions
)
SELECT
    m.id
FROM
    v1_match m
JOIN
    distinct_match_ids dm ON dm.v1_match_id = m.id
ORDER BY
    m.id
FOR UPDATE
`

type ResetMatchConditionsParams struct {
	Matchids     []int64  `json:"matchids"`
	Conditionids []int64  `json:"conditionids"`
	Datas        [][]byte `json:"datas"`
}

// NOTE: we have to break this into a separate query because CTEs can't see modified rows
// on the same target table without using RETURNING.
func (q *Queries) ResetMatchConditions(ctx context.Context, db DBTX, arg ResetMatchConditionsParams) ([]int64, error) {
	rows, err := db.Query(ctx, resetMatchConditions, arg.Matchids, arg.Conditionids, arg.Datas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const saveSatisfiedMatchConditions = `-- name: SaveSatisfiedMatchConditions :many
WITH match_counts AS (
    SELECT
        v1_match_id,
        COUNT(DISTINCT CASE WHEN action = 'CREATE' THEN or_group_id END) AS total_create_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'CREATE' THEN or_group_id END) AS satisfied_create_groups,
        COUNT(DISTINCT CASE WHEN action = 'QUEUE' THEN or_group_id END) AS total_queue_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'QUEUE' THEN or_group_id END) AS satisfied_queue_groups,
        COUNT(DISTINCT CASE WHEN action = 'CANCEL' THEN or_group_id END) AS total_cancel_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'CANCEL' THEN or_group_id END) AS satisfied_cancel_groups,
        COUNT(DISTINCT CASE WHEN action = 'SKIP' THEN or_group_id END) AS total_skip_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'SKIP' THEN or_group_id END) AS satisfied_skip_groups,
        COUNT(DISTINCT CASE WHEN action = 'CREATE_MATCH' THEN or_group_id END) AS total_create_match_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'CREATE_MATCH' THEN or_group_id END) AS satisfied_create_match_groups
    FROM v1_match_condition main
    WHERE v1_match_id = ANY($1::bigint[])
    GROUP BY v1_match_id
), result_matches AS (
    SELECT
        m.id, m.tenant_id, m.kind, m.is_satisfied, m.existing_data, m.signal_task_id, m.signal_task_inserted_at, m.signal_external_id, m.signal_key, m.trigger_dag_id, m.trigger_dag_inserted_at, m.trigger_step_id, m.trigger_step_index, m.trigger_external_id, m.trigger_workflow_run_id, m.trigger_parent_task_external_id, m.trigger_parent_task_id, m.trigger_parent_task_inserted_at, m.trigger_child_index, m.trigger_child_key, m.trigger_existing_task_id, m.trigger_existing_task_inserted_at,
        CASE WHEN
            (mc.total_skip_groups > 0 AND mc.total_skip_groups = mc.satisfied_skip_groups) THEN 'SKIP'
            WHEN (mc.total_cancel_groups > 0 AND mc.total_cancel_groups = mc.satisfied_cancel_groups) THEN 'CANCEL'
            WHEN (mc.total_create_groups > 0 AND mc.total_create_groups = mc.satisfied_create_groups) THEN 'CREATE'
            WHEN (mc.total_queue_groups > 0 AND mc.total_queue_groups = mc.satisfied_queue_groups) THEN 'QUEUE'
            WHEN (mc.total_create_match_groups > 0 AND mc.total_create_match_groups = mc.satisfied_create_match_groups) THEN 'CREATE_MATCH'
        END::v1_match_condition_action AS action
    FROM
        v1_match m
    JOIN
        match_counts mc ON m.id = mc.v1_match_id
    WHERE
        (
            (mc.total_create_groups > 0 AND mc.total_create_groups = mc.satisfied_create_groups)
            OR (mc.total_queue_groups > 0 AND mc.total_queue_groups = mc.satisfied_queue_groups)
            OR (mc.total_cancel_groups > 0 AND mc.total_cancel_groups = mc.satisfied_cancel_groups)
            OR (mc.total_skip_groups > 0 AND mc.total_skip_groups = mc.satisfied_skip_groups)
            OR (mc.total_create_match_groups > 0 AND mc.total_create_match_groups = mc.satisfied_create_match_groups)
        )
), locked_conditions AS (
    SELECT
        m.v1_match_id,
        m.id
    FROM
        v1_match_condition m
    JOIN
        result_matches r ON r.id = m.v1_match_id
    ORDER BY
        m.id
    FOR UPDATE
), deleted_conditions AS (
    DELETE FROM
        v1_match_condition
    WHERE
        (v1_match_id, id) IN (SELECT v1_match_id, id FROM locked_conditions)
    RETURNING
        v1_match_id AS id
), matches_with_data AS (
    SELECT
        m.id,
        m.action,
        (
            SELECT jsonb_object_agg(action, aggregated_1)
            FROM (
                SELECT action, jsonb_object_agg(readable_data_key, data_array) AS aggregated_1
                FROM (
                    SELECT mc.action, readable_data_key, jsonb_agg(data) AS data_array
                    FROM v1_match_condition mc
                    WHERE mc.v1_match_id = m.id AND mc.is_satisfied AND mc.action = m.action
                    GROUP BY mc.action, readable_data_key
                ) t
                GROUP BY action
            ) s
        )::jsonb AS mc_aggregated_data
    FROM
        result_matches m
    GROUP BY
        m.id, m.action
), deleted_matches AS (
    DELETE FROM
        v1_match
    WHERE
        id IN (SELECT id FROM deleted_conditions)
)
SELECT
    rm.id, rm.tenant_id, rm.kind, rm.is_satisfied, rm.existing_data, rm.signal_task_id, rm.signal_task_inserted_at, rm.signal_external_id, rm.signal_key, rm.trigger_dag_id, rm.trigger_dag_inserted_at, rm.trigger_step_id, rm.trigger_step_index, rm.trigger_external_id, rm.trigger_workflow_run_id, rm.trigger_parent_task_external_id, rm.trigger_parent_task_id, rm.trigger_parent_task_inserted_at, rm.trigger_child_index, rm.trigger_child_key, rm.trigger_existing_task_id, rm.trigger_existing_task_inserted_at, rm.action,
    COALESCE(rm.existing_data || d.mc_aggregated_data, d.mc_aggregated_data)::jsonb AS mc_aggregated_data
FROM
    result_matches rm
LEFT JOIN
    matches_with_data d ON rm.id = d.id
`

type SaveSatisfiedMatchConditionsRow struct {
	ID                            int64                  `json:"id"`
	TenantID                      pgtype.UUID            `json:"tenant_id"`
	Kind                          V1MatchKind            `json:"kind"`
	IsSatisfied                   bool                   `json:"is_satisfied"`
	ExistingData                  []byte                 `json:"existing_data"`
	SignalTaskID                  pgtype.Int8            `json:"signal_task_id"`
	SignalTaskInsertedAt          pgtype.Timestamptz     `json:"signal_task_inserted_at"`
	SignalExternalID              pgtype.UUID            `json:"signal_external_id"`
	SignalKey                     pgtype.Text            `json:"signal_key"`
	TriggerDagID                  pgtype.Int8            `json:"trigger_dag_id"`
	TriggerDagInsertedAt          pgtype.Timestamptz     `json:"trigger_dag_inserted_at"`
	TriggerStepID                 pgtype.UUID            `json:"trigger_step_id"`
	TriggerStepIndex              pgtype.Int8            `json:"trigger_step_index"`
	TriggerExternalID             pgtype.UUID            `json:"trigger_external_id"`
	TriggerWorkflowRunID          pgtype.UUID            `json:"trigger_workflow_run_id"`
	TriggerParentTaskExternalID   pgtype.UUID            `json:"trigger_parent_task_external_id"`
	TriggerParentTaskID           pgtype.Int8            `json:"trigger_parent_task_id"`
	TriggerParentTaskInsertedAt   pgtype.Timestamptz     `json:"trigger_parent_task_inserted_at"`
	TriggerChildIndex             pgtype.Int8            `json:"trigger_child_index"`
	TriggerChildKey               pgtype.Text            `json:"trigger_child_key"`
	TriggerExistingTaskID         pgtype.Int8            `json:"trigger_existing_task_id"`
	TriggerExistingTaskInsertedAt pgtype.Timestamptz     `json:"trigger_existing_task_inserted_at"`
	Action                        V1MatchConditionAction `json:"action"`
	McAggregatedData              []byte                 `json:"mc_aggregated_data"`
}

// NOTE: we have to break this into a separate query because CTEs can't see modified rows
// on the same target table without using RETURNING.
// Additionally, since we've placed a FOR UPDATE lock in the previous query, we're guaranteeing
// that only one transaction can update these rows,so this should be concurrency-safe.
func (q *Queries) SaveSatisfiedMatchConditions(ctx context.Context, db DBTX, matchids []int64) ([]*SaveSatisfiedMatchConditionsRow, error) {
	rows, err := db.Query(ctx, saveSatisfiedMatchConditions, matchids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*SaveSatisfiedMatchConditionsRow
	for rows.Next() {
		var i SaveSatisfiedMatchConditionsRow
		if err := rows.Scan(
			&i.ID,
			&i.TenantID,
			&i.Kind,
			&i.IsSatisfied,
			&i.ExistingData,
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
			&i.TriggerParentTaskExternalID,
			&i.TriggerParentTaskID,
			&i.TriggerParentTaskInsertedAt,
			&i.TriggerChildIndex,
			&i.TriggerChildKey,
			&i.TriggerExistingTaskID,
			&i.TriggerExistingTaskInsertedAt,
			&i.Action,
			&i.McAggregatedData,
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
