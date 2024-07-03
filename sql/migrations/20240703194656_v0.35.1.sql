-- Create index "Event_createdAt_idx" to table: "Event"
CREATE INDEX "Event_createdAt_idx" ON "Event" ("createdAt");
-- Create index "Event_tenantId_createdAt_idx" to table: "Event"
CREATE INDEX "Event_tenantId_createdAt_idx" ON "Event" ("tenantId", "createdAt");
-- Create index "Event_tenantId_idx" to table: "Event"
CREATE INDEX "Event_tenantId_idx" ON "Event" ("tenantId");
-- Create index "WorkflowRun_createdAt_idx" to table: "WorkflowRun"
CREATE INDEX "WorkflowRun_createdAt_idx" ON "WorkflowRun" ("createdAt");
-- Create index "WorkflowRun_finishedAt_idx" to table: "WorkflowRun"
CREATE INDEX "WorkflowRun_finishedAt_idx" ON "WorkflowRun" ("finishedAt");
-- Create index "WorkflowRun_status_idx" to table: "WorkflowRun"
CREATE INDEX "WorkflowRun_status_idx" ON "WorkflowRun" ("status");
-- Create index "WorkflowRun_tenantId_createdAt_idx" to table: "WorkflowRun"
CREATE INDEX "WorkflowRun_tenantId_createdAt_idx" ON "WorkflowRun" ("tenantId", "createdAt");
-- Create index "WorkflowRun_tenantId_idx" to table: "WorkflowRun"
CREATE INDEX "WorkflowRun_tenantId_idx" ON "WorkflowRun" ("tenantId");
-- Create index "WorkflowRun_workflowVersionId_idx" to table: "WorkflowRun"
CREATE INDEX "WorkflowRun_workflowVersionId_idx" ON "WorkflowRun" ("workflowVersionId");
-- Create index "WorkflowRunTriggeredBy_eventId_idx" to table: "WorkflowRunTriggeredBy"
CREATE INDEX "WorkflowRunTriggeredBy_eventId_idx" ON "WorkflowRunTriggeredBy" ("eventId");
-- Create index "WorkflowRunTriggeredBy_parentId_idx" to table: "WorkflowRunTriggeredBy"
CREATE INDEX "WorkflowRunTriggeredBy_parentId_idx" ON "WorkflowRunTriggeredBy" ("parentId");
-- Create index "WorkflowRunTriggeredBy_tenantId_idx" to table: "WorkflowRunTriggeredBy"
CREATE INDEX "WorkflowRunTriggeredBy_tenantId_idx" ON "WorkflowRunTriggeredBy" ("tenantId");
