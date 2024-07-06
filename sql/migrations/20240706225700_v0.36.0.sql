-- Create index "Action_tenantId_idx" to table: "Action"
CREATE INDEX "Action_tenantId_idx" ON "Action" ("tenantId");
-- Create index "Job_tenantId_idx" to table: "Job"
CREATE INDEX "Job_tenantId_idx" ON "Job" ("tenantId");
-- Create index "Step_jobId_idx" to table: "Step"
CREATE INDEX "Step_jobId_idx" ON "Step" ("jobId");
-- Create index "Step_tenantId_idx" to table: "Step"
CREATE INDEX "Step_tenantId_idx" ON "Step" ("tenantId");
-- Create index "StepRun_status_idx" to table: "StepRun"
CREATE INDEX "StepRun_status_idx" ON "StepRun" ("status");
-- Create index "StepRun_timeoutAt_idx" to table: "StepRun"
CREATE INDEX "StepRun_timeoutAt_idx" ON "StepRun" ("timeoutAt");
-- Create index "TenantResourceLimit_tenantId_idx" to table: "TenantResourceLimit"
CREATE INDEX "TenantResourceLimit_tenantId_idx" ON "TenantResourceLimit" ("tenantId");
-- Create index "TenantResourceLimitAlert_tenantId_idx" to table: "TenantResourceLimitAlert"
CREATE INDEX "TenantResourceLimitAlert_tenantId_idx" ON "TenantResourceLimitAlert" ("tenantId");
-- Create index "Ticker_isActive_idx" to table: "Ticker"
CREATE INDEX "Ticker_isActive_idx" ON "Ticker" ("isActive");
-- Create index "Ticker_lastHeartbeatAt_idx" to table: "Ticker"
CREATE INDEX "Ticker_lastHeartbeatAt_idx" ON "Ticker" ("lastHeartbeatAt");
-- Create index "Worker_isActive_idx" to table: "Worker"
CREATE INDEX "Worker_isActive_idx" ON "Worker" ("isActive");
-- Create index "Worker_lastHeartbeatAt_idx" to table: "Worker"
CREATE INDEX "Worker_lastHeartbeatAt_idx" ON "Worker" ("lastHeartbeatAt");
-- Create index "Worker_tenantId_idx" to table: "Worker"
CREATE INDEX "Worker_tenantId_idx" ON "Worker" ("tenantId");
-- Create index "Workflow_tenantId_idx" to table: "Workflow"
CREATE INDEX "Workflow_tenantId_idx" ON "Workflow" ("tenantId");
-- Create index "WorkflowVersion_workflowId_idx" to table: "WorkflowVersion"
CREATE INDEX "WorkflowVersion_workflowId_idx" ON "WorkflowVersion" ("workflowId");
