-- CreateIndex
CREATE INDEX "GetGroupKeyRun_deletedAt_idx" ON "GetGroupKeyRun"("deletedAt");

-- CreateIndex
CREATE INDEX "JobRun_deletedAt_idx" ON "JobRun"("deletedAt");

-- CreateIndex
CREATE INDEX "StepRun_deletedAt_idx" ON "StepRun"("deletedAt");

-- CreateIndex
CREATE INDEX "WorkflowVersion_deletedAt_idx" ON "WorkflowVersion"("deletedAt");
