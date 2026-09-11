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

-- name: GetServerlessEndpointById :one
-- Resolves an endpoint before its tenant is known, for the API's resource populator, which
-- checks the returned tenant_id against the caller's tenant.
SELECT *
FROM v1_serverless_endpoint
WHERE id = @id::UUID;

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
-- Incremental refresh of a tenant's routing cache through v1_serverless_endpoint_version_idx.
-- A row's version is the later of updated_at (configuration and registered_actions writes)
-- and status_changed_at (health transitions written by the owner), so every write the cache
-- needs to see surfaces here. Keyset on (version, id) from the last row the caller applied.
SELECT *
FROM v1_serverless_endpoint
WHERE
    tenant_id = @tenantId::UUID
    AND (GREATEST(updated_at, COALESCE(status_changed_at, updated_at)), id) > (@since::TIMESTAMPTZ, @sinceId::UUID)
ORDER BY GREATEST(updated_at, COALESCE(status_changed_at, updated_at)), id;

-- name: UpdateServerlessEndpointStatus :one
-- Written by the owner on a healthy/unhealthy transition only. Deliberately leaves updated_at
-- alone: a health flip is not a routing change. The write's own timestamp is returned so the
-- writer's cache can order it against rows read before or after it.
UPDATE v1_serverless_endpoint
SET
    healthy = @healthy::BOOLEAN,
    status_error = sqlc.narg('statusError')::TEXT,
    status_changed_at = NOW()
WHERE id = @id::UUID
RETURNING status_changed_at;

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

-- name: DeleteExpiredServerlessProcesses :one
-- Sweeps rows of processes that expired before the cutoff. Rows are kept for a while after
-- expiry so ClaimServerlessLeases takes their units over first; whatever a swept process still
-- owns is released in the same statement, so deleting an owner row never strands its units.
WITH deleted AS (
    DELETE FROM v1_serverless_process
    WHERE expires_at < @cutoff::TIMESTAMPTZ
    RETURNING process_id
), released AS (
    UPDATE v1_serverless_lease l
    SET process_id = NULL, claimed_at = NULL
    FROM deleted d
    WHERE l.process_id = d.process_id
    RETURNING l.tenant_id
)
SELECT
    (SELECT COUNT(*) FROM deleted)::BIGINT AS deleted_processes,
    (SELECT COUNT(*) FROM released)::BIGINT AS released_units;

-- name: DeleteServerlessProcess :exec
-- The last step of a graceful shutdown. The process released its leases beforehand; any it
-- still holds after a failed release are released here so the row deletion cannot strand them.
WITH released AS (
    UPDATE v1_serverless_lease
    SET process_id = NULL, claimed_at = NULL
    WHERE process_id = @processId::UUID
    RETURNING tenant_id
)
DELETE FROM v1_serverless_process
WHERE process_id = @processId::UUID;

-- name: InsertServerlessLeaseIfAbsent :exec
INSERT INTO v1_serverless_lease (tenant_id, shard)
VALUES (@tenantId::UUID, @shard::INT)
ON CONFLICT (tenant_id, shard) DO NOTHING;

-- name: ClaimServerlessLeases :many
-- Claims up to @claimLimit units for @processId. Only units with endpoints are claimable: an
-- empty unit (shard growth, every endpoint deleted) has nothing to poll and is left unowned
-- until an endpoint lands on it. Unowned units come first, walked in (tenant_id, shard) order
-- from @afterTenantId/@afterShard through v1_serverless_lease_claimable_idx (the caller starts
-- at a random key and wraps around), then units of processes whose heartbeat row has expired,
-- walked per dead process through v1_serverless_lease_owner_idx. Neither walk sorts the
-- candidate population. Liveness is decided here, in the statement's own snapshot, never from
-- a process list read earlier: an owner that heartbeated since the caller looked is live and
-- keeps its units, and the caller must itself be live to claim at all, so a process whose row
-- expired or was swept cannot take units until its next heartbeat. FOR UPDATE SKIP LOCKED lets
-- concurrent claimers race without blocking; a unit is claimed by exactly one of them.
WITH unowned AS (
    SELECT l.tenant_id, l.shard
    FROM v1_serverless_lease l
    WHERE
        l.process_id IS NULL
        AND l.endpoint_count > 0
        AND (l.tenant_id, l.shard) > (@afterTenantId::UUID, @afterShard::INT)
    ORDER BY l.tenant_id, l.shard
    LIMIT @claimLimit::INT
    FOR UPDATE SKIP LOCKED
), abandoned AS (
    SELECT a.tenant_id, a.shard
    FROM v1_serverless_process p
    CROSS JOIN LATERAL (
        SELECT l.tenant_id, l.shard
        FROM v1_serverless_lease l
        WHERE l.process_id = p.process_id AND l.endpoint_count > 0
        ORDER BY l.tenant_id, l.shard
        LIMIT @claimLimit::INT
        FOR UPDATE SKIP LOCKED
    ) a
    WHERE p.expires_at < now()
    LIMIT @claimLimit::INT
), claimable AS (
    SELECT tenant_id, shard FROM unowned
    UNION ALL
    SELECT tenant_id, shard FROM abandoned
    LIMIT @claimLimit::INT
)
UPDATE v1_serverless_lease l
SET process_id = @processId::UUID, claimed_at = now()
FROM claimable c
WHERE
    l.tenant_id = c.tenant_id
    AND l.shard = c.shard
    AND EXISTS (
        SELECT 1
        FROM v1_serverless_process me
        WHERE me.process_id = @processId::UUID AND me.expires_at >= now()
    )
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

-- name: CountClaimableServerlessLeases :one
-- Counts what a process may claim under the rules of ClaimServerlessLeases: unowned units
-- with endpoints (v1_serverless_lease_claimable_idx, index only) plus units with endpoints
-- still held by processes whose heartbeat row has expired (v1_serverless_lease_owner_idx), so
-- a survivor's fair share includes the work of dead processes. Empty units are not counted,
-- as they are not claimed: a window of empty rows would otherwise report a claimable
-- population of zero weight and hide the populated units behind it. Each side is a sample
-- of at most @countLimit units, and the abandoned side is bounded as a whole, not per dead
-- process: a caller only needs to know the claimable population up to its own claim budget
-- for one tick, and a full sample tells it the backlog is at least that large.
WITH unowned AS (
    SELECT COUNT(*) AS n, COALESCE(SUM(u.endpoint_count), 0) AS w
    FROM (
        SELECT endpoint_count
        FROM v1_serverless_lease
        WHERE process_id IS NULL AND endpoint_count > 0
        LIMIT @countLimit::BIGINT
    ) u
), abandoned AS (
    SELECT COUNT(*) AS n, COALESCE(SUM(o.endpoint_count), 0) AS w
    FROM (
        SELECT a.endpoint_count
        FROM v1_serverless_process p
        CROSS JOIN LATERAL (
            SELECT l.endpoint_count
            FROM v1_serverless_lease l
            WHERE l.process_id = p.process_id AND l.endpoint_count > 0
            LIMIT @countLimit::BIGINT
        ) a
        WHERE p.expires_at < now()
        LIMIT @countLimit::BIGINT
    ) o
)
SELECT
    (unowned.n + abandoned.n)::BIGINT AS unit_count,
    (unowned.w + abandoned.w)::BIGINT AS endpoint_count
FROM unowned, abandoned;

-- name: IncrementServerlessLeaseEndpointCount :exec
UPDATE v1_serverless_lease
SET endpoint_count = endpoint_count + @delta::INT
WHERE
    tenant_id = @tenantId::UUID
    AND shard = @shard::INT;
