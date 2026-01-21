-- +goose Up
-- +goose NO TRANSACTION

-- +goose StatementBegin
-- This supports ListConcurrencyStrategiesByStepId (tenant_id + step_id = ANY(...)).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v1_step_concurrency_tenant_step_id
    ON v1_step_concurrency(tenant_id, step_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_v1_step_concurrency_tenant_step_id;
-- +goose StatementEnd
