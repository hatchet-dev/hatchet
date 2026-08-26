-- +goose Up
-- +goose NO TRANSACTION

-- For v1_statuses_olap
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_statuses_olap_tenant_inserted_at_idx
ON v1_statuses_olap (tenant_id, inserted_at)
INCLUDE (readable_status);

-- +goose Down
-- +goose NO TRANSACTION
-- For v1_statuses_olap
DROP INDEX CONCURRENTLY IF EXISTS v1_statuses_olap_tenant_inserted_at_idx;
