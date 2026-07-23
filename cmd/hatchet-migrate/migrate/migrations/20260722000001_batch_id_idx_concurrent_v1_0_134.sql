-- +goose Up
-- +goose NO TRANSACTION

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_task_runtime_batch_id_idx
    ON v1_task_runtime USING BTREE (batch_id)
    WHERE batch_id IS NOT NULL;

-- +goose Down
-- +goose NO TRANSACTION

DROP INDEX CONCURRENTLY IF EXISTS v1_task_runtime_batch_id_idx;
