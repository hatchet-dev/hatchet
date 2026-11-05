-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'INCOMING_WEBHOOK';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
