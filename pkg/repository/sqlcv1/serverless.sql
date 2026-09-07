-- Serverless operator tables. Endpoints and tenants are written by the API server; processes and
-- leases by the serverless operator (out of process or in-engine). Steady-state writes are one
-- process heartbeat per process: lease rows are only written on ownership changes and endpoint
-- rows only on status transitions and workflow changes.

-- name: CreateServerlessEndpoint :one
-- The shard is derived from the id so it is stable for the endpoint's lifetime. The cast to
-- bigint before abs() avoids the integer overflow abs(-2147483648) would raise.
INSERT INTO v1_serverless_endpoint (
    id,
    tenant_id,
    name,
    kind,
    healthcheck_url,
    trigger_url,
    signing_secret_enc,
    slots,
    durable_slots,
    request_timeout_seconds,
    poll_interval_seconds,
    inline_wait_budget_ms,
    labels,
    enabled,
    shard
) VALUES (
    @id::UUID,
    @tenantId::UUID,
    @name::TEXT,
    @kind::v1_serverless_endpoint_kind,
    @healthcheckUrl::TEXT,
    @triggerUrl::TEXT,
    @signingSecretEnc::TEXT,
    @slots::INT,
    @durableSlots::INT,
    @requestTimeoutSeconds::INT,
    @pollIntervalSeconds::INT,
    @inlineWaitBudgetMs::INT,
    @labels::JSONB,
    @enabled::BOOLEAN,
    (abs(hashtext((@id::UUID)::text)::bigint) % @shardCount::INT)::INT
)
RETURNING *;

-- name: GetServerlessEndpoint :one
SELECT *
FROM v1_serverless_endpoint
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID;

-- name: ListServerlessEndpoints :many
SELECT *
FROM v1_serverless_endpoint
WHERE tenant_id = @tenantId::UUID
ORDER BY created_at DESC, id DESC
LIMIT @endpointLimit::BIGINT
OFFSET @endpointOffset::BIGINT;

-- name: CountServerlessEndpoints :one
SELECT COUNT(*)
FROM v1_serverless_endpoint
WHERE tenant_id = @tenantId::UUID;

-- name: UpdateServerlessEndpoint :one
-- namespace and shard are never updatable: the namespace prefixes everything the endpoint has
-- registered with the engine, and the shard decides which lease unit owns the endpoint.
UPDATE v1_serverless_endpoint
SET
    name = COALESCE(sqlc.narg('name')::TEXT, name),
    kind = COALESCE(sqlc.narg('kind')::v1_serverless_endpoint_kind, kind),
    healthcheck_url = COALESCE(sqlc.narg('healthcheckUrl')::TEXT, healthcheck_url),
    trigger_url = COALESCE(sqlc.narg('triggerUrl')::TEXT, trigger_url),
    signing_secret_enc = COALESCE(sqlc.narg('signingSecretEnc')::TEXT, signing_secret_enc),
    slots = COALESCE(sqlc.narg('slots')::INT, slots),
    durable_slots = COALESCE(sqlc.narg('durableSlots')::INT, durable_slots),
    request_timeout_seconds = COALESCE(sqlc.narg('requestTimeoutSeconds')::INT, request_timeout_seconds),
    poll_interval_seconds = COALESCE(sqlc.narg('pollIntervalSeconds')::INT, poll_interval_seconds),
    inline_wait_budget_ms = COALESCE(sqlc.narg('inlineWaitBudgetMs')::INT, inline_wait_budget_ms),
    labels = COALESCE(sqlc.narg('labels')::JSONB, labels),
    enabled = COALESCE(sqlc.narg('enabled')::BOOLEAN, enabled),
    updated_at = NOW()
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
RETURNING *;

-- name: DeleteServerlessEndpoint :one
DELETE FROM v1_serverless_endpoint
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
RETURNING *;

-- name: ListServerlessEndpointsForUnits :many
-- Endpoints of the given (tenant, shard) units, keyset-paged by id through
-- v1_serverless_endpoint_unit_idx. tenantIds and shards are parallel arrays paired with unnest;
-- pass afterId = '00000000-0000-0000-0000-000000000000' for the first page.
SELECT e.*
FROM v1_serverless_endpoint e
JOIN (
    -- parallel unnest zips the two arrays into (tenant_id, shard) pairs
    SELECT
        unnest(@tenantIds::UUID[]) AS tenant_id,
        unnest(@shards::INT[]) AS shard
) AS u ON e.tenant_id = u.tenant_id AND e.shard = u.shard
WHERE e.id > @afterId::UUID
ORDER BY e.id
LIMIT @endpointLimit::BIGINT;

-- name: ListServerlessEndpointsForTenant :many
-- Full load of a tenant's routing cache. Disabled endpoints are included so callers can decide
-- what to route; the routing cache filters on enabled itself.
SELECT *
FROM v1_serverless_endpoint
WHERE tenant_id = @tenantId::UUID
ORDER BY id;

-- name: ListServerlessEndpointsUpdatedSince :many
-- Incremental refresh of a tenant's routing cache through v1_serverless_endpoint_updated_idx.
-- Every write the cache needs to see (config changes, registered_actions) bumps updated_at;
-- health flips do not, so they never appear here.
SELECT *
FROM v1_serverless_endpoint
WHERE
    tenant_id = @tenantId::UUID
    AND updated_at > @since::TIMESTAMPTZ
ORDER BY updated_at, id;

-- name: UpdateServerlessEndpointStatus :exec
-- Written by the owner on a healthy/unhealthy transition only. Deliberately leaves updated_at
-- alone: a health flip is not a routing change, so the routing caches of other processes must
-- not reload the endpoint for it.
UPDATE v1_serverless_endpoint
SET
    healthy = @healthy::BOOLEAN,
    status_error = sqlc.narg('statusError')::TEXT,
    status_changed_at = NOW()
