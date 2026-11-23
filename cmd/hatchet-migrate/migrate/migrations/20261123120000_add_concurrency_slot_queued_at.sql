-- +goose Up
-- +goose StatementBegin
-- Added without a default first so existing rows stay NULL (no rewrite of a hot table) and the
-- dashboard falls back to task_inserted_at for them; only slots created afterwards get a time.
ALTER TABLE v1_concurrency_slot ADD COLUMN queued_at TIMESTAMPTZ;
ALTER TABLE v1_concurrency_slot ALTER COLUMN queued_at SET DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_concurrency_slot DROP COLUMN queued_at;
-- +goose StatementEnd
