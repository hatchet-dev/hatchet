-- Create index "GetGroupKeyRun_status_timeoutAt_tickerId_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_status_timeoutAt_tickerId_idx" ON "GetGroupKeyRun" ("status", "timeoutAt", "tickerId");
-- Drop index "StepRun_tenantId_status_requeueAfter_createdAt_idx" from table: "StepRun"
DROP INDEX CONCURRENTLY IF EXISTS "StepRun_tenantId_status_requeueAfter_createdAt_idx";
-- Create index "StepRun_createdAt_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_createdAt_idx" ON "StepRun" ("createdAt");
-- Create index "StepRun_requeueAfter_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_requeueAfter_idx" ON "StepRun" ("requeueAfter");
-- Create index "StepRun_status_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_idx" ON "StepRun" ("status");
-- Create index "StepRun_status_timeoutAt_tickerId_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_timeoutAt_tickerId_idx" ON "StepRun" ("status", "timeoutAt", "tickerId");
-- Create index "StepRun_tenantId_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_tenantId_idx" ON "StepRun" ("tenantId");
