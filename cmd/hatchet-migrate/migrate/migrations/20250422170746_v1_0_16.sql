-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_tasks_olap
ADD COLUMN step_readable_id TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_tasks_olap
DROP COLUMN step_readable_id;
-- +goose StatementEnd
