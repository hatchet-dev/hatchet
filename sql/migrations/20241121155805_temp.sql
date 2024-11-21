-- Create enum type "WorkflowTriggerCronRefMethods"
CREATE TYPE "WorkflowTriggerCronRefMethods" AS ENUM ('DEFAULT', 'API');
-- Modify "WorkflowTriggerCronRef" table
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN "method" "WorkflowTriggerCronRefMethods" NOT NULL DEFAULT 'DEFAULT';
