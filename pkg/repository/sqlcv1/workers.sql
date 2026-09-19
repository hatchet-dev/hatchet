-- name: ListManyWorkerLabels :many
SELECT
    "id",
    "key",
    "intValue",
    "strValue",
    "createdAt",
    "updatedAt",
    "workerId"
FROM "WorkerLabel" wl
WHERE wl."workerId" = ANY(@workerIds::uuid[]);

-- name: ListWorkerSlotConfigs :many
SELECT
    worker_id,
    slot_type,
    max_units
FROM
    v1_worker_slot_config
WHERE
    tenant_id = @tenantId::uuid
    AND worker_id = ANY(@workerIds::uuid[]);

-- name: CreateWorkerSlotConfigs :exec
INSERT INTO v1_worker_slot_config (
    tenant_id,
    worker_id,
    slot_type,
    max_units,
    created_at,
    updated_at
)
SELECT
    @tenantId::uuid,
    @workerId::uuid,
    unnest(@slotTypes::text[]),
    unnest(@maxUnits::integer[]),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
-- NOTE: ON CONFLICT can be removed after the 0_76_d migration is run to remove insert triggers added in 0_76
ON CONFLICT (tenant_id, worker_id, slot_type) DO UPDATE SET
    max_units = EXCLUDED.max_units,
    updated_at = CURRENT_TIMESTAMP;

-- name: ListAvailableSlotsForWorkers :many
WITH worker_capacities AS (
    SELECT
        worker_id,
        max_units
    FROM
        v1_worker_slot_config
    WHERE
        tenant_id = @tenantId::uuid
        AND worker_id = ANY(@workerIds::uuid[])
        AND slot_type = @slotType::text
), worker_used_slots AS (
    SELECT
        runtime.worker_id,
        (
            COALESCE(SUM(CASE WHEN tr.batch_id IS NULL THEN runtime.units ELSE 0 END), 0)::integer
            + COUNT(DISTINCT tr.batch_id)::integer
        ) AS used_units
    FROM
        v1_task_runtime_slot runtime
    LEFT JOIN v1_task_runtime tr ON tr.task_id = runtime.task_id
        AND tr.task_inserted_at = runtime.task_inserted_at
        AND tr.retry_count = runtime.retry_count
    WHERE
        runtime.tenant_id = @tenantId::uuid
        AND runtime.worker_id = ANY(@workerIds::uuid[])
        AND runtime.slot_type = @slotType::text
    GROUP BY
        runtime.worker_id
)
SELECT
    wc.worker_id AS "id",
    wc.max_units - COALESCE(wus.used_units, 0) AS "availableSlots"
FROM
    worker_capacities wc
LEFT JOIN
    worker_used_slots wus ON wc.worker_id = wus.worker_id;

-- name: ListAvailableSlotsForWorkersAndTypes :many
WITH worker_capacities AS (
    SELECT
        worker_id,
        slot_type,
        max_units
    FROM
        v1_worker_slot_config
    WHERE
        tenant_id = @tenantId::uuid
        AND worker_id = ANY(@workerIds::uuid[])
        AND slot_type = ANY(@slotTypes::text[])
), worker_used_slots AS (
    SELECT
        runtime.worker_id,
        runtime.slot_type,
        (
            COALESCE(SUM(CASE WHEN tr.batch_id IS NULL THEN runtime.units ELSE 0 END), 0)::integer
            + COUNT(DISTINCT tr.batch_id)::integer
        ) AS used_units
    FROM
        v1_task_runtime_slot runtime
    LEFT JOIN v1_task_runtime tr ON tr.task_id = runtime.task_id
        AND tr.task_inserted_at = runtime.task_inserted_at
        AND tr.retry_count = runtime.retry_count
    WHERE
        runtime.tenant_id = @tenantId::uuid
        AND runtime.worker_id = ANY(@workerIds::uuid[])
        AND runtime.slot_type = ANY(@slotTypes::text[])
    GROUP BY
        runtime.worker_id,
        runtime.slot_type
)
SELECT
    wc.worker_id AS "id",
    wc.slot_type AS "slotType",
    wc.max_units - COALESCE(wus.used_units, 0) AS "availableSlots"
FROM
    worker_capacities wc
