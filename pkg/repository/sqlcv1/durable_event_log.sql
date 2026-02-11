-- name: GetOrCreateEventLogFile :one
INSERT INTO v1_durable_event_log_file (
    tenant_id, durable_task_id, durable_task_inserted_at,
    latest_inserted_at, latest_node_id, latest_branch_id, latest_branch_first_parent_node_id
) VALUES (
    @tenantId::UUID, @durableTaskId::BIGINT, @durableTaskInsertedAt::TIMESTAMPTZ,
    NOW(), @nodeId::BIGINT, 1, 0
)
ON CONFLICT (durable_task_id, durable_task_inserted_at)
DO UPDATE SET
    latest_node_id = GREATEST(v1_durable_event_log_file.latest_node_id, EXCLUDED.latest_node_id),
    latest_inserted_at = NOW()
RETURNING *
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
    SET latest_node_id = GREATEST(f.latest_node_id, i.node_id)
    FROM inputs i
    WHERE
        f.durable_task_id = i.durable_task_id
        AND f.durable_task_inserted_at = i.durable_task_inserted_at
)

SELECT *, false AS already_exists FROM inserts
UNION ALL
SELECT e.*, true AS already_exists
FROM v1_durable_event_log_entry e
JOIN inputs i ON
    e.durable_task_id = i.durable_task_id
    AND e.durable_task_inserted_at = i.durable_task_inserted_at
    AND e.node_id = i.node_id
WHERE NOT EXISTS (SELECT 1 FROM inserts)
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

SELECT *, false AS already_exists FROM ins
UNION ALL
SELECT c.*, true AS already_exists
FROM v1_durable_event_log_callback c
JOIN inputs i ON
    c.durable_task_id = i.durable_task_id
    AND c.durable_task_inserted_at = i.durable_task_inserted_at
    AND c.node_id = i.node_id
WHERE NOT EXISTS (SELECT 1 FROM ins)
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
