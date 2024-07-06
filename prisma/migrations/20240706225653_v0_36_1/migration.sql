-- CreateIndex
CREATE INDEX "Action_tenantId_idx" ON "Action"("tenantId");

-- CreateIndex
CREATE INDEX "Job_tenantId_idx" ON "Job"("tenantId");

-- CreateIndex
CREATE INDEX "Step_tenantId_idx" ON "Step"("tenantId");

-- CreateIndex
CREATE INDEX "Step_jobId_idx" ON "Step"("jobId");

-- CreateIndex
CREATE INDEX "StepRun_timeoutAt_idx" ON "StepRun"("timeoutAt");

-- CreateIndex
CREATE INDEX "StepRun_status_idx" ON "StepRun"("status");

-- CreateIndex
CREATE INDEX "TenantResourceLimit_tenantId_idx" ON "TenantResourceLimit"("tenantId");

-- CreateIndex
CREATE INDEX "TenantResourceLimitAlert_tenantId_idx" ON "TenantResourceLimitAlert"("tenantId");

-- CreateIndex
CREATE INDEX "Ticker_isActive_idx" ON "Ticker"("isActive");

-- CreateIndex
CREATE INDEX "Ticker_lastHeartbeatAt_idx" ON "Ticker"("lastHeartbeatAt");

-- CreateIndex
CREATE INDEX "Worker_isActive_idx" ON "Worker"("isActive");

-- CreateIndex
CREATE INDEX "Worker_lastHeartbeatAt_idx" ON "Worker"("lastHeartbeatAt");

-- CreateIndex
CREATE INDEX "Worker_tenantId_idx" ON "Worker"("tenantId");

-- CreateIndex
CREATE INDEX "Workflow_tenantId_idx" ON "Workflow"("tenantId");

-- CreateIndex
CREATE INDEX "WorkflowVersion_workflowId_idx" ON "WorkflowVersion"("workflowId");
