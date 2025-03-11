-- +goose Up
-- +goose NO TRANSACTION

-- Create index "GetGroupKeyRun_createdAt_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_createdAt_idx" ON "GetGroupKeyRun" ("createdAt");
-- Create index "GetGroupKeyRun_status_deletedAt_timeoutAt_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_status_deletedAt_timeoutAt_idx" ON "GetGroupKeyRun" ("status", "deletedAt", "timeoutAt");
-- Create index "GetGroupKeyRun_tenantId_deletedAt_status_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_tenantId_deletedAt_status_idx" ON "GetGroupKeyRun" ("tenantId", "deletedAt", "status");
-- Create index "GetGroupKeyRun_tenantId_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_tenantId_idx" ON "GetGroupKeyRun" ("tenantId");
-- Create index "GetGroupKeyRun_workerId_idx" to table: "GetGroupKeyRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "GetGroupKeyRun_workerId_idx" ON "GetGroupKeyRun" ("workerId");
