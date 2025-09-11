-- +goose Up
/*
  Warnings:

  - A unique constraint covering the columns `[parentId,parentStepRunId,childKey]` on the table `WorkflowRun` will be added. If there are existing duplicate values, this will fail.
  - A unique constraint covering the columns `[parentId,parentStepRunId,childKey]` on the table `WorkflowTriggerScheduledRef` will be added. If there are existing duplicate values, this will fail.

*/
-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "childIndex" INTEGER,
ADD COLUMN     "childKey" TEXT,
ADD COLUMN     "parentId" UUID,
ADD COLUMN     "parentStepRunId" UUID;

-- AlterTable
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN     "childIndex" INTEGER,
ADD COLUMN     "childKey" TEXT,
ADD COLUMN     "parentStepRunId" UUID,
ADD COLUMN     "parentWorkflowRunId" UUID;

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRun_parentId_parentStepRunId_childKey_key" ON "WorkflowRun"("parentId", "parentStepRunId", "childKey");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerScheduledRef_parentId_parentStepRunId_childK_key" ON "WorkflowTriggerScheduledRef"("parentId", "parentStepRunId", "childKey");

-- AddForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_parentWorkflowRunId_fkey" FOREIGN KEY ("parentWorkflowRunId") REFERENCES "WorkflowRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRun" ADD CONSTRAINT "WorkflowRun_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRun" ADD CONSTRAINT "WorkflowRun_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;
