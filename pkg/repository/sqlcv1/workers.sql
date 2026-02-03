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

-- name: ListWorkerSlotCapacities :many
SELECT
    worker_id,
    slot_type,
    max_units
FROM
    v1_worker_slot_capacity
WHERE
    tenant_id = @tenantId::uuid
    AND worker_id = ANY(@workerIds::uuid[]);

-- name: UpsertWorkerSlotCapacities :exec
INSERT INTO v1_worker_slot_capacity (
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
ON CONFLICT (tenant_id, worker_id, slot_type) DO UPDATE
SET
    max_units = EXCLUDED.max_units,
    updated_at = CURRENT_TIMESTAMP;

-- name: ListWorkersWithSlotCount :many
SELECT
    sqlc.embed(workers),
    ww."url" AS "webhookUrl",
    ww."id" AS "webhookId",
    -- TODO do we still need this?
    COALESCE((
        SELECT COALESCE(cap.max_units, 0)
        FROM v1_worker_slot_capacity cap
        WHERE
            cap.tenant_id = workers."tenantId"
            AND cap.worker_id = workers."id"
            AND cap.slot_type = 'default'::text
    ) - (
        SELECT COALESCE(SUM(runtime.units), 0)
        FROM v1_task_runtime_slot runtime
        WHERE
            runtime.tenant_id = workers."tenantId" AND
            runtime.worker_id = workers."id" AND
            runtime.slot_type = 'default'::text
    ), 0)::int AS "remainingSlots"
    ,
    COALESCE((
        (
            SELECT COALESCE(cap.max_units, 0)
            FROM v1_worker_slot_capacity cap
            WHERE
                cap.tenant_id = workers."tenantId"
                AND cap.worker_id = workers."id"
                AND cap.slot_type = 'durable'::text
        ) - (
            SELECT COALESCE(SUM(runtime.units), 0)
            FROM v1_task_runtime_slot runtime
            WHERE
                runtime.tenant_id = workers."tenantId" AND
                runtime.worker_id = workers."id" AND
                runtime.slot_type = 'durable'::text
        )
    ), 0)::int AS "remainingDurableSlots"
FROM
    "Worker" workers
LEFT JOIN
    "WebhookWorker" ww ON workers."webhookId" = ww."id"
WHERE
    workers."tenantId" = @tenantId
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
        workers."maxRuns" IS NULL OR
        (sqlc.narg('assignable')::boolean AND workers."maxRuns" > (
            SELECT COUNT(*)
            FROM "StepRun" srs
            WHERE srs."workerId" = workers."id" AND srs."status" = 'RUNNING'
        ))
    )
GROUP BY
    workers."id", ww."url", ww."id";

-- name: GetWorkerById :one
SELECT
    sqlc.embed(w),
    ww."url" AS "webhookUrl",
    COALESCE((
        SELECT COALESCE(cap.max_units, 0)
        FROM v1_worker_slot_capacity cap
        WHERE
            cap.tenant_id = w."tenantId"
            AND cap.worker_id = w."id"
            AND cap.slot_type = 'default'::text
    ) - (
        SELECT COALESCE(SUM(runtime.units), 0)
        FROM v1_task_runtime_slot runtime
        WHERE
            runtime.tenant_id = w."tenantId" AND
            runtime.worker_id = w."id" AND
            runtime.slot_type = 'default'::text
    ), 0)::int AS "remainingSlots"
    ,
    COALESCE((
        (
            SELECT COALESCE(cap.max_units, 0)
            FROM v1_worker_slot_capacity cap
            WHERE
                cap.tenant_id = w."tenantId"
                AND cap.worker_id = w."id"
                AND cap.slot_type = 'durable'::text
        ) - (
            SELECT COALESCE(SUM(runtime.units), 0)
            FROM v1_task_runtime_slot runtime
            WHERE
                runtime.tenant_id = w."tenantId" AND
                runtime.worker_id = w."id" AND
                runtime.slot_type = 'durable'::text
        )
    ), 0)::int AS "remainingDurableSlots"
FROM
    "Worker" w
LEFT JOIN
    "WebhookWorker" ww ON w."webhookId" = ww."id"
WHERE
    w."id" = @id::uuid;

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
FROM v1_worker_slot_capacity wc
JOIN "Worker" w ON w."id" = wc.worker_id AND w."tenantId" = wc.tenant_id
WHERE
    w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false
