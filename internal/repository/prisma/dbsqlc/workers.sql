-- name: ListWorkersWithStepCount :many
SELECT
    sqlc.embed(workers),
    COUNT(runs."id") FILTER (WHERE runs."status" = 'RUNNING') AS "runningStepRuns"
FROM
    "Worker" workers
LEFT JOIN
    "StepRun" AS runs ON runs."workerId" = workers."id" AND runs."status" = 'RUNNING'
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
    workers."id";

-- name: GetWorkerForEngine :one
SELECT
    w."id" AS "id",
    w."tenantId" AS "tenantId",
    w."dispatcherId" AS "dispatcherId"
FROM
    "Worker" w
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
    "status",
    "dispatcherId",
    "maxRuns"
) VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @name::text,
    'ACTIVE',
    @dispatcherId::uuid,
    sqlc.narg('maxRuns')::int
) RETURNING *;

-- name: UpdateWorker :one
UPDATE
    "Worker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "status" = coalesce(sqlc.narg('status')::"WorkerStatus", "status"),
    "dispatcherId" = coalesce(sqlc.narg('dispatcherId')::uuid, "dispatcherId"),
    "maxRuns" = coalesce(sqlc.narg('maxRuns')::int, "maxRuns"),
    "lastHeartbeatAt" = coalesce(sqlc.narg('lastHeartbeatAt')::timestamp, "lastHeartbeatAt")
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