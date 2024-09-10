-- Create "TimeoutQueueItem" table
CREATE TABLE "TimeoutQueueItem" ("id" bigserial NOT NULL, "stepRunId" uuid NOT NULL, "retryCount" integer NOT NULL, "timeoutAt" timestamp(3) NOT NULL, "tenantId" uuid NOT NULL, "isQueued" boolean NOT NULL, PRIMARY KEY ("id"));
-- Create index "TimeoutQueueItem_stepRunId_retryCount_key" to table: "TimeoutQueueItem"
CREATE UNIQUE INDEX "TimeoutQueueItem_stepRunId_retryCount_key" ON "TimeoutQueueItem" ("stepRunId", "retryCount");
-- Create index "TimeoutQueueItem_tenantId_isQueued_timeoutAt_idx" to table: "TimeoutQueueItem"
CREATE INDEX "TimeoutQueueItem_tenantId_isQueued_timeoutAt_idx" ON "TimeoutQueueItem" ("tenantId", "isQueued", "timeoutAt");