GROUP BY wc.tenant_id
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
WITH inputs AS (
    SELECT UNNEST(@workerIds::UUID[]) AS "workerId"
)

SELECT
    w."id" AS "workerId",
    a."actionId" AS actionId
FROM "Worker" w
JOIN inputs i ON w."id" = i."workerId"
LEFT JOIN "_ActionToWorker" aw ON w.id = aw."B"
LEFT JOIN "Action" a ON aw."A" = a.id
WHERE
    a."tenantId" = @tenantId::UUID
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
SELECT
    w."id" AS "id",
    w."tenantId" AS "tenantId",
    w."dispatcherId" AS "dispatcherId",
    d."lastHeartbeatAt" AS "dispatcherLastHeartbeatAt",
    w."isActive" AS "isActive",
    w."lastListenerEstablished" AS "lastListenerEstablished"
FROM
    "Worker" w
LEFT JOIN
    "Dispatcher" d ON w."dispatcherId" = d."id"
WHERE
    w."tenantId" = @tenantId
    AND w."id" = @id;

-- name: ListWorkerLabels :many
SELECT
    "id",
    "key",
    "intValue",
    "strValue",
    "createdAt",
    "updatedAt"
FROM "WorkerLabel" wl
WHERE wl."workerId" = @workerId::uuid;

-- name: UpdateWorker :one
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "dispatcherId" = coalesce(sqlc.narg('dispatcherId')::uuid, "dispatcherId"),
    "maxRuns" = coalesce(sqlc.narg('maxRuns')::int, "maxRuns"),
    "durableMaxRuns" = coalesce(sqlc.narg('durableMaxRuns')::int, "durableMaxRuns"),
    "lastHeartbeatAt" = coalesce(sqlc.narg('lastHeartbeatAt')::timestamp, "lastHeartbeatAt"),
    "isActive" = coalesce(sqlc.narg('isActive')::boolean, "isActive"),
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

-- name: UpdateWorkerHeartbeat :one
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "lastHeartbeatAt" = sqlc.narg('lastHeartbeatAt')::timestamp
WHERE
    "id" = @id::uuid
RETURNING *;

-- name: DeleteWorker :one
DELETE FROM
  "Worker"
WHERE
  "id" = @id::uuid
RETURNING *;

-- name: UpdateWorkerActiveStatus :one
UPDATE "Worker"
SET
    "isActive" = @isActive::boolean,
    "lastListenerEstablished" = sqlc.narg('lastListenerEstablished')::timestamp
WHERE
    "id" = @id::uuid
    AND (
        "lastListenerEstablished" IS NULL
        OR "lastListenerEstablished" <= sqlc.narg('lastListenerEstablished')::timestamp
        )
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

-- name: DeleteOldWorkers :one
WITH for_delete AS (
    SELECT
        "id"
    FROM "Worker" w
    WHERE
        w."tenantId" = @tenantId::uuid AND
        w."lastHeartbeatAt" < @lastHeartbeatBefore::timestamp
    LIMIT sqlc.arg('limit') + 1
), expired_with_limit AS (
    SELECT
        for_delete."id" as "id"
    FROM for_delete
    LIMIT sqlc.arg('limit')
), has_more AS (
    SELECT
        CASE
            WHEN COUNT(*) > sqlc.arg('limit') THEN TRUE
            ELSE FALSE
        END as has_more
    FROM for_delete
), delete_events AS (
    DELETE FROM "WorkerAssignEvent" wae
    WHERE wae."workerId" IN (SELECT "id" FROM expired_with_limit)
    RETURNING wae."id"
)
DELETE FROM "Worker" w
WHERE w."id" IN (SELECT "id" FROM expired_with_limit)
RETURNING
    (SELECT has_more FROM has_more) as has_more;

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
    "maxRuns",
    "durableMaxRuns",
    "webhookId",
    "type",
    "sdkVersion",
    "language",
    "languageVersion",
    "os",
    "runtimeExtra"
) VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @name::text,
    @dispatcherId::uuid,
    sqlc.narg('maxRuns')::int,
    coalesce(sqlc.narg('durableMaxRuns')::int, 0),
    sqlc.narg('webhookId')::uuid,
    sqlc.narg('type')::"WorkerType",
    sqlc.narg('sdkVersion')::text,
    sqlc.narg('language')::"WorkerSDKS",
    sqlc.narg('languageVersion')::text,
    sqlc.narg('os')::text,
    sqlc.narg('runtimeExtra')::text
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
