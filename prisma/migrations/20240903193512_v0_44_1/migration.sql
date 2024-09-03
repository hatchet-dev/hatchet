-- DropIndex
DROP INDEX "WorkerAssignEvent_workerId_idx";

-- CreateIndex
CREATE INDEX "WorkerAssignEvent_workerId_id_idx" ON "WorkerAssignEvent"("workerId", "id");
