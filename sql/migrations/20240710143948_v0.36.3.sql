-- atlas:txmode none

-- Create index "StepRun_status_timeoutAt_tickerId_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_timeoutAt_tickerId_idx" ON "StepRun" ("status", "timeoutAt", "tickerId");
