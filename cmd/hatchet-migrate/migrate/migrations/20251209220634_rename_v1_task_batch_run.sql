-- +goose Up
-- +goose StatementBegin
-- Rename v1_task_batch_run -> v1_batch_runtime and drop unused completed_at column.
ALTER TABLE IF EXISTS v1_task_batch_run RENAME TO v1_batch_runtime;

-- Rename primary key constraint to match new table name.
ALTER TABLE IF EXISTS v1_batch_runtime
    RENAME CONSTRAINT v1_task_batch_run_pkey TO v1_batch_runtime_pkey;

-- Drop partial index that referenced completed_at and replace with a simple key index.
DROP INDEX IF EXISTS v1_task_batch_run_active_key_idx;
CREATE INDEX IF NOT EXISTS v1_batch_runtime_key_idx
    ON v1_batch_runtime (tenant_id, step_id, batch_key);

-- Remove unused completed_at column.
ALTER TABLE IF EXISTS v1_batch_runtime DROP COLUMN IF EXISTS completed_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Restore completed_at column and original names.
ALTER TABLE IF EXISTS v1_batch_runtime ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

DROP INDEX IF EXISTS v1_batch_runtime_key_idx;

ALTER TABLE IF EXISTS v1_batch_runtime
    RENAME CONSTRAINT v1_batch_runtime_pkey TO v1_task_batch_run_pkey;

ALTER TABLE IF EXISTS v1_batch_runtime RENAME TO v1_task_batch_run;

-- Recreate the original partial index.
CREATE INDEX IF NOT EXISTS v1_task_batch_run_active_key_idx
    ON v1_task_batch_run (tenant_id, step_id, batch_key)
    WHERE completed_at IS NULL;
-- +goose StatementEnd
