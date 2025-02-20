-- CreateIndex
CREATE INDEX "Event_tenantId_idx" ON "Event"("tenantId");

-- CreateIndex
CREATE INDEX "Event_createdAt_idx" ON "Event"("createdAt");

-- CreateIndex
CREATE INDEX "Event_tenantId_createdAt_idx" ON "Event"("tenantId", "createdAt");

-- CreateIndex
CREATE INDEX "WorkflowRun_tenantId_idx" ON "WorkflowRun"("tenantId");

-- CreateIndex
CREATE INDEX "WorkflowRun_workflowVersionId_idx" ON "WorkflowRun"("workflowVersionId");

-- CreateIndex
CREATE INDEX "WorkflowRun_createdAt_idx" ON "WorkflowRun"("createdAt");

-- CreateIndex
CREATE INDEX "WorkflowRun_tenantId_createdAt_idx" ON "WorkflowRun"("tenantId", "createdAt");

-- CreateIndex
CREATE INDEX "WorkflowRun_finishedAt_idx" ON "WorkflowRun"("finishedAt");

-- CreateIndex
CREATE INDEX "WorkflowRun_status_idx" ON "WorkflowRun"("status");

-- CreateIndex
CREATE INDEX "WorkflowRunTriggeredBy_tenantId_idx" ON "WorkflowRunTriggeredBy"("tenantId");

-- CreateIndex
CREATE INDEX "WorkflowRunTriggeredBy_eventId_idx" ON "WorkflowRunTriggeredBy"("eventId");

-- CreateIndex
CREATE INDEX "WorkflowRunTriggeredBy_parentId_idx" ON "WorkflowRunTriggeredBy"("parentId");