WHERE id = @id::UUID;

-- name: UpdateServerlessEndpointRegisteredActions :exec
-- Written by the owner when a healthcheck changes the endpoint's workflows. Bumps updated_at so
-- ListServerlessEndpointsUpdatedSince surfaces the new action set to every registration for
-- the tenant.
UPDATE v1_serverless_endpoint
SET
    registered_actions = @registeredActions::TEXT[],
    updated_at = NOW()
WHERE id = @id::UUID;

-- name: UpsertServerlessTenant :one
-- Creates the tenant's serverless settings row with defaults if it does not exist and returns
-- the current row either way. The no-op update makes RETURNING work on conflict.
INSERT INTO v1_serverless_tenant (tenant_id)
VALUES (@tenantId::UUID)
ON CONFLICT (tenant_id) DO UPDATE
SET tenant_id = EXCLUDED.tenant_id
RETURNING *;

-- name: GetServerlessTenant :one
SELECT *
FROM v1_serverless_tenant
WHERE tenant_id = @tenantId::UUID;

-- name: UpdateServerlessTenantShardCount :one
UPDATE v1_serverless_tenant
SET shard_count = @shardCount::INT
WHERE tenant_id = @tenantId::UUID
RETURNING *;

-- name: UpsertServerlessProcess :exec
-- The process heartbeat. expires_at is computed in SQL so process clock skew does not matter.
INSERT INTO v1_serverless_process (process_id, expires_at, unit_count, endpoint_count, hostname, version)
VALUES (
    @processId::UUID,
    now() + @ttl::interval,
    @unitCount::INT,
    @endpointCount::INT,
    sqlc.narg('hostname')::TEXT,
    sqlc.narg('version')::TEXT
)
ON CONFLICT (process_id) DO UPDATE SET
    expires_at = EXCLUDED.expires_at,
    unit_count = EXCLUDED.unit_count,
    endpoint_count = EXCLUDED.endpoint_count;

-- name: ListServerlessProcesses :many
-- All process rows with liveness resolved in SQL, again so clock skew between processes does
-- not matter. Callers split the result into live processes and dead process ids.
SELECT
    sqlc.embed(p),
    (p.expires_at < now())::BOOLEAN AS expired
FROM v1_serverless_process p
ORDER BY p.process_id;

-- name: DeleteExpiredServerlessProcesses :execrows
-- Sweeps rows of processes that expired before the cutoff. Rows are kept for a while after
-- expiry so ClaimServerlessLeases can still see the dead ids of recently crashed processes.
DELETE FROM v1_serverless_process
WHERE expires_at < @cutoff::TIMESTAMPTZ;

-- name: DeleteServerlessProcess :exec
DELETE FROM v1_serverless_process
WHERE process_id = @processId::UUID;

-- name: InsertServerlessLeaseIfAbsent :exec
INSERT INTO v1_serverless_lease (tenant_id, shard)
VALUES (@tenantId::UUID, @shard::INT)
ON CONFLICT (tenant_id, shard) DO NOTHING;

-- name: ClaimServerlessLeases :many
-- Claims up to @claimLimit units that are unowned or owned by a dead process. FOR UPDATE SKIP LOCKED
-- lets concurrent claimers race without blocking; a unit is claimed by exactly one of them.
WITH claimable AS (
    SELECT l.tenant_id, l.shard, l.endpoint_count
    FROM v1_serverless_lease l
    WHERE l.process_id IS NULL OR l.process_id = ANY(@deadIds::UUID[])
    ORDER BY random()
    LIMIT @claimLimit::INT
    FOR UPDATE SKIP LOCKED
)
UPDATE v1_serverless_lease l
SET process_id = @processId::UUID, claimed_at = now()
FROM claimable c
WHERE l.tenant_id = c.tenant_id AND l.shard = c.shard
RETURNING l.tenant_id, l.shard, l.endpoint_count;

-- name: ShedServerlessLeases :many
-- Releases the given units. Guarded by process_id so a process that lost a unit to a takeover
-- (after being declared dead) cannot release the new owner's lease.
UPDATE v1_serverless_lease l
SET process_id = NULL, claimed_at = NULL
FROM (
    SELECT
        unnest(@tenantIds::UUID[]) AS tenant_id,
        unnest(@shards::INT[]) AS shard
) AS u
WHERE
    l.tenant_id = u.tenant_id
    AND l.shard = u.shard
    AND l.process_id = @processId::UUID
RETURNING l.tenant_id, l.shard, l.endpoint_count;

-- name: ReleaseAllServerlessLeases :execrows
UPDATE v1_serverless_lease
SET process_id = NULL, claimed_at = NULL
WHERE process_id = @processId::UUID;

-- name: ListOwnedServerlessLeases :many
SELECT *
FROM v1_serverless_lease
WHERE process_id = @processId::UUID
ORDER BY tenant_id, shard;

-- name: CountUnownedServerlessLeases :one
-- Served by v1_serverless_lease_unowned_idx (partial on process_id IS NULL).
SELECT
    COUNT(*)::BIGINT AS unit_count,
    COALESCE(SUM(endpoint_count), 0)::BIGINT AS endpoint_count
FROM v1_serverless_lease
WHERE process_id IS NULL;

-- name: IncrementServerlessLeaseEndpointCount :exec
UPDATE v1_serverless_lease
SET endpoint_count = endpoint_count + @delta::INT
WHERE
    tenant_id = @tenantId::UUID
    AND shard = @shard::INT;
