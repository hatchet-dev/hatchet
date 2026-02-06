-- +goose Up
-- +goose StatementBegin
ALTER TYPE v1_payload_type ADD VALUE IF NOT EXISTS 'DURABLE_EVENT_LOG_ENTRY_DATA';
ALTER TYPE v1_payload_type ADD VALUE IF NOT EXISTS 'DURABLE_EVENT_LOG_CALLBACK_RESULT_DATA';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Intentionally no down
-- +goose StatementEnd
