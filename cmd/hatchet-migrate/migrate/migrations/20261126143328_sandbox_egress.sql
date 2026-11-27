-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_sandbox ADD COLUMN IF NOT EXISTS egress JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_sandbox DROP COLUMN IF EXISTS egress;
-- +goose StatementEnd
