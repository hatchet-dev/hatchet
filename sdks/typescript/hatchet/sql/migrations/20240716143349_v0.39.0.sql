-- atlas:txmode none

-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_deletedAt_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_deletedAt_idx" ON "GetGroupKeyRun" ("deletedAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "JobRun_deletedAt_idx" to table: "JobRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "JobRun_deletedAt_idx" ON "JobRun" ("deletedAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_deletedAt_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_deletedAt_idx" ON "StepRun" ("deletedAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Workflow_deletedAt_idx" to table: "Workflow"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Workflow_deletedAt_idx" ON "Workflow" ("deletedAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_deletedAt_idx" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_deletedAt_idx" ON "WorkflowRun" ("deletedAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowVersion_deletedAt_idx" to table: "WorkflowVersion"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowVersion_deletedAt_idx" ON "WorkflowVersion" ("deletedAt");
