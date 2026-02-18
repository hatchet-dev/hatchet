-- +goose Up
-- +goose StatementBegin

ALTER TABLE v1_task_runtime
    ADD COLUMN IF NOT EXISTS evicted_at TIMESTAMPTZ DEFAULT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE v1_task_runtime
    DROP COLUMN IF EXISTS evicted_at;

-- +goose StatementEnd
