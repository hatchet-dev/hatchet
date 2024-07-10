-- DropIndex
DROP INDEX "StepRun_tenantId_status_requeueAfter_createdAt_idx";

-- CreateIndex
CREATE INDEX "StepRun_tenantId_idx" ON "StepRun"("tenantId");

-- CreateIndex
CREATE INDEX "StepRun_timeoutAt_idx" ON "StepRun"("timeoutAt");

-- CreateIndex
CREATE INDEX "StepRun_requeueAfter_idx" ON "StepRun"("requeueAfter");

-- CreateIndex
CREATE INDEX "StepRun_createdAt_idx" ON "StepRun"("createdAt");

-- CreateIndex
CREATE INDEX "StepRun_status_idx" ON "StepRun"("status");
