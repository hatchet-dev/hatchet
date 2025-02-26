-- +goose Up
-- CreateEnum
CREATE TYPE "ConcurrencyLimitStrategy" AS ENUM ('CANCEL_IN_PROGRESS', 'DROP_NEWEST', 'QUEUE_NEWEST');

-- AlterEnum
ALTER TYPE "WorkflowRunStatus" ADD VALUE 'QUEUED';

-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "concurrencyGroupId" TEXT;

-- AlterTable
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN     "input" JSONB;

-- CreateTable
CREATE TABLE "WorkflowConcurrency" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "workflowVersionId" UUID NOT NULL,
    "getConcurrencyGroupId" UUID,
    "maxRuns" INTEGER NOT NULL DEFAULT 1,
    "limitStrategy" "ConcurrencyLimitStrategy" NOT NULL DEFAULT 'CANCEL_IN_PROGRESS',

    CONSTRAINT "WorkflowConcurrency_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "GetGroupKeyRun" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "workflowRunId" TEXT NOT NULL,
    "workerId" UUID,
    "tickerId" UUID,
    "status" "StepRunStatus" NOT NULL DEFAULT 'PENDING',
    "input" JSONB,
    "output" TEXT,
    "requeueAfter" TIMESTAMP(3),
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,

    CONSTRAINT "GetGroupKeyRun_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowConcurrency_id_key" ON "WorkflowConcurrency"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowConcurrency_workflowVersionId_key" ON "WorkflowConcurrency"("workflowVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "GetGroupKeyRun_id_key" ON "GetGroupKeyRun"("id");

-- CreateIndex
CREATE UNIQUE INDEX "GetGroupKeyRun_tenantId_workflowRunId_key" ON "GetGroupKeyRun"("tenantId", "workflowRunId");

-- AddForeignKey
ALTER TABLE "WorkflowConcurrency" ADD CONSTRAINT "WorkflowConcurrency_workflowVersionId_fkey" FOREIGN KEY ("workflowVersionId") REFERENCES "WorkflowVersion"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowConcurrency" ADD CONSTRAINT "WorkflowConcurrency_getConcurrencyGroupId_fkey" FOREIGN KEY ("getConcurrencyGroupId") REFERENCES "Action"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_tenantId_workflowRunId_fkey" FOREIGN KEY ("tenantId", "workflowRunId") REFERENCES "WorkflowRun"("tenantId", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker"("id") ON DELETE SET NULL ON UPDATE CASCADE;
