-- +goose Up
-- +goose StatementBegin
-- Runs inside goose's transaction: every type this file uses is created here, so nothing needs
-- the autocommit treatment an ALTER TYPE ... ADD VALUE would. The retired SERVERLESS operator
-- kind is never added; the serverless operator is a GRPC operator with leasing manager SELF,
-- and its rows are unique under v1_operator_tenant_name_kind_key like every other kind.

CREATE TYPE v1_serverless_endpoint_kind AS ENUM ('GENERIC_HTTP', 'CLOUDFLARE_WORKERS');

-- Customer configuration. Written by the API server; the operator writes only the status
-- columns, and only on state transitions (never per poll).
CREATE TABLE v1_serverless_endpoint (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    kind v1_serverless_endpoint_kind NOT NULL DEFAULT 'CLOUDFLARE_WORKERS',
    healthcheck_url TEXT NOT NULL,
    trigger_url TEXT NOT NULL,
    -- enc.EncryptString(secret, SigningSecretEncryptionDataID)
    signing_secret_enc TEXT NOT NULL,
    request_timeout_seconds INT NOT NULL DEFAULT 60,
    poll_interval_seconds INT NOT NULL DEFAULT 30,
    inline_wait_budget_ms INT NOT NULL DEFAULT 5000,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    -- hashtext(id::text) % shard_count, set on insert
    shard INT NOT NULL DEFAULT 0,
    -- status, written on transitions only
    healthy BOOLEAN,
    status_error TEXT,
    status_changed_at TIMESTAMPTZ,
    -- the action ids the endpoint's last accepted healthcheck declared, as registered with the
    -- engine; written by the owner on healthcheck change. Several endpoints of a tenant may
    -- declare the same action; each then serves it, like several workers would.
    registered_actions TEXT[] NOT NULL DEFAULT '{}',
    -- the subset of registered_actions whose task asked for an invocation websocket
    stream_actions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Set by the delete API together with updated_at, so the routing caches' incremental
    -- refresh sees the deletion through the version index and drops the endpoint; every other
    -- read filters deleted rows out. The operator purges the rows after a grace period.
    deleted_at TIMESTAMPTZ,
    CONSTRAINT v1_serverless_endpoint_pkey PRIMARY KEY (id)
);

-- Names are unique among the live endpoints of a tenant; a deleted endpoint's name is free.
CREATE UNIQUE INDEX v1_serverless_endpoint_tenant_name_key ON v1_serverless_endpoint (tenant_id, name) WHERE deleted_at IS NULL;

-- endpoints of an owned unit (owner: polling) and of a served tenant (routing cache)
CREATE INDEX v1_serverless_endpoint_unit_idx ON v1_serverless_endpoint (tenant_id, shard, id);

-- incremental refresh of the routing cache, keyed by row version: the later of updated_at
-- (configuration and registered_actions writes) and status_changed_at (health transitions)
CREATE INDEX v1_serverless_endpoint_version_idx ON v1_serverless_endpoint (tenant_id, GREATEST(updated_at, COALESCE(status_changed_at, updated_at)), id);

CREATE TABLE v1_serverless_tenant (
    tenant_id UUID NOT NULL,
    -- > 1 splits a hot tenant across processes
    shard_count INT NOT NULL DEFAULT 1,
    CONSTRAINT v1_serverless_tenant_pkey PRIMARY KEY (tenant_id)
);

-- One row per live serverless operator process (the shape of pgoutbox's consumer_sessions): the
-- only steady-state write. Liveness is expires_at, refreshed by the process heartbeat.
CREATE TABLE v1_serverless_process (
    -- out of process: random per start; in-engine: dispatcher id
    process_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    unit_count INT NOT NULL DEFAULT 0,
    endpoint_count INT NOT NULL DEFAULT 0,
    hostname TEXT,
    version TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_serverless_process_pkey PRIMARY KEY (process_id)
);

-- Lease unit (tenant, shard). One row per unit, created by the API with the tenant's first
-- endpoint (and per shard when shard_count grows). process_id NULL = unowned. Owner liveness
-- is resolved through v1_serverless_process (pgoutbox's "NULL expiry defers to the session"),
-- so holding a unit costs zero writes.
CREATE TABLE v1_serverless_lease (
    tenant_id UUID NOT NULL,
    shard INT NOT NULL DEFAULT 0,
    process_id UUID,
    claimed_at TIMESTAMPTZ,
    -- maintained by the API on endpoint create/delete; fair-share weight
    endpoint_count INT NOT NULL DEFAULT 0,
    CONSTRAINT v1_serverless_lease_pkey PRIMARY KEY (tenant_id, shard)
);

CREATE INDEX v1_serverless_lease_owner_idx ON v1_serverless_lease (process_id, tenant_id, shard);

-- claims walk unowned units with endpoints in key order from a random start; covering so the
-- claimable count is index only. Empty units are never claimed, so they are not in the index.
CREATE INDEX v1_serverless_lease_claimable_idx ON v1_serverless_lease (tenant_id, shard) INCLUDE (endpoint_count) WHERE process_id IS NULL AND endpoint_count > 0;

-- The gRPC operator service counts the action links of every worker of an operator once per
-- Listen stream (CountOperatorWorkerActions); the workers are found by (tenant, operator).
CREATE INDEX "Worker_tenantId_operatorId_idx" ON "Worker" ("tenantId", "operatorId");

-- Gates the serverless operator per tenant: endpoint creation over the API and the operator's
-- lease claims both require it.
ALTER TABLE tenant_entitlement ADD COLUMN serverless_operator BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenant_entitlement DROP COLUMN IF EXISTS serverless_operator;
DROP INDEX IF EXISTS "Worker_tenantId_operatorId_idx";
DROP INDEX IF EXISTS v1_serverless_lease_claimable_idx;
DROP INDEX IF EXISTS v1_serverless_lease_owner_idx;
DROP TABLE IF EXISTS v1_serverless_lease;
DROP TABLE IF EXISTS v1_serverless_process;
DROP TABLE IF EXISTS v1_serverless_tenant;
DROP INDEX IF EXISTS v1_serverless_endpoint_version_idx;
DROP INDEX IF EXISTS v1_serverless_endpoint_unit_idx;
DROP INDEX IF EXISTS v1_serverless_endpoint_tenant_name_key;
DROP TABLE IF EXISTS v1_serverless_endpoint;
DROP TYPE IF EXISTS v1_serverless_endpoint_kind;
-- +goose StatementEnd
