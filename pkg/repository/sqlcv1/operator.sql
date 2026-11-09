-- name: CreateOperator :one
INSERT INTO v1_operator (
    tenant_id,
    name,
    kind,
    leasing_manager,
    config
) VALUES (
    @tenantId::UUID,
    @name::TEXT,
    @kind::v1_operator_kind,
    @leasing_manager::v1_operator_leasing_manager,
    @config::JSONB
)
ON CONFLICT (tenant_id, name, kind) DO NOTHING
RETURNING *;

-- name: GetOperator :one
SELECT *
FROM v1_operator
WHERE
    id = @id::UUID;

-- name: ListOperators :many
SELECT *
FROM v1_operator
WHERE
    tenant_id = @tenantId::UUID
    AND (
        sqlc.narg('kind')::v1_operator_kind IS NULL
        OR kind = sqlc.narg('kind')::v1_operator_kind
    )
ORDER BY created_at DESC, id DESC
LIMIT @operatorLimit::BIGINT
OFFSET @operatorOffset::BIGINT;

-- name: CountOperators :one
SELECT COUNT(*)
FROM v1_operator
WHERE
    tenant_id = @tenantId::UUID
    AND (
        sqlc.narg('kind')::v1_operator_kind IS NULL
        OR kind = sqlc.narg('kind')::v1_operator_kind
    );

-- name: UpdateOperator :one
UPDATE v1_operator
SET
    name = COALESCE(sqlc.narg('name')::TEXT, name),
    config = COALESCE(sqlc.narg('config')::JSONB, config),
    updated_at = NOW(),
    worker_id = COALESCE(sqlc.narg('workerId')::UUID, worker_id)
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
RETURNING *;

-- name: LockOperator :one
-- Takes the operator's row lock for the rest of the transaction. A delta on one of the
-- operator's workers takes it after the worker's own row lock, always in that order, so the
-- per-operator action budget is checked against a sum no concurrent delta is changing.
SELECT id
FROM v1_operator
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
FOR UPDATE;

-- name: DeleteOperator :one
DELETE FROM v1_operator
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
RETURNING *;

-- name: ClaimOperators :many
-- The DISPATCHER rows this dispatcher should run, among the ones it can build: every row it
-- holds, plus free rows up to its fair share, ceil(rows / live dispatchers), less what it holds.
-- A row is held by the dispatcher of the worker its worker_id points at while that dispatcher is
-- live. worker_id is written by the registration the claimer makes for each row it opens.
--
-- Held rows are returned unlocked and whatever the share: the claimer tears down every running
-- operator missing from the result, and SKIP LOCKED would leave out a held row whenever an
-- action delta holds its lock (LockOperator). The share only limits new claims, so no running
-- operator is interrupted to even out the load.
--
-- A row stays free until its registration writes worker_id, so two dispatchers can both take
-- it. The one that registers first loses: the row is missing from its next result and it tears
-- the operator down.
WITH live_dispatchers AS (
    -- self counts as live despite a late heartbeat, so it keeps its rows until another
    -- dispatcher takes one over and moves its worker_id
    SELECT "id"
    FROM "Dispatcher"
    WHERE
        "isActive" = TRUE
        AND ("lastHeartbeatAt" > NOW() - INTERVAL '15 seconds' OR "id" = @dispatcherId::UUID)
), candidates AS (
    SELECT
        v1_operator.id,
        ld."id" AS holder_id
    FROM v1_operator
    JOIN "Tenant" t ON t."id" = v1_operator.tenant_id AND t."deletedAt" IS NULL
    LEFT JOIN "Worker" w ON w."id" = v1_operator.worker_id
    LEFT JOIN live_dispatchers ld ON ld."id" = w."dispatcherId"
    WHERE
        -- Only DISPATCHER rows are claimed, whatever their kind. A SELF row keeps itself alive
        -- (a Listen stream out of process, its own leaser in process) and registers its own
        -- workers, so the claimer never claims or reconciles it.
        v1_operator.leasing_manager = 'DISPATCHER'
        -- A row no dispatcher builds is never registered, so it would stay free and use up
        -- every dispatcher's share.
        AND (
            (v1_operator.kind <> 'GRPC' AND v1_operator.kind::TEXT = ANY(@kinds::TEXT[]))
            OR (v1_operator.kind = 'GRPC' AND v1_operator.name = ANY(@grpcNames::TEXT[]))
        )
), held AS (
    SELECT id
    FROM candidates
    WHERE holder_id = @dispatcherId::UUID
), taken AS (
    -- Random order, so a row that fails to open on every poll does not take the last free slot
    -- every time. Locked so that claims running at the same instant take disjoint rows.
    SELECT v1_operator.id
    FROM v1_operator
    WHERE v1_operator.id IN (SELECT id FROM candidates WHERE holder_id IS NULL)
    ORDER BY random()
    LIMIT GREATEST(
        CEIL(
            (SELECT COUNT(*) FROM candidates)::NUMERIC
            / GREATEST((SELECT COUNT(*) FROM live_dispatchers), 1)
        )::BIGINT - (SELECT COUNT(*) FROM held),
        0
    )
    FOR UPDATE OF v1_operator SKIP LOCKED
)
SELECT v1_operator.*
FROM v1_operator
WHERE
    -- A dispatcher that has started shutting down marks itself inactive and claims nothing
    -- more, so an operator released by another dispatcher never lands on one that is leaving.
    EXISTS (
        SELECT 1
        FROM "Dispatcher" d
        WHERE d."id" = @dispatcherId::UUID AND d."isActive" = TRUE
    )
    AND (
        v1_operator.id IN (SELECT id FROM held)
        OR v1_operator.id IN (SELECT id FROM taken)
    )
ORDER BY v1_operator.id;

-- name: UpsertOperator :one
-- Registers an operator by (tenant, name, kind), the row a session registers under by name. The
-- row carries no config. A repeat registration takes the leasing manager it names: a row a
-- dispatcher was claiming that registers as SELF leaves the claim set on the claimer's next poll,
-- and the other way round. A SELF row never gets a worker_id: each registration creates its own worker, linked
-- back via "Worker"."operatorId".
INSERT INTO v1_operator (
    tenant_id,
    name,
    kind,
    leasing_manager,
    config
) VALUES (
    @tenantId::UUID,
    @name::TEXT,
    @kind::v1_operator_kind,
    @leasing_manager::v1_operator_leasing_manager,
    '{}'::JSONB
)
ON CONFLICT (tenant_id, name, kind) DO UPDATE
SET
    leasing_manager = EXCLUDED.leasing_manager,
    updated_at = NOW()
RETURNING *;

-- name: ListDAGOrchestrationActionsForTenant :many
SELECT DISTINCT s."actionId"::text AS action
FROM "Step" s
JOIN "Job" j ON j."id" = s."jobId"
JOIN "WorkflowVersion" wv ON wv."id" = j."workflowVersionId"
WHERE
    s."tenantId" = @tenantId::UUID
    AND s."isDagOrchestrator" = true
    AND s."deletedAt" IS NULL
    AND wv."deletedAt" IS NULL
ORDER BY action;

-- name: CountEvictedDAGOrchestratorRuns :one
SELECT COUNT(*)
FROM v1_task_runtime rt
JOIN v1_task t ON (t.id, t.inserted_at) = (rt.task_id, rt.task_inserted_at)
JOIN "Step" s ON s."id" = t.step_id
WHERE
    rt.tenant_id = @tenantId::uuid
    AND rt.evicted_at IS NOT NULL
    AND s."isDagOrchestrator" = TRUE;
