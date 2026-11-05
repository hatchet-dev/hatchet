-- +goose Up
-- +goose NO TRANSACTION

-- For v1_statuses_olap
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v1_statuses_olap_query_optim ON v1_statuses_olap(tenant_id, workflow_id);

-- +goose Down
-- +goose NO TRANSACTION
-- For v1_statuses_olap
DROP INDEX IF EXISTS idx_v1_statuses_olap_query_optim;
