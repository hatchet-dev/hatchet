-- +goose Up
-- Create "TimeoutQueueItem" table
CREATE TABLE "TimeoutQueueItem" ("id" bigserial NOT NULL, "stepRunId" uuid NOT NULL, "retryCount" integer NOT NULL, "timeoutAt" timestamp(3) NOT NULL, "tenantId" uuid NOT NULL, "isQueued" boolean NOT NULL, PRIMARY KEY ("id"));
-- Create index "TimeoutQueueItem_stepRunId_retryCount_key" to table: "TimeoutQueueItem"
CREATE UNIQUE INDEX "TimeoutQueueItem_stepRunId_retryCount_key" ON "TimeoutQueueItem" ("stepRunId", "retryCount");
-- Create index "TimeoutQueueItem_tenantId_isQueued_timeoutAt_idx" to table: "TimeoutQueueItem"
CREATE INDEX "TimeoutQueueItem_tenantId_isQueued_timeoutAt_idx" ON "TimeoutQueueItem" ("tenantId", "isQueued", "timeoutAt");

-- Migrate all running and assigned step runs to TimeoutQueueItem
INSERT INTO "TimeoutQueueItem" ("stepRunId", "retryCount", "timeoutAt", "tenantId", "isQueued")
SELECT 
    "id" AS "stepRunId",
    "retryCount",
    "timeoutAt",
    "tenantId",
    true
FROM 
    "StepRun"
WHERE 
    "status" IN ('RUNNING', 'ASSIGNED')
    AND "timeoutAt" IS NOT NULL
ON CONFLICT DO NOTHING;
