-- Modify "Event" table
ALTER TABLE "Event" DROP CONSTRAINT "Event_tenantId_fkey";
-- Modify "GetGroupKeyRun" table
ALTER TABLE "GetGroupKeyRun" DROP CONSTRAINT "GetGroupKeyRun_tenantId_fkey";
-- Modify "Job" table
ALTER TABLE "Job" DROP CONSTRAINT "Job_tenantId_fkey";
-- Modify "JobRun" table
ALTER TABLE "JobRun" DROP CONSTRAINT "JobRun_tenantId_fkey", DROP CONSTRAINT "JobRun_tickerId_fkey";
-- Modify "LogLine" table
ALTER TABLE "LogLine" DROP CONSTRAINT "LogLine_tenantId_fkey";
-- Modify "RateLimit" table
ALTER TABLE "RateLimit" DROP CONSTRAINT "RateLimit_tenantId_fkey";
-- Modify "Step" table
ALTER TABLE "Step" DROP CONSTRAINT "Step_tenantId_fkey";
-- Modify "StepRateLimit" table
ALTER TABLE "StepRateLimit" DROP CONSTRAINT "StepRateLimit_tenantId_fkey";
-- Modify "StepRun" table
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_stepId_fkey", DROP CONSTRAINT "StepRun_tenantId_fkey", DROP CONSTRAINT "StepRun_tickerId_fkey";
-- Create index "StepRunExpressionEval_stepRunId_idx" to table: "StepRunExpressionEval"
CREATE INDEX "StepRunExpressionEval_stepRunId_idx" ON "StepRunExpressionEval" ("stepRunId");
-- Modify "StreamEvent" table
ALTER TABLE "StreamEvent" DROP CONSTRAINT "StreamEvent_tenantId_fkey";
-- Modify "Worker" table
ALTER TABLE "Worker" DROP CONSTRAINT "Worker_tenantId_fkey";
-- Modify "Workflow" table
ALTER TABLE "Workflow" DROP CONSTRAINT "Workflow_tenantId_fkey";
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" DROP CONSTRAINT "WorkflowRun_tenantId_fkey", DROP CONSTRAINT "WorkflowRun_workflowVersionId_fkey";
-- Modify "WorkflowRunDedupe" table
ALTER TABLE "WorkflowRunDedupe" DROP CONSTRAINT "WorkflowRunDedupe_tenantId_fkey";
-- Modify "WorkflowRunTriggeredBy" table
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_eventId_fkey", DROP CONSTRAINT "WorkflowRunTriggeredBy_parentId_fkey", DROP CONSTRAINT "WorkflowRunTriggeredBy_tenantId_fkey";
