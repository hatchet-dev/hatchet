-- +goose Up
-- +goose NO TRANSACTION

-- For v1_runs_olap
CREATE INDEX IF NOT EXISTS idx_v1_runs_olap_external_id ON v1_runs_olap(parent_task_external_id) WHERE parent_task_external_id IS NOT NULL;

-- For v1_statuses_olap
CREATE INDEX IF NOT EXISTS idx_v1_statuses_olap_query_optim ON v1_statuses_olap(tenant_id, inserted_at, workflow_id);

-- +goose Down
-- +goose NO TRANSACTION

-- For v1_runs_olap
DROP INDEX IF EXISTS idx_v1_runs_olap_external_id;

-- For v1_statuses_olap
DROP INDEX IF EXISTS idx_v1_statuses_olap_query_optim;
