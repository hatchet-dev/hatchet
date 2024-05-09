-- CreateEnum
CREATE TYPE "StepRunEventReason" AS ENUM ('REQUEUED_NO_WORKER', 'REQUEUED_RATE_LIMIT', 'SCHEDULING_TIMED_OUT', 'STARTED', 'FINISHED', 'FAILED', 'RETRYING', 'CANCELLED', 'TIMED_OUT');

-- CreateEnum
CREATE TYPE "StepRunEventSeverity" AS ENUM ('INFO', 'WARNING', 'CRITICAL');

-- CreateTable
CREATE TABLE "StepRunEvent" (
    "id" BIGSERIAL NOT NULL,
    "timeFirstSeen" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "timeLastSeen" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stepRunId" UUID NOT NULL,
    "reason" "StepRunEventReason" NOT NULL,
    "severity" "StepRunEventSeverity" NOT NULL,
    "message" TEXT NOT NULL,
    "count" INTEGER NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "StepRunEvent_id_key" ON "StepRunEvent"("id");

-- CreateIndex
CREATE INDEX "StepRunEvent_stepRunId_idx" ON "StepRunEvent"("stepRunId");

-- AddForeignKey
ALTER TABLE "StepRunEvent" ADD CONSTRAINT "StepRunEvent_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
