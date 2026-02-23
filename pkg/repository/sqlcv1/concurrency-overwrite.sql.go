package sqlcv1

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const runChildGroupRoundRobin = `-- name: RunChildGroupRoundRobin :many
WITH filled_parent_slots AS (
    SELECT sort_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, completed_child_strategy_ids, child_strategy_ids, priority, key, is_filled
    FROM v1_workflow_concurrency_slot wcs
    WHERE
        wcs.tenant_id = $1::uuid
        AND wcs.strategy_id = $2::bigint
        AND wcs.is_filled = TRUE
), eligible_slots_per_group AS (
    SELECT cs_all.sort_id, cs_all.task_id, cs_all.task_inserted_at, cs_all.task_retry_count, cs_all.external_id, cs_all.tenant_id, cs_all.workflow_id, cs_all.workflow_version_id, cs_all.workflow_run_id, cs_all.strategy_id, cs_all.parent_strategy_id, cs_all.priority, cs_all.key, cs_all.is_filled, cs_all.next_parent_strategy_ids, cs_all.next_strategy_ids, cs_all.next_keys, cs_all.queue_to_notify, cs_all.schedule_timeout_at
    FROM v1_concurrency_slot cs_all
    JOIN
        filled_parent_slots wcs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs_all.parent_strategy_id, cs_all.workflow_version_id, cs_all.workflow_run_id)
    WHERE
        cs_all.tenant_id = $1::uuid
        AND cs_all.strategy_id = $3::bigint
        AND (
            cs_all.schedule_timeout_at >= NOW() OR
            cs_all.is_filled = TRUE
        )
), schedule_timeout_slots AS (
    SELECT
        sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = $1::uuid AND
        strategy_id = $3::bigint AND
        schedule_timeout_at < NOW() AND
        is_filled = FALSE
    ORDER BY
        task_id, task_inserted_at
    FOR UPDATE
    LIMIT 1000
), eligible_slots AS (
    SELECT
        cs.sort_id, cs.task_id, cs.task_inserted_at, cs.task_retry_count, cs.external_id, cs.tenant_id, cs.workflow_id, cs.workflow_version_id, cs.workflow_run_id, cs.strategy_id, cs.parent_strategy_id, cs.priority, cs.key, cs.is_filled, cs.next_parent_strategy_ids, cs.next_strategy_ids, cs.next_keys, cs.queue_to_notify, cs.schedule_timeout_at
    FROM
        v1_concurrency_slot cs
    WHERE
        (cs.task_id, cs.task_inserted_at, cs.task_retry_count, cs.strategy_id) IN (
            SELECT
                task_id, task_inserted_at, task_retry_count, strategy_id
            FROM
                eligible_slots_per_group
        )
        AND cs.is_filled = FALSE
    ORDER BY
        task_id, task_inserted_at
    FOR UPDATE
), updated_slots AS (
    UPDATE
        v1_concurrency_slot
    SET
        is_filled = TRUE
    FROM
        eligible_slots
    WHERE
        v1_concurrency_slot.task_id = eligible_slots.task_id AND
        v1_concurrency_slot.task_inserted_at = eligible_slots.task_inserted_at AND
        v1_concurrency_slot.task_retry_count = eligible_slots.task_retry_count AND
        v1_concurrency_slot.tenant_id = eligible_slots.tenant_id AND
        v1_concurrency_slot.strategy_id = eligible_slots.strategy_id AND
        v1_concurrency_slot.key = eligible_slots.key
    RETURNING
        v1_concurrency_slot.sort_id, v1_concurrency_slot.task_id, v1_concurrency_slot.task_inserted_at, v1_concurrency_slot.task_retry_count, v1_concurrency_slot.external_id, v1_concurrency_slot.tenant_id, v1_concurrency_slot.workflow_id, v1_concurrency_slot.workflow_version_id, v1_concurrency_slot.workflow_run_id, v1_concurrency_slot.strategy_id, v1_concurrency_slot.parent_strategy_id, v1_concurrency_slot.priority, v1_concurrency_slot.key, v1_concurrency_slot.is_filled, v1_concurrency_slot.next_parent_strategy_ids, v1_concurrency_slot.next_strategy_ids, v1_concurrency_slot.next_keys, v1_concurrency_slot.queue_to_notify, v1_concurrency_slot.schedule_timeout_at
), deleted_slots AS (
    DELETE FROM
        v1_concurrency_slot
    WHERE
        (task_inserted_at, task_id, task_retry_count) IN (
            SELECT
                c.task_inserted_at,
                c.task_id,
                c.task_retry_count
            FROM
                schedule_timeout_slots c
        )
)
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'RUNNING' AS "operation"
FROM
    updated_slots
`

