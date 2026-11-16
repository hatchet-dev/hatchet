-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry
    ADD COLUMN child_task_is_failure BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN child_task_error_message TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry
    DROP COLUMN child_task_is_failure,
    DROP COLUMN child_task_error_message;
-- +goose StatementEnd