LEFT JOIN
    worker_used_slots wus ON wc.worker_id = wus.worker_id AND wc.slot_type = wus.slot_type;

-- name: ListWorkers :many
SELECT
    sqlc.embed(workers)
FROM
    "Worker" workers
WHERE
    workers."tenantId" = @tenantId
    AND (
        COALESCE(sqlc.narg('includeOperators')::boolean, FALSE)
        OR NOT EXISTS (
            -- hide dag operators
            SELECT 1
            FROM v1_operator op
            WHERE
                op.id = workers."operatorId"
                AND op.kind = 'DAG'
        )
    )
    AND (
        sqlc.narg('actionId')::text IS NULL OR
        workers."id" IN (
            SELECT "_ActionToWorker"."B"
            FROM "_ActionToWorker"
            INNER JOIN "Action" ON "Action"."id" = "_ActionToWorker"."A"
            WHERE "Action"."tenantId" = @tenantId AND "Action"."actionId" = sqlc.narg('actionId')::text
        )
    )
    AND (
        sqlc.narg('lastHeartbeatAfter')::timestamp IS NULL OR
        workers."lastHeartbeatAt" > sqlc.narg('lastHeartbeatAfter')::timestamp
    )
    AND (
        sqlc.narg('assignable')::boolean IS NULL OR
        (sqlc.narg('assignable')::boolean AND (
            SELECT COALESCE(SUM(cap.max_units), 0)
            FROM v1_worker_slot_config cap
            WHERE cap.tenant_id = workers."tenantId" AND cap.worker_id = workers."id"
        ) > (
            SELECT COALESCE(SUM(runtime.units), 0)
            FROM v1_task_runtime_slot runtime
            WHERE runtime.tenant_id = workers."tenantId" AND runtime.worker_id = workers."id"
        ))
    )
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR
        CASE
            WHEN workers."lastHeartbeatAt" IS NULL OR workers."lastHeartbeatAt" <= NOW() - INTERVAL '5 seconds' THEN 'INACTIVE'
            WHEN workers."isPaused" = true THEN 'PAUSED'
            ELSE 'ACTIVE'
        END = ANY(sqlc.narg('statuses')::text[])
    )
    AND (
        sqlc.narg('labelKeys')::text[] IS NULL
        OR sqlc.narg('labelValues')::text[] IS NULL
        OR (
            SELECT BOOL_AND(
                EXISTS (
                    SELECT 1
                    FROM "WorkerLabel" wl
                    WHERE wl."workerId" = workers."id"
                        AND wl."key" = lf.k
                        AND (
                            wl."strValue" = lf.v
                            OR wl."intValue"::text = lf.v
                        )
                )
            )
            FROM (
                SELECT
                    UNNEST(sqlc.narg('labelKeys')::text[]) AS k,
                    UNNEST(sqlc.narg('labelValues')::text[]) AS v
            ) AS lf
        )
    )
ORDER BY
    workers."createdAt" DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 10000);

-- name: CountWorkers :one
SELECT count(*)
FROM
    "Worker" workers
