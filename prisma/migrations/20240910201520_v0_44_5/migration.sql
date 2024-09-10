-- CreateTable
CREATE TABLE "TimeoutQueueItem" (
    "id" BIGSERIAL NOT NULL,
    "stepRunId" UUID NOT NULL,
    "retryCount" INTEGER NOT NULL,
    "timeoutAt" TIMESTAMP(3) NOT NULL,
    "tenantId" UUID NOT NULL,
    "isQueued" BOOLEAN NOT NULL,

    CONSTRAINT "TimeoutQueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "TimeoutQueueItem_tenantId_isQueued_timeoutAt_idx" ON "TimeoutQueueItem"("tenantId", "isQueued", "timeoutAt");

-- CreateIndex
CREATE UNIQUE INDEX "TimeoutQueueItem_stepRunId_retryCount_key" ON "TimeoutQueueItem"("stepRunId", "retryCount");
