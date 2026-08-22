-- name: GetAndLockLogFile :one
SELECT *
FROM v1_durable_event_log_file
WHERE
    durable_task_id = @durableTaskId::BIGINT
    AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
    AND tenant_id = @tenantId::UUID
FOR UPDATE
;

-- name: GetAndLockLogFileWithBranchPoints :many
WITH locked_file AS (
    SELECT *
    FROM v1_durable_event_log_file
    WHERE
        durable_task_id = @durableTaskId::BIGINT
        AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
        AND tenant_id = @tenantId::UUID
    FOR UPDATE
)

SELECT
    sqlc.embed(to_embed),
    bp.*
FROM locked_file lf
-- note: intentionally using the params for the join so we can prune partitions
JOIN v1_durable_event_log_file to_embed
    ON (to_embed.durable_task_id, to_embed.durable_task_inserted_at, to_embed.tenant_id) = (@durableTaskId::BIGINT, @durableTaskInsertedAt::TIMESTAMPTZ, @tenantId::UUID)
LEFT JOIN v1_durable_event_log_branch_point bp
    ON (bp.durable_task_id, bp.durable_task_inserted_at, bp.tenant_id) = (@durableTaskId::BIGINT, @durableTaskInsertedAt::TIMESTAMPTZ, @tenantId::UUID)
;

-- name: IncrementLogFileInvocationCounts :many
WITH inputs AS (
    SELECT
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@tenantIds::UUID[]) AS tenant_id
)

INSERT INTO v1_durable_event_log_file (
    tenant_id,
    durable_task_id,
    durable_task_inserted_at,
    latest_invocation_count,
    latest_inserted_at,
    latest_node_id,
    latest_branch_id
)
SELECT
    tenant_id,
    durable_task_id,
    durable_task_inserted_at,
    1,
    NOW(),
    0,
    1
FROM inputs
ON CONFLICT (durable_task_id, durable_task_inserted_at) DO UPDATE
SET
    latest_invocation_count = v1_durable_event_log_file.latest_invocation_count + 1,
    latest_node_id = 0
RETURNING v1_durable_event_log_file.*
;

-- name: UpdateLogFile :one
UPDATE v1_durable_event_log_file
SET
    -- important: need `GREATEST` here to avoid moving the `latest_node_id` backwards in the case of child spawning with
    -- a child_key set, which, if the child was cached, would not create a new log entry and thus not move the latest node forward
    latest_node_id = GREATEST(v1_durable_event_log_file.latest_node_id, COALESCE(sqlc.narg('nodeId')::BIGINT, v1_durable_event_log_file.latest_node_id)),
    latest_invocation_count = COALESCE(sqlc.narg('invocationCount')::INTEGER, v1_durable_event_log_file.latest_invocation_count),
    latest_branch_id = COALESCE(sqlc.narg('branchId')::BIGINT, v1_durable_event_log_file.latest_branch_id),
    latest_satisfied_order = COALESCE(sqlc.narg('latestSatisfiedOrder')::BIGINT, v1_durable_event_log_file.latest_satisfied_order)
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
RETURNING *;

-- name: CreateDurableEventLogBranchPoint :exec
INSERT INTO v1_durable_event_log_branch_point (
    tenant_id,
    durable_task_id,
    durable_task_inserted_at,
    first_node_id_in_new_branch,
    parent_branch_id,
    next_branch_id,
    replay_child_external_ids
)
VALUES (
    @tenantId::UUID,
    @durableTaskId::BIGINT,
    @durableTaskInsertedAt::TIMESTAMPTZ,
    @firstNodeIdInNewBranch::BIGINT,
    @parentBranchId::BIGINT,
    @nextBranchId::BIGINT,
    sqlc.narg('replayChildExternalIds')::UUID[]
)
RETURNING *
;

-- name: GetDurableEventLogEntry :one
SELECT *
FROM v1_durable_event_log_entry
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND branch_id = @branchId::BIGINT
  AND node_id = @nodeId::BIGINT;


-- name: ListDurableEventLogEntriesBeforeNode :many
SELECT *
FROM v1_durable_event_log_entry
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND tenant_id = @tenantId::UUID
  AND node_id < @nodeId::BIGINT
ORDER BY node_id ASC, branch_id ASC;

