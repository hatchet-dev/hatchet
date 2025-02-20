-- Add value to enum type: "InternalQueue"
ALTER TYPE "InternalQueue" ADD VALUE 'WORKFLOW_RUN_PAUSED';
-- Modify "Workflow" table
ALTER TABLE "Workflow" ADD COLUMN "isPaused" boolean NULL DEFAULT false;
