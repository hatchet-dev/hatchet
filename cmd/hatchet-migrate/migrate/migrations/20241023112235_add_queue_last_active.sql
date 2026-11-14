-- +goose Up
-- Modify "Queue" table
ALTER TABLE "Queue" ADD COLUMN "lastActive" timestamp(3) NULL;
-- Create index "Queue_tenantId_lastActive_idx" to table: "Queue"
CREATE INDEX "Queue_tenantId_lastActive_idx" ON "Queue" ("tenantId", "lastActive");

-- Upsert the lastActive column for all queues, to avoid very old queues from being picked
-- up by the scheduler
WITH unique_queues AS (
    SELECT DISTINCT ON ("tenantId", "queue")
        "tenantId",
        "queue"
    FROM
        "StepRun"
    WHERE
        "createdAt" > NOW() - INTERVAL '1 day'
)
INSERT INTO "Queue" (
    "tenantId",
    "name",
    "lastActive"
)
SELECT
    "tenantId",
    "queue",
    NOW()
FROM
    unique_queues
ON CONFLICT ("tenantId", "name") DO UPDATE
SET
    "lastActive" = NOW();
