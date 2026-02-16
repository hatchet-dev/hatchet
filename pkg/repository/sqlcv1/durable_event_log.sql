-- name: GetAndLockLogFile :one
SELECT *
FROM v1_durable_event_log_file
WHERE durable_task_id = @durableTaskId::BIGINT
    AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
FOR UPDATE;

-- name: UpdateLogFileNodeIdInvocationCount :one
UPDATE v1_durable_event_log_file
SET
    latest_node_id = COALESCE(sqlc.narg('nodeId')::BIGINT, v1_durable_event_log_file.latest_node_id),
    latest_invocation_count = COALESCE(sqlc.narg('invocationCount')::BIGINT, v1_durable_event_log_file.latest_invocation_count)
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
RETURNING *;

-- name: CreateEventLogFile :one
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
    0,
    NOW(),
    1,
    1,
    0
)
ON CONFLICT (durable_task_id, durable_task_inserted_at)
DO UPDATE SET
    latest_node_id = GREATEST(v1_durable_event_log_file.latest_node_id, EXCLUDED.latest_node_id),
    latest_inserted_at = NOW(),
    latest_invocation_count = GREATEST(v1_durable_event_log_file.latest_invocation_count, EXCLUDED.latest_invocation_count)
RETURNING *
;

-- name: GetDurableEventLogEntry :one
SELECT *
FROM v1_durable_event_log_entry
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND node_id = @nodeId::BIGINT;

-- name: CreateDurableEventLogEntry :one
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
    idempotency_key
)
VALUES (
    @tenantId::UUID,
    @externalId::UUID,
    @durableTaskId::BIGINT,
    @durableTaskInsertedAt::TIMESTAMPTZ,
    NOW(),
    @kind::v1_durable_event_log_kind,
    @nodeId::BIGINT,
    sqlc.narg('parentNodeId')::BIGINT,
    @branchId::BIGINT,
    @idempotencyKey::BYTEA
)
ON CONFLICT (durable_task_id, durable_task_inserted_at, node_id) DO NOTHING
RETURNING *
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
    external_id
)
VALUES (
    @tenantId::UUID,
    @durableTaskId::BIGINT,
    @durableTaskInsertedAt::TIMESTAMPTZ,
    @insertedAt::TIMESTAMPTZ,
    @kind::v1_durable_event_log_kind,
    @nodeId::BIGINT,
    @isSatisfied::BOOLEAN,
    @externalId::UUID
)
ON CONFLICT (durable_task_id, durable_task_inserted_at, node_id) DO NOTHING
RETURNING *
;

-- name: UpdateDurableEventLogCallbacksSatisfied :many
WITH inputs AS (
    SELECT
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@nodeIds::BIGINT[]) AS node_id
)

UPDATE v1_durable_event_log_callback
SET is_satisfied = true
FROM inputs
WHERE v1_durable_event_log_callback.durable_task_id = inputs.durable_task_id
  AND v1_durable_event_log_callback.durable_task_inserted_at = inputs.durable_task_inserted_at
  AND v1_durable_event_log_callback.node_id = inputs.node_id
RETURNING v1_durable_event_log_callback.*
;

-- name: ListSatisfiedCallbacks :many
WITH tasks AS (
    SELECT t.*
    FROM v1_lookup_table lt
    JOIN v1_task t ON (t.id, t.inserted_at) = (lt.task_id, lt.inserted_at)
    WHERE lt.external_id = ANY(@taskExternalIds::UUID[])
)

SELECT cb.*, t.external_id AS task_external_id
FROM v1_durable_event_log_callback cb
JOIN tasks t ON (t.id, t.inserted_at) = (cb.durable_task_id, cb.durable_task_inserted_at)
WHERE
    cb.node_id = ANY(@nodeIds::BIGINT[])
    AND cb.is_satisfied
;
