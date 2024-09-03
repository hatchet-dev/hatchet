/*
  Warnings:

  - A unique constraint covering the columns `[stepRunId,workerId,isAssigned,retryCount]` on the table `WorkerSemaphoreQueueItem` will be added. If there are existing duplicate values, this will fail.

*/
-- DropIndex
DROP INDEX "WorkerSemaphoreQueueItem_stepRunId_workerId_retryCount_key";

-- CreateIndex
CREATE UNIQUE INDEX "WorkerSemaphoreQueueItem_stepRunId_workerId_isAssigned_retr_key" ON "WorkerSemaphoreQueueItem"("stepRunId", "workerId", "isAssigned", "retryCount");
