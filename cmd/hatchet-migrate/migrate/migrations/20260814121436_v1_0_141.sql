-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry ADD COLUMN triggered_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry DROP COLUMN triggered_at;
-- +goose StatementEnd
