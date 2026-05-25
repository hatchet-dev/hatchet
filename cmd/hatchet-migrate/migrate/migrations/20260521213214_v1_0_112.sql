-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry ADD COLUMN child_task_external_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry DROP COLUMN child_task_external_id;
-- +goose StatementEnd
