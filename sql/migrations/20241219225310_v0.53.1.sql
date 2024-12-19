-- Modify "LogLine" table
ALTER TABLE  "LogLine" DROP CONSTRAINT IF EXISTS "LogLine_stepRunId_fkey";
-- Drop index "StepRun_id_key" from table: "StepRun"
DROP INDEX IF EXISTS "StepRun_id_key";
-- Modify "StepRun" table
ALTER TABLE "StepRun" DROP CONSTRAINT IF EXISTS "StepRun_jobRunId_fkey", DROP CONSTRAINT IF EXISTS "StepRun_workerId_fkey";
-- Create index "StepRun_id_key" to table: "StepRun"
CREATE UNIQUE INDEX IF NOT EXISTS "StepRun_id_key" ON "StepRun" ("id", "status");
-- Create index "StepRun_status_tenantId_idx" to table: "StepRun"
CREATE INDEX  IF NOT EXISTS "StepRun_status_tenantId_idx" ON "StepRun" ("status", "tenantId");
-- Modify "StepRunResultArchive" table
ALTER TABLE "StepRunResultArchive" DROP CONSTRAINT IF EXISTS "StepRunResultArchive_stepRunId_fkey";
-- Modify "StreamEvent" table
ALTER TABLE "StreamEvent" DROP CONSTRAINT IF EXISTS "StreamEvent_stepRunId_fkey";
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" DROP CONSTRAINT IF EXISTS "WorkflowRun_parentStepRunId_fkey";
-- Modify "WorkflowTriggerScheduledRef" table
ALTER TABLE "WorkflowTriggerScheduledRef" DROP CONSTRAINT IF EXISTS "WorkflowTriggerScheduledRef_parentStepRunId_fkey";
-- Modify "_StepRunOrder" table
ALTER TABLE "_StepRunOrder" DROP CONSTRAINT IF EXISTS "_StepRunOrder_A_fkey", DROP CONSTRAINT IF EXISTS "_StepRunOrder_B_fkey";
