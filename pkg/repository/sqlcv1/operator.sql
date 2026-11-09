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
-- The DISPATCHER rows this dispatcher should run, among the ones it can build. A DAG row runs on
-- every live dispatcher and any other row on one, each host with a worker of its own. A
-- dispatcher hosts a row while it is live and has an active worker of it, and the hosts with the
-- oldest workers, up to the row's replica count, hold it. A host that does not hold the row finds
-- it missing from its result and tears it down, which is how two dispatchers that took the same
-- free replica settle.
--
-- The result is every row this dispatcher holds, marked held, plus rows with a free replica it
-- does not host, up to its fair share of all replicas, ceil(replicas / live dispatchers), less
-- what it holds. Held rows are returned unlocked and whatever the share: the claimer tears down
-- every running operator the result does not mark held, and SKIP LOCKED would leave out a held
-- row whenever an action delta holds its lock (LockOperator). The share only limits new claims,
-- so no running operator is interrupted to even out the load. The exception is a stranded row,
-- short of replicas and already hosted by every dispatcher under its share: nobody else can take
-- it, so a dispatcher at its share takes one per poll.
WITH live_dispatchers AS (
    -- self counts as live despite a late heartbeat, so it keeps its rows until another
    -- dispatcher's registration releases its worker of one (ReleaseDeadOperatorWorkers)
    SELECT "id"
    FROM "Dispatcher"
    WHERE
        "isActive" = TRUE
        AND ("lastHeartbeatAt" > NOW() - INTERVAL '15 seconds' OR "id" = @dispatcherId::UUID)
), candidates AS (
    SELECT
        v1_operator.id,
        v1_operator.tenant_id,
        CASE WHEN v1_operator.kind = 'DAG' THEN (SELECT COUNT(*) FROM live_dispatchers) ELSE 1 END AS replicas
    FROM v1_operator
    JOIN "Tenant" t ON t."id" = v1_operator.tenant_id AND t."deletedAt" IS NULL
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
), hosts AS (
    -- DISTINCT ON: a dispatcher can briefly have two active workers of a row, one still tearing
    -- down while it registers the next, and it hosts the row once
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
    SELECT ld."id" AS dispatcher_id
    FROM live_dispatchers ld
    LEFT JOIN hosts h ON h.dispatcher_id = ld."id" AND h.rank <= h.replicas
    GROUP BY ld."id"
    HAVING COUNT(h.id) < (SELECT size FROM share)
), open_rows AS (
    -- stranded: every dispatcher under its share already hosts the row, so only a dispatcher at
    -- its share can add a replica
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
    -- Fewest hosts first, so every row gets a first replica before any gets a second, and random
    -- among equals, so a row that fails to open on every poll does not take the last free slot
    -- every time. Locked so that claims running at the same instant take disjoint rows.
    SELECT v1_operator.id
    FROM v1_operator
    JOIN open_rows o ON o.id = v1_operator.id
    WHERE (SELECT size FROM room) > 0 OR o.stranded
    ORDER BY o.host_count, random()
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
