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
    updated_at = NOW()
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
-- The DISPATCHER rows this dispatcher should run, among the ones it can build. A row runs on up
-- to its replica count of dispatchers at once (@dagReplicas for DAG rows, one for every other
-- kind), each with a worker of its own. A dispatcher holds a row while it is live and has an
-- active worker of it. When more dispatchers hold a row than it has replicas, the ones with the
-- oldest workers keep it and the rest find it missing from their result and tear it down, which
-- is how two dispatchers that claimed the same free replica settle.
--
-- The result is every row this dispatcher holds, marked held, plus rows with a free replica it
-- does not host, up to its fair share of all replicas, ceil(replicas / live dispatchers), less
-- what it holds. Held rows are returned whatever the share and are not locked: the claimer tears
-- down every running operator missing from the result, and SKIP LOCKED would leave out a held
-- row whenever an action delta holds its lock. A dispatcher over its share keeps its rows; the
-- share only limits new claims, except that a dispatcher at its share still takes one row per
-- poll that is short of replicas and already hosted by every dispatcher under its share, which
-- nobody else could take.
WITH live_dispatchers AS (
    -- this dispatcher is live even with a late heartbeat, so that it keeps the rows it holds
    SELECT "id"
    FROM "Dispatcher"
    WHERE
        "isActive" = TRUE
        AND ("lastHeartbeatAt" > NOW() - INTERVAL '15 seconds' OR "id" = @dispatcherId::UUID)
), candidates AS (
    SELECT
        v1_operator.id,
        v1_operator.tenant_id,
        -- a dispatcher hosts a row once, so a row has no more replicas than live dispatchers
        LEAST(
            CASE WHEN v1_operator.kind = 'DAG' THEN GREATEST(@dagReplicas::INT, 1) ELSE 1 END,
            (SELECT COUNT(*) FROM live_dispatchers)
        ) AS replicas
    FROM v1_operator
    JOIN "Tenant" t ON t."id" = v1_operator.tenant_id AND t."deletedAt" IS NULL
    WHERE
        -- Only DISPATCHER rows are claimed, whatever their kind. A SELF row keeps itself alive
        -- (a Listen stream out of process, its own leaser in process) and registers its own
        -- workers, so the claimer never claims or reconciles it.
        v1_operator.leasing_manager = 'DISPATCHER'
        -- Only rows the caller can build, GRPC rows by name and every other kind by kind. A row
        -- no dispatcher builds is never registered, so it would stay free and use up every
        -- dispatcher's share.
        AND (
            (v1_operator.kind <> 'GRPC' AND v1_operator.kind::TEXT = ANY(@kinds::TEXT[]))
            OR (v1_operator.kind = 'GRPC' AND v1_operator.name = ANY(@grpcNames::TEXT[]))
        )
), hosts AS (
    -- the live dispatchers with an active worker of each row, ranked by their oldest one
    SELECT
        c.id,
        h.dispatcher_id,
        c.replicas,
        ROW_NUMBER() OVER (PARTITION BY c.id ORDER BY h.created_at, h.worker_id) AS rank
    FROM candidates c
    JOIN LATERAL (
        SELECT DISTINCT ON (w."dispatcherId")
            w."dispatcherId" AS dispatcher_id,
            w."createdAt" AS created_at,
            w."id" AS worker_id
        FROM "Worker" w
        JOIN live_dispatchers ld ON ld."id" = w."dispatcherId"
        WHERE
            w."tenantId" = c.tenant_id
            AND w."operatorId" = c.id
            AND w."isActive" = TRUE
        ORDER BY w."dispatcherId", w."createdAt", w."id"
    ) h ON TRUE
), held AS (
    SELECT id
    FROM hosts
    WHERE dispatcher_id = @dispatcherId::UUID AND rank <= replicas
), share AS (
    SELECT CEIL(
        (SELECT COALESCE(SUM(replicas), 0) FROM candidates)::NUMERIC
        / GREATEST((SELECT COUNT(*) FROM live_dispatchers), 1)
    )::BIGINT AS size
), under_share AS (
    -- the live dispatchers holding fewer rows than their share
    SELECT ld."id" AS dispatcher_id
    FROM live_dispatchers ld
    LEFT JOIN hosts h ON h.dispatcher_id = ld."id" AND h.rank <= h.replicas
    GROUP BY ld."id"
    HAVING COUNT(h.id) < (SELECT size FROM share)
), open_rows AS (
    -- rows with a free replica that this dispatcher does not host, not even one it is about to
    -- drop; stranded when every dispatcher under its share already hosts the row
    SELECT
        c.id,
        COUNT(h.id) AS host_count,
        NOT EXISTS (
            SELECT 1
            FROM under_share u
            WHERE NOT EXISTS (SELECT 1 FROM hosts uh WHERE uh.id = c.id AND uh.dispatcher_id = u.dispatcher_id)
        ) AS stranded
    FROM candidates c
    LEFT JOIN hosts h ON h.id = c.id
    GROUP BY c.id, c.replicas
    HAVING
        COUNT(h.id) < c.replicas
        AND COUNT(*) FILTER (WHERE h.dispatcher_id = @dispatcherId::UUID) = 0
), room AS (
    SELECT (SELECT size FROM share) - (SELECT COUNT(*) FROM held) AS size
), taken AS (
    -- Fewest hosts first, so the share goes to rows short of a first replica before any row gets
    -- another; at its share, one stranded row. Locked so that concurrent claims take disjoint rows.
    SELECT v1_operator.id
    FROM v1_operator
    JOIN open_rows o ON o.id = v1_operator.id
    WHERE (SELECT size FROM room) > 0 OR o.stranded
    ORDER BY o.host_count, v1_operator.id
    LIMIT GREATEST((SELECT size FROM room), 1)
    FOR UPDATE OF v1_operator SKIP LOCKED
)
SELECT
    sqlc.embed(v1_operator),
    (v1_operator.id IN (SELECT id FROM held))::BOOLEAN AS held
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
-- and the other way round. Each registration creates its own worker, linked back via
-- "Worker"."operatorId".
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
