-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
ALTER TABLE v1_task_runtime
    ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS acked_at TIMESTAMPTZ
;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_task_runtime_tenant_unacked_assigned_at_idx ON v1_task_runtime (tenant_id, assigned_at) WHERE assigned_at IS NOT NULL AND acked_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS v1_task_runtime_tenant_unacked_assigned_at_idx;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_runtime
    DROP COLUMN IF EXISTS acked_at,
    DROP COLUMN IF EXISTS assigned_at
;
-- +goose StatementEnd
