-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_workflow_concurrency_slot_cancel_newest_query_idx ON v1_workflow_concurrency_slot (tenant_id, strategy_id ASC, key ASC, priority DESC, is_filled DESC, sort_id ASC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_workflow_concurrency_slot_cancel_in_progress_query_idx ON v1_workflow_concurrency_slot (tenant_id, strategy_id ASC, key ASC, priority DESC, is_filled ASC, sort_id DESC);
DROP INDEX CONCURRENTLY IF EXISTS v1_workflow_concurrency_slot_query_idx;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS v1_workflow_concurrency_slot_cancel_newest_query_idx;
DROP INDEX CONCURRENTLY IF EXISTS v1_workflow_concurrency_slot_cancel_in_progress_query_idx;
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_workflow_concurrency_slot_query_idx ON v1_workflow_concurrency_slot (tenant_id, strategy_id ASC, key ASC, priority DESC, sort_id ASC);
