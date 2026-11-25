-- +goose Up
-- +goose StatementBegin
ALTER TYPE v1_sandbox_provider_kind ADD VALUE IF NOT EXISTS 'MODAL';
ALTER TYPE v1_sandbox_provider_kind ADD VALUE IF NOT EXISTS 'VERCEL';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
