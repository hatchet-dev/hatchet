-- Modify "WorkerAssignEvent" table
ALTER TABLE "WorkerAssignEvent" DROP CONSTRAINT "WorkerAssignEvent_workerId_fkey", ALTER COLUMN "workerId" DROP NOT NULL, ADD CONSTRAINT "WorkerAssignEvent_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE SET NULL NOT VALID;
