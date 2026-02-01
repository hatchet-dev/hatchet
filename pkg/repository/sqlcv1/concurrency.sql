-- name: ListActiveConcurrencyStrategies :many
SELECT
    sc.*
FROM
    v1_step_concurrency sc
JOIN
    "WorkflowVersion" wv ON wv."id" = sc.workflow_version_id
WHERE
    sc.tenant_id = @tenantId::uuid AND
    sc.is_active = TRUE;

-- name: ListConcurrencyStrategiesByWorkflowVersionId :many
SELECT c.*, s."readableId" AS step_readable_id
FROM v1_step_concurrency c
JOIN "Step" s ON s.id = c.step_id
WHERE
    tenant_id = @tenantId::UUID
    AND workflow_version_id = @workflowVersionId::UUID
    AND workflow_id = @workflowId::UUID
    AND c.id NOT IN (
        SELECT UNNEST(child_strategy_ids)
        FROM v1_workflow_concurrency
        WHERE
            tenant_id = @tenantId::UUID
            AND workflow_version_id = @workflowVersionId::UUID
            AND workflow_id = @workflowId::UUID
    )
;

-- name: GetWorkflowConcurrencyQueueCounts :many
SELECT
    w."name" AS "workflowName",
    wcs.key,
    COUNT(*) AS "count"
FROM
    v1_workflow_concurrency_slot wcs
JOIN
    "Workflow" w ON w.id = wcs.workflow_id
WHERE
    wcs.tenant_id = @tenantId::uuid
    AND wcs.is_filled = FALSE
GROUP BY
    w."name",
    wcs.key;

-- name: ListConcurrencyStrategiesByStepId :many
SELECT
    *
FROM
    v1_step_concurrency
WHERE
    tenant_id = @tenantId::uuid AND
    step_id = ANY(@stepIds::uuid[])
;

-- name: CheckStrategyActive :one
-- A strategy is active if the workflow is not deleted, and it is attached to the latest workflow version or it has
-- at least one concurrency slot that is not filled (the concurrency slot could be on the parent).
WITH latest_workflow_version AS (
    SELECT DISTINCT ON("workflowId")
        "workflowId",
        wv."id" AS "workflowVersionId"
    FROM
        "WorkflowVersion" wv
    WHERE
        wv."workflowId" = @workflowId::uuid
        AND wv."deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
    LIMIT 1
), first_active_strategy AS (
    -- Get the first active strategy for the workflow version
    SELECT
        sc.id
    FROM
        v1_step_concurrency sc
    WHERE
        sc.tenant_id = @tenantId::uuid AND
        sc.workflow_id = @workflowId::uuid AND
        sc.workflow_version_id = @workflowVersionId::uuid AND
        sc.is_active = TRUE
    ORDER BY
        sc.id ASC
    LIMIT 1
), active_slot AS (
    SELECT
        *
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = @tenantId::uuid AND
        strategy_id = @strategyId::bigint
        -- Note we don't check for is_filled=False, because is_filled=True could imply that the task
        -- gets retried and the slot is still active.
    LIMIT 1
), active_parent_slot AS (
    SELECT
        wcs.*
    FROM
        v1_concurrency_slot cs
    JOIN
        v1_workflow_concurrency_slot wcs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
    WHERE
        cs.tenant_id = @tenantId::uuid AND
        cs.strategy_id = @strategyId::bigint
        -- Note we don't check for is_filled=False, because is_filled=True could imply that the task
        -- gets retried and the slot is still active.
    LIMIT 1
), is_active AS (
    SELECT
        EXISTS(SELECT 1 FROM latest_workflow_version) AND
        (
            -- We must match the first active strategy, otherwise we could have another concurrency strategy
            -- that is active and has this concurrency strategy as a child.
            (first_active_strategy.id != @strategyId::bigint) OR
            latest_workflow_version."workflowVersionId" = @workflowVersionId::uuid OR
            EXISTS(SELECT 1 FROM active_slot) OR
            EXISTS(SELECT 1 FROM active_parent_slot)
        ) AS "isActive"
    FROM
        latest_workflow_version, first_active_strategy
)
SELECT COALESCE((SELECT "isActive" FROM is_active), FALSE)::bool AS "isActive";

-- name: SetConcurrencyStrategyInactive :exec
UPDATE
    v1_step_concurrency
SET
    is_active = FALSE
WHERE
    workflow_id = @workflowId::uuid AND
    workflow_version_id = @workflowVersionId::uuid AND
    step_id = @stepId::uuid AND
    id = @strategyId::bigint;

-- name: AdvisoryLock :exec
SELECT pg_advisory_xact_lock(@key::bigint);

