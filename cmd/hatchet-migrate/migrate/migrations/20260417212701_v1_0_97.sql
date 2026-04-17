-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry
    ADD COLUMN user_message TEXT,
    ADD COLUMN readable_summary TEXT,
    ADD COLUMN satisfied_at TIMESTAMPTZ
;
ALTER TABLE v1_tasks_olap ADD COLUMN is_durable BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry
    DROP COLUMN user_message,
    DROP COLUMN readable_summary,
    DROP COLUMN satisfied_at
;
ALTER TABLE v1_tasks_olap DROP COLUMN is_durable;
-- +goose StatementEnd
