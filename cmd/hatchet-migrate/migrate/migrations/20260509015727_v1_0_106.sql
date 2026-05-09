-- +goose Up
-- +goose NO TRANSACTION

DROP INDEX CONCURRENTLY IF EXISTS v1_queue_item_task_idx;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS v1_queue_item_task_idx ON v1_queue_item (
    task_id ASC,
    task_inserted_at ASC,
    retry_count ASC
);

-- +goose Down
-- +goose NO TRANSACTION

DROP INDEX CONCURRENTLY IF EXISTS v1_queue_item_task_idx;

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_queue_item_task_idx ON v1_queue_item (
    task_id ASC,
    task_inserted_at ASC,
    retry_count ASC
);
