-- +goose NO TRANSACTION

-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_task_runtime_tenant_evicted_at_idx
    ON v1_task_runtime (tenant_id, evicted_at, task_id, task_inserted_at, retry_count)
    WHERE evicted_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS v1_task_runtime_tenant_evicted_at_idx;
-- +goose StatementEnd
