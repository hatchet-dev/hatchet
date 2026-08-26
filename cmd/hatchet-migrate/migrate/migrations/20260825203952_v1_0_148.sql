-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_dags_olap
    ADD COLUMN latest_retry_count INT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_dags_olap
    DROP COLUMN latest_retry_count;
-- +goose StatementEnd
