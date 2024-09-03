-- Drop index "WorkerAssignEvent_workerId_idx" from table: "WorkerAssignEvent"
DROP INDEX "WorkerAssignEvent_workerId_idx";
-- Create index "WorkerAssignEvent_workerId_id_idx" to table: "WorkerAssignEvent"
CREATE INDEX "WorkerAssignEvent_workerId_id_idx" ON "WorkerAssignEvent" ("workerId", "id");
-- Modify "WorkerSemaphoreQueueItem" table
ALTER TABLE "WorkerSemaphoreQueueItem" ADD COLUMN "isAssigned" boolean NOT NULL DEFAULT false;
