-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX IF NOT EXISTS v1_task_events_olap_tenant_time_idx ON v1_task_events_olap (tenant_id, task_inserted_at)
    WITH (timescaledb.transaction_per_chunk);

-- +goose Down
DROP INDEX IF EXISTS v1_task_events_olap_tenant_time_idx;
