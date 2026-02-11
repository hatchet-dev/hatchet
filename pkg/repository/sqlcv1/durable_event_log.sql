-- name: GetOrCreateEventLogFile :one
WITH existing_log_file AS (
    SELECT *
    FROM v1_durable_event_log_file
    WHERE durable_task_id = @durableTaskId::BIGINT
      AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
), upsert_result AS (
    INSERT INTO v1_durable_event_log_file (
        tenant_id,
        durable_task_id,
        durable_task_inserted_at,
        latest_invocation_count,
        latest_inserted_at,
        latest_node_id,
        latest_branch_id,
        latest_branch_first_parent_node_id
    ) VALUES (
        @tenantId::UUID,
        @durableTaskId::BIGINT,
        @durableTaskInsertedAt::TIMESTAMPTZ,
        @invocationCount::BIGINT,
        NOW(),
        0,
        1,
        0
    )
    ON CONFLICT (durable_task_id, durable_task_inserted_at)
    DO UPDATE SET
        latest_node_id = GREATEST(v1_durable_event_log_file.latest_node_id, EXCLUDED.latest_node_id),
        latest_inserted_at = NOW(),
        latest_invocation_count = GREATEST(v1_durable_event_log_file.latest_invocation_count, EXCLUDED.latest_invocation_count)
    RETURNING *
)

SELECT
    r.*,
    COALESCE(e.latest_invocation_count, 0) < @invocationCount::BIGINT AS is_new_invocation
FROM upsert_result r
LEFT JOIN existing_log_file e USING (durable_task_id, durable_task_inserted_at)
;

-- name: GetDurableEventLogEntry :one
SELECT *
FROM v1_durable_event_log_entry
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND node_id = @nodeId::BIGINT;

-- name: CreateDurableEventLogEntry :one
WITH ins AS (
    INSERT INTO v1_durable_event_log_entry (
        tenant_id,
        external_id,
        durable_task_id,
        durable_task_inserted_at,
        inserted_at,
        kind,
        node_id,
        parent_node_id,
        branch_id,
        data_hash,
        data_hash_alg
    )
    VALUES (
        @tenantId::UUID,
        @externalId::UUID,
        @durableTaskId::BIGINT,
        @durableTaskInsertedAt::TIMESTAMPTZ,
        NOW(),
        @kind::v1_durable_event_log_entry_kind,
        @nodeId::BIGINT,
        sqlc.narg('parentNodeId')::BIGINT,
        @branchId::BIGINT,
        @dataHash::BYTEA,
        @dataHashAlg::TEXT
    )
    ON CONFLICT (durable_task_id, durable_task_inserted_at, node_id) DO NOTHING
    RETURNING *
), node_id_update AS (
    -- todo: this should probably be figured out at the repo level
    UPDATE v1_durable_event_log_file AS f
    SET latest_node_id = GREATEST(f.latest_node_id, i.node_id)
    FROM ins i
    WHERE
        f.durable_task_id = i.durable_task_id
        AND f.durable_task_inserted_at = i.durable_task_inserted_at
)

SELECT *
FROM ins
;

-- name: GetDurableEventLogCallback :one
SELECT *
FROM v1_durable_event_log_callback
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND node_id = @nodeId::BIGINT
;

-- name: CreateDurableEventLogCallback :one
INSERT INTO v1_durable_event_log_callback (
    tenant_id,
    durable_task_id,
    durable_task_inserted_at,
    inserted_at,
    kind,
    node_id,
    is_satisfied,
    external_id,
    dispatcher_id
)
VALUES (
    @tenantId::UUID,
    @durableTaskId::BIGINT,
    @durableTaskInsertedAt::TIMESTAMPTZ,
    @insertedAt::TIMESTAMPTZ,
    @kind::v1_durable_event_log_callback_kind,
    @nodeId::BIGINT,
    @isSatisfied::BOOLEAN,
    @externalId::UUID,
    @dispatcherId::UUID
)
ON CONFLICT (durable_task_id, durable_task_inserted_at, node_id) DO NOTHING
RETURNING *
;

-- name: UpdateDurableEventLogCallbackSatisfied :one
UPDATE v1_durable_event_log_callback
SET is_satisfied = @isSatisfied::BOOLEAN
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND node_id = @nodeId::BIGINT
RETURNING *
;

-- name: GetSatisfiedCallbacks :many
SELECT cb.*, t.external_id AS task_external_id
FROM v1_durable_event_log_callback cb
JOIN v1_task t ON t.id = cb.durable_task_id
    AND t.inserted_at = cb.durable_task_inserted_at
    AND t.tenant_id = @tenantId::UUID
WHERE (t.external_id, cb.node_id) IN (
    SELECT
        unnest(@taskExternalIds::uuid[]),
        unnest(@nodeIds::bigint[])
)
  AND cb.is_satisfied = TRUE
;
