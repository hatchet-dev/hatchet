-- +goose Up
-- +goose StatementBegin
-- ix_v1_runs_olap_tenant_ins_at_status_wf has the same leading columns and order, so it serves every query this index did.
-- Dropping a partitioned index locks v1_runs_olap and then each partition; fail fast instead of queueing writers behind a long query.
-- lock_timeout bounds each lock wait, statement_timeout bounds the total time v1_runs_olap stays locked.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '15s';
DROP INDEX IF EXISTS ix_v1_runs_olap_tenant_ins_at_status;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS ix_v1_runs_olap_tenant_ins_at_status ON v1_runs_olap (tenant_id, inserted_at DESC, readable_status);
-- +goose StatementEnd
