-- atlas:txmode none

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
