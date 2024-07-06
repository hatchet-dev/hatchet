-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Action_tenantId_idx" to table: "Action"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Action_tenantId_idx" ON "Action" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Job_tenantId_idx" to table: "Job"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Job_tenantId_idx" ON "Job" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Step_jobId_idx" to table: "Step"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Step_jobId_idx" ON "Step" ("jobId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Step_tenantId_idx" to table: "Step"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Step_tenantId_idx" ON "Step" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_idx" ON "StepRun" ("status");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_timeoutAt_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_timeoutAt_idx" ON "StepRun" ("timeoutAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "TenantResourceLimit_tenantId_idx" to table: "TenantResourceLimit"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "TenantResourceLimit_tenantId_idx" ON "TenantResourceLimit" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "TenantResourceLimitAlert_tenantId_idx" to table: "TenantResourceLimitAlert"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "TenantResourceLimitAlert_tenantId_idx" ON "TenantResourceLimitAlert" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Ticker_isActive_idx" to table: "Ticker"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Ticker_isActive_idx" ON "Ticker" ("isActive");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Ticker_lastHeartbeatAt_idx" to table: "Ticker"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Ticker_lastHeartbeatAt_idx" ON "Ticker" ("lastHeartbeatAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Worker_isActive_idx" to table: "Worker"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Worker_isActive_idx" ON "Worker" ("isActive");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Worker_lastHeartbeatAt_idx" to table: "Worker"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Worker_lastHeartbeatAt_idx" ON "Worker" ("lastHeartbeatAt");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Worker_tenantId_idx" to table: "Worker"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Worker_tenantId_idx" ON "Worker" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "Workflow_tenantId_idx" to table: "Workflow"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Workflow_tenantId_idx" ON "Workflow" ("tenantId");
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowVersion_workflowId_idx" to table: "WorkflowVersion"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowVersion_workflowId_idx" ON "WorkflowVersion" ("workflowId");