WHERE
    workers."tenantId" = @tenantId
    AND (
        COALESCE(sqlc.narg('includeOperators')::boolean, FALSE)
        OR NOT EXISTS (
            -- hide dag operators
            SELECT 1
            FROM v1_operator op
            WHERE
                op.id = workers."operatorId"
                AND op.kind = 'DAG'
        )
    )
    AND (
        sqlc.narg('actionId')::text IS NULL OR
        workers."id" IN (
            SELECT "_ActionToWorker"."B"
            FROM "_ActionToWorker"
            INNER JOIN "Action" ON "Action"."id" = "_ActionToWorker"."A"
            WHERE "Action"."tenantId" = @tenantId AND "Action"."actionId" = sqlc.narg('actionId')::text
        )
    )
    AND (
        sqlc.narg('lastHeartbeatAfter')::timestamp IS NULL OR
        workers."lastHeartbeatAt" > sqlc.narg('lastHeartbeatAfter')::timestamp
    )
    AND (
        sqlc.narg('assignable')::boolean IS NULL OR
        (sqlc.narg('assignable')::boolean AND (
            SELECT COALESCE(SUM(cap.max_units), 0)
            FROM v1_worker_slot_config cap
            WHERE cap.tenant_id = workers."tenantId" AND cap.worker_id = workers."id"
        ) > (
            SELECT COALESCE(SUM(runtime.units), 0)
            FROM v1_task_runtime_slot runtime
            WHERE runtime.tenant_id = workers."tenantId" AND runtime.worker_id = workers."id"
        ))
    )
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR
        CASE
            WHEN workers."lastHeartbeatAt" IS NULL OR workers."lastHeartbeatAt" <= NOW() - INTERVAL '5 seconds' THEN 'INACTIVE'
            WHEN workers."isPaused" = true THEN 'PAUSED'
            ELSE 'ACTIVE'
        END = ANY(sqlc.narg('statuses')::text[])
    )
    AND (
        sqlc.narg('labelKeys')::text[] IS NULL
        OR sqlc.narg('labelValues')::text[] IS NULL
        OR (
            SELECT BOOL_AND(
                EXISTS (
                    SELECT 1
                    FROM "WorkerLabel" wl
                    WHERE wl."workerId" = workers."id"
                        AND wl."key" = lf.k
                        AND (
                            wl."strValue" = lf.v
                            OR wl."intValue"::text = lf.v
                        )
                )
            )
            FROM (
                SELECT
                    UNNEST(sqlc.narg('labelKeys')::text[]) AS k,
                    UNNEST(sqlc.narg('labelValues')::text[]) AS v
            ) AS lf
        )
    );

-- name: GetWorkerById :one
SELECT
    sqlc.embed(w),
    w."maxRuns" - (
        SELECT
            COALESCE(SUM(CASE WHEN runtime.batch_id IS NULL THEN 1 ELSE 0 END), 0)::integer
            + COUNT(DISTINCT runtime.batch_id)::integer
        FROM v1_task_runtime runtime
        WHERE
            runtime.tenant_id = w."tenantId" AND
            runtime.worker_id = w."id"
    ) AS "remainingSlots"
FROM
    "Worker" w
WHERE
    w."id" = @id::uuid;

-- name: GetActiveWorkerById :one
SELECT
    sqlc.embed(w),
    ww."url" AS "webhookUrl",
    w."maxRuns" - (
        SELECT COUNT(*)
        FROM v1_task_runtime runtime
        WHERE
            runtime.tenant_id = w."tenantId" AND
            runtime.worker_id = w."id"
    ) AS "remainingSlots"
FROM
    "Worker" w
LEFT JOIN
    "WebhookWorker" ww ON w."webhookId" = ww."id"
WHERE
    w."id" = @id::uuid
    AND w."tenantId" = @tenantId::uuid
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false;

-- name: ListSemaphoreSlotsWithStateForWorker :many
SELECT
    *
FROM
    v1_task_runtime runtime
JOIN
    v1_task ON runtime.task_id = v1_task.id AND runtime.task_inserted_at = v1_task.inserted_at
WHERE
    runtime.tenant_id = @tenantId::uuid
    AND runtime.worker_id = @workerId::uuid
LIMIT
    COALESCE(sqlc.narg('limit')::int, 100);

-- name: ListTotalActiveSlotsPerTenant :many
SELECT
    wc.tenant_id AS "tenantId",
    SUM(wc.max_units) AS "totalActiveSlots"
FROM v1_worker_slot_config wc
JOIN "Worker" w ON w."id" = wc.worker_id AND w."tenantId" = wc.tenant_id
WHERE
    w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false
    -- a worker the in-process operator host created is engine infrastructure and is not
    -- metered; every other worker, an operator's or an SDK's, counts (see
    -- CreateWorkerOpts.ExemptFromLimits in worker.go)
    AND NOT w."exemptFromLimits"
GROUP BY wc.tenant_id
;

-- name: ListActiveSlotsPerTenantAndSlotType :many
SELECT
    wc.tenant_id AS "tenantId",
    wc.slot_type AS "slotType",
    SUM(wc.max_units) AS "activeSlots"
FROM v1_worker_slot_config wc
JOIN "Worker" w ON w."id" = wc.worker_id AND w."tenantId" = wc.tenant_id
WHERE
    w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false
    -- a worker the in-process operator host created is engine infrastructure and is not
    -- metered; every other worker, an operator's or an SDK's, counts (see
    -- CreateWorkerOpts.ExemptFromLimits in worker.go)
    AND NOT w."exemptFromLimits"
GROUP BY wc.tenant_id, wc.slot_type
;

-- name: ListActiveSDKsPerTenant :many
SELECT
    "tenantId",
    COALESCE("language"::TEXT, 'unknown')::TEXT AS "language",
    COALESCE("languageVersion", 'unknown') AS "languageVersion",
    COALESCE("sdkVersion", 'unknown') AS "sdkVersion",
    COALESCE("os", 'unknown') AS "os",
    COUNT(*) AS "count"
FROM "Worker"
WHERE
    "dispatcherId" IS NOT NULL
    AND "lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND "isActive" = true
    AND "isPaused" = false
GROUP BY "tenantId", "language", "languageVersion", "sdkVersion", "os"
;

-- name: ListActiveWorkersPerTenant :many
SELECT "tenantId", COUNT(*)
FROM "Worker"
WHERE
    "dispatcherId" IS NOT NULL
    AND "lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND "isActive" = true
    AND "isPaused" = false
GROUP BY "tenantId"
;

-- name: GetWorkerActionsByWorkerId :many
SELECT
    w."id" AS "workerId",
    a."actionId" AS actionId
FROM "Worker" w
JOIN "_ActionToWorker" aw ON w.id = aw."B"
JOIN "Action" a ON aw."A" = a.id
WHERE
    a."tenantId" = @tenantId::UUID
    AND w.id = ANY(@workerIds::UUID[])
;

-- name: GetWorkerActionsByWorkerActionHash :many
-- NOTE: workers with the same action hash have identical action sets by construction,
-- so we only look up the actions for a single representative worker per hash. Joining
-- through every worker with a matching hash degrades badly when inactive workers
-- accumulate, since each hash can match tens of thousands of historical workers.
SELECT
    rep.action_hash,
    a."actionId" AS action_id
FROM (
    SELECT
        h.hash::BYTEA AS action_hash,
        (
            SELECT w.id
            FROM "Worker" w
            WHERE
                w."tenantId" = @tenantId::UUID
                AND w."actionHash" = h.hash
            LIMIT 1
        ) AS worker_id
    FROM unnest(@actionHashes::BYTEA[]) AS h(hash)
) rep
JOIN "_ActionToWorker" aw ON aw."B" = rep.worker_id
JOIN "Action" a ON a.id = aw."A"
;

-- name: GetWorkerWorkflowsByWorkerId :many
SELECT wf.*
FROM "Worker" w
JOIN "_ActionToWorker" aw ON w.id = aw."B"
JOIN "Action" a ON aw."A" = a.id
JOIN "Step" s ON s."actionId" = a."actionId" AND s."tenantId" = a."tenantId"
JOIN "Job" j ON j."id" = s."jobId" AND j."tenantId" = a."tenantId"
JOIN "WorkflowVersion" wv ON wv."id" = j."workflowVersionId"
JOIN "Workflow" wf ON wf."id" = wv."workflowId" AND wf."tenantId" = a."tenantId"
WHERE
    w."id" = @workerId::UUID
    AND w."tenantId" = @tenantId::UUID
;

-- name: GetWorkerForEngine :one
-- "actionHash" is NULL while the hash refresh that follows a delta is pending; a session that
-- opens on such a worker refreshes it first.
SELECT
    w."id" AS "id",
    w."tenantId" AS "tenantId",
    w."dispatcherId" AS "dispatcherId",
    w."lastHeartbeatAt" AS "lastHeartbeatAt",
    d."lastHeartbeatAt" AS "dispatcherLastHeartbeatAt",
    w."isActive" AS "isActive",
    w."lastListenerEstablished" AS "lastListenerEstablished",
    w."operatorId" AS "operatorId",
    w."actionHash" AS "actionHash"
FROM
    "Worker" w
LEFT JOIN
    "Dispatcher" d ON w."dispatcherId" = d."id"
WHERE
    w."tenantId" = @tenantId
    AND w."id" = @id;

-- name: ListWorkerLabels :many
SELECT
    "workerId",
    "id",
    "key",
    "intValue",
    "strValue",
    "createdAt",
    "updatedAt"
FROM "WorkerLabel" wl
WHERE wl."workerId" = ANY(@workerIds::uuid[]);

