-- +goose Up
-- +goose StatementBegin
ALTER TYPE v1_incoming_webhook_source_name ADD VALUE IF NOT EXISTS 'LINEAR';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally blank
-- +goose StatementEnd
