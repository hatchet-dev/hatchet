-- atlas:txmode none

CREATE INDEX CONCURRENTLY "Worker_lastHeartbeatAt_idx" ON "Worker" ("lastHeartbeatAt", "tenantId");

