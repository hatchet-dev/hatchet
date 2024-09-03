-- Drop index "WorkerSemaphoreQueueItem_stepRunId_workerId_retryCount_key" from table: "WorkerSemaphoreQueueItem"
DROP INDEX "WorkerSemaphoreQueueItem_stepRunId_workerId_retryCount_key";
-- Create index "WorkerSemaphoreQueueItem_stepRunId_workerId_isAssigned_retr_key" to table: "WorkerSemaphoreQueueItem"
CREATE UNIQUE INDEX "WorkerSemaphoreQueueItem_stepRunId_workerId_isAssigned_retr_key" ON "WorkerSemaphoreQueueItem" ("stepRunId", "workerId", "isAssigned", "retryCount");
