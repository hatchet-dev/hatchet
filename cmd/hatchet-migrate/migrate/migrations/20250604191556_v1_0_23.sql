-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_events_olap ADD COLUMN scope TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_events_olap DROP COLUMN scope;
-- +goose StatementEnd
