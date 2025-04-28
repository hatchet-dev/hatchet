-- name: RunChildGroupRoundRobin :many
-- Used for round-robin scheduling when a strategy has a parent strategy. It inherits the concurrency
-- settings of the parent, so we just set the is_filled flag to true if the parent slot is filled.
WITH filled_parent_slots AS (
    SELECT *
    FROM v1_workflow_concurrency_slot wcs
    WHERE
        wcs.tenant_id = @tenantId::uuid
        AND wcs.strategy_id = @parentStrategyId::bigint
        AND wcs.is_filled = TRUE
), eligible_slots_per_group AS (
    SELECT cs_all.*
    FROM v1_concurrency_slot cs_all
    JOIN
        filled_parent_slots wcs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs_all.parent_strategy_id, cs_all.workflow_version_id, cs_all.workflow_run_id)
    WHERE
        cs_all.tenant_id = @tenantId::uuid
        AND cs_all.strategy_id = @strategyId::bigint
        AND (
            cs_all.schedule_timeout_at >= NOW() OR
            cs_all.is_filled = TRUE
        )
), schedule_timeout_slots AS (
    SELECT
        *
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = @tenantId::uuid AND
        strategy_id = @strategyId::bigint AND
        schedule_timeout_at < NOW() AND
        is_filled = FALSE
    ORDER BY
        task_id, task_inserted_at
    FOR UPDATE
), eligible_slots AS (
    SELECT
        cs.*
    FROM
        v1_concurrency_slot cs
    JOIN
        eligible_slots_per_group es ON cs.task_id = es.task_id
    WHERE
        cs.task_inserted_at = es.task_inserted_at
        AND cs.task_retry_count = es.task_retry_count
        AND cs.strategy_id = es.strategy_id
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
        v1_concurrency_slot.*
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
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'RUNNING' AS "operation"
FROM
    updated_slots;

-- name: RunChildCancelInProgress :many
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
        cs.tenant_id = @tenantId::uuid AND
        cs.strategy_id = @strategyId::bigint AND
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
        *
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = @tenantId::uuid AND
        strategy_id = @strategyId::bigint AND
        schedule_timeout_at < NOW() AND
        is_filled = FALSE
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
        rn <= @maxRuns::int
), slots_to_cancel AS (
    SELECT
        cs.*
    FROM
        v1_concurrency_slot cs
    JOIN
        tmp_workflow_concurrency_slot wcs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
    WHERE
        cs.tenant_id = @tenantId::uuid AND
        cs.strategy_id = @strategyId::bigint AND
        (cs.task_inserted_at, cs.task_id, cs.task_retry_count) NOT IN (
            SELECT
                ers.task_inserted_at,
                ers.task_id,
                ers.task_retry_count
            FROM
                eligible_running_slots ers
        )
    ORDER BY
        cs.task_id, cs.task_inserted_at
    FOR UPDATE
), slots_to_run AS (
    SELECT
        *
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
        )
    ORDER BY
        task_id, task_inserted_at
    FOR UPDATE
), updated_slots AS (
    UPDATE
        v1_concurrency_slot
    SET
        is_filled = TRUE
    FROM
        slots_to_run
    WHERE
        v1_concurrency_slot.task_id = slots_to_run.task_id AND
        v1_concurrency_slot.task_inserted_at = slots_to_run.task_inserted_at AND
        v1_concurrency_slot.task_retry_count = slots_to_run.task_retry_count AND
        v1_concurrency_slot.key = slots_to_run.key AND
        v1_concurrency_slot.is_filled = FALSE
    RETURNING
        v1_concurrency_slot.*
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
                slots_to_cancel c
        )
)
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'CANCELLED' AS "operation"
FROM
    slots_to_cancel
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
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'RUNNING' AS "operation"
FROM
    updated_slots;

-- name: RunChildCancelNewest :many
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
        cs.tenant_id = @tenantId::uuid AND
        cs.strategy_id = @strategyId::bigint AND
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
        *
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = @tenantId::uuid AND
        strategy_id = @strategyId::bigint AND
        schedule_timeout_at < NOW() AND
        is_filled = FALSE
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
        rn <= @maxRuns::int
), slots_to_cancel AS (
    SELECT
        cs.*
    FROM
        v1_concurrency_slot cs
    JOIN
        tmp_workflow_concurrency_slot wcs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
    WHERE
        cs.tenant_id = @tenantId::uuid AND
        cs.strategy_id = @strategyId::bigint AND
        (cs.task_inserted_at, cs.task_id, cs.task_retry_count) NOT IN (
            SELECT
                ers.task_inserted_at,
                ers.task_id,
                ers.task_retry_count
            FROM
                eligible_running_slots ers
        )
    ORDER BY
        cs.task_id ASC, cs.task_inserted_at ASC
    FOR UPDATE
), slots_to_run AS (
    SELECT
        *
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
        )
    ORDER BY
        task_id ASC, task_inserted_at ASC
    FOR UPDATE
), updated_slots AS (
    UPDATE
        v1_concurrency_slot
    SET
        is_filled = TRUE
    FROM
        slots_to_run
    WHERE
        v1_concurrency_slot.task_id = slots_to_run.task_id AND
        v1_concurrency_slot.task_inserted_at = slots_to_run.task_inserted_at AND
        v1_concurrency_slot.task_retry_count = slots_to_run.task_retry_count AND
        v1_concurrency_slot.key = slots_to_run.key AND
        v1_concurrency_slot.is_filled = FALSE
    RETURNING
        v1_concurrency_slot.*
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
                slots_to_cancel c
        )
)
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'CANCELLED' AS "operation"
FROM
    slots_to_cancel
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
UNION ALL
SELECT
    sort_id, task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, parent_strategy_id, priority, key, is_filled, ARRAY_REPLACE(next_parent_strategy_ids, NULL, -1)::BIGINT[] AS next_parent_strategy_ids, next_strategy_ids, next_keys, queue_to_notify, schedule_timeout_at,
    'RUNNING' AS "operation"
FROM
    updated_slots;
