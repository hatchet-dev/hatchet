-- +goose Up
-- +goose NO TRANSACTION

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_task_events_olap_tenant_id_inserted_at_idx
    ON v1_task_events_olap (tenant_id ASC, inserted_at ASC);

-- +goose Down
DROP INDEX IF EXISTS v1_task_events_olap_tenant_id_inserted_at_idx;
