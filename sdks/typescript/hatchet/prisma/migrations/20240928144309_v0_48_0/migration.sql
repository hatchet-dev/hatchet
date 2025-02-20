-- AlterEnum
ALTER TYPE "InternalQueue" ADD VALUE 'WORKFLOW_RUN_PAUSED';

-- AlterTable
ALTER TABLE "Workflow" ADD COLUMN     "isPaused" BOOLEAN DEFAULT false;