-- name: TryAdvisoryLock :one
SELECT pg_try_advisory_xact_lock(@key::bigint) AS "locked";

-- name: RunParentGroupRoundRobin :exec
WITH eligible_slots_per_group AS (
    SELECT wsc.*
    FROM (
        SELECT DISTINCT key
        FROM v1_workflow_concurrency_slot
        WHERE
            tenant_id = @tenantId::uuid
            AND strategy_id = @strategyId::bigint
    ) distinct_keys
    JOIN LATERAL (
        SELECT *
        FROM v1_workflow_concurrency_slot wcs_all
        WHERE
            wcs_all.key = distinct_keys.key
            AND wcs_all.tenant_id = @tenantId::uuid
            AND wcs_all.strategy_id = @strategyId::bigint
        ORDER BY wcs_all.priority DESC, wcs_all.sort_id ASC
        LIMIT @maxRuns::int
    ) wsc ON true
), eligible_slots AS (
    SELECT
        *
    FROM
        v1_workflow_concurrency_slot
    WHERE
        (strategy_id, workflow_version_id, workflow_run_id) IN (
            SELECT
                es.strategy_id,
                es.workflow_version_id,
                es.workflow_run_id
            FROM
                eligible_slots_per_group es
        )
        AND is_filled = FALSE
    ORDER BY
        strategy_id, workflow_version_id, workflow_run_id
    FOR UPDATE
)
UPDATE
    v1_workflow_concurrency_slot
SET
    is_filled = TRUE
FROM
    eligible_slots
WHERE
    v1_workflow_concurrency_slot.strategy_id = eligible_slots.strategy_id AND
    v1_workflow_concurrency_slot.workflow_version_id = eligible_slots.workflow_version_id AND
    v1_workflow_concurrency_slot.workflow_run_id = eligible_slots.workflow_run_id;

