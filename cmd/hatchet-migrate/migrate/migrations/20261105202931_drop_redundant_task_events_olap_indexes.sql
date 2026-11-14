-- +goose no transaction
-- +goose Up

-- +goose StatementBegin
DROP INDEX IF EXISTS v1_task_events_olap_task_id_idx;
DROP INDEX IF EXISTS v1_task_events_olap_task_inserted_at_idx;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally blank
-- +goose StatementEnd
