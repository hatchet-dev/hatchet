-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Step"
    ADD COLUMN batch_size INTEGER,
    ADD COLUMN batch_flush_interval_ms INTEGER;

ALTER TABLE v1_task_runtime 
    ADD COLUMN batch_id UUID
    ADD COLUMN batch_size INTEGER
    ADD COLUMN batch_index INTEGER;

CREATE INDEX IF NOT EXISTS v1_task_runtime_batch_id_idx
    ON v1_task_runtime USING BTREE (batch_id)
    WHERE batch_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Step"
    DROP COLUMN IF EXISTS batch_size,
    DROP COLUMN IF EXISTS batch_flush_interval_ms;

DROP INDEX IF EXISTS v1_task_runtime_batch_id_idx;

ALTER TABLE v1_task_runtime 
    DROP COLUMN IF EXISTS batch_index,
    DROP COLUMN IF EXISTS batch_size,
    DROP COLUMN IF EXISTS batch_id;
-- +goose StatementEnd
