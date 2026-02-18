-- name: GetLogFile :one
SELECT *
FROM v1_durable_event_log_file
WHERE durable_task_id = @durableTaskId::BIGINT
    AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
;

-- name: UpdateLogFile :one
UPDATE v1_durable_event_log_file
SET
    latest_node_id = COALESCE(sqlc.narg('nodeId')::BIGINT, v1_durable_event_log_file.latest_node_id),
    latest_invocation_count = COALESCE(sqlc.narg('invocationCount')::BIGINT, v1_durable_event_log_file.latest_invocation_count),
    latest_branch_id = COALESCE(sqlc.narg('branchId')::BIGINT, v1_durable_event_log_file.latest_branch_id),
    latest_branch_first_parent_node_id = COALESCE(sqlc.narg('branchFirstParentNodeId')::BIGINT, v1_durable_event_log_file.latest_branch_first_parent_node_id)
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
  AND branch_id = @branchId::BIGINT
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
    parent_branch_id,
    idempotency_key,
    is_satisfied
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
    sqlc.narg('parentBranchId')::BIGINT,
    @idempotencyKey::BYTEA,
    @isSatisfied::BOOLEAN
)
ON CONFLICT (durable_task_id, durable_task_inserted_at, branch_id, node_id) DO NOTHING
RETURNING *
;


-- name: UpdateDurableEventLogEntriesSatisfied :many
WITH inputs AS (
    SELECT
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id
)

UPDATE v1_durable_event_log_entry
SET is_satisfied = true
FROM inputs
WHERE v1_durable_event_log_entry.durable_task_id = inputs.durable_task_id
  AND v1_durable_event_log_entry.durable_task_inserted_at = inputs.durable_task_inserted_at
  AND v1_durable_event_log_entry.node_id = inputs.node_id
  AND v1_durable_event_log_entry.branch_id = inputs.branch_id
RETURNING v1_durable_event_log_entry.*
;

-- name: ListSatisfiedEntries :many
WITH tasks AS (
    SELECT t.*
    FROM v1_lookup_table lt
    JOIN v1_task t ON (t.id, t.inserted_at) = (lt.task_id, lt.inserted_at)
    WHERE lt.external_id = ANY(@taskExternalIds::UUID[])
), nodes_and_branches AS (
    SELECT
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id
)

SELECT e.*, t.external_id AS task_external_id
FROM v1_durable_event_log_entry e
JOIN tasks t ON (t.id, t.inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
WHERE
    (e.branch_id, e.node_id) IN (SELECT branch_id, node_id FROM nodes_and_branches)
    AND e.is_satisfied
;
