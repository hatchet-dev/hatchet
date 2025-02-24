-- name: ListActiveConcurrencyStrategies :many
WITH earliest_workflow_versions AS (
    -- We select the earliest workflow versions with an active concurrency queue. The reason
    -- is that we'd like to drain previous concurrency queues gracefully, otherwise we risk
    -- running more tasks than we should.
    SELECT DISTINCT ON("workflowId")
        "workflowId",
        wv."id" AS "workflowVersionId"
    FROM
        v1_step_concurrency sc
    JOIN
        "WorkflowVersion" wv ON wv."id" = sc.workflow_version_id
    WHERE
        sc.tenant_id = @tenantId::uuid AND
        sc.is_active = TRUE
    ORDER BY
        "workflowId", wv."order" ASC
)
SELECT
    sc.*
FROM
    v1_step_concurrency sc
JOIN
    "WorkflowVersion" wv ON wv."id" = sc.workflow_version_id
JOIN
    earliest_workflow_versions ewv ON ewv."workflowVersionId" = wv."id"
WHERE
    sc.tenant_id = @tenantId::uuid AND
    sc.is_active = TRUE;

-- name: ListConcurrencyStrategiesByStepId :many
SELECT
    *
FROM
    v1_step_concurrency
WHERE
    tenant_id = @tenantId::uuid AND
    step_id = ANY(@stepIds::uuid[])
ORDER BY
    id ASC;

-- name: CheckStrategyActive :one
-- A strategy is active if the workflow is not deleted, and it is attached to the latest workflow version or it has
-- at least one concurrency slot that is not filled.
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
), is_active AS (
    SELECT
        EXISTS(SELECT 1 FROM latest_workflow_version) AND
        (
            -- We must match the first active strategy, otherwise we could have another concurrency strategy
            -- that is active and has this concurrency strategy as a child.
            (first_active_strategy.id != @strategyId::bigint) OR
            latest_workflow_version."workflowVersionId" = @workflowVersionId::uuid OR
            EXISTS(SELECT 1 FROM active_slot)
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

-- name: ConcurrencyAdvisoryLock :exec
SELECT pg_advisory_xact_lock(@key::bigint);

-- name: RunGroupRoundRobin :many
WITH slots AS (
    SELECT
        task_id,
        task_inserted_at,
        task_retry_count,
        key,
        strategy_id,
        tenant_id,
        is_filled,
        row_number() OVER (PARTITION BY key ORDER BY priority DESC, task_id ASC, task_inserted_at ASC) AS rn,
        row_number() OVER (ORDER BY priority DESC, task_id ASC, task_inserted_at ASC) AS seqnum
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
    ORDER BY
        task_id, task_inserted_at
    FOR UPDATE
), eligible_slots_per_group AS (
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
), eligible_slots AS (
    SELECT
        *
    FROM
        v1_concurrency_slot
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
            ORDER BY
                rn, seqnum
            LIMIT (@maxRuns::int) * (SELECT COUNT(DISTINCT key) FROM slots)
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
    *,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    *,
    'RUNNING' AS "operation"
FROM
    updated_slots;

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
        row_number() OVER (PARTITION BY key ORDER BY task_id DESC, task_inserted_at DESC) AS rn,
        row_number() OVER (ORDER BY task_id DESC, task_inserted_at DESC) AS seqnum
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
    *,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    *,
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
    *,
    'RUNNING' AS "operation"
FROM
    updated_slots;

-- name: RunCancelNewest :many
WITH slots AS (
    SELECT
        task_id,
        task_inserted_at,
        task_retry_count,
        tenant_id,
        strategy_id,
        key,
        is_filled,
        row_number() OVER (PARTITION BY key ORDER BY task_id ASC, task_inserted_at ASC) AS rn,
        row_number() OVER (ORDER BY task_id ASC, task_inserted_at ASC) AS seqnum
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
    *,
    'SCHEDULING_TIMED_OUT' AS "operation"
FROM
    schedule_timeout_slots
UNION ALL
SELECT
    *,
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
    *,
    'RUNNING' AS "operation"
FROM
    updated_slots;
