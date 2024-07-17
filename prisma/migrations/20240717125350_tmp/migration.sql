-- CreateEnum
CREATE TYPE "StickyStrategy" AS ENUM ('SOFT', 'HARD');

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN     "sticky" "StickyStrategy";

-- CreateTable
CREATE TABLE "WorkflowRunStickyState" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "workflowRunId" UUID NOT NULL,
    "desiredWorkerId" UUID,
    "strategy" "StickyStrategy" NOT NULL,

    CONSTRAINT "WorkflowRunStickyState_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunStickyState_workflowRunId_key" ON "WorkflowRunStickyState"("workflowRunId");

-- AddForeignKey
ALTER TABLE "WorkflowRunStickyState" ADD CONSTRAINT "WorkflowRunStickyState_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
