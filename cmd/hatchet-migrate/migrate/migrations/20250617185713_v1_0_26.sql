-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_filter ADD COLUMN payload_hash TEXT GENERATED ALWAYS AS (MD5(payload::TEXT)) STORED;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_filter DROP COLUMN payload_hash;
-- +goose StatementEnd
