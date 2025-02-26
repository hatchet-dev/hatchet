-- +goose Up
-- Create enum type "InternalQueue"
CREATE TYPE "InternalQueue" AS ENUM ('WORKER_SEMAPHORE_COUNT', 'STEP_RUN_UPDATE');
-- Create "InternalQueueItem" table
CREATE TABLE "InternalQueueItem" ("id" bigserial NOT NULL, "queue" "InternalQueue" NOT NULL, "isQueued" boolean NOT NULL, "data" jsonb NULL, "tenantId" uuid NOT NULL, "priority" integer NOT NULL DEFAULT 1, "uniqueKey" text NULL, PRIMARY KEY ("id"), CONSTRAINT "InternalQueueItem_priority_check" CHECK ((priority >= 1) AND (priority <= 4)));
-- Create index "InternalQueueItem_isQueued_tenantId_queue_priority_id_idx" to table: "InternalQueueItem"
CREATE INDEX "InternalQueueItem_isQueued_tenantId_queue_priority_id_idx" ON "InternalQueueItem" ("isQueued", "tenantId", "queue", "priority" DESC, "id");
-- Create index "InternalQueueItem_tenantId_queue_uniqueKey_key" to table: "InternalQueueItem"
CREATE UNIQUE INDEX "InternalQueueItem_tenantId_queue_uniqueKey_key" ON "InternalQueueItem" ("tenantId", "queue", "uniqueKey");
-- Create "WorkerAssignEvent" table
CREATE TABLE "WorkerAssignEvent" ("id" bigserial NOT NULL, "workerId" uuid NOT NULL, "assignedStepRuns" jsonb NULL, PRIMARY KEY ("id"), CONSTRAINT "WorkerAssignEvent_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkerAssignEvent_workerId_id_idx" to table: "WorkerAssignEvent"
CREATE INDEX "WorkerAssignEvent_workerId_id_idx" ON "WorkerAssignEvent" ("workerId", "id");
-- Create "WorkerSemaphoreCount" table
CREATE TABLE "WorkerSemaphoreCount" ("workerId" uuid NOT NULL, "count" integer NOT NULL, PRIMARY KEY ("workerId"), CONSTRAINT "WorkerSemaphoreCount_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkerSemaphoreCount_workerId_idx" to table: "WorkerSemaphoreCount"
CREATE INDEX "WorkerSemaphoreCount_workerId_idx" ON "WorkerSemaphoreCount" ("workerId");
-- Create index "WorkerSemaphoreCount_workerId_key" to table: "WorkerSemaphoreCount"
CREATE UNIQUE INDEX "WorkerSemaphoreCount_workerId_key" ON "WorkerSemaphoreCount" ("workerId");
