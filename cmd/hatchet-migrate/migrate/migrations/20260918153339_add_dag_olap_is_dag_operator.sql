-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_dags_olap ADD COLUMN is_dag_operator BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_dags_olap DROP COLUMN is_dag_operator;
-- +goose StatementEnd
