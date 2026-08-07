-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_file ADD COLUMN latest_satisfied_order BIGINT NOT NULL DEFAULT 0;
ALTER TABLE v1_durable_event_log_entry ADD COLUMN satisfied_order BIGINT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry DROP COLUMN satisfied_order;
ALTER TABLE v1_durable_event_log_file DROP COLUMN latest_satisfied_order;
-- +goose StatementEnd
