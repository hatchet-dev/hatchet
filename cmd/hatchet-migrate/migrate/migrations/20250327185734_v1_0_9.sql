-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_dags_olap ADD COLUMN total_tasks INT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_dags_olap DROP COLUMN total_tasks;
-- +goose StatementEnd
