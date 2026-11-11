-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY "Worker_tenantId_operatorId_active_idx" ON "Worker" ("tenantId", "operatorId") WHERE "isActive" AND "operatorId" IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS "Worker_tenantId_operatorId_active_idx";
-- +goose StatementEnd
