-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_dags_olap
    ADD COLUMN is_operator_run BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN latest_retry_count INT NOT NULL DEFAULT 0;

-- Matt note: going to run this manually for instances we've turned on, don't think it's practical to run during the migration
-- but we do need it to prevent dags from hanging
-- UPDATE v1_dags_olap d
-- SET is_operator_run = TRUE
-- FROM v1_dag_to_task_olap dt
-- WHERE
--     (d.id, d.inserted_at) = (dt.dag_id, dt.dag_inserted_at)
--     AND (dt.dag_id, dt.dag_inserted_at) = (dt.task_id, dt.task_inserted_at)
--     AND d.inserted_at > << timestamp here >>;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_dags_olap
    DROP COLUMN is_operator_run,
    DROP COLUMN latest_retry_count;
-- +goose StatementEnd
