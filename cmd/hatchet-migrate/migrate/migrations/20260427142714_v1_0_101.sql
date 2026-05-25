-- +goose Up
-- +goose StatementBegin
ANALYZE v1_runs_olap, v1_tasks_olap, v1_dags_olap;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- empty
-- +goose StatementEnd
