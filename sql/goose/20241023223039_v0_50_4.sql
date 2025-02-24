-- +goose Up
-- +goose NO TRANSACTION

-- Create index "WorkflowRun_parentId_parentStepRunId_childIndex_key" to table: "WorkflowRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_parentId_parentStepRunId_childIndex_key" 
ON "WorkflowRun" ("parentId", "parentStepRunId", "childIndex") 
WHERE ("deletedAt" IS NULL);
