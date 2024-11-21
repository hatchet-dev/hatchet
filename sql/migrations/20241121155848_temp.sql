-- Create enum type "WorkflowTriggerScheduledRefMethods"
CREATE TYPE "WorkflowTriggerScheduledRefMethods" AS ENUM ('DEFAULT', 'API');
-- Modify "WorkflowTriggerScheduledRef" table
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN "method" "WorkflowTriggerScheduledRefMethods" NOT NULL DEFAULT 'DEFAULT';
