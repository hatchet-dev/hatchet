-- +goose Up
-- +goose NO TRANSACTION

-- Create index "Event_createdAt_idx" to table: "Event"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Event_createdAt_idx" ON "Event" ("createdAt");
-- Create index "Event_tenantId_createdAt_idx" to table: "Event"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Event_tenantId_createdAt_idx" ON "Event" ("tenantId", "createdAt");
-- Create index "Event_tenantId_idx" to table: "Event"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Event_tenantId_idx" ON "Event" ("tenantId");
-- Create index "WorkflowRun_createdAt_idx" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_createdAt_idx" ON "WorkflowRun" ("createdAt");
-- Create index "WorkflowRun_finishedAt_idx" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_finishedAt_idx" ON "WorkflowRun" ("finishedAt");
-- Create index "WorkflowRun_status_idx" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_status_idx" ON "WorkflowRun" ("status");
-- Create index "WorkflowRun_tenantId_createdAt_idx" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_tenantId_createdAt_idx" ON "WorkflowRun" ("tenantId", "createdAt");
-- Create index "WorkflowRun_tenantId_idx" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_tenantId_idx" ON "WorkflowRun" ("tenantId");
-- Create index "WorkflowRun_workflowVersionId_idx" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_workflowVersionId_idx" ON "WorkflowRun" ("workflowVersionId");
-- Create index "WorkflowRunTriggeredBy_eventId_idx" to table: "WorkflowRunTriggeredBy"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRunTriggeredBy_eventId_idx" ON "WorkflowRunTriggeredBy" ("eventId");
-- Create index "WorkflowRunTriggeredBy_parentId_idx" to table: "WorkflowRunTriggeredBy"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRunTriggeredBy_parentId_idx" ON "WorkflowRunTriggeredBy" ("parentId");
-- Create index "WorkflowRunTriggeredBy_tenantId_idx" to table: "WorkflowRunTriggeredBy"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRunTriggeredBy_tenantId_idx" ON "WorkflowRunTriggeredBy" ("tenantId");
