-- DropIndex
DROP INDEX "StepRun_tenantId_status_createdAt_idx";

-- CreateIndex
CREATE INDEX "StepRun_tenantId_idx" ON "StepRun"("tenantId");

-- CreateIndex
CREATE INDEX "StepRun_createdAt_idx" ON "StepRun"("createdAt");

-- CreateIndex
CREATE INDEX "StepRun_status_idx" ON "StepRun"("status");