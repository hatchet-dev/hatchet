-- atlas:txmode none

-- REVERSE: DROP INDEX CONCURRENTLY IF EXISTS "Action_tenantId_idx" to table: "Action"
DROP INDEX CONCURRENTLY IF EXISTS "Action_tenantId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Job_tenantId_idx" to table: "Job"
DROP INDEX CONCURRENTLY IF EXISTS "Job_tenantId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Step_jobId_idx" to table: "Step"
DROP INDEX CONCURRENTLY IF EXISTS "Step_jobId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Step_tenantId_idx" to table: "Step"
DROP INDEX CONCURRENTLY IF EXISTS "Step_tenantId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "StepRun_status_idx" to table: "StepRun"
DROP INDEX CONCURRENTLY IF EXISTS "StepRun_status_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "StepRun_timeoutAt_idx" to table: "StepRun"
DROP INDEX CONCURRENTLY IF EXISTS "StepRun_timeoutAt_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "TenantResourceLimit_tenantId_idx" to table: "TenantResourceLimit"
DROP INDEX CONCURRENTLY IF EXISTS "TenantResourceLimit_tenantId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "TenantResourceLimitAlert_tenantId_idx" to table: "TenantResourceLimitAlert"
DROP INDEX CONCURRENTLY IF EXISTS "TenantResourceLimitAlert_tenantId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Ticker_isActive_idx" to table: "Ticker"
DROP INDEX CONCURRENTLY IF EXISTS "Ticker_isActive_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Ticker_lastHeartbeatAt_idx" to table: "Ticker"
DROP INDEX CONCURRENTLY IF EXISTS "Ticker_lastHeartbeatAt_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Worker_isActive_idx" to table: "Worker"
DROP INDEX CONCURRENTLY IF EXISTS "Worker_isActive_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Worker_lastHeartbeatAt_idx" to table: "Worker"
DROP INDEX CONCURRENTLY IF EXISTS "Worker_lastHeartbeatAt_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Worker_tenantId_idx" to table: "Worker"
DROP INDEX CONCURRENTLY IF EXISTS "Worker_tenantId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "Workflow_tenantId_idx" to table: "Workflow"
DROP INDEX CONCURRENTLY IF EXISTS "Workflow_tenantId_idx";
-- REVERSE: CREATE INDEX CONCURRENTLY IF EXISTS "WorkflowVersion_workflowId_idx" to table: "WorkflowVersion"
DROP INDEX CONCURRENTLY IF EXISTS "WorkflowVersion_workflowId_idx";