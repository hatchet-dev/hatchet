-- CreateEnum
CREATE TYPE "StickyStrategy" AS ENUM ('SOFT', 'HARD');

-- CreateEnum
CREATE TYPE "WorkerLabelComparator" AS ENUM ('EQUAL', 'NOT_EQUAL', 'GREATER_THAN', 'GREATER_THAN_OR_EQUAL', 'LESS_THAN', 'LESS_THAN_OR_EQUAL');

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN     "sticky" "StickyStrategy";

-- CreateTable
CREATE TABLE "StepDesiredWorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stepId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,
    "required" BOOLEAN NOT NULL,
    "comparator" "WorkerLabelComparator" NOT NULL,
    "weight" INTEGER NOT NULL,

    CONSTRAINT "StepDesiredWorkerLabel_pkey" PRIMARY KEY ("id")
);

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

-- CreateTable
CREATE TABLE "WorkflowRunDedupe" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "workflowId" UUID NOT NULL,
    "workflowRunId" UUID NOT NULL,
    "value" TEXT NOT NULL
);

-- CreateTable
CREATE TABLE "WorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "workerId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,

    CONSTRAINT "WorkerLabel_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "StepDesiredWorkerLabel_stepId_idx" ON "StepDesiredWorkerLabel"("stepId");

-- CreateIndex
CREATE UNIQUE INDEX "StepDesiredWorkerLabel_stepId_key_key" ON "StepDesiredWorkerLabel"("stepId", "key");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunStickyState_workflowRunId_key" ON "WorkflowRunStickyState"("workflowRunId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunDedupe_id_key" ON "WorkflowRunDedupe"("id");

-- CreateIndex
CREATE INDEX "WorkflowRunDedupe_tenantId_value_idx" ON "WorkflowRunDedupe"("tenantId", "value");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunDedupe_tenantId_workflowId_value_key" ON "WorkflowRunDedupe"("tenantId", "workflowId", "value");

-- CreateIndex
CREATE INDEX "WorkerLabel_workerId_idx" ON "WorkerLabel"("workerId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkerLabel_workerId_key_key" ON "WorkerLabel"("workerId", "key");

-- AddForeignKey
ALTER TABLE "StepDesiredWorkerLabel" ADD CONSTRAINT "StepDesiredWorkerLabel_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunStickyState" ADD CONSTRAINT "WorkflowRunStickyState_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunDedupe" ADD CONSTRAINT "WorkflowRunDedupe_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerLabel" ADD CONSTRAINT "WorkerLabel_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
