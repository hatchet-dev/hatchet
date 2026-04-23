-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN step_name TEXT;
ALTER TABLE v1_tasks_olap ADD COLUMN step_name TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN step_name;
ALTER TABLE v1_tasks_olap DROP COLUMN step_name;
-- +goose StatementEnd
