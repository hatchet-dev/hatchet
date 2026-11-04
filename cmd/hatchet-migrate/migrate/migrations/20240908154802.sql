-- +goose Up
INSERT INTO "WorkerSemaphoreCount" ("workerId", "count")
SELECT
    "workerId",
    COUNT(*) as "count"
FROM
    "WorkerSemaphoreSlot"
JOIN
    "Worker" w ON "WorkerSemaphoreSlot"."workerId" = w."id"
WHERE
    "stepRunId" IS NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '15 seconds'
GROUP BY
    "workerId"
ON CONFLICT ("workerId")
DO UPDATE SET "count" = EXCLUDED."count";
