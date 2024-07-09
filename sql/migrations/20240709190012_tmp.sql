-- Drop index "StepRun_tenantId_status_createdAt_idx" from table: "StepRun"
DROP INDEX "StepRun_tenantId_status_createdAt_idx";
-- Create index "StepRun_createdAt_idx" to table: "StepRun"
CREATE INDEX "StepRun_createdAt_idx" ON "StepRun" ("createdAt");
-- Create index "StepRun_status_idx" to table: "StepRun"
CREATE INDEX "StepRun_status_idx" ON "StepRun" ("status");
-- Create index "StepRun_tenantId_idx" to table: "StepRun"
CREATE INDEX "StepRun_tenantId_idx" ON "StepRun" ("tenantId");
