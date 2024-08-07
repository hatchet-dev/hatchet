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

-- CreateTable
CREATE TABLE "Queue" (
    "id" BIGSERIAL NOT NULL,
    "tenantId" UUID NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "Queue_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "QueueItem" (
    "id" BIGSERIAL NOT NULL,
    "stepRunId" UUID,
    "stepId" UUID,
    "actionId" TEXT,
    "scheduleTimeoutAt" TIMESTAMP(3),
    "stepTimeout" TEXT,
    "isQueued" BOOLEAN NOT NULL,
    "tenantId" UUID NOT NULL,
    "queue" TEXT NOT NULL,

    CONSTRAINT "QueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "StepRunQueue_tenantId_queue_key" ON "StepRunQueue"("tenantId", "queue");

-- CreateIndex
CREATE UNIQUE INDEX "StepRunPtr_tenantId_key" ON "StepRunPtr"("tenantId");

-- CreateIndex
CREATE UNIQUE INDEX "Queue_tenantId_name_key" ON "Queue"("tenantId", "name");

-- CreateIndex
CREATE INDEX "QueueItem_isQueued_queue_idx" ON "QueueItem"("isQueued", "queue");

-- CreateIndex
CREATE INDEX "StepRun_status_tenantId_deletedAt_queueOrder_idx" ON "StepRun"("status", "tenantId", "deletedAt", "queueOrder");
