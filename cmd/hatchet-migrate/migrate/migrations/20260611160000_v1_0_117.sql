-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_file ADD COLUMN IF NOT EXISTS latest_satisfied_order BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry ADD COLUMN IF NOT EXISTS satisfied_order BIGINT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry DROP COLUMN IF EXISTS satisfied_order;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_file DROP COLUMN IF EXISTS latest_satisfied_order;
-- +goose StatementEnd
