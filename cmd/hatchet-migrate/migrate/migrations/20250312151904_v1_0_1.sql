-- +goose Up
-- +goose StatementBegin

DROP TRIGGER IF EXISTS after_v1_task_runtime_delete ON v1_task_runtime;

DROP FUNCTION IF EXISTS after_v1_task_runtime_delete_function();
-- +goose StatementEnd
