-- Modify "LogLine" table
ALTER TABLE "LogLine" DROP CONSTRAINT "LogLine_stepRunId_fkey";
-- Drop index "StepRun_id_key" from table: "StepRun"
DROP INDEX "StepRun_id_key";
-- Modify "StepRun" table
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_jobRunId_fkey", DROP CONSTRAINT "StepRun_workerId_fkey";
-- Create index "StepRun_id_key" to table: "StepRun"
CREATE UNIQUE INDEX "StepRun_id_key" ON "StepRun" ("id", "status");
-- Create index "StepRun_status_tenantId_idx" to table: "StepRun"
CREATE INDEX "StepRun_status_tenantId_idx" ON "StepRun" ("status", "tenantId");
-- Modify "StepRunResultArchive" table
ALTER TABLE "StepRunResultArchive" DROP CONSTRAINT "StepRunResultArchive_stepRunId_fkey";
-- Modify "StreamEvent" table
ALTER TABLE "StreamEvent" DROP CONSTRAINT "StreamEvent_stepRunId_fkey";
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" DROP CONSTRAINT "WorkflowRun_parentStepRunId_fkey";
-- Modify "WorkflowTriggerScheduledRef" table
ALTER TABLE "WorkflowTriggerScheduledRef" DROP CONSTRAINT "WorkflowTriggerScheduledRef_parentStepRunId_fkey";
-- Modify "_StepRunOrder" table
ALTER TABLE "_StepRunOrder" DROP CONSTRAINT "_StepRunOrder_A_fkey", DROP CONSTRAINT "_StepRunOrder_B_fkey";
