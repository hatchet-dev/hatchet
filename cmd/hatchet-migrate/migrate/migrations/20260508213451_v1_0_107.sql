-- +goose no transaction

-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Worker_tenantId_actionHash_idx" ON "Worker" ("tenantId", "actionHash");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS "Worker_tenantId_actionHash_idx";
-- +goose StatementEnd
