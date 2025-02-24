-- +goose Up
-- +goose NO TRANSACTION

-- Drop index "SemaphoreQueueItem_stepRunId_workerId_key" from table: "SemaphoreQueueItem"
DROP INDEX CONCURRENTLY IF EXISTS "SemaphoreQueueItem_stepRunId_workerId_key";
-- Modify "SemaphoreQueueItem" table
ALTER TABLE "SemaphoreQueueItem" DROP CONSTRAINT "SemaphoreQueueItem_pkey", DROP COLUMN "id", ADD PRIMARY KEY ("stepRunId");
-- Create index "SemaphoreQueueItem_stepRunId_key" to table: "SemaphoreQueueItem"
CREATE UNIQUE INDEX "SemaphoreQueueItem_stepRunId_key" ON "SemaphoreQueueItem" ("stepRunId");
-- Drop index "StepRun_tenantId_status_timeoutAt_idx" from table: "StepRun"
DROP INDEX CONCURRENTLY IF EXISTS "StepRun_tenantId_status_timeoutAt_idx";
-- Modify "StepRunResultArchive" table
ALTER TABLE "StepRunResultArchive" ADD COLUMN "retryCount" integer NOT NULL DEFAULT 0;

-- Drop index "StepRun_jobRunId_status_tenantId_idx" from table: "StepRun"
DROP INDEX CONCURRENTLY IF EXISTS "StepRun_jobRunId_status_tenantId_idx";

-- Create new partial index "StepRun_jobRunId_status_tenantId_idx" on table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_jobRunId_status_tenantId_idx" 
ON "StepRun" ("jobRunId", "status", "tenantId") 
WHERE "status" = 'PENDING';
