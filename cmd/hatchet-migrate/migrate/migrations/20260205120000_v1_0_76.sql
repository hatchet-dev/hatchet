-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_idempotency_key
    ADD COLUMN last_denied_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_idempotency_key
    DROP COLUMN last_denied_at;
-- +goose StatementEnd