type RunChildGroupRoundRobinParams struct {
	Tenantid         uuid.UUID `json:"tenantid"`
	Parentstrategyid int64     `json:"parentstrategyid"`
	Strategyid       int64     `json:"strategyid"`
}

type RunChildGroupRoundRobinRow struct {
	SortID                pgtype.Int8        `json:"sort_id"`
	TaskID                int64              `json:"task_id"`
	TaskInsertedAt        pgtype.Timestamptz `json:"task_inserted_at"`
	TaskRetryCount        int32              `json:"task_retry_count"`
	ExternalID            uuid.UUID          `json:"external_id"`
	TenantID              uuid.UUID          `json:"tenant_id"`
	WorkflowID            uuid.UUID          `json:"workflow_id"`
	WorkflowVersionID     uuid.UUID          `json:"workflow_version_id"`
	WorkflowRunID         uuid.UUID          `json:"workflow_run_id"`
	StrategyID            int64              `json:"strategy_id"`
	ParentStrategyID      pgtype.Int8        `json:"parent_strategy_id"`
	Priority              int32              `json:"priority"`
	Key                   string             `json:"key"`
	IsFilled              bool               `json:"is_filled"`
	NextParentStrategyIds []pgtype.Int8      `json:"next_parent_strategy_ids"`
	NextStrategyIds       []int64            `json:"next_strategy_ids"`
	NextKeys              []string           `json:"next_keys"`
	QueueToNotify         string             `json:"queue_to_notify"`
	ScheduleTimeoutAt     pgtype.Timestamp   `json:"schedule_timeout_at"`
	Operation             string             `json:"operation"`
}

