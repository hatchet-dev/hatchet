-- +goose Up
-- +goose NO TRANSACTION

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_batched_queue_item_step_priority_idx
    ON v1_batched_queue_item (tenant_id ASC, step_id ASC, priority DESC, id ASC);

-- +goose Down
-- +goose NO TRANSACTION

DROP INDEX CONCURRENTLY IF EXISTS v1_batched_queue_item_step_priority_idx;