-- name: RunGroupRoundRobin :many
-- Used for round-robin scheduling when a strategy doesn't have a parent strategy
WITH eligible_slots_per_group AS (
    SELECT cs.*
    FROM (
        SELECT DISTINCT key
        FROM v1_concurrency_slot
        WHERE
            tenant_id = @tenantId::uuid
            AND strategy_id = @strategyId::bigint
    ) distinct_keys
    JOIN LATERAL (
        SELECT *
        FROM v1_concurrency_slot wcs_all
        WHERE
            wcs_all.key = distinct_keys.key
            AND wcs_all.tenant_id = @tenantId::uuid
            AND wcs_all.strategy_id = @strategyId::bigint
        ORDER BY wcs_all.sort_id ASC
        LIMIT @maxRuns::int
    ) cs ON true
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
    LIMIT 1000
), eligible_slots AS (
    SELECT
        cs.*
    FROM
        v1_concurrency_slot cs
    WHERE
        (task_inserted_at, task_id, task_retry_count, tenant_id, strategy_id) IN (
            SELECT
                es.task_inserted_at,
                es.task_id,
                es.task_retry_count,
                es.tenant_id,
                es.strategy_id
            FROM
                eligible_slots_per_group es
        )
        AND is_filled = FALSE
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
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
    'RUNNING' AS "operation"
FROM
    updated_slots;


-- name: RunParentCancelInProgress :exec
WITH locked_workflow_concurrency_slots AS (
    SELECT *
    FROM v1_workflow_concurrency_slot
    WHERE (strategy_id, workflow_version_id, workflow_run_id) IN (
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            tmp_workflow_concurrency_slot
    )
    ORDER BY strategy_id, workflow_version_id, workflow_run_id
    FOR UPDATE
), eligible_running_slots AS (
    SELECT wsc.*
    FROM (
        SELECT DISTINCT key
        FROM locked_workflow_concurrency_slots
        WHERE
            tenant_id = @tenantId::uuid
            AND strategy_id = @strategyId::bigint
    ) distinct_keys
    JOIN LATERAL (
        SELECT *
        FROM locked_workflow_concurrency_slots wcs_all
        WHERE
            wcs_all.key = distinct_keys.key
            AND wcs_all.tenant_id = @tenantId::uuid
            AND wcs_all.strategy_id = @strategyId::bigint
        ORDER BY wcs_all.sort_id DESC
        LIMIT @maxRuns::int
    ) wsc ON true
), slots_to_run AS (
    SELECT
        *
    FROM
        v1_workflow_concurrency_slot
    WHERE
        (strategy_id, workflow_version_id, workflow_run_id) IN (
            SELECT
                ers.strategy_id,
                ers.workflow_version_id,
                ers.workflow_run_id
            FROM
                eligible_running_slots ers
        )
    ORDER BY
        strategy_id, workflow_version_id, workflow_run_id
    FOR UPDATE
), update_tmp_table AS (
    UPDATE
        tmp_workflow_concurrency_slot wsc
    SET
        is_filled = TRUE
    FROM
        slots_to_run
    WHERE
        wsc.strategy_id = slots_to_run.strategy_id AND
        wsc.workflow_version_id = slots_to_run.workflow_version_id AND
        wsc.workflow_run_id = slots_to_run.workflow_run_id
)
UPDATE
    v1_workflow_concurrency_slot wsc
SET
    is_filled = TRUE
FROM
    slots_to_run sr
WHERE
    wsc.strategy_id = sr.strategy_id AND
    wsc.workflow_version_id = sr.workflow_version_id AND
    wsc.workflow_run_id = sr.workflow_run_id;



-- name: RunCancelInProgress :many
WITH slots AS (
    SELECT
        task_id,
        task_inserted_at,
        task_retry_count,
        tenant_id,
        strategy_id,
        key,
        is_filled,
        -- Order slots by rn desc, seqnum desc to ensure that the most recent tasks will be run
        row_number() OVER (PARTITION BY key ORDER BY sort_id DESC) AS rn,
        row_number() OVER (ORDER BY sort_id DESC) AS seqnum
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = @tenantId::uuid AND
        strategy_id = @strategyId::bigint AND
        (
            schedule_timeout_at >= NOW() OR
            is_filled = TRUE
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
        rn <= @maxRuns::int
), slots_to_cancel AS (
    SELECT
        *
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = @tenantId::uuid AND
        strategy_id = @strategyId::bigint AND
        (task_inserted_at, task_id, task_retry_count) NOT IN (
            SELECT
                ers.task_inserted_at,
                ers.task_id,
                ers.task_retry_count
            FROM
                eligible_running_slots ers
        )
    ORDER BY
        task_id, task_inserted_at
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
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
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
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
    'RUNNING' AS "operation"
FROM
    updated_slots;

-- name: RunParentCancelNewest :exec
WITH locked_workflow_concurrency_slots AS (
    SELECT *
    FROM v1_workflow_concurrency_slot
    WHERE (strategy_id, workflow_version_id, workflow_run_id) IN (
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            tmp_workflow_concurrency_slot
    )
    ORDER BY strategy_id, workflow_version_id, workflow_run_id
    FOR UPDATE
), eligible_running_slots AS (
    SELECT wsc.*
    FROM (
        SELECT DISTINCT key
        FROM locked_workflow_concurrency_slots
        WHERE
            tenant_id = @tenantId::uuid
            AND strategy_id = @strategyId::bigint
    ) distinct_keys
    JOIN LATERAL (
        SELECT *
        FROM locked_workflow_concurrency_slots wcs_all
        WHERE
            wcs_all.key = distinct_keys.key
            AND wcs_all.tenant_id = @tenantId::uuid
            AND wcs_all.strategy_id = @strategyId::bigint
        ORDER BY wcs_all.sort_id ASC
        LIMIT @maxRuns::int
    ) wsc ON true
), slots_to_run AS (
    SELECT
        *
    FROM
        v1_workflow_concurrency_slot
    WHERE
        (strategy_id, workflow_version_id, workflow_run_id) IN (
            SELECT
                ers.strategy_id,
                ers.workflow_version_id,
                ers.workflow_run_id
            FROM
                eligible_running_slots ers
        )
    ORDER BY
        strategy_id, workflow_version_id, workflow_run_id
    FOR UPDATE
), update_tmp_table AS (
    UPDATE
        tmp_workflow_concurrency_slot wsc
    SET
        is_filled = TRUE
    FROM
        slots_to_run
    WHERE
        wsc.strategy_id = slots_to_run.strategy_id AND
        wsc.workflow_version_id = slots_to_run.workflow_version_id AND
        wsc.workflow_run_id = slots_to_run.workflow_run_id
)
UPDATE
    v1_workflow_concurrency_slot wsc
SET
    is_filled = TRUE
FROM
    slots_to_run sr
WHERE
    wsc.strategy_id = sr.strategy_id AND
    wsc.workflow_version_id = sr.workflow_version_id AND
    wsc.workflow_run_id = sr.workflow_run_id;



-- name: RunCancelNewest :many
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
    WHERE
        cs.tenant_id = @tenantId::uuid AND
        cs.strategy_id = @strategyId::bigint AND
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
        rn <= @maxRuns::int
), slots_to_cancel AS (
    SELECT
        *
    FROM
        v1_concurrency_slot
    WHERE
        tenant_id = @tenantId::uuid AND
        strategy_id = @strategyId::bigint AND
        (task_inserted_at, task_id, task_retry_count) NOT IN (
            SELECT
                ers.task_inserted_at,
                ers.task_id,
                ers.task_retry_count
            FROM
                eligible_running_slots ers
        )
    ORDER BY
        task_id ASC, task_inserted_at ASC
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
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
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
    task_id,
    task_inserted_at,
    task_retry_count,
    tenant_id,
    next_strategy_ids,
    external_id,
    workflow_run_id,
    queue_to_notify,
    'RUNNING' AS "operation"
FROM
    updated_slots;

-- name: GetConcurrencySlotStatus :many
-- Returns the current concurrency queue status for a task including other tasks in the same queue
WITH task_slots AS (
    SELECT
        cs.strategy_id,
        cs.key,
        cs.sort_id,
        cs.is_filled,
        sc.expression,
        sc.max_concurrency
    FROM v1_concurrency_slot cs
    JOIN v1_step_concurrency sc ON sc.id = cs.strategy_id
    WHERE
        cs.task_id = @taskId::bigint
        AND cs.task_inserted_at = @taskInsertedAt::timestamptz
        AND cs.tenant_id = @tenantId::uuid
),
-- Subquery to get pending tasks with display names (ordered by queue position)
pending_tasks AS (
    SELECT
        ts.strategy_id,
        ts.key,
        cs2.external_id,
        t.display_name,
        ROW_NUMBER() OVER (PARTITION BY ts.strategy_id, ts.key ORDER BY cs2.sort_id) as queue_pos
    FROM task_slots ts
    JOIN v1_concurrency_slot cs2 ON cs2.strategy_id = ts.strategy_id AND cs2.key = ts.key AND cs2.is_filled = FALSE
    JOIN v1_task t ON t.id = cs2.task_id AND t.inserted_at = cs2.task_inserted_at
),
-- Subquery to get running tasks with display names
running_tasks AS (
    SELECT
        ts.strategy_id,
        ts.key,
        cs2.external_id,
        t.display_name
    FROM task_slots ts
    JOIN v1_concurrency_slot cs2 ON cs2.strategy_id = ts.strategy_id AND cs2.key = ts.key AND cs2.is_filled = TRUE
    JOIN v1_task t ON t.id = cs2.task_id AND t.inserted_at = cs2.task_inserted_at
)
SELECT
    ts.key,
    ts.expression,
    ts.max_concurrency,
    ts.is_filled,
    -- Queue position: count of unfilled slots with smaller sort_id
    (
        SELECT COUNT(*)
        FROM v1_concurrency_slot cs2
        WHERE cs2.strategy_id = ts.strategy_id
          AND cs2.key = ts.key
          AND cs2.is_filled = FALSE
          AND cs2.sort_id < ts.sort_id
    )::int AS queue_position,
    -- Pending count: total unfilled slots
    (
        SELECT COUNT(*)
        FROM v1_concurrency_slot cs2
        WHERE cs2.strategy_id = ts.strategy_id
          AND cs2.key = ts.key
          AND cs2.is_filled = FALSE
    )::int AS pending_count,
    -- Running count: filled slots
    (
        SELECT COUNT(*)
        FROM v1_concurrency_slot cs2
        WHERE cs2.strategy_id = ts.strategy_id
          AND cs2.key = ts.key
          AND cs2.is_filled = TRUE
    )::int AS running_count,
    -- Pending task external IDs (ordered by queue position, limited to 10)
    (
        SELECT COALESCE(array_agg(pt.external_id ORDER BY pt.queue_pos), ARRAY[]::uuid[])
        FROM pending_tasks pt
        WHERE pt.strategy_id = ts.strategy_id AND pt.key = ts.key AND pt.queue_pos <= 10
    ) AS pending_task_external_ids,
    -- Pending task display names (ordered by queue position, limited to 10)
    (
        SELECT COALESCE(array_agg(pt.display_name ORDER BY pt.queue_pos), ARRAY[]::text[])
        FROM pending_tasks pt
        WHERE pt.strategy_id = ts.strategy_id AND pt.key = ts.key AND pt.queue_pos <= 10
    ) AS pending_task_display_names,
    -- Running task external IDs (limited to 10)
    (
        SELECT COALESCE(array_agg(rt.external_id), ARRAY[]::uuid[])
        FROM (SELECT * FROM running_tasks rt2 WHERE rt2.strategy_id = ts.strategy_id AND rt2.key = ts.key LIMIT 10) rt
    ) AS running_task_external_ids,
    -- Running task display names (limited to 10)
    (
        SELECT COALESCE(array_agg(rt.display_name), ARRAY[]::text[])
        FROM (SELECT * FROM running_tasks rt2 WHERE rt2.strategy_id = ts.strategy_id AND rt2.key = ts.key LIMIT 10) rt
    ) AS running_task_display_names
FROM task_slots ts;
