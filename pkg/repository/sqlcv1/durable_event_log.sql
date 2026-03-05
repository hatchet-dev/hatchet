-- name: GetAndLockLogFile :one
SELECT *
FROM v1_durable_event_log_file
WHERE
    durable_task_id = @durableTaskId::BIGINT
    AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
    AND tenant_id = @tenantId::UUID
FOR UPDATE
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
    latest_branch_id,
    latest_branch_first_parent_node_id
)
SELECT
    tenant_id,
    durable_task_id,
    durable_task_inserted_at,
    1,
    NOW(),
    0,
    1,
    0
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
    latest_node_id = COALESCE(sqlc.narg('nodeId')::BIGINT, v1_durable_event_log_file.latest_node_id),
    latest_invocation_count = COALESCE(sqlc.narg('invocationCount')::INTEGER, v1_durable_event_log_file.latest_invocation_count),
    latest_branch_id = COALESCE(sqlc.narg('branchId')::BIGINT, v1_durable_event_log_file.latest_branch_id),
    latest_branch_first_parent_node_id = COALESCE(sqlc.narg('branchFirstParentNodeId')::BIGINT, v1_durable_event_log_file.latest_branch_first_parent_node_id)
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
RETURNING *;

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
    invocation_count,
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
    @invocationCount::INTEGER,
    @idempotencyKey::BYTEA,
    @isSatisfied::BOOLEAN
)
ON CONFLICT (durable_task_id, durable_task_inserted_at, branch_id, node_id) DO NOTHING
RETURNING *
;

-- name: UpdateDurableEventLogEntryInvocationCount :one
UPDATE v1_durable_event_log_entry
SET
    invocation_count = @invocationCount::INTEGER,
    idempotency_key = @idempotencyKey::BYTEA
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND branch_id = @branchId::BIGINT
  AND node_id = @nodeId::BIGINT
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

SELECT e.*, twn.external_id AS task_external_id
FROM v1_durable_event_log_entry e
JOIN tasks_with_nodes twn ON (twn.id, twn.inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
WHERE
    e.branch_id = twn.requested_branch_id
    AND e.node_id = twn.requested_node_id
    AND e.is_satisfied
;

-- name: MarkDurableEventLogEntrySatisfied :one
UPDATE v1_durable_event_log_entry
SET is_satisfied = true
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND branch_id = @branchId::BIGINT
  AND node_id = @nodeId::BIGINT
RETURNING *
;


-- name: BulkGetDurableEventLogEntries :many
WITH inputs AS (
    SELECT
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@nodeIds::BIGINT[]) AS node_id
)
SELECT e.*
FROM v1_durable_event_log_entry e
JOIN inputs i ON e.durable_task_id = i.durable_task_id
    AND e.durable_task_inserted_at = i.durable_task_inserted_at
    AND e.branch_id = i.branch_id
    AND e.node_id = i.node_id;

-- name: BulkCreateDurableEventLogEntries :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@kinds::text[]) AS kind,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@parentNodeIds::BIGINT[]) AS parent_node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@parentBranchIds::BIGINT[]) AS parent_branch_id,
        UNNEST(@invocationCounts::INTEGER[]) AS invocation_count,
        UNNEST(@idempotencyKeys::BYTEA[]) AS idempotency_key,
        UNNEST(@isSatisfieds::BOOLEAN[]) AS is_satisfied
)
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
    invocation_count,
    idempotency_key,
    is_satisfied
)
SELECT
    i.tenant_id,
    i.external_id,
    i.durable_task_id,
    i.durable_task_inserted_at,
    NOW(),
    i.kind::v1_durable_event_log_kind,
    i.node_id,
    i.parent_node_id,
    i.branch_id,
    i.parent_branch_id,
    i.invocation_count,
    i.idempotency_key,
    i.is_satisfied
FROM inputs i
ON CONFLICT (durable_task_id, durable_task_inserted_at, branch_id, node_id) DO NOTHING
RETURNING *;

-- name: BulkUpdateDurableEventLogEntryInvocationCounts :many
WITH inputs AS (
    SELECT
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@invocationCounts::INTEGER[]) AS invocation_count,
        UNNEST(@idempotencyKeys::BYTEA[]) AS idempotency_key
)
UPDATE v1_durable_event_log_entry
SET
    invocation_count = inputs.invocation_count,
    idempotency_key = inputs.idempotency_key
FROM inputs
WHERE v1_durable_event_log_entry.durable_task_id = inputs.durable_task_id
  AND v1_durable_event_log_entry.durable_task_inserted_at = inputs.durable_task_inserted_at
  AND v1_durable_event_log_entry.branch_id = inputs.branch_id
  AND v1_durable_event_log_entry.node_id = inputs.node_id
RETURNING v1_durable_event_log_entry.*;

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
