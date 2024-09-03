-- Create "WorkerSemaphoreQueueItem" table
CREATE TABLE "WorkerSemaphoreQueueItem" ("id" bigserial NOT NULL, "stepRunId" uuid NOT NULL, "workerId" uuid NOT NULL, "retryCount" integer NOT NULL, "isProcessed" boolean NOT NULL, PRIMARY KEY ("id"));
-- Create index "WorkerSemaphoreQueueItem_isProcessed_workerId_id_idx" to table: "WorkerSemaphoreQueueItem"
CREATE INDEX "WorkerSemaphoreQueueItem_isProcessed_workerId_id_idx" ON "WorkerSemaphoreQueueItem" ("isProcessed", "workerId", "id");
-- Create index "WorkerSemaphoreQueueItem_stepRunId_workerId_retryCount_key" to table: "WorkerSemaphoreQueueItem"
CREATE UNIQUE INDEX "WorkerSemaphoreQueueItem_stepRunId_workerId_retryCount_key" ON "WorkerSemaphoreQueueItem" ("stepRunId", "workerId", "retryCount");
-- Create "WorkerAssignEvent" table
CREATE TABLE "WorkerAssignEvent" ("id" bigserial NOT NULL, "workerId" uuid NOT NULL, "assignedStepRuns" jsonb NULL, PRIMARY KEY ("id"), CONSTRAINT "WorkerAssignEvent_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkerAssignEvent_workerId_idx" to table: "WorkerAssignEvent"
CREATE INDEX "WorkerAssignEvent_workerId_idx" ON "WorkerAssignEvent" ("workerId");
-- Create "WorkerSemaphoreCount" table
CREATE TABLE "WorkerSemaphoreCount" ("workerId" uuid NOT NULL, "count" integer NOT NULL, PRIMARY KEY ("workerId"), CONSTRAINT "WorkerSemaphoreCount_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkerSemaphoreCount_workerId_idx" to table: "WorkerSemaphoreCount"
CREATE INDEX "WorkerSemaphoreCount_workerId_idx" ON "WorkerSemaphoreCount" ("workerId");
-- Create index "WorkerSemaphoreCount_workerId_key" to table: "WorkerSemaphoreCount"
CREATE UNIQUE INDEX "WorkerSemaphoreCount_workerId_key" ON "WorkerSemaphoreCount" ("workerId");
