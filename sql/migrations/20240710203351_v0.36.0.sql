-- atlas:txmode none

-- Drop index "StepRun_tenantId_status_requeueAfter_createdAt_idx" from table: "StepRun"
DROP INDEX CONCURRENTLY IF EXISTS "StepRun_tenantId_status_requeueAfter_createdAt_idx";
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_createdAt_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_createdAt_idx" ON "StepRun" ("createdAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_jobRunId_status_tenantId_requeueAfter_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_jobRunId_status_tenantId_requeueAfter_idx" ON "StepRun" ("jobRunId", "status", "tenantId", "requeueAfter");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_timeoutAt_tickerId_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_timeoutAt_tickerId_idx" ON "StepRun" ("status", "timeoutAt", "tickerId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_tenantId_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_tenantId_idx" ON "StepRun" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_workerId_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_workerId_idx" ON "StepRun" ("workerId");
