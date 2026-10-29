-- +goose Up
-- +goose StatementBegin
-- Runs inside goose's transaction: every type this file uses is created here, so nothing needs
-- the autocommit treatment an ALTER TYPE ... ADD VALUE would. The sandbox operator is a GRPC
-- operator with leasing manager SELF, like the serverless operator.

CREATE TYPE v1_sandbox_provider_kind AS ENUM ('E2B', 'DAYTONA', 'MODAL', 'SUPABASE', 'ARCHIL', 'FREESTYLE');

-- One sandbox vendor account a tenant connected. Written by the API server; the operator
-- writes only the status columns, and only on state transitions.
CREATE TABLE v1_sandbox_provider (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    kind v1_sandbox_provider_kind NOT NULL,
    -- enc.EncryptString(key, ApiKeyEncryptionDataID); never returned by the API
    api_key_enc TEXT NOT NULL,
    -- vendor-specific settings: base url, team or project, region
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- template a sandbox is created from when the SDK names none
    default_template TEXT,
    max_idle_ttl_seconds INT NOT NULL DEFAULT 86400,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    -- hashtext(id::text) % provider_partition_count, set on insert
    provider_partition INT NOT NULL DEFAULT 0,
    -- status, written on transitions only
    healthy BOOLEAN,
    status_error TEXT,
    status_changed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Set by the delete API together with updated_at; every other read filters deleted rows
    -- out. The operator purges the rows after a grace period.
    deleted_at TIMESTAMPTZ,
    CONSTRAINT v1_sandbox_provider_pkey PRIMARY KEY (id)
);

-- Names are unique among the live providers of a tenant; a deleted provider's name is free.
CREATE UNIQUE INDEX v1_sandbox_provider_tenant_name_key ON v1_sandbox_provider (tenant_id, name) WHERE deleted_at IS NULL;

-- providers of an owned unit (owner: sessions) and of a served tenant
CREATE INDEX v1_sandbox_provider_unit_idx ON v1_sandbox_provider (tenant_id, provider_partition, id);

-- incremental refresh keyed by row version: the later of updated_at and status_changed_at
CREATE INDEX v1_sandbox_provider_version_idx ON v1_sandbox_provider (tenant_id, GREATEST(updated_at, COALESCE(status_changed_at, updated_at)), id);

CREATE TYPE v1_sandbox_status AS ENUM ('CREATING', 'RUNNING', 'PAUSED', 'DESTROYED', 'FAILED');

