-- CreateIndex
CREATE INDEX "JobRun_workflowRunId_tenantId_idx" ON "JobRun"("workflowRunId", "tenantId");

-- CreateIndex
CREATE INDEX "StepRun_tenantId_status_input_requeueAfter_createdAt_idx" ON "StepRun"("tenantId", "status", "input", "requeueAfter", "createdAt");

-- CreateIndex
CREATE INDEX "StepRun_stepId_idx" ON "StepRun"("stepId");

-- CreateIndex
CREATE INDEX "StepRun_jobRunId_status_idx" ON "StepRun"("jobRunId", "status");

-- CreateIndex
CREATE INDEX "StepRun_id_tenantId_idx" ON "StepRun"("id", "tenantId");

-- CreateIndex
CREATE INDEX "StepRun_jobRunId_tenantId_order_idx" ON "StepRun"("jobRunId", "tenantId", "order");
