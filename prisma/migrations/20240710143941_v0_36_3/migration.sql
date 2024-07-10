-- CreateIndex
CREATE INDEX "StepRun_status_timeoutAt_tickerId_idx" ON "StepRun"("status", "timeoutAt", "tickerId");
