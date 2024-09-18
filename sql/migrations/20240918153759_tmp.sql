-- Drop index "SemaphoreQueueItem_stepRunId_idx" from table: "SemaphoreQueueItem"
DROP INDEX "SemaphoreQueueItem_stepRunId_idx";
-- Modify "StepRun" table
ALTER TABLE "StepRun" ALTER COLUMN "id_uuid" SET NOT NULL;
