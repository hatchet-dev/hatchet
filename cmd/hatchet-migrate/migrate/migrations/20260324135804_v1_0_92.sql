-- +goose Up
-- +goose NO TRANSACTION
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_log_line_tenant_id_level_idx ON v1_log_line (tenant_id ASC, level ASC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS v1_log_line_tenant_id_level_idx;
-- +goose StatementEnd