-- name: UpdateWorker :one
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "dispatcherId" = coalesce(sqlc.narg('dispatcherId')::uuid, "dispatcherId"),
    "lastHeartbeatAt" = coalesce(sqlc.narg('lastHeartbeatAt')::timestamp, "lastHeartbeatAt"),
    "isPaused" = coalesce(sqlc.narg('isPaused')::boolean, "isPaused")
WHERE
    "id" = @id::uuid
RETURNING *;

-- name: LinkActionsToWorker :exec
INSERT INTO "_ActionToWorker" (
    "A",
    "B"
) SELECT
    unnest(@actionIds::uuid[]),
    @workerId::uuid
ON CONFLICT DO NOTHING;

-- name: LockWorkerActionHash :one
-- Serializes concurrent action set changes on one worker: the caller holds the row lock for the
-- rest of its transaction. The tenant is part of the predicate so a caller that pairs a tenant
-- with another tenant's worker finds no row and mutates nothing.
SELECT "actionHash"
FROM "Worker"
WHERE
    "id" = @workerId::uuid
    AND "tenantId" = @tenantId::uuid
FOR UPDATE;

-- name: InsertMissingActions :many
-- Creates the Action rows in @actions that do not exist yet and returns the rows it created.
-- Existing rows are left untouched (no lock, no rewrite), so concurrent transactions that
-- reference the same actions do not contend on them. @actions must be lower-cased, free of
-- duplicates and sorted: two transactions creating overlapping sets then take their row locks
-- in the same order and cannot deadlock. An action another transaction creates concurrently is
-- absent from the result; the caller resolves it with ListActionsByActionIds afterwards.
INSERT INTO "Action" (
    "id",
    "actionId",
    "tenantId"
)
SELECT
    gen_random_uuid(),
    a.action,
    @tenantId::uuid
FROM unnest(@actions::text[]) AS a(action)
ON CONFLICT ("tenantId", "actionId") DO NOTHING
RETURNING "id", "actionId";

-- name: LinkActionsToWorkerReturning :many
-- Links the worker to the given action rows and returns the action row ids that were newly
-- linked, so the caller knows whether the set changed. The Worker and Action rows are joined
-- on the tenant: an action of another tenant, or a worker of another tenant, is never linked.
INSERT INTO "_ActionToWorker" (
    "A",
    "B"
) SELECT
    a."id",
    w."id"
FROM "Worker" w
JOIN "Action" a ON a."tenantId" = w."tenantId"
WHERE
    w."id" = @workerId::uuid
    AND w."tenantId" = @tenantId::uuid
    AND a."id" = ANY(@actionIds::uuid[])
ORDER BY a."id"
ON CONFLICT DO NOTHING
RETURNING "A";

-- name: ListActionsByActionIds :many
-- Resolves action ids to their rows. @actionIds are compared as stored (lower-cased).
SELECT "id", "actionId"
FROM "Action"
WHERE
    "tenantId" = @tenantId::uuid
    AND "actionId" = ANY(@actionIds::text[]);

-- name: UnlinkActionsFromWorkerReturning :many
-- Unlinks the given action rows from the worker and returns the action row ids that were
-- actually unlinked. The Worker and Action rows are joined on the tenant, as in
-- LinkActionsToWorkerReturning.
DELETE FROM "_ActionToWorker" aw
USING "Worker" w, "Action" a
WHERE
    aw."B" = w."id"
    AND aw."A" = a."id"
    AND w."id" = @workerId::uuid
    AND w."tenantId" = @tenantId::uuid
    AND a."tenantId" = w."tenantId"
    AND a."id" = ANY(@actionIds::uuid[])
RETURNING aw."A";

-- name: ComputeWorkerActionHash :one
-- The canonical digest of the worker's linked action set: sha256 over the action ids sorted
-- by byte order, each followed by ";". An id cannot contain the separator (ParseActionID
-- rejects it), so no id can be read as the boundary between two others. It is the same
-- function hashActions computes in Go, byte for byte, so a worker created with an initial set
-- and a worker built by deltas hash equal for the same set. The empty set hashes to sha256 of
-- no bytes.
SELECT sha256(coalesce(
    string_agg(
        convert_to(a."actionId", 'UTF8') || ';'::bytea,
        ''::bytea
        ORDER BY a."actionId" COLLATE "C"
    ),
    ''::bytea
))::bytea AS "hash"
FROM "_ActionToWorker" aw
JOIN "Action" a ON a."id" = aw."A"
WHERE aw."B" = @workerId::uuid;

