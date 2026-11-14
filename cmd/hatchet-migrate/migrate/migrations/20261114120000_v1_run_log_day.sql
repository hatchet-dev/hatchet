-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_run_log_day (
    day DATE PRIMARY KEY,
    state TEXT NOT NULL CHECK (state IN ('ARCHIVED', 'DROPPING', 'DROPPED')),
    task_count BIGINT NOT NULL,
    event_count BIGINT NOT NULL,
    max_event_id BIGINT NOT NULL,
    tenant_days INTEGER NOT NULL,
    bundles BIGINT NOT NULL,
    bytes_compressed BIGINT NOT NULL,
    bytes_uncompressed BIGINT NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_run_log_day;
-- +goose StatementEnd
