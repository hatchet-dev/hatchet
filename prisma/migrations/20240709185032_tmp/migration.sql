-- DropIndex
DROP INDEX "StepRun_tenantId_status_requeueAfter_createdAt_idx";

-- CreateIndex
CREATE INDEX "StepRun_tenantId_status_createdAt_idx" ON "StepRun"("tenantId", "status", "createdAt");