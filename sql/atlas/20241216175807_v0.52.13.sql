-- atlas:txmode none

CREATE INDEX CONCURRENTLY IF NOT EXISTS "LogLine_tenantId_stepRunId_idx" ON "LogLine" ("tenantId", "stepRunId" ASC);
