-- +goose Up
-- +goose NO TRANSACTION

CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowRun_parentStepRunId" ON "WorkflowRun"("parentStepRunId" ASC);

-- Add value to enum type: "StepRunEventReason"
ALTER TYPE "StepRunEventReason" ADD VALUE 'WORKFLOW_RUN_GROUP_KEY_SUCCEEDED';
-- Add value to enum type: "StepRunEventReason"
ALTER TYPE "StepRunEventReason" ADD VALUE 'WORKFLOW_RUN_GROUP_KEY_FAILED';
-- Add value to enum type: "InternalQueue"
ALTER TYPE "InternalQueue" ADD VALUE 'WORKFLOW_RUN_UPDATE';
-- Modify "StepRunEvent" table
ALTER TABLE "StepRunEvent" DROP CONSTRAINT "StepRunEvent_stepRunId_fkey", ALTER COLUMN "stepRunId" DROP NOT NULL, ADD COLUMN "workflowRunId" uuid NULL;
-- Create index "StepRunEvent_workflowRunId_idx" to table: "StepRunEvent"
CREATE INDEX "StepRunEvent_workflowRunId_idx" ON "StepRunEvent" ("workflowRunId");
-- Modify "WorkflowConcurrency" table
ALTER TABLE "WorkflowConcurrency" ADD COLUMN "concurrencyGroupExpression" text NULL;
