-- +goose Up
-- +goose NO TRANSACTION

-- For v1_runs_olap
CREATE INDEX IF NOT EXISTS idx_v1_runs_olap_external_id ON v1_runs_olap(parent_task_external_id);
CREATE INDEX IF NOT EXISTS idx_v1_runs_olap_workflow_tenant ON v1_runs_olap(tenant_id, workflow_id);

-- For v1_statuses_olap
CREATE INDEX IF NOT EXISTS idx_v1_statuses_olap_tenant_workflow_inserted ON v1_statuses_olap(tenant_id, workflow_id, inserted_at);
CREATE INDEX IF NOT EXISTS idx_v1_statuses_olap_external_id_status ON v1_statuses_olap(readable_status);


-- +goose Down
-- +goose NO TRANSACTION

-- For v1_runs_olap
DROP INDEX IF EXISTS idx_v1_runs_olap_external_id;
DROP INDEX IF EXISTS idx_v1_runs_olap_workflow_tenant;

-- For v1_statuses_olap
DROP INDEX IF EXISTS idx_v1_statuses_olap_tenant_workflow_inserted;
DROP INDEX IF EXISTS idx_v1_statuses_olap_external_id_status;
