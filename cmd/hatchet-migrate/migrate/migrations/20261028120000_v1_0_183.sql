-- +goose Up
-- +goose StatementBegin
-- The new value is not used in this transaction, so ADD VALUE may run inside it. The down step
-- cannot remove an enum value, so the guard lets this migration run again after it.
ALTER TYPE v1_serverless_endpoint_kind ADD VALUE IF NOT EXISTS 'VERCEL';

CREATE TYPE v1_serverless_endpoint_event_kind AS ENUM (
    'HEALTHCHECK_FAILED',
    'UNHEALTHY',
    'HEALTHY',
    'CATALOG_REGISTERED',
    'CATALOG_REJECTED',
    'DELIVERY_FAILED'
);

-- What the serverless operator observed about an endpoint, for the dashboard. Written by the
-- owning process on failure-streak starts, status transitions, catalog changes and rate-limited
-- delivery failures, never once per poll. Partitioned by day like the other core tables, and
-- dropped with them after the core retention period; reads also stop at the tenant's retention.
CREATE TABLE v1_serverless_endpoint_event (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind v1_serverless_endpoint_event_kind NOT NULL,
    message TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- also serves an endpoint's events newest first, scanned backward
    CONSTRAINT v1_serverless_endpoint_event_pkey PRIMARY KEY (tenant_id, endpoint_id, created_at, id)
) PARTITION BY RANGE(created_at);

SELECT create_v1_range_partition('v1_serverless_endpoint_event', (NOW() AT TIME ZONE 'UTC')::DATE);
SELECT create_v1_range_partition('v1_serverless_endpoint_event', (NOW() AT TIME ZONE 'UTC')::DATE + 1);

-- Bounds a whole invocation, so default to a common platform limit (Vercel's default maxDuration).
ALTER TABLE v1_serverless_endpoint ALTER COLUMN request_timeout_seconds SET DEFAULT 300;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_serverless_endpoint ALTER COLUMN request_timeout_seconds SET DEFAULT 60;
DROP TABLE v1_serverless_endpoint_event;
DROP TYPE v1_serverless_endpoint_event_kind;
-- A value cannot be removed from an enum; VERCEL stays on v1_serverless_endpoint_kind.
-- +goose StatementEnd
