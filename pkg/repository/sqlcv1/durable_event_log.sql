-- name: CreateDurableEventLogFile :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@latestInsertedAts::TIMESTAMPTZ[]) AS latest_inserted_at,
        UNNEST(@latestNodeIds::BIGINT[]) AS latest_node_id,
        UNNEST(@latestBranchIds::BIGINT[]) AS latest_branch_id,
        UNNEST(@latestBranchFirstParentNodeIds::BIGINT[]) AS latest_branch_first_parent_node_id
)
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
    i.tenant_id,
    i.durable_task_id,
    i.durable_task_inserted_at,
    i.latest_inserted_at,
    i.latest_node_id,
    i.latest_branch_id,
    i.latest_branch_first_parent_node_id
FROM
    inputs i
RETURNING *
;

-- name: GetOrCreateEventLogFileForTask :one
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

SELECT *
FROM to_insert
;

-- name: CreateDurableEventLogEntries :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(CAST(@kinds::TEXT[] AS v1_durable_event_log_entry_kind[])) AS kind,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@parentNodeIds::BIGINT[]) AS parent_node_id,
        UNNEST(@branchIds::BIGINT[]) AS branch_id,
        UNNEST(@dataHashes::BYTEA[]) AS data_hash,
        UNNEST(@dataHashAlgs::TEXT[]) AS data_hash_alg,
        -- todo: probably need an override here since this can be null
        UNNEST(@childRunExternalIds::TEXT[]) AS triggered_run_external_id
), latest_node_ids AS (
    SELECT
        durable_task_id,
        durable_task_inserted_at,
        MAX(node_id) AS latest_node_id
    FROM inputs
    GROUP BY durable_task_id, durable_task_inserted_at
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
        data_hash_alg,
        triggered_run_external_id
    )
    SELECT
        i.tenant_id,
        i.external_id,
        i.durable_task_id,
        i.durable_task_inserted_at,
        i.inserted_at,
        i.kind,
        i.node_id,
        -- todo: check on if 0 is a safe sentinel value here or if we're zero-indexing the node id
        NULLIF(i.parent_node_id, 0),
        i.branch_id,
        i.data_hash,
        i.data_hash_alg,
        CASE
            WHEN i.triggered_run_external_id = '' THEN NULL
            ELSE i.triggered_run_external_id::UUID
        END
    FROM
        inputs i
    ORDER BY
        i.durable_task_id,
        i.durable_task_inserted_at,
        i.node_id
    -- todo: conflict resolution here
    RETURNING *
), node_id_update AS (
    -- todo: might need to explicitly lock here before the initial select / inserts
    UPDATE v1_durable_event_log_file AS f
    SET latest_node_id = GREATEST(f.latest_node_id, l.latest_node_id)
    FROM latest_node_ids l
    WHERE
        f.durable_task_id = l.durable_task_id
        AND f.durable_task_inserted_at = l.durable_task_inserted_at
)

SELECT *
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

-- name: CreateDurableEventLogCallbacks :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@durableTaskIds::BIGINT[]) AS durable_task_id,
        UNNEST(@durableTaskInsertedAts::TIMESTAMPTZ[]) AS durable_task_inserted_at,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(CAST(@kinds::TEXT[] AS v1_durable_event_log_callback_kind[])) AS kind,
        UNNEST(@nodeIds::BIGINT[]) AS node_id,
        UNNEST(@isSatisfieds::BOOLEAN[]) AS is_satisfied,
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@dispatcherIds::UUID[]) AS dispatcher_id
)
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
RETURNING *
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
