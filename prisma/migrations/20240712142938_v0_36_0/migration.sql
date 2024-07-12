-- DropIndex
DROP INDEX "StepRun_tenantId_status_requeueAfter_createdAt_idx";

-- CreateIndex
CREATE INDEX "StepRun_tenantId_idx" ON "StepRun"("tenantId");

-- CreateIndex
CREATE INDEX "StepRun_workerId_idx" ON "StepRun"("workerId");

-- CreateIndex
CREATE INDEX "StepRun_createdAt_idx" ON "StepRun"("createdAt");

-- CreateIndex
CREATE INDEX "StepRun_jobRunId_status_tenantId_idx" ON "StepRun"("jobRunId", "status", "tenantId");

-- CreateIndex
CREATE INDEX "StepRun_tenantId_status_timeoutAt_idx" ON "StepRun"("tenantId", "status", "timeoutAt");
