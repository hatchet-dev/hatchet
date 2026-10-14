-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task_events_olap ADD COLUMN is_dag_orchestrator BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_events_olap DROP COLUMN is_dag_orchestrator;
-- +goose StatementEnd
