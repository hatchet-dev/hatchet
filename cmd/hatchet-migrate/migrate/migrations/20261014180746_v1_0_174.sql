-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task
    ADD COLUMN triggers JSONB,
    ADD COLUMN dag_parent_task_run_ids UUID[],
    ADD COLUMN child_task_desired_worker_labels JSONB,
    ADD COLUMN task_input_payload_version SMALLINT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task
    DROP COLUMN triggers,
    DROP COLUMN dag_parent_task_run_ids,
    DROP COLUMN child_task_desired_worker_labels,
    DROP COLUMN task_input_payload_version;
-- +goose StatementEnd
