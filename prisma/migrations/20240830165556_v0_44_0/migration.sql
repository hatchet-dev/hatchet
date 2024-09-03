-- CreateTable
CREATE TABLE "WorkerSemaphoreQueueItem" (
    "id" BIGSERIAL NOT NULL,
    "stepRunId" UUID NOT NULL,
    "workerId" UUID NOT NULL,
    "retryCount" INTEGER NOT NULL,
    "isProcessed" BOOLEAN NOT NULL,

    CONSTRAINT "WorkerSemaphoreQueueItem_pkey" PRIMARY KEY ("id")
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
CREATE INDEX "WorkerSemaphoreQueueItem_isProcessed_workerId_id_idx" ON "WorkerSemaphoreQueueItem"("isProcessed", "workerId", "id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkerSemaphoreQueueItem_stepRunId_workerId_retryCount_key" ON "WorkerSemaphoreQueueItem"("stepRunId", "workerId", "retryCount");

-- CreateIndex
CREATE UNIQUE INDEX "WorkerSemaphoreCount_workerId_key" ON "WorkerSemaphoreCount"("workerId");

-- CreateIndex
CREATE INDEX "WorkerSemaphoreCount_workerId_idx" ON "WorkerSemaphoreCount"("workerId");

-- CreateIndex
CREATE INDEX "WorkerAssignEvent_workerId_idx" ON "WorkerAssignEvent"("workerId");

-- AddForeignKey
ALTER TABLE "WorkerSemaphoreCount" ADD CONSTRAINT "WorkerSemaphoreCount_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerAssignEvent" ADD CONSTRAINT "WorkerAssignEvent_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
