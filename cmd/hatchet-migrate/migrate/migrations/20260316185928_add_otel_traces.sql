-- +goose Up

CREATE TYPE v1_otel_span_kind AS ENUM ('UNSPECIFIED', 'INTERNAL', 'SERVER', 'CLIENT', 'PRODUCER', 'CONSUMER');
CREATE TYPE v1_otel_status_code AS ENUM ('UNSET', 'OK', 'ERROR');

CREATE TABLE v1_otel_trace (
    tenant_id       UUID NOT NULL,
    trace_id        BYTEA NOT NULL,
    span_id         BYTEA NOT NULL,
    parent_span_id  TEXT,
    span_name       TEXT NOT NULL,
    span_kind       v1_otel_span_kind NOT NULL DEFAULT 'UNSPECIFIED',
    service_name    TEXT NOT NULL DEFAULT 'unknown',
    status_code     v1_otel_status_code NOT NULL DEFAULT 'UNSET',
    status_message  TEXT,
    duration_ns     BIGINT NOT NULL DEFAULT 0,
    resource_attributes JSONB,
    span_attributes     JSONB,
    scope_name      TEXT,
    scope_version   TEXT,
    task_run_external_id    UUID,
    workflow_run_external_id UUID,
    retry_count     INT NOT NULL DEFAULT 0,
    start_time      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, trace_id, start_time, span_id)
) PARTITION BY RANGE (start_time);

CREATE INDEX idx_v1_otel_trace_task_lookup
    ON v1_otel_trace (tenant_id, task_run_external_id)
    WHERE task_run_external_id IS NOT NULL;

CREATE INDEX idx_v1_otel_trace_workflow_lookup
    ON v1_otel_trace (tenant_id, workflow_run_external_id)
    WHERE workflow_run_external_id IS NOT NULL;

SELECT create_v1_range_partition('v1_otel_trace'::TEXT, NOW()::DATE);
SELECT create_v1_range_partition('v1_otel_trace'::TEXT, (NOW() + INTERVAL '1 day')::DATE);

CREATE TABLE v1_otel_trace_lookup_table (
    tenant_id       UUID NOT NULL,
    external_id     UUID NOT NULL,
    retry_count     INT NOT NULL,
    trace_id        BYTEA NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, external_id, retry_count, start_time)
) PARTITION BY RANGE (start_time);

SELECT create_v1_range_partition('v1_otel_trace_lookup_table'::TEXT, NOW()::DATE);
SELECT create_v1_range_partition('v1_otel_trace_lookup_table'::TEXT, (NOW() + INTERVAL '1 day')::DATE);

-- +goose Down
DROP TABLE v1_otel_trace_lookup_table;
DROP TABLE v1_otel_trace;
DROP TYPE v1_otel_status_code;
DROP TYPE v1_otel_span_kind;
