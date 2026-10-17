-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task
    ADD COLUMN triggers JSONB,
    ADD COLUMN dag_parent_task_run_ids UUID[],
    ADD COLUMN child_task_desired_worker_labels JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task
    DROP COLUMN triggers,
    DROP COLUMN dag_parent_task_run_ids,
    DROP COLUMN child_task_desired_worker_labels;
-- +goose StatementEnd
