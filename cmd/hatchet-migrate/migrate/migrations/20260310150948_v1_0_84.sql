-- +goose Up

CREATE TABLE v1_otel_traces (
    id              BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id       UUID NOT NULL,
    trace_id        TEXT NOT NULL,
    span_id         TEXT NOT NULL,
    parent_span_id  TEXT NOT NULL DEFAULT '',
    span_name       TEXT NOT NULL,
    span_kind       TEXT NOT NULL DEFAULT 'INTERNAL',
    service_name    TEXT NOT NULL DEFAULT 'unknown',
    status_code     TEXT NOT NULL DEFAULT 'UNSET',
    status_message  TEXT NOT NULL DEFAULT '',
    duration_ns     BIGINT NOT NULL DEFAULT 0,
    resource_attributes JSONB NOT NULL DEFAULT '{}',
    span_attributes     JSONB NOT NULL DEFAULT '{}',
    scope_name      TEXT NOT NULL DEFAULT '',
    scope_version   TEXT NOT NULL DEFAULT '',
    task_run_external_id    UUID,
    workflow_run_external_id UUID,
    start_time      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (id, start_time)
) PARTITION BY RANGE (start_time);

CREATE INDEX idx_v1_otel_traces_task_lookup
    ON v1_otel_traces (tenant_id, task_run_external_id)
    WHERE task_run_external_id IS NOT NULL;

CREATE INDEX idx_v1_otel_traces_trace
    ON v1_otel_traces (tenant_id, trace_id, start_time);

-- +goose StatementBegin
SELECT create_v1_range_partition('v1_otel_traces'::text, CURRENT_DATE::date);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS v1_otel_traces;
