-- Create "SemaphoreQueueItem" table
CREATE TABLE "SemaphoreQueueItem" ("id" bigserial NOT NULL, "stepRunId" uuid NOT NULL, "workerId" uuid NOT NULL, "tenantId" uuid NOT NULL, PRIMARY KEY ("id"));
-- Create index "SemaphoreQueueItem_stepRunId_workerId_key" to table: "SemaphoreQueueItem"
CREATE UNIQUE INDEX "SemaphoreQueueItem_stepRunId_workerId_key" ON "SemaphoreQueueItem" ("stepRunId", "workerId");
-- Create index "SemaphoreQueueItem_tenantId_workerId_idx" to table: "SemaphoreQueueItem"
CREATE INDEX "SemaphoreQueueItem_tenantId_workerId_idx" ON "SemaphoreQueueItem" ("tenantId", "workerId");
