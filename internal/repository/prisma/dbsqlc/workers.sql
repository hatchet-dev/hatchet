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
            FROM "StepRun"
            WHERE runs."workerId" = workers."id" AND runs."status" = 'RUNNING'
        ))
    )
GROUP BY
    workers."id";