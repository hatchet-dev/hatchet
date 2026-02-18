-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_task_runtime_tenant_worker_not_evicted_idx ON v1_task_runtime (tenant_id, worker_id) WHERE evicted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS v1_task_runtime_tenant_worker_not_evicted_idx;
