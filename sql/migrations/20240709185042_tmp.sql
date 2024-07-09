-- Drop index "StepRun_tenantId_status_requeueAfter_createdAt_idx" from table: "StepRun"
DROP INDEX "StepRun_tenantId_status_requeueAfter_createdAt_idx";
-- Create index "StepRun_tenantId_status_createdAt_idx" to table: "StepRun"
CREATE INDEX "StepRun_tenantId_status_createdAt_idx" ON "StepRun" ("tenantId", "status", "createdAt");
