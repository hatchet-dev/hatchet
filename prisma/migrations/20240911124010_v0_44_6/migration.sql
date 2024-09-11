-- CreateTable
CREATE TABLE "SemaphoreQueueItem" (
    "id" BIGSERIAL NOT NULL,
    "stepRunId" UUID NOT NULL,
    "workerId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "SemaphoreQueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "SemaphoreQueueItem_tenantId_workerId_idx" ON "SemaphoreQueueItem"("tenantId", "workerId");

-- CreateIndex
CREATE UNIQUE INDEX "SemaphoreQueueItem_stepRunId_workerId_key" ON "SemaphoreQueueItem"("stepRunId", "workerId");