-- name: UpdateDurableEventLogEntriesSatisfied :many
WITH inputs AS (
    SELECT
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@childTaskIsFailures::BOOLEAN[]) AS child_task_is_failure,
        UNNEST(@childTaskErrorMessages::TEXT[]) AS child_task_error_message
), locked_log_files AS (
    SELECT *
    FROM v1_durable_event_log_file
    WHERE (durable_task_id, durable_task_inserted_at) IN (
        SELECT durable_task_id, durable_task_inserted_at
        FROM inputs
    )
    ORDER BY durable_task_id, durable_task_inserted_at
    FOR UPDATE
), satisfied_orders_to_apply AS (
    SELECT
        e.durable_task_id,
        e.durable_task_inserted_at,
        e.branch_id,
        e.node_id,
        llf.latest_satisfied_order + ROW_NUMBER() OVER (
            PARTITION BY e.durable_task_id, e.durable_task_inserted_at
            ORDER BY e.branch_id ASC, e.node_id ASC
        ) AS satisfied_order
    FROM v1_durable_event_log_entry e
    JOIN locked_log_files llf USING (durable_task_id, durable_task_inserted_at)
    WHERE
        e.satisfied_order IS NULL
        AND (durable_task_id, durable_task_inserted_at, branch_id, node_id) IN (
            SELECT durable_task_id, durable_task_inserted_at, branch_id, node_id
            FROM inputs
        )
), updated AS (
    UPDATE v1_durable_event_log_entry e
    SET
        is_satisfied = true,
        satisfied_at = COALESCE(e.satisfied_at, NOW()),
        satisfied_order = COALESCE(e.satisfied_order, so.satisfied_order),
        child_task_is_failure = inputs.child_task_is_failure,
        child_task_error_message = CASE WHEN inputs.child_task_is_failure THEN NULLIF(inputs.child_task_error_message, '') ELSE NULL END
    FROM inputs
    LEFT JOIN satisfied_orders_to_apply so USING(durable_task_id, durable_task_inserted_at, branch_id, node_id)
    WHERE e.durable_task_id = inputs.durable_task_id
      AND e.durable_task_inserted_at = inputs.durable_task_inserted_at
      AND e.node_id = inputs.node_id
      AND e.branch_id = inputs.branch_id
    RETURNING e.*
), max_satisfied_orders_to_apply AS (
    SELECT durable_task_id, durable_task_inserted_at, MAX(satisfied_order) AS satisfied_order
    FROM satisfied_orders_to_apply
    GROUP BY durable_task_id, durable_task_inserted_at
), log_file_updates AS (
    UPDATE v1_durable_event_log_file lf
    SET latest_satisfied_order = GREATEST(lf.latest_satisfied_order, so.satisfied_order)
    FROM max_satisfied_orders_to_apply so
    WHERE (lf.durable_task_id, lf.durable_task_inserted_at) = (so.durable_task_id, so.durable_task_inserted_at)
)

SELECT updated.*, lf.latest_invocation_count AS invocation_count
FROM updated
JOIN v1_durable_event_log_file lf ON (lf.durable_task_id, lf.durable_task_inserted_at) = (updated.durable_task_id, updated.durable_task_inserted_at)
;

-- name: ListSatisfiedEntries :many
WITH inputs AS (
    SELECT
        UNNEST(@taskExternalIds::UUID[]) AS external_id,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id
), tasks_with_nodes AS (
    SELECT t.*, i.node_id AS requested_node_id, i.branch_id AS requested_branch_id
    FROM inputs i
    JOIN v1_lookup_table lt ON lt.external_id = i.external_id
    JOIN v1_task t ON (t.id, t.inserted_at) = (lt.task_id, lt.inserted_at)
)

SELECT
    e.*,
    twn.external_id AS task_external_id,
    lf.latest_invocation_count AS invocation_count
FROM v1_durable_event_log_entry e
JOIN tasks_with_nodes twn ON (twn.id, twn.inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
JOIN v1_durable_event_log_file lf ON (lf.durable_task_id, lf.durable_task_inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
WHERE
    e.branch_id = twn.requested_branch_id
    AND e.node_id = twn.requested_node_id
    AND e.is_satisfied
;

-- name: MarkDurableEventLogEntrySatisfied :one
UPDATE v1_durable_event_log_entry
SET
    is_satisfied = true,
    satisfied_at = COALESCE(satisfied_at, NOW())
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND branch_id = @branchId::BIGINT
  AND node_id = @nodeId::BIGINT
RETURNING *
;


-- name: BulkGetDurableEventLogEntries :many
WITH inputs AS (
    SELECT
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@nodeIds::BIGINT[]) AS node_id
)
SELECT e.*, lf.latest_invocation_count AS invocation_count
FROM v1_durable_event_log_entry e
JOIN inputs i ON e.branch_id = i.branch_id AND e.node_id = i.node_id
JOIN v1_durable_event_log_file lf ON (lf.durable_task_id, lf.durable_task_inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
WHERE e.durable_task_id = @durableTaskId::BIGINT
  AND e.durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ;

-- name: GetDurableEventLogEntriesByChildTaskExternalIds :many
SELECT e.*, lf.latest_invocation_count AS invocation_count
FROM v1_durable_event_log_entry e
JOIN v1_durable_event_log_file lf ON (lf.durable_task_id, lf.durable_task_inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
WHERE
    e.durable_task_id = @durableTaskId::BIGINT
    AND e.durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
    AND e.child_task_external_id = ANY(@childTaskExternalIds::UUID[])
ORDER BY e.child_task_external_id, e.node_id ASC;

-- name: BulkCreateDurableEventLogEntries :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@childTaskExternalIds::UUID[]) AS child_task_external_id,
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@kinds::text[]) AS kind,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@idempotencyKeys::BYTEA[]) AS idempotency_key,
        UNNEST(@isSatisfieds::BOOLEAN[]) AS is_satisfied,
        UNNEST(@userMessages::TEXT[]) AS user_message,
        UNNEST(@waitDatas::TEXT[]) AS wait_data
), inserts AS (
    INSERT INTO v1_durable_event_log_entry (
        tenant_id,
        external_id,
        child_task_external_id,
        durable_task_id,
        durable_task_inserted_at,
        inserted_at,
        kind,
        node_id,
        branch_id,
        idempotency_key,
        is_satisfied,
        user_message,
        wait_data,
        -- !!IMPORTANT: Writing the `triggered_at` explicitly as `NULL` since it has a `DEFAULT CURRENT_TIMESTAMP`,
        -- so we write the explicit null to avoid it being set to the current timestamp on insert
        triggered_at
    )
    SELECT
        i.tenant_id,
        i.external_id,
        NULLIF(i.child_task_external_id, '00000000-0000-0000-0000-000000000000'::UUID),
        i.durable_task_id,
        i.durable_task_inserted_at,
        NOW(),
        i.kind::v1_durable_event_log_kind,
        i.node_id,
        i.branch_id,
        i.idempotency_key,
        i.is_satisfied,
        NULLIF(i.user_message, ''),
        NULLIF(i.wait_data, '')::JSONB,
        NULL::TIMESTAMPTZ
    FROM inputs i
    ON CONFLICT (durable_task_id, durable_task_inserted_at, branch_id, node_id) DO NOTHING
    RETURNING *
)

