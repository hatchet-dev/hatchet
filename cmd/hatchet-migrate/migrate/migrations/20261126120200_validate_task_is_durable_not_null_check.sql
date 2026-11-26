-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task VALIDATE CONSTRAINT v1_task_is_durable_not_null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally blank
-- +goose StatementEnd
