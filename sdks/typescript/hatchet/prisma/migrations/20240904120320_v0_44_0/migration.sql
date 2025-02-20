-- CreateEnum
CREATE TYPE "InternalQueue" AS ENUM ('WORKER_SEMAPHORE_COUNT', 'STEP_RUN_UPDATE');

-- CreateTable
CREATE TABLE "InternalQueueItem" (
    "id" BIGSERIAL NOT NULL,
    "queue" "InternalQueue" NOT NULL,
    "isQueued" BOOLEAN NOT NULL,
    "data" JSONB,
    "tenantId" UUID NOT NULL,
    "priority" INTEGER NOT NULL DEFAULT 1,
    "uniqueKey" TEXT,

    CONSTRAINT "InternalQueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkerSemaphoreCount" (
    "workerId" UUID NOT NULL,
    "count" INTEGER NOT NULL,

    CONSTRAINT "WorkerSemaphoreCount_pkey" PRIMARY KEY ("workerId")
);

-- CreateTable
CREATE TABLE "WorkerAssignEvent" (
    "id" BIGSERIAL NOT NULL,
    "workerId" UUID NOT NULL,
    "assignedStepRuns" JSONB,

    CONSTRAINT "WorkerAssignEvent_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "InternalQueueItem_isQueued_tenantId_queue_priority_id_idx" ON "InternalQueueItem"("isQueued", "tenantId", "queue", "priority" DESC, "id");

-- CreateIndex
CREATE UNIQUE INDEX "InternalQueueItem_tenantId_queue_uniqueKey_key" ON "InternalQueueItem"("tenantId", "queue", "uniqueKey");

-- CreateIndex
CREATE UNIQUE INDEX "WorkerSemaphoreCount_workerId_key" ON "WorkerSemaphoreCount"("workerId");

-- CreateIndex
CREATE INDEX "WorkerSemaphoreCount_workerId_idx" ON "WorkerSemaphoreCount"("workerId");

-- CreateIndex
CREATE INDEX "WorkerAssignEvent_workerId_id_idx" ON "WorkerAssignEvent"("workerId", "id");

-- AddForeignKey
ALTER TABLE "WorkerSemaphoreCount" ADD CONSTRAINT "WorkerSemaphoreCount_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerAssignEvent" ADD CONSTRAINT "WorkerAssignEvent_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