// Used for round-robin scheduling when a strategy has a parent strategy. It inherits the concurrency
// settings of the parent, so we just set the is_filled flag to true if the parent slot is filled.
func (q *Queries) RunChildGroupRoundRobin(ctx context.Context, db DBTX, arg RunChildGroupRoundRobinParams) ([]*RunChildGroupRoundRobinRow, error) {
	rows, err := db.Query(ctx, runChildGroupRoundRobin, arg.Tenantid, arg.Parentstrategyid, arg.Strategyid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*RunChildGroupRoundRobinRow
	for rows.Next() {
		var i RunChildGroupRoundRobinRow
		if err := rows.Scan(
			&i.SortID,
			&i.TaskID,
			&i.TaskInsertedAt,
			&i.TaskRetryCount,
			&i.ExternalID,
			&i.TenantID,
			&i.WorkflowID,
			&i.WorkflowVersionID,
			&i.WorkflowRunID,
			&i.StrategyID,
			&i.ParentStrategyID,
			&i.Priority,
			&i.Key,
			&i.IsFilled,
			&i.NextParentStrategyIds,
			&i.NextStrategyIds,
			&i.NextKeys,
			&i.QueueToNotify,
			&i.ScheduleTimeoutAt,
			&i.Operation,
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

const runChildCancelInProgress = `-- name: RunChildCancelInProgress :many
WITH slots AS (
    SELECT
        task_id,
        task_inserted_at,
        task_retry_count,
        cs.tenant_id,
        cs.strategy_id,
        cs.key,
        cs.is_filled,
        -- Order slots by rn desc, seqnum desc to ensure that the most recent tasks will be run
        row_number() OVER (PARTITION BY cs.key ORDER BY cs.sort_id DESC) AS rn,
        row_number() OVER (ORDER BY cs.sort_id DESC) AS seqnum
    FROM
        v1_concurrency_slot cs
    JOIN
        tmp_workflow_concurrency_slot wcs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
    WHERE
        cs.tenant_id = $1::uuid AND
        cs.strategy_id = $2::bigint AND
        (
            cs.parent_strategy_id IS NULL OR
            wcs.is_filled = TRUE
        ) AND
        (
            schedule_timeout_at >= NOW() OR
            cs.is_filled = TRUE
        )
), schedule_timeout_slots AS (
    SELECT
        sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = $1::uuid AND
        strategy_id = $2::bigint AND
        schedule_timeout_at < NOW() AND
        is_filled = FALSE
    LIMIT 1000
), eligible_running_slots AS (
    SELECT
        task_id,
        task_inserted_at,
        task_retry_count,
        tenant_id,
        strategy_id,
        key,
        is_filled,
        rn,
        seqnum
    FROM
        slots
    WHERE
        rn <= $3::int
), all_slots AS (
    SELECT
        sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
        CASE
            WHEN (task_inserted_at, task_id, task_retry_count, tenant_id, strategy_id) IN (
                SELECT
                    ers.task_inserted_at,
                    ers.task_id,
                    ers.task_retry_count,
                    ers.tenant_id,
                    ers.strategy_id
                FROM
                    eligible_running_slots ers
                ORDER BY
                    rn, seqnum
            ) THEN 'run'
            WHEN (
                tenant_id = $1::uuid AND
                strategy_id = $2::bigint AND
                (task_inserted_at, task_id, task_retry_count) NOT IN (
                    SELECT
                        ers.task_inserted_at,
                        ers.task_id,
                        ers.task_retry_count
                    FROM
                        eligible_running_slots ers
                ) AND
                (parent_strategy_id, workflow_version_id, workflow_run_id) IN (
                    SELECT wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
                    FROM
                        tmp_workflow_concurrency_slot wcs
                )
            ) THEN 'cancel'
            ELSE NULL
        END AS operation
    FROM
        v1_concurrency_slot
    WHERE
        (task_inserted_at, task_id, task_retry_count, tenant_id, strategy_id) IN (
            SELECT
                ers.task_inserted_at,
                ers.task_id,
                ers.task_retry_count,
                ers.tenant_id,
                ers.strategy_id
            FROM
                eligible_running_slots ers
            ORDER BY
                rn, seqnum
        ) OR (
            tenant_id = $1::uuid AND
            strategy_id = $2::bigint AND
            (task_inserted_at, task_id, task_retry_count) NOT IN (
                SELECT
                    ers.task_inserted_at,
                    ers.task_id,
                    ers.task_retry_count
                FROM
                    eligible_running_slots ers
            ) AND
            (parent_strategy_id, workflow_version_id, workflow_run_id) IN (
                SELECT wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
                FROM tmp_workflow_concurrency_slot wcs
            )
        )
    ORDER BY task_id ASC, task_inserted_at ASC, task_retry_count ASC
    FOR UPDATE
), updated_slots AS (
    UPDATE
        v1_concurrency_slot
    SET
        is_filled = TRUE
    FROM
        all_slots s
    WHERE
        v1_concurrency_slot.task_id = s.task_id AND
        v1_concurrency_slot.task_inserted_at = s.task_inserted_at AND
        v1_concurrency_slot.task_retry_count = s.task_retry_count AND
        v1_concurrency_slot.tenant_id = s.tenant_id AND
        v1_concurrency_slot.strategy_id = s.strategy_id AND
        v1_concurrency_slot.key = s.key AND
        v1_concurrency_slot.is_filled = FALSE AND
        s.operation = 'run'
    RETURNING
        v1_concurrency_slot.sort_id, v1_concurrency_slot.task_id, v1_concurrency_slot.task_inserted_at, v1_concurrency_slot.task_retry_count, v1_concurrency_slot.external_id, v1_concurrency_slot.tenant_id, v1_concurrency_slot.workflow_id, v1_concurrency_slot.workflow_version_id, v1_concurrency_slot.workflow_run_id, v1_concurrency_slot.strategy_id, v1_concurrency_slot.parent_strategy_id, v1_concurrency_slot.priority, v1_concurrency_slot.key, v1_concurrency_slot.is_filled, v1_concurrency_slot.next_parent_strategy_ids, v1_concurrency_slot.next_strategy_ids, v1_concurrency_slot.next_keys, v1_concurrency_slot.queue_to_notify, v1_concurrency_slot.schedule_timeout_at
), deleted_slots AS (
    DELETE FROM
        v1_concurrency_slot
    WHERE
        (task_inserted_at, task_id, task_retry_count) IN (
            SELECT
                s.task_inserted_at,
                s.task_id,
                s.task_retry_count
            FROM
                all_slots s
            WHERE
                s.operation = 'cancel'
        )
)
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'CANCELLED' AS "operation"
FROM
    all_slots
WHERE
    -- not in the schedule_timeout_slots
    (task_inserted_at, task_id, task_retry_count) NOT IN (
        SELECT
            c.task_inserted_at,
            c.task_id,
            c.task_retry_count
        FROM
            schedule_timeout_slots c
    )
    AND operation = 'cancel'
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'RUNNING' AS "operation"
FROM
    updated_slots
`

type RunChildCancelInProgressParams struct {
	Tenantid   uuid.UUID `json:"tenantid"`
	Strategyid int64     `json:"strategyid"`
	Maxruns    int32     `json:"maxruns"`
}

type RunChildCancelInProgressRow struct {
	SortID                pgtype.Int8        `json:"sort_id"`
	TaskID                int64              `json:"task_id"`
	TaskInsertedAt        pgtype.Timestamptz `json:"task_inserted_at"`
	TaskRetryCount        int32              `json:"task_retry_count"`
	ExternalID            uuid.UUID          `json:"external_id"`
	TenantID              uuid.UUID          `json:"tenant_id"`
	WorkflowID            uuid.UUID          `json:"workflow_id"`
	WorkflowVersionID     uuid.UUID          `json:"workflow_version_id"`
	WorkflowRunID         uuid.UUID          `json:"workflow_run_id"`
	StrategyID            int64              `json:"strategy_id"`
	ParentStrategyID      pgtype.Int8        `json:"parent_strategy_id"`
	Priority              int32              `json:"priority"`
	Key                   string             `json:"key"`
	IsFilled              bool               `json:"is_filled"`
	NextParentStrategyIds []pgtype.Int8      `json:"next_parent_strategy_ids"`
	NextStrategyIds       []int64            `json:"next_strategy_ids"`
	NextKeys              []string           `json:"next_keys"`
	QueueToNotify         string             `json:"queue_to_notify"`
	ScheduleTimeoutAt     pgtype.Timestamp   `json:"schedule_timeout_at"`
	Operation             string             `json:"operation"`
}

func (q *Queries) RunChildCancelInProgress(ctx context.Context, db DBTX, arg RunChildCancelInProgressParams) ([]*RunChildCancelInProgressRow, error) {
	rows, err := db.Query(ctx, runChildCancelInProgress, arg.Tenantid, arg.Strategyid, arg.Maxruns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*RunChildCancelInProgressRow
	for rows.Next() {
		var i RunChildCancelInProgressRow
		if err := rows.Scan(
			&i.SortID,
			&i.TaskID,
			&i.TaskInsertedAt,
			&i.TaskRetryCount,
			&i.ExternalID,
			&i.TenantID,
			&i.WorkflowID,
			&i.WorkflowVersionID,
			&i.WorkflowRunID,
			&i.StrategyID,
			&i.ParentStrategyID,
			&i.Priority,
			&i.Key,
			&i.IsFilled,
			&i.NextParentStrategyIds,
			&i.NextStrategyIds,
			&i.NextKeys,
			&i.QueueToNotify,
			&i.ScheduleTimeoutAt,
			&i.Operation,
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

const runChildCancelNewest = `-- name: RunChildCancelNewest :many
WITH slots AS (
    SELECT
        task_id,
        task_inserted_at,
        task_retry_count,
        cs.tenant_id,
        cs.strategy_id,
        cs.key,
        cs.is_filled,
        -- Order slots by rn desc, seqnum desc to ensure that the most recent tasks will be run
        row_number() OVER (PARTITION BY cs.key ORDER BY cs.sort_id ASC) AS rn,
        row_number() OVER (ORDER BY cs.sort_id ASC) AS seqnum
    FROM
        v1_concurrency_slot cs
    JOIN
        tmp_workflow_concurrency_slot wcs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
    WHERE
        cs.tenant_id = $1::uuid AND
        cs.strategy_id = $2::bigint AND
        (
            cs.parent_strategy_id IS NULL OR
            wcs.is_filled = TRUE
        ) AND
        (
            schedule_timeout_at >= NOW() OR
            cs.is_filled = TRUE
        )
), schedule_timeout_slots AS (
    SELECT
        sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = $1::uuid AND
        strategy_id = $2::bigint AND
        schedule_timeout_at < NOW() AND
        is_filled = FALSE
    LIMIT 1000
), eligible_running_slots AS (
    SELECT
        task_id,
        task_inserted_at,
        task_retry_count,
        tenant_id,
        strategy_id,
        key,
        is_filled,
        rn,
        seqnum
    FROM
        slots
    WHERE
        rn <= $3::int
), all_slots AS (
    SELECT
        cs.sort_id, cs.task_id, cs.task_inserted_at, cs.task_retry_count, cs.external_id, cs.tenant_id, cs.workflow_id, cs.workflow_version_id, cs.workflow_run_id, cs.strategy_id, cs.parent_strategy_id, cs.priority, cs.key, cs.is_filled, cs.next_parent_strategy_ids, cs.next_strategy_ids, cs.next_keys, cs.queue_to_notify, cs.schedule_timeout_at,
        CASE
            WHEN (cs.task_inserted_at, cs.task_id, cs.task_retry_count, cs.tenant_id, cs.strategy_id) IN (
                SELECT
                    ers.task_inserted_at,
                    ers.task_id,
                    ers.task_retry_count,
                    ers.tenant_id,
                    ers.strategy_id
                FROM
                    eligible_running_slots ers
                ORDER BY
                    rn, seqnum
            ) THEN 'run'
            WHEN (
                cs.tenant_id = $1::uuid AND
                cs.strategy_id = $2::bigint AND
                (cs.task_inserted_at, cs.task_id, cs.task_retry_count) NOT IN (
                    SELECT
                        ers.task_inserted_at,
                        ers.task_id,
                        ers.task_retry_count
                    FROM
                        eligible_running_slots ers
                ) AND
                (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id) IN (
                    SELECT wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
                    FROM
                        tmp_workflow_concurrency_slot wcs
                )
            ) THEN 'cancel'
            ELSE NULL
        END AS operation
    FROM
        v1_concurrency_slot cs
    WHERE
        (cs.task_inserted_at, cs.task_id, cs.task_retry_count, cs.tenant_id, cs.strategy_id) IN (
            SELECT
                ers.task_inserted_at,
                ers.task_id,
                ers.task_retry_count,
                ers.tenant_id,
                ers.strategy_id
            FROM
                eligible_running_slots ers
            ORDER BY
                rn, seqnum
        ) OR (
            cs.tenant_id = $1::uuid AND
            cs.strategy_id = $2::bigint AND
            (cs.task_inserted_at, cs.task_id, cs.task_retry_count) NOT IN (
                SELECT
                    ers.task_inserted_at,
                    ers.task_id,
                    ers.task_retry_count
                FROM
                    eligible_running_slots ers
            ) AND
            (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id) IN (
                SELECT wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
                FROM tmp_workflow_concurrency_slot wcs
            )
        )
    ORDER BY
        cs.task_id ASC, cs.task_inserted_at ASC, cs.task_retry_count ASC
    FOR UPDATE
), updated_slots AS (
    UPDATE
        v1_concurrency_slot
    SET
        is_filled = TRUE
    FROM
        all_slots s
    WHERE
        v1_concurrency_slot.task_id = s.task_id AND
        v1_concurrency_slot.task_inserted_at = s.task_inserted_at AND
        v1_concurrency_slot.task_retry_count = s.task_retry_count AND
        v1_concurrency_slot.tenant_id = s.tenant_id AND
        v1_concurrency_slot.strategy_id = s.strategy_id AND
        v1_concurrency_slot.key = s.key AND
        v1_concurrency_slot.is_filled = FALSE AND
        s.operation = 'run'
    RETURNING
        v1_concurrency_slot.sort_id, v1_concurrency_slot.task_id, v1_concurrency_slot.task_inserted_at, v1_concurrency_slot.task_retry_count, v1_concurrency_slot.external_id, v1_concurrency_slot.tenant_id, v1_concurrency_slot.workflow_id, v1_concurrency_slot.workflow_version_id, v1_concurrency_slot.workflow_run_id, v1_concurrency_slot.strategy_id, v1_concurrency_slot.parent_strategy_id, v1_concurrency_slot.priority, v1_concurrency_slot.key, v1_concurrency_slot.is_filled, v1_concurrency_slot.next_parent_strategy_ids, v1_concurrency_slot.next_strategy_ids, v1_concurrency_slot.next_keys, v1_concurrency_slot.queue_to_notify, v1_concurrency_slot.schedule_timeout_at
), deleted_slots AS (
    DELETE FROM
        v1_concurrency_slot
    WHERE
        (task_inserted_at, task_id, task_retry_count) IN (
            SELECT
                s.task_inserted_at,
                s.task_id,
                s.task_retry_count
            FROM
                all_slots s
            WHERE
                s.operation = 'cancel'
        )
)
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'CANCELLED' AS "operation"
FROM
    all_slots
WHERE
    -- not in the schedule_timeout_slots
    (task_inserted_at, task_id, task_retry_count) NOT IN (
        SELECT
            c.task_inserted_at,
            c.task_id,
            c.task_retry_count
        FROM
            schedule_timeout_slots c
    )
    AND operation = 'cancel'
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'RUNNING' AS "operation"
FROM
    updated_slots
`

type RunChildCancelNewestParams struct {
	Tenantid   uuid.UUID `json:"tenantid"`
	Strategyid int64     `json:"strategyid"`
	Maxruns    int32     `json:"maxruns"`
}

type RunChildCancelNewestRow struct {
	SortID                pgtype.Int8        `json:"sort_id"`
	TaskID                int64              `json:"task_id"`
	TaskInsertedAt        pgtype.Timestamptz `json:"task_inserted_at"`
	TaskRetryCount        int32              `json:"task_retry_count"`
	ExternalID            uuid.UUID          `json:"external_id"`
	TenantID              uuid.UUID          `json:"tenant_id"`
	WorkflowID            uuid.UUID          `json:"workflow_id"`
	WorkflowVersionID     uuid.UUID          `json:"workflow_version_id"`
	WorkflowRunID         uuid.UUID          `json:"workflow_run_id"`
	StrategyID            int64              `json:"strategy_id"`
	ParentStrategyID      pgtype.Int8        `json:"parent_strategy_id"`
	Priority              int32              `json:"priority"`
	Key                   string             `json:"key"`
	IsFilled              bool               `json:"is_filled"`
	NextParentStrategyIds []pgtype.Int8      `json:"next_parent_strategy_ids"`
	NextStrategyIds       []int64            `json:"next_strategy_ids"`
	NextKeys              []string           `json:"next_keys"`
	QueueToNotify         string             `json:"queue_to_notify"`
	ScheduleTimeoutAt     pgtype.Timestamp   `json:"schedule_timeout_at"`
	Operation             string             `json:"operation"`
}

func (q *Queries) RunChildCancelNewest(ctx context.Context, db DBTX, arg RunChildCancelNewestParams) ([]*RunChildCancelNewestRow, error) {
	rows, err := db.Query(ctx, runChildCancelNewest, arg.Tenantid, arg.Strategyid, arg.Maxruns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*RunChildCancelNewestRow
	for rows.Next() {
		var i RunChildCancelNewestRow
		if err := rows.Scan(
			&i.SortID,
			&i.TaskID,
			&i.TaskInsertedAt,
			&i.TaskRetryCount,
			&i.ExternalID,
			&i.TenantID,
			&i.WorkflowID,
			&i.WorkflowVersionID,
			&i.WorkflowRunID,
			&i.StrategyID,
			&i.ParentStrategyID,
			&i.Priority,
			&i.Key,
			&i.IsFilled,
			&i.NextParentStrategyIds,
			&i.NextStrategyIds,
			&i.NextKeys,
			&i.QueueToNotify,
			&i.ScheduleTimeoutAt,
			&i.Operation,
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
