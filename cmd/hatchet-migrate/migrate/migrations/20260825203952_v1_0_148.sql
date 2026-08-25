-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_dags_olap
    ADD COLUMN is_operator_run BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN latest_retry_count INT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_dags_olap
    DROP COLUMN is_operator_run,
    DROP COLUMN latest_retry_count;
-- +goose StatementEnd
