-- +goose Up
-- +goose StatementBegin
ANALYZE v1_lookup_table_olap, v1_dag_to_task_olap, v1_statuses_olap, v1_task_events_olap;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- no op
-- +goose StatementEnd