-- name: RecountWorkerActions :exec
-- Sets "operatorActionCount" to the worker's real link count, for the paths that link
-- without returning what they linked.
UPDATE "Worker" w
SET "operatorActionCount" = (SELECT count(*) FROM "_ActionToWorker" aw WHERE aw."B" = w."id")
WHERE w."id" = @workerId::uuid;

-- name: SettleWorkerActionsDelta :one
-- Records a delta's effect on the worker row under the caller's row lock:
-- "operatorActionCount" moves by the links the delta created minus the links it removed, and
-- "actionHash" is cleared
-- until the session refreshes it at the end of the delta sequence. Returns the operator the
-- worker belongs to, NULL for an SDK worker, so the caller knows whose budget to check.
UPDATE "Worker" w
SET
    "operatorActionCount" = "operatorActionCount" + sqlc.arg('added')::integer - sqlc.arg('removed')::integer,
    "actionHash" = NULL
WHERE w."id" = @workerId::uuid
RETURNING w."operatorId";

-- name: SumOperatorWorkerActionCounts :one
-- The action links held by every worker of the operator, from the per-worker counts. The
-- caller holds the operator's row lock (LockOperator), so the sum is consistent with the
-- delta it is checking.
SELECT coalesce(sum(w."operatorActionCount"), 0)::bigint
FROM "Worker" w
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."operatorId" = @operatorId::uuid;

-- name: CountOperatorWorkerActions :one
-- The action links held by every worker of the operator, from the per-worker counts. It is
-- the unlocked reading of SumOperatorWorkerActionCounts, for reporting; the budget check
-- inside a delta uses the locked one.
SELECT coalesce(sum(w."operatorActionCount"), 0)::bigint
FROM "Worker" w
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."operatorId" = @operatorId::uuid;

-- name: UpdateWorkerHeartbeat :one
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "lastHeartbeatAt" = sqlc.narg('lastHeartbeatAt')::timestamp
WHERE
    "id" = @id::uuid
RETURNING *;

-- name: UpdateWorkerHeartbeats :exec
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "lastHeartbeatAt" = @lastHeartbeatAt::timestamp
WHERE
    "id" = ANY(@ids::uuid[]);

-- name: PauseWorkers :exec
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "isPaused" = TRUE
WHERE
    "id" = ANY(@ids::uuid[]);

-- name: DeleteWorker :one
DELETE FROM
  "Worker"
WHERE
  "id" = @id::uuid
RETURNING *;

-- name: ActivateWorkerListener :one
-- Marks the worker active for the given listener session. lastListenerEstablished is
-- stamped alongside the session id because Heartbeat uses it to tell a worker that never
-- opened a listener from one whose listener has gone away.
UPDATE "Worker"
SET
    "isActive" = TRUE,
    "lastListenerSessionId" = @sessionId::uuid,
    "lastListenerEstablished" = CURRENT_TIMESTAMP
WHERE
    "id" = @id::uuid
    AND "tenantId" = @tenantId::uuid
RETURNING *;

-- name: SetWorkerPausedForListener :one
-- Sets the worker's pause on behalf of a listener session, only while that session is the one
-- recorded on the row. A superseded session must not change the scheduling state the newer
-- session owns, so this returns no rows in that case.
UPDATE "Worker"
SET
    "isPaused" = @paused::boolean,
    "updatedAt" = CURRENT_TIMESTAMP
WHERE
    "id" = @id::uuid
    AND "tenantId" = @tenantId::uuid
    AND "lastListenerSessionId" = @sessionId::uuid
RETURNING "id";

-- name: DeactivateWorkerListener :one
-- Marks the worker inactive only while the given session is still the one recorded on the
-- row. A session whose id is no longer on the row was superseded by a newer session and
-- must not touch it, so this returns no rows in that case.
UPDATE "Worker"
SET
    "isActive" = FALSE
WHERE
    "id" = @id::uuid
    AND "tenantId" = @tenantId::uuid
    AND "lastListenerSessionId" = @sessionId::uuid
