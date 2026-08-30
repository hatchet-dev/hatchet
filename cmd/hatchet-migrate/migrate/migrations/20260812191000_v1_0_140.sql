-- +goose Up
-- +goose NO TRANSACTION

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_concurrency_slot_timeout_idx
    ON v1_concurrency_slot (tenant_id, strategy_id, task_id, task_inserted_at)
    WHERE is_filled = FALSE;

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_workflow_concurrency_slot_filled_idx
    ON v1_workflow_concurrency_slot (tenant_id, strategy_id, workflow_version_id, workflow_run_id)
    WHERE is_filled = TRUE;

-- +goose Down
-- +goose NO TRANSACTION

DROP INDEX CONCURRENTLY IF EXISTS v1_concurrency_slot_timeout_idx;
DROP INDEX CONCURRENTLY IF EXISTS v1_workflow_concurrency_slot_filled_idx;
