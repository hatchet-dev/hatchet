-- CreateIndex
CREATE INDEX "GetGroupKeyRun_tenantId_idx" ON "GetGroupKeyRun"("tenantId");

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_workerId_idx" ON "GetGroupKeyRun"("workerId");

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_createdAt_idx" ON "GetGroupKeyRun"("createdAt");

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_tenantId_deletedAt_status_idx" ON "GetGroupKeyRun"("tenantId", "deletedAt", "status");

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_status_deletedAt_timeoutAt_idx" ON "GetGroupKeyRun"("status", "deletedAt", "timeoutAt");
