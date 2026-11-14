-- +goose Up
-- +goose StatementBegin
ANALYZE v1_lookup_table, v1_dag_data, v1_dag_to_task, v1_task_expression_eval;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- no op
-- +goose StatementEnd
