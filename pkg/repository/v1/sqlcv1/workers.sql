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

-- name: ListWorkersWithSlotCount :many
SELECT
    sqlc.embed(workers),
    ww."url" AS "webhookUrl",
    ww."id" AS "webhookId",
    workers."maxRuns" - (
        SELECT COUNT(*)
        FROM v1_task_runtime runtime
        WHERE
            runtime.tenant_id = workers."tenantId" AND
            runtime.worker_id = workers."id"
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
