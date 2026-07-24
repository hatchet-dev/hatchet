-- name: CreateOperator :one
INSERT INTO v1_operator (
    tenant_id,
    name,
    kind,
    config
) VALUES (
    @tenantId::UUID,
    @name::TEXT,
    @kind::v1_operator_kind,
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
WHERE v1_operator.id IN (SELECT id FROM operators_on_inactive_dispatchers) OR
v1_operator.id IN (SELECT id FROM unassigned_operators) OR
v1_operator.id IN (SELECT id FROM operators_already_assigned_to_dispatcher)
ORDER BY v1_operator.id
FOR UPDATE SKIP LOCKED;

-- name: CreateOperatorWorker :one
-- Creates a fresh worker for a single operator instance, linked back to the operator via
-- "operatorId". Each time an operator is instantiated on a dispatcher it gets its own
-- worker; older workers age out via the normal worker-inactivity path.
INSERT INTO "Worker" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "name",
    "dispatcherId",
    "type",
    "actionHash",
    "operatorId",
    "isActive"
) VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @name::text,
    @dispatcherId::uuid,
    'SELFHOSTED',
    @actionHash::bytea,
    @operatorId::uuid,
    -- operator workers have no gRPC listener to activate them, so they are born active.
    true
) RETURNING *;

-- name: UpdateWorkerActionsHash :exec
UPDATE
    "Worker" w
SET
    "actionHash" = @actionHash::bytea
WHERE
    w."id" = @workerId::uuid;

-- name: ListDAGWorkflowIdsForTenant :many
-- Returns the ids of all DAG workflows for a tenant. The DAG operator registers these as
-- worker actions so tasks for those workflows are routed to it.
SELECT DISTINCT w."id"
FROM "Workflow" w
JOIN "WorkflowVersion" wv ON wv."workflowId" = w."id"
WHERE
    w."tenantId" = @tenantId::UUID
    AND w."deletedAt" IS NULL
    AND wv."deletedAt" IS NULL
    AND wv."kind" = 'DAG'
ORDER BY w."id";
