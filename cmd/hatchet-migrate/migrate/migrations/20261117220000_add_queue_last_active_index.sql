-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY v1_queue_tenant_id_last_active_idx ON v1_queue (tenant_id, last_active);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS v1_queue_tenant_id_last_active_idx;
-- +goose StatementEnd
