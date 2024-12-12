-- atlas:txmode none

CREATE INDEX CONCURRENTLY "Worker_tenantId_lastHeartbeatAt_idx" ON "Worker" ("tenantId", "lastHeartbeatAt");

