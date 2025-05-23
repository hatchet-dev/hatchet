-- name: ListWorkersWithSlotCount :many
SELECT
    sqlc.embed(workers),
    ww."url" AS "webhookUrl",
    ww."id" AS "webhookId",
    workers."maxRuns" - (
        SELECT COUNT(*)
        FROM "SemaphoreQueueItem" sqi
        WHERE
            sqi."tenantId" = workers."tenantId" AND
            sqi."workerId" = workers."id"
    ) AS "remainingSlots"
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
    w."maxRuns" - (
        SELECT COUNT(*)
        FROM "SemaphoreQueueItem" sqi
        WHERE
            sqi."tenantId" = w."tenantId" AND
            sqi."workerId" = w."id"
    ) AS "remainingSlots"
FROM
    "Worker" w
LEFT JOIN
    "WebhookWorker" ww ON w."webhookId" = ww."id"
WHERE
    w."id" = @id::uuid;

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

-- name: ListSemaphoreSlotsWithStateForWorker :many
SELECT
    sr."id" AS "stepRunId",
    sr."status" AS "status",
    s."actionId",
    sr."timeoutAt" AS "timeoutAt",
    sr."startedAt" AS "startedAt",
    jr."workflowRunId" AS "workflowRunId"
FROM
    "SemaphoreQueueItem" sqi
JOIN
    "StepRun" sr ON sr."id" = sqi."stepRunId"
JOIN
    "JobRun" jr ON sr."jobRunId" = jr."id"
JOIN
    "Step" s ON sr."stepId" = s."id"
WHERE
    sqi."tenantId" = @tenantId::uuid
    AND sqi."workerId" = @workerId::uuid
ORDER BY
    sr."createdAt" DESC
LIMIT
    COALESCE(sqlc.narg('limit')::int, 100);

-- name: ListRecentAssignedEventsForWorker :many
SELECT
    "workerId",
    "assignedStepRuns"
FROM
    "WorkerAssignEvent"
WHERE
    "workerId" = @workerId::uuid
ORDER BY "id" DESC
LIMIT
    COALESCE(sqlc.narg('limit')::int, 100);

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

-- name: CreateWorker :one
INSERT INTO "Worker" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "name",
    "dispatcherId",
    "maxRuns",
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
    sqlc.narg('webhookId')::uuid,
    sqlc.narg('type')::"WorkerType",
    sqlc.narg('sdkVersion')::text,
    sqlc.narg('language')::"WorkerSDKS",
    sqlc.narg('languageVersion')::text,
    sqlc.narg('os')::text,
    sqlc.narg('runtimeExtra')::text
) RETURNING *;

-- name: GetWorkerByWebhookId :one
SELECT
    *
FROM
    "Worker"
WHERE
    "webhookId" = @webhookId::uuid
    AND "tenantId" = @tenantId::uuid;

-- name: UpdateWorkerHeartbeat :one
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "lastHeartbeatAt" = sqlc.narg('lastHeartbeatAt')::timestamp
WHERE
    "id" = @id::uuid
RETURNING *;

-- name: UpdateWorker :one
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "dispatcherId" = coalesce(sqlc.narg('dispatcherId')::uuid, "dispatcherId"),
    "maxRuns" = coalesce(sqlc.narg('maxRuns')::int, "maxRuns"),
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

-- name: DeleteWorker :one
DELETE FROM
  "Worker"
WHERE
  "id" = @id::uuid
RETURNING *;

-- name: UpdateWorkersByWebhookId :many
UPDATE "Worker"
SET "isActive" = @isActive::boolean
WHERE
  "tenantId" = @tenantId::uuid AND
  "webhookId" = @webhookId::uuid
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

-- name: DeleteOldWorkerAssignEvents :one
-- delete worker assign events outside of the first <maxRuns> events for a worker
WITH for_delete AS (
    SELECT
        "id"
    FROM "WorkerAssignEvent" wae
    WHERE
        wae."workerId" = @workerId::uuid
    ORDER BY wae."id" DESC
    OFFSET sqlc.arg('maxRuns')::int
    LIMIT sqlc.arg('limit')::int + 1
), has_more AS (
    SELECT
        CASE
            WHEN COUNT(*) > sqlc.arg('limit') THEN TRUE
            ELSE FALSE
        END as has_more
    FROM for_delete
)
DELETE FROM "WorkerAssignEvent" wae
WHERE wae."id" IN (SELECT "id" FROM for_delete)
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
