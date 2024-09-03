-- CreateEnum
CREATE TYPE "WorkflowRunEventType" AS ENUM ('PENDING', 'QUEUED', 'RUNNING', 'SUCCEEDED', 'RETRIED', 'FAILED', 'QUEUE_DEPTH');

-- CreateTable
CREATE TABLE "WorkflowRunEvent" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "workflowRunId" UUID NOT NULL,
    "eventType" "WorkflowRunEventType" NOT NULL,

    CONSTRAINT "WorkflowRunEvent_pkey" PRIMARY KEY ("id","createdAt")
);

-- CreateIndex
CREATE INDEX "WorkflowRunEvent_createdAt_idx" ON "WorkflowRunEvent"("createdAt" DESC);

-- AddForeignKey
ALTER TABLE "WorkflowRunEvent" ADD CONSTRAINT "WorkflowRunEvent_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunEvent" ADD CONSTRAINT "WorkflowRunEvent_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