-- One sandbox the SDK created. The id is the creating run's id or a caller-supplied id
-- (lowercase letters, digits and hyphens, up to 63 characters). latest_snapshot_id is the
-- pointer $connect resolves; it only ever moves forward.
CREATE TABLE v1_sandbox (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    -- the vendor's own sandbox id, once created
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

-- Snapshot lineage. The id is the run id of the turn that made the snapshot, so a retried
-- turn overwrites its own snapshot and replay stays safe. parent_id is the snapshot the
-- sandbox ran from when the turn started; two snapshots with the same parent are a fork.
CREATE TABLE v1_sandbox_snapshot (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    sandbox_id TEXT NOT NULL,
    parent_id TEXT,
    -- the vendor's snapshot or template id
    external_id TEXT NOT NULL,
    size_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_sandbox_snapshot_pkey PRIMARY KEY (tenant_id, id)
);

CREATE INDEX v1_sandbox_snapshot_sandbox_idx ON v1_sandbox_snapshot (tenant_id, sandbox_id, created_at DESC);

-- One row per turn that ran, keyed by the task run id, so a retried turn returns the result
-- the first attempt produced instead of running the command again. prev_snapshot_id is the
-- sandbox's latest snapshot when the turn started. The operator prunes rows older than a day.
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

-- What the sandbox operator observed about a provider, for the dashboard. Written on status
-- transitions and failures, never once per turn. Partitioned by day like the other core
-- tables and dropped with them after the core retention period.
CREATE TABLE v1_sandbox_event (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    sandbox_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind v1_sandbox_event_kind NOT NULL,
    message TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- also serves a provider's events newest first, scanned backward
    CONSTRAINT v1_sandbox_event_pkey PRIMARY KEY (tenant_id, provider_id, created_at, id)
) PARTITION BY RANGE(created_at);

SELECT create_v1_range_partition('v1_sandbox_event', (NOW() AT TIME ZONE 'UTC')::DATE);
SELECT create_v1_range_partition('v1_sandbox_event', (NOW() AT TIME ZONE 'UTC')::DATE + 1);

CREATE TABLE v1_sandbox_tenant (
    tenant_id UUID NOT NULL,
    -- > 1 splits a hot tenant across processes
    provider_partition_count INT NOT NULL DEFAULT 1,
    CONSTRAINT v1_sandbox_tenant_pkey PRIMARY KEY (tenant_id)
);

-- One row per live sandbox operator process: the only steady-state write. Liveness is
-- expires_at, refreshed by the process heartbeat.
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

-- Lease unit (tenant, partition). One row per unit, created by the API with the tenant's first
-- provider. process_id NULL = unowned. Owner liveness is resolved through v1_sandbox_process,
-- so holding a unit costs zero writes.
CREATE TABLE v1_sandbox_lease (
    tenant_id UUID NOT NULL,
    provider_partition INT NOT NULL DEFAULT 0,
    process_id UUID,
    claimed_at TIMESTAMPTZ,
    -- maintained by the API on provider create/delete; fair-share weight
    provider_count INT NOT NULL DEFAULT 0,
    CONSTRAINT v1_sandbox_lease_pkey PRIMARY KEY (tenant_id, provider_partition)
);

CREATE INDEX v1_sandbox_lease_owner_idx ON v1_sandbox_lease (process_id, tenant_id, provider_partition);

-- claims walk unowned units with providers in key order from a random start; covering so the
-- claimable count is index only. Empty units are never claimed, so they are not in the index.
CREATE INDEX v1_sandbox_lease_claimable_idx ON v1_sandbox_lease (tenant_id, provider_partition) INCLUDE (provider_count) WHERE process_id IS NULL AND provider_count > 0;

-- Gates the sandbox operator per tenant: provider creation over the API and the operator's
-- lease claims both require it.
ALTER TABLE tenant_entitlement ADD COLUMN sandbox_operator BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenant_entitlement DROP COLUMN IF EXISTS sandbox_operator;
DROP INDEX IF EXISTS v1_sandbox_lease_claimable_idx;
DROP INDEX IF EXISTS v1_sandbox_lease_owner_idx;
DROP TABLE IF EXISTS v1_sandbox_lease;
DROP TABLE IF EXISTS v1_sandbox_process;
DROP TABLE IF EXISTS v1_sandbox_tenant;
DROP TABLE IF EXISTS v1_sandbox_event;
DROP TYPE IF EXISTS v1_sandbox_event_kind;
DROP INDEX IF EXISTS v1_sandbox_turn_created_at_idx;
DROP TABLE IF EXISTS v1_sandbox_turn;
DROP INDEX IF EXISTS v1_sandbox_snapshot_sandbox_idx;
DROP TABLE IF EXISTS v1_sandbox_snapshot;
DROP INDEX IF EXISTS v1_sandbox_provider_idx;
DROP TABLE IF EXISTS v1_sandbox;
DROP TYPE IF EXISTS v1_sandbox_status;
DROP INDEX IF EXISTS v1_sandbox_provider_version_idx;
DROP INDEX IF EXISTS v1_sandbox_provider_unit_idx;
DROP INDEX IF EXISTS v1_sandbox_provider_tenant_name_key;
DROP TABLE IF EXISTS v1_sandbox_provider;
DROP TYPE IF EXISTS v1_sandbox_provider_kind;
-- +goose StatementEnd
