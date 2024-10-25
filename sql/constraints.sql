-- NOTE: this is a SQL script file that contains the constraints for the database
-- it is needed because prisma does not support constraints yet

-- Modify "QueueItem" table
ALTER TABLE "QueueItem" ADD CONSTRAINT "QueueItem_priority_check" CHECK ("priority" >= 1 AND "priority" <= 4);

-- Modify "InternalQueueItem" table
ALTER TABLE "InternalQueueItem" ADD CONSTRAINT "InternalQueueItem_priority_check" CHECK ("priority" >= 1 AND "priority" <= 4);

CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_jobRunId_status_tenantId_idx"
ON "StepRun" ("jobRunId", "status", "tenantId")
WHERE "status" = 'PENDING';

CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_parentStepRunId" ON "WorkflowRun"("parentStepRunId" ASC);

-- Additional indexes on workflow run
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflowrun_concurrency ON "WorkflowRun" ("concurrencyGroupId", "createdAt");
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflowrun_main ON "WorkflowRun" ("tenantId", "deletedAt", "status", "workflowVersionId", "createdAt");

-- Additional indexes on workflow
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_version_workflow_id_order
ON "WorkflowVersion" ("workflowId", "order" DESC)
WHERE "deletedAt" IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_tenant_id
ON "Workflow" ("tenantId");

-- Additional indexes on WorkflowTriggers
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_triggers_workflow_version_id
ON "WorkflowTriggers" ("workflowVersionId");

-- Additional indexes on WorkflowTriggerEventRef
CREATE INDEX idx_workflow_trigger_event_ref_event_key_parent_id
ON "WorkflowTriggerEventRef" ("eventKey", "parentId");

-- Additional indexes on WorkflowRun
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_parentId_parentStepRunId_childIndex_key"
ON "WorkflowRun"("parentId", "parentStepRunId", "childIndex")
WHERE "deletedAt" IS NULL;
