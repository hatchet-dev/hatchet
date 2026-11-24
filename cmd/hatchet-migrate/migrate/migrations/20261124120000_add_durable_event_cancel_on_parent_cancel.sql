-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry ADD COLUMN cancel_on_parent_cancel BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry DROP COLUMN cancel_on_parent_cancel;
-- +goose StatementEnd
