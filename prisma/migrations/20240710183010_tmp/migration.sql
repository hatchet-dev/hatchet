-- DropIndex
DROP INDEX "StepRun_timeoutAt_idx";

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_status_timeoutAt_tickerId_idx" ON "GetGroupKeyRun"("status", "timeoutAt", "tickerId");
