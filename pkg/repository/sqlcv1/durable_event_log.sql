-- name: GetOrCreateDurableEventLogFile :one
WITH to_insert AS (
    SELECT
        @tenantId::UUID AS tenant_id,
        @durableTaskId::BIGINT AS durable_task_id,
        @durableTaskInsertedAt::TIMESTAMPTZ AS durable_task_inserted_at,
        @latestInsertedAt::TIMESTAMPTZ AS latest_inserted_at,
        @latestNodeId::BIGINT AS latest_node_id,
        @latestBranchId::BIGINT AS latest_branch_id,
        @latestBranchFirstParentNodeId::BIGINT AS latest_branch_first_parent_node_id
), ins AS (
    INSERT INTO v1_durable_event_log_file (
        tenant_id,
        durable_task_id,
        durable_task_inserted_at,
        latest_inserted_at,
        latest_node_id,
        latest_branch_id,
        latest_branch_first_parent_node_id
    )
    SELECT
        tenant_id,
        durable_task_id,
        durable_task_inserted_at,
        latest_inserted_at,
        latest_node_id,
        latest_branch_id,
        latest_branch_first_parent_node_id
    FROM to_insert
    ON CONFLICT (durable_task_id, durable_task_inserted_at) DO NOTHING
)

SELECT
    *,
    (SELECT COUNT(*) FROM ins) = 0 AS already_exists
FROM to_insert
;

-- name: GetOrCreateDurableEventLogEntry :one
WITH inputs AS (
    SELECT
        @tenantId::UUID AS tenant_id,
        @externalId::UUID AS external_id,
        @durableTaskId::BIGINT AS durable_task_id,
        @durableTaskInsertedAt::TIMESTAMPTZ AS durable_task_inserted_at,
        @insertedAt::TIMESTAMPTZ AS inserted_at,
        @kind::v1_durable_event_log_entry_kind AS kind,
        @nodeId::BIGINT AS node_id,
        sqlc.narg('parentNodeId')::BIGINT AS parent_node_id,
        @branchId::BIGINT AS branch_id,
        @dataHash::BYTEA AS data_hash,
        @dataHashAlg::TEXT AS data_hash_alg
), inserts AS (
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
    SELECT
        i.tenant_id,
        i.external_id,
        i.durable_task_id,
        i.durable_task_inserted_at,
        i.inserted_at,
        i.kind,
        i.node_id,
        i.parent_node_id,
        i.branch_id,
        i.data_hash,
        i.data_hash_alg
    FROM
        inputs i
    ON CONFLICT (durable_task_id, durable_task_inserted_at, node_id) DO NOTHING
    RETURNING *
), node_id_update AS (
    -- todo: this should probably be figured out at the repo level
    UPDATE v1_durable_event_log_file AS f
    SET latest_node_id = GREATEST(f.latest_node_id, i.latest_node_id)
    FROM inputs i
    WHERE
        f.durable_task_id = i.durable_task_id
        AND f.durable_task_inserted_at = i.durable_task_inserted_at
)

SELECT
    *,
    (SELECT COUNT(*) FROM inserts) = 0 AS already_exists
FROM inserts
;

-- name: ListDurableEventLogEntries :many
SELECT *
FROM v1_durable_event_log_entry
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
ORDER BY node_id ASC
;

-- name: GetDurableEventLogEntry :one
SELECT *
FROM v1_durable_event_log_entry
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND node_id = @nodeId::BIGINT
;

-- name: GetOrCreateDurableEventLogCallback :one
WITH inputs AS (
    SELECT
        @tenantId::UUID AS tenant_id,
        @durableTaskId::BIGINT AS durable_task_id,
        @durableTaskInsertedAt::TIMESTAMPTZ AS durable_task_inserted_at,
        @insertedAt::TIMESTAMPTZ AS inserted_at,
        @kind::v1_durable_event_log_callback_kind AS kind,
        @nodeId::BIGINT AS node_id,
        @isSatisfied::BOOLEAN AS is_satisfied,
        @externalId::UUID AS external_id,
        @dispatcherId::UUID AS dispatcher_id
), ins AS (
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
    SELECT
        i.tenant_id,
        i.durable_task_id,
        i.durable_task_inserted_at,
        i.inserted_at,
        i.kind,
        i.node_id,
        i.is_satisfied,
        i.external_id,
        i.dispatcher_id
    FROM
        inputs i
    ON CONFLICT (durable_task_id, durable_task_inserted_at, node_id) DO NOTHING
    RETURNING *
)

SELECT
    *,
    (SELECT COUNT(*) FROM ins) = 0 AS already_exists
FROM inputs
;

-- name: GetDurableEventLogCallback :one
SELECT *
FROM v1_durable_event_log_callback
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND node_id = @nodeId::BIGINT
;

-- name: ListDurableEventLogCallbacks :many
SELECT *
FROM v1_durable_event_log_callback
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
ORDER BY inserted_at ASC
;

-- name: UpdateDurableEventLogCallbackSatisfied :one
UPDATE v1_durable_event_log_callback
SET is_satisfied = @isSatisfied::BOOLEAN
WHERE durable_task_id = @durableTaskId::BIGINT
  AND durable_task_inserted_at = @durableTaskInsertedAt::TIMESTAMPTZ
  AND node_id = @nodeId::BIGINT
RETURNING *
;
