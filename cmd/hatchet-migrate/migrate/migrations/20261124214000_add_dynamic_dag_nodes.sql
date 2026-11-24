-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN dynamic_dag_nodes JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN dynamic_dag_nodes;
-- +goose StatementEnd
