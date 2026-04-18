-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry
    ADD COLUMN user_message TEXT,
    ADD COLUMN wait_data JSONB,
    ADD COLUMN satisfied_at TIMESTAMPTZ,
    DROP COLUMN IF EXISTS readable_summary
;
ALTER TABLE v1_tasks_olap ADD COLUMN is_durable BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry
    DROP COLUMN user_message,
    DROP COLUMN wait_data,
    DROP COLUMN satisfied_at,
    ADD COLUMN readable_summary TEXT
;
ALTER TABLE v1_tasks_olap DROP COLUMN is_durable;
-- +goose StatementEnd
