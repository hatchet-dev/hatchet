-- AlterEnum
ALTER TYPE "StepRunEventReason" ADD VALUE 'SENT_TO_WORKER';

-- AlterEnum
ALTER TYPE "StepRunStatus" ADD VALUE 'CANCELLING';

-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "queue" TEXT NOT NULL DEFAULT 'default',
ADD COLUMN     "queueOrder" BIGINT NOT NULL DEFAULT 0;

-- CreateTable
CREATE TABLE "StepRunQueue" (
    "id" BIGSERIAL NOT NULL,
    "queue" TEXT NOT NULL,
    "blockAddr" BIGINT NOT NULL,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "StepRunQueue_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "StepRunPtr" (
    "maxAssignedBlockAddr" BIGINT NOT NULL DEFAULT 0,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "StepRunPtr_pkey" PRIMARY KEY ("tenantId")
);

-- CreateIndex
CREATE UNIQUE INDEX "StepRunQueue_tenantId_queue_key" ON "StepRunQueue"("tenantId", "queue");

-- CreateIndex
CREATE UNIQUE INDEX "StepRunPtr_tenantId_key" ON "StepRunPtr"("tenantId");

-- CreateIndex
CREATE INDEX "StepRun_status_tenantId_deletedAt_queueOrder_idx" ON "StepRun"("status", "tenantId", "deletedAt", "queueOrder");
