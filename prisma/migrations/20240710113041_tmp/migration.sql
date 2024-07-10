-- DropIndex
DROP INDEX "Action_tenantId_idx";

-- DropIndex
DROP INDEX "Job_tenantId_idx";

-- DropIndex
DROP INDEX "Step_jobId_idx";

-- DropIndex
DROP INDEX "Step_tenantId_idx";

-- DropIndex
DROP INDEX "StepRun_tenantId_status_requeueAfter_createdAt_idx";

-- DropIndex
DROP INDEX "TenantResourceLimit_tenantId_idx";

-- DropIndex
DROP INDEX "TenantResourceLimitAlert_tenantId_idx";

-- DropIndex
DROP INDEX "Worker_isActive_idx";

-- DropIndex
DROP INDEX "Worker_lastHeartbeatAt_idx";

-- DropIndex
DROP INDEX "Worker_tenantId_idx";

-- DropIndex
DROP INDEX "Workflow_tenantId_idx";

-- DropIndex
DROP INDEX "WorkflowVersion_workflowId_idx";

-- CreateIndex
CREATE INDEX "StepRun_tenantId_idx" ON "StepRun"("tenantId");

-- CreateIndex
CREATE INDEX "StepRun_requeueAfter_idx" ON "StepRun"("requeueAfter");

-- CreateIndex
CREATE INDEX "StepRun_createdAt_idx" ON "StepRun"("createdAt");
