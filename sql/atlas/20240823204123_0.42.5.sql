-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "priority" integer NULL;
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" ADD COLUMN "priority" integer NULL;
-- Modify "WorkflowVersion" table
ALTER TABLE "WorkflowVersion" ADD COLUMN "defaultPriority" integer NULL;
