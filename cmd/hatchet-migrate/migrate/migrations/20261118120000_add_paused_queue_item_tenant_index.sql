-- +goose NO TRANSACTION
-- +goose Up
-- An interrupted concurrent build leaves an invalid index of this name; dropping it first lets the migration run again.
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS v1_paused_workflow_queue_item_tenant_idx;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX CONCURRENTLY v1_paused_workflow_queue_item_tenant_idx ON v1_paused_workflow_queue_item (tenant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS v1_paused_workflow_queue_item_tenant_idx;
-- +goose StatementEnd
