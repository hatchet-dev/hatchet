-- AlterEnum
ALTER TYPE "StepRunEventReason" ADD VALUE 'SENT_TO_WORKER';

-- AlterEnum
ALTER TYPE "StepRunStatus" ADD VALUE 'CANCELLING';

-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "queue" TEXT NOT NULL DEFAULT 'default';

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
    "priority" INTEGER NOT NULL DEFAULT 1,
    "isQueued" BOOLEAN NOT NULL,
    "tenantId" UUID NOT NULL,
    "queue" TEXT NOT NULL,
    "sticky" "StickyStrategy",
    "desiredWorkerId" UUID,

    CONSTRAINT "QueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "Queue_tenantId_name_key" ON "Queue"("tenantId", "name");

-- CreateIndex
CREATE INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx" ON "QueueItem"("isQueued", "priority", "tenantId", "queue", "id");
