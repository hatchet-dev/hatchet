-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload_wal ALTER COLUMN operation SET DEFAULT 'CREATE';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload_wal ALTER COLUMN operation DROP DEFAULT;
-- +goose StatementEnd
