-- +goose Up
-- CreateIndex
CREATE INDEX "JobRun_workflowRunId_tenantId_idx" ON "JobRun" ("workflowRunId", "tenantId");

-- CreateIndex
CREATE INDEX "StepRun_tenantId_status_requeueAfter_createdAt_idx" ON "StepRun" ("tenantId", "status", "requeueAfter", "createdAt");

-- CreateIndex
CREATE INDEX "StepRun_stepId_idx" ON "StepRun" ("stepId");

-- CreateIndex
CREATE INDEX "StepRun_jobRunId_status_idx" ON "StepRun" ("jobRunId", "status");

-- CreateIndex
CREATE INDEX "StepRun_id_tenantId_idx" ON "StepRun" ("id", "tenantId");

-- CreateIndex
CREATE INDEX "StepRun_jobRunId_tenantId_order_idx" ON "StepRun" ("jobRunId", "tenantId", "order");

-- CreateEnum
CREATE TYPE "StepRunEventReason" AS ENUM (
    'REQUEUED_NO_WORKER',
    'REQUEUED_RATE_LIMIT',
    'SCHEDULING_TIMED_OUT',
    'ASSIGNED',
    'STARTED',
    'FINISHED',
    'FAILED',
    'RETRYING',
    'CANCELLED'
);

-- CreateEnum
CREATE TYPE "StepRunEventSeverity" AS ENUM ('INFO', 'WARNING', 'CRITICAL');

-- CreateTable
CREATE TABLE
    "StepRunEvent" (
        "id" BIGSERIAL NOT NULL,
        "timeFirstSeen" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "timeLastSeen" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "stepRunId" UUID NOT NULL,
        "reason" "StepRunEventReason" NOT NULL,
        "severity" "StepRunEventSeverity" NOT NULL,
        "message" TEXT NOT NULL,
        "count" INTEGER NOT NULL,
        "data" JSONB
    );

-- CreateIndex
CREATE UNIQUE INDEX "StepRunEvent_id_key" ON "StepRunEvent" ("id");

-- CreateIndex
CREATE INDEX "StepRunEvent_stepRunId_idx" ON "StepRunEvent" ("stepRunId");

-- AddForeignKey
ALTER TABLE "StepRunEvent" ADD CONSTRAINT "StepRunEvent_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun" ("id") ON DELETE CASCADE ON UPDATE CASCADE;