RETURNING *;

-- name: UpsertWorkerLabel :one
INSERT INTO "WorkerLabel" (
    "createdAt",
    "updatedAt",
    "workerId",
    "key",
    "intValue",
    "strValue"
) VALUES (
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @workerId::uuid,
    @key::text,
    sqlc.narg('intValue')::int,
    sqlc.narg('strValue')::text
) ON CONFLICT ("workerId", "key") DO UPDATE
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "intValue" = sqlc.narg('intValue')::int,
    "strValue" = sqlc.narg('strValue')::text
RETURNING *;

-- name: CleanupOldWorkers :execresult
WITH old_workers AS (
    SELECT "id"
    FROM "Worker"
    WHERE "tenantId" = @tenantId::uuid
      AND "lastHeartbeatAt" < @lastHeartbeatBefore::timestamp
    LIMIT @batchSize::int
), deleted_worker_slot_configs AS (
    DELETE FROM v1_worker_slot_config
    WHERE worker_id IN (SELECT "id" FROM old_workers)
)
DELETE FROM "Worker"
WHERE "id" IN (SELECT "id" FROM old_workers);

-- name: ListDispatcherIdsForWorkers :many
SELECT
    "id" as "workerId",
    "dispatcherId"
FROM
    "Worker"
WHERE
    "tenantId" = @tenantId::uuid
    AND "id" = ANY(@workerIds::uuid[]);

-- name: UpsertService :one
INSERT INTO "Service" (
    "id",
    "createdAt",
    "updatedAt",
    "name",
    "tenantId"
)
VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @name::text,
    @tenantId::uuid
)
ON CONFLICT ("tenantId", "name") DO UPDATE
SET
    "updatedAt" = CURRENT_TIMESTAMP
WHERE
    "Service"."tenantId" = @tenantId AND "Service"."name" = @name::text
RETURNING *;

-- name: CreateWorker :one
INSERT INTO "Worker" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "name",
    "dispatcherId",
    "type",
    "sdkVersion",
    "language",
    "languageVersion",
    "os",
    "runtimeExtra",
    "actionHash",
    "operatorActionCount",
    "operatorId",
    "exemptFromLimits"
) VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @name::text,
    @dispatcherId::uuid,
    sqlc.narg('type')::"WorkerType",
    sqlc.narg('sdkVersion')::text,
    sqlc.narg('language')::"WorkerSDKS",
    sqlc.narg('languageVersion')::text,
    sqlc.narg('os')::text,
    sqlc.narg('runtimeExtra')::text,
    @actionHash::bytea,
    -- the size of the initial action set the caller links right after
    @operatorActionCount::integer,
    -- set for workers backing an operator connection; NULL for SDK workers
    sqlc.narg('operatorId')::uuid,
    -- true only for workers the in-process operator host creates; the limit queries skip them
    @exemptFromLimits::boolean
) RETURNING *;

-- name: LinkServicesToWorker :exec
INSERT INTO "_ServiceToWorker" (
    "A",
    "B"
)
VALUES (
    unnest(@services::uuid[]),
    @workerId::uuid
)
ON CONFLICT DO NOTHING;

-- name: UpdateWorkerDurableTaskDispatcherId :exec
UPDATE "Worker"
SET
    "durableTaskDispatcherId" = @dispatcherId::UUID,
    "updatedAt" = CURRENT_TIMESTAMP
WHERE
    "id" = @workerId::uuid
    AND "tenantId" = @tenantId::uuid
    AND "durableTaskDispatcherId" IS DISTINCT FROM @dispatcherId::UUID;

-- name: ListDurableTaskDispatcherIdsForTasks :many
WITH tasks AS (
    SELECT
        UNNEST(@taskIds::BIGINT[]) AS task_id,
        UNNEST(@taskInsertedAts::TIMESTAMPTZ[]) AS task_inserted_at
)

SELECT
    rt.*,
    w."durableTaskDispatcherId"
FROM v1_task_runtime rt
LEFT JOIN "Worker" w ON rt.worker_id = w.id
WHERE
    rt.tenant_id = @tenantId::uuid
    AND (rt.task_id, rt.task_inserted_at) IN (
        SELECT task_id, task_inserted_at
        FROM tasks
    )
;
