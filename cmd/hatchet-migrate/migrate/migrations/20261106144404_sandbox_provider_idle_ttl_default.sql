-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_sandbox_provider ALTER COLUMN max_idle_ttl_seconds SET DEFAULT 900;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_sandbox_provider ALTER COLUMN max_idle_ttl_seconds SET DEFAULT 3600;
-- +goose StatementEnd
