-- +goose no transaction

-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS ix_v1_tasks_olap_additional_metadata_gin;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally blank
-- +goose StatementEnd
