-- +goose Up

ALTER TABLE v1_task ADD COLUMN durable_invocation_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE v1_task_runtime ADD COLUMN durable_invocation_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE v1_task DROP COLUMN IF EXISTS durable_invocation_count;
ALTER TABLE v1_task_runtime DROP COLUMN IF EXISTS durable_invocation_count;
