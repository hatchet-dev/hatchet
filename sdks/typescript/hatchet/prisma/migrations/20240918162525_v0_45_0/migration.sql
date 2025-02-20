-- AlterEnum
ALTER TYPE "InternalQueue" ADD VALUE 'WORKFLOW_RUN_UPDATE';

-- AlterEnum
-- This migration adds more than one value to an enum.
-- With PostgreSQL versions 11 and earlier, this is not possible
-- in a single migration. This can be worked around by creating
-- multiple migrations, each migration adding only one value to
-- the enum.


ALTER TYPE "StepRunEventReason" ADD VALUE 'WORKFLOW_RUN_GROUP_KEY_SUCCEEDED';
ALTER TYPE "StepRunEventReason" ADD VALUE 'WORKFLOW_RUN_GROUP_KEY_FAILED';

-- DropForeignKey
ALTER TABLE "StepRunEvent" DROP CONSTRAINT "StepRunEvent_stepRunId_fkey";

-- AlterTable
ALTER TABLE "StepRunEvent" ADD COLUMN     "workflowRunId" UUID,
ALTER COLUMN "stepRunId" DROP NOT NULL;

-- AlterTable
ALTER TABLE "WorkflowConcurrency" ADD COLUMN     "concurrencyGroupExpression" TEXT;

-- CreateIndex
CREATE INDEX "StepRunEvent_workflowRunId_idx" ON "StepRunEvent"("workflowRunId");
