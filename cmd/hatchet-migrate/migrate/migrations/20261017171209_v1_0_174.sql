-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY IF NOT EXISTS "TenantResourceLimitAlert_tenantId_idx" ON "TenantResourceLimitAlert" ("tenantId");

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX CONCURRENTLY IF EXISTS "TenantResourceLimitAlert_tenantId_idx";
