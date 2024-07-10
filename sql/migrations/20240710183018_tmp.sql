-- atlas:txmode none

-- Create index "GetGroupKeyRun_status_timeoutAt_tickerId_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_status_timeoutAt_tickerId_idx" ON "GetGroupKeyRun" ("status", "timeoutAt", "tickerId");
-- Drop index "StepRun_timeoutAt_idx" from table: "StepRun"
DROP INDEX CONCURRENTLY NOT EXISTS "StepRun_timeoutAt_idx";
