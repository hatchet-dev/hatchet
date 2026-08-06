-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_cagg_task_statuses_minute (
    bucket TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count BIGINT NOT NULL DEFAULT 0,
    running_count BIGINT NOT NULL DEFAULT 0,
    completed_count BIGINT NOT NULL DEFAULT 0,
    cancelled_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    evicted_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (bucket, tenant_id, workflow_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_cagg_task_statuses_minute;
-- +goose StatementEnd