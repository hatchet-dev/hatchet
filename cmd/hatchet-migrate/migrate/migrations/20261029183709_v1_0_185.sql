-- +goose Up
-- +goose StatementBegin

CREATE TYPE v1_sandbox_provider_kind AS ENUM ('E2B', 'SUPABASE', 'ARCHIL', 'FREESTYLE');

CREATE TABLE v1_sandbox_provider (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    kind v1_sandbox_provider_kind NOT NULL,
    api_key_enc TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    default_template TEXT,
    max_idle_ttl_seconds INT NOT NULL DEFAULT 86400,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    provider_partition INT NOT NULL DEFAULT 0,
    healthy BOOLEAN,
    status_error TEXT,
    status_changed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT v1_sandbox_provider_pkey PRIMARY KEY (id)
);

CREATE UNIQUE INDEX v1_sandbox_provider_tenant_name_key ON v1_sandbox_provider (tenant_id, name) WHERE deleted_at IS NULL;

CREATE INDEX v1_sandbox_provider_unit_idx ON v1_sandbox_provider (tenant_id, provider_partition, id);

CREATE INDEX v1_sandbox_provider_version_idx ON v1_sandbox_provider (tenant_id, GREATEST(updated_at, COALESCE(status_changed_at, updated_at)), id);

CREATE TYPE v1_sandbox_status AS ENUM ('CREATING', 'RUNNING', 'PAUSED', 'DESTROYED', 'FAILED');

CREATE TABLE v1_sandbox (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    external_id TEXT,
    template TEXT,
    status v1_sandbox_status NOT NULL DEFAULT 'CREATING',
    status_error TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    idle_ttl_ms BIGINT NOT NULL DEFAULT 0,
    latest_snapshot_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT v1_sandbox_pkey PRIMARY KEY (tenant_id, id)
);

CREATE INDEX v1_sandbox_provider_idx ON v1_sandbox (tenant_id, provider_id, created_at DESC);

CREATE TABLE v1_sandbox_snapshot (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    sandbox_id TEXT NOT NULL,
    parent_id TEXT,
    external_id TEXT NOT NULL,
    size_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_sandbox_snapshot_pkey PRIMARY KEY (tenant_id, id)
);

CREATE INDEX v1_sandbox_snapshot_sandbox_idx ON v1_sandbox_snapshot (tenant_id, sandbox_id, created_at DESC);

CREATE TABLE v1_sandbox_turn (
    run_id TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    sandbox_id TEXT NOT NULL,
    prev_snapshot_id TEXT,
    result JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_sandbox_turn_pkey PRIMARY KEY (tenant_id, run_id)
);

CREATE INDEX v1_sandbox_turn_created_at_idx ON v1_sandbox_turn (created_at);

CREATE TYPE v1_sandbox_event_kind AS ENUM (
    'HEALTHCHECK_FAILED',
    'UNHEALTHY',
    'HEALTHY',
    'SANDBOX_CREATED',
    'SANDBOX_CREATE_FAILED',
    'TURN_FAILED',
    'SNAPSHOT_FAILED',
    'SANDBOX_DESTROYED'
);

CREATE TABLE v1_sandbox_event (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    sandbox_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind v1_sandbox_event_kind NOT NULL,
    message TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT v1_sandbox_event_pkey PRIMARY KEY (tenant_id, provider_id, created_at, id)
) PARTITION BY RANGE(created_at);

SELECT create_v1_range_partition('v1_sandbox_event', (NOW() AT TIME ZONE 'UTC')::DATE);
SELECT create_v1_range_partition('v1_sandbox_event', (NOW() AT TIME ZONE 'UTC')::DATE + 1);

CREATE TABLE v1_sandbox_tenant (
    tenant_id UUID NOT NULL,
    provider_partition_count INT NOT NULL DEFAULT 1,
    CONSTRAINT v1_sandbox_tenant_pkey PRIMARY KEY (tenant_id)
);

CREATE TABLE v1_sandbox_process (
    process_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    unit_count INT NOT NULL DEFAULT 0,
    provider_count INT NOT NULL DEFAULT 0,
    hostname TEXT,
    version TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_sandbox_process_pkey PRIMARY KEY (process_id)
);

CREATE TABLE v1_sandbox_lease (
    tenant_id UUID NOT NULL,
    provider_partition INT NOT NULL DEFAULT 0,
    process_id UUID,
    claimed_at TIMESTAMPTZ,
    provider_count INT NOT NULL DEFAULT 0,
    CONSTRAINT v1_sandbox_lease_pkey PRIMARY KEY (tenant_id, provider_partition)
);

CREATE INDEX v1_sandbox_lease_owner_idx ON v1_sandbox_lease (process_id, tenant_id, provider_partition);

CREATE INDEX v1_sandbox_lease_claimable_idx ON v1_sandbox_lease (tenant_id, provider_partition) INCLUDE (provider_count) WHERE process_id IS NULL AND provider_count > 0;

ALTER TABLE tenant_entitlement ADD COLUMN sandbox_operator BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenant_entitlement DROP COLUMN IF EXISTS sandbox_operator;
DROP TABLE IF EXISTS v1_sandbox_lease;
DROP TABLE IF EXISTS v1_sandbox_process;
DROP TABLE IF EXISTS v1_sandbox_tenant;
DROP TABLE IF EXISTS v1_sandbox_event;
DROP TYPE IF EXISTS v1_sandbox_event_kind;
DROP TABLE IF EXISTS v1_sandbox_turn;
DROP TABLE IF EXISTS v1_sandbox_snapshot;
DROP TABLE IF EXISTS v1_sandbox;
DROP TYPE IF EXISTS v1_sandbox_status;
DROP TABLE IF EXISTS v1_sandbox_provider;
DROP TYPE IF EXISTS v1_sandbox_provider_kind;
-- +goose StatementEnd
