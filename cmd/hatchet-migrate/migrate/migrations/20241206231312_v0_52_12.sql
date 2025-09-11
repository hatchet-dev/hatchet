-- +goose Up
-- +goose NO TRANSACTION

CREATE INDEX CONCURRENTLY "Worker_tenantId_lastHeartbeatAt_idx" ON "Worker" ("tenantId", "lastHeartbeatAt");


-- Modify "WorkerAssignEvent" table
ALTER TABLE "WorkerAssignEvent" DROP CONSTRAINT "WorkerAssignEvent_workerId_fkey", ALTER COLUMN "workerId" DROP NOT NULL, ADD CONSTRAINT "WorkerAssignEvent_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE SET NULL NOT VALID;
