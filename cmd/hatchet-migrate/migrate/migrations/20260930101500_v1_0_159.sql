-- +goose Up
-- +goose StatementBegin
-- ix_v1_runs_olap_tenant_ins_at_status_wf has the same leading columns and order, so it serves every query this index did.
-- Dropping a partitioned index locks v1_runs_olap and all partitions; fail fast instead of queueing writers behind a long query.
SET LOCAL lock_timeout = '5s';
DROP INDEX IF EXISTS ix_v1_runs_olap_tenant_ins_at_status;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS ix_v1_runs_olap_tenant_ins_at_status ON v1_runs_olap (tenant_id, inserted_at DESC, readable_status);
-- +goose StatementEnd
