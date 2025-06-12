-- +goose Up
-- +goose NO TRANSACTION

-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v1_step_concurrency_tenant_active
    ON v1_step_concurrency(tenant_id, is_active)
    WHERE is_active = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_v1_step_concurrency_tenant_active;
-- +goose StatementEnd
