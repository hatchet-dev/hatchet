-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_filter
ADD COLUMN is_declarative BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_filter
DROP COLUMN is_declarative;
-- +goose StatementEnd
