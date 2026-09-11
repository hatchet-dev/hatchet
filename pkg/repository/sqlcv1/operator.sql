-- name: CreateOperator :one
INSERT INTO v1_operator (
    tenant_id,
    name,
    kind,
    leasing,
    config
) VALUES (
    @tenantId::UUID,
    @name::TEXT,
    @kind::v1_operator_kind,
    @leasing::v1_operator_leasing,
    @config::JSONB
)
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
WITH operators_on_inactive_dispatchers AS (
    SELECT v1_operator.id
    FROM v1_operator
    JOIN "Worker" w ON w."id" = v1_operator.worker_id
    WHERE
        w."dispatcherId" IS NULL OR
        w."dispatcherId" IN (
            SELECT "id"
            FROM "Dispatcher"
            WHERE
                "lastHeartbeatAt" < NOW () - INTERVAL '15 seconds'
        )
), unassigned_operators AS (
    SELECT v1_operator.id
    FROM v1_operator
    WHERE v1_operator.worker_id IS NULL
), operators_already_assigned_to_dispatcher AS (
    SELECT v1_operator.id
    FROM v1_operator
    JOIN "Worker" w ON w."id" = v1_operator.worker_id
    WHERE w."dispatcherId" = @dispatcherId::UUID
)
SELECT *
FROM v1_operator
WHERE
    -- Only engine-leased rows are claimed, whatever their kind. A SELF row keeps itself alive
    -- (a Listen stream out of process, its own leaser in process) and registers its own
    -- workers, so the claimer never claims or reconciles it.
    v1_operator.leasing = 'MANAGED'
    AND (
        v1_operator.id IN (SELECT id FROM operators_on_inactive_dispatchers) OR
        v1_operator.id IN (SELECT id FROM unassigned_operators) OR
        v1_operator.id IN (SELECT id FROM operators_already_assigned_to_dispatcher)
    )
ORDER BY v1_operator.id
FOR UPDATE SKIP LOCKED;

-- name: UpsertOperator :one
-- Registers an operator by (tenant, name, kind), the row a session registers under by name. The
-- row carries no config. A repeat registration takes the leasing it names: a row the engine was
-- leasing that registers as SELF leaves the claim set on the claimer's next poll, and the other
-- way round. A SELF row never gets a worker_id: each registration creates its own worker, linked
-- back via "Worker"."operatorId".
INSERT INTO v1_operator (
    tenant_id,
    name,
    kind,
    leasing,
    config
) VALUES (
    @tenantId::UUID,
    @name::TEXT,
    @kind::v1_operator_kind,
    @leasing::v1_operator_leasing,
    '{}'::JSONB
)
ON CONFLICT (tenant_id, name, kind) DO UPDATE
SET
    leasing = EXCLUDED.leasing,
    updated_at = NOW()
RETURNING *;

-- name: UpdateWorkerActionsHash :exec
UPDATE
    "Worker" w
SET
    "actionHash" = @actionHash::bytea
WHERE
    w."id" = @workerId::uuid;

-- name: TenantHasDAGOperator :one
SELECT EXISTS(
    SELECT 1
    FROM v1_operator
    WHERE
        tenant_id = @tenantId::UUID
        AND kind = 'DAG'
) AS has_operator;

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
