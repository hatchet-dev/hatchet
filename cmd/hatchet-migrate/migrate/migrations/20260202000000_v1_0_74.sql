-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_durable_event_log (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    event_type TEXT NOT NULL,
    key TEXT NOT NULL,
    data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, task_inserted_at, key)
);

CREATE UNIQUE INDEX v1_durable_event_log_lookup_idx ON v1_durable_event_log (
    tenant_id ASC,
    task_id ASC,
    task_inserted_at ASC,
    retry_count ASC,
    key ASC
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS v1_durable_event_log_lookup_idx;
DROP TABLE IF EXISTS v1_durable_event_log;
-- +goose StatementEnd
