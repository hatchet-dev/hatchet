-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_concurrency_slot_workflow_unfilled_idx ON v1_concurrency_slot (tenant_id, workflow_id) WHERE is_filled = FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS v1_concurrency_slot_workflow_unfilled_idx;
-- +goose StatementEnd