SELECT i.*, lf.latest_invocation_count AS invocation_count
FROM inserts i
JOIN v1_durable_event_log_file lf ON (lf.durable_task_id, lf.durable_task_inserted_at) = (i.durable_task_id, i.durable_task_inserted_at)
;


-- name: GetDurableTaskLogFiles :many
WITH inputs AS (
    SELECT
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@tenantIds::UUID[]) AS tenant_id
)

SELECT *
FROM v1_durable_event_log_file lf
WHERE (lf.durable_task_id, lf.durable_task_inserted_at, lf.tenant_id) IN (
    SELECT durable_task_id, durable_task_inserted_at, tenant_id
    FROM inputs
)
;

-- name: ClaimDurableEventLogEntriesForTrigger :many
WITH inputs AS (
    SELECT
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at
), to_claim AS (
    SELECT
        e.durable_task_id,
        e.durable_task_inserted_at,
        e.branch_id,
        e.node_id
    FROM
        v1_durable_event_log_entry e
    JOIN
        inputs i ON e.durable_task_id = i.durable_task_id
            AND e.durable_task_inserted_at = i.durable_task_inserted_at
            AND e.node_id = i.node_id
            AND e.branch_id = i.branch_id
    WHERE
        e.triggered_at IS NULL
    ORDER BY e.durable_task_id, e.durable_task_inserted_at, e.branch_id, e.node_id
    FOR UPDATE
)

UPDATE v1_durable_event_log_entry e
SET triggered_at = NOW()
FROM to_claim c
WHERE e.durable_task_id = c.durable_task_id
  AND e.durable_task_inserted_at = c.durable_task_inserted_at
  AND e.branch_id = c.branch_id
  AND e.node_id = c.node_id
RETURNING e.*
;

-- name: ListDurableEventLogForTask :many
SELECT e.*, t.external_id AS durable_task_external_id, t.display_name AS durable_task_display_name
FROM v1_durable_event_log_entry e
JOIN v1_task t ON (t.id, t.inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
WHERE e.durable_task_id = @durableTaskId::BIGINT
  AND e.durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND e.tenant_id = @tenantId::UUID
ORDER BY e.branch_id ASC, e.node_id ASC
OFFSET @eventLogOffset::BIGINT
LIMIT @eventLogLimit::BIGINT
;

-- name: UpsertDurableChildSignalCreatedEvents :many
WITH input AS (
    SELECT
        UNNEST(@eventKeys::TEXT[]) AS event_key,
        UNNEST(@childExternalIds::UUID[]) AS child_external_id
)

INSERT INTO v1_task_event (
    tenant_id,
    task_id,
    task_inserted_at,
    retry_count,
    event_type,
    event_key,
    child_external_id
)
SELECT
    @tenantId::UUID,
    @durableTaskId::BIGINT,
    @durableTaskInsertedAt::TIMESTAMPTZ,
    -1,
    'SIGNAL_CREATED',
    i.event_key,
    i.child_external_id
FROM input i
ON CONFLICT (tenant_id, task_id, task_inserted_at, event_type, event_key) WHERE event_key IS NOT NULL
DO UPDATE SET child_external_id = COALESCE(v1_task_event.child_external_id, EXCLUDED.child_external_id)
RETURNING
    v1_task_event.event_key,
    v1_task_event.child_external_id
;
