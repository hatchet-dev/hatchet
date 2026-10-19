-- name: UpsertRateLimit :one
INSERT INTO "RateLimit" (
    "tenantId",
    "key",
    "limitValue",
    "value",
    "window"
) VALUES (
    @tenantId::uuid,
    @key::text,
    sqlc.arg('limit')::int,
    sqlc.arg('limit')::int,
    COALESCE(sqlc.narg('window')::text, '1 minute')
) ON CONFLICT ("tenantId", "key") DO UPDATE SET
    "limitValue" = sqlc.arg('limit')::int,
    "window" = COALESCE(sqlc.narg('window')::text, '1 minute'),
    "value" = CASE WHEN EXCLUDED."limitValue" < "RateLimit"."value" THEN EXCLUDED."limitValue" ELSE "RateLimit"."value" END
RETURNING *;

-- name: UpsertDynamicRateLimitDefinitions :exec
WITH input_values AS (
    SELECT
        "key", "limitValue", "window"
    FROM
        (
            SELECT
                unnest(@keys::text[]) AS "key",
                unnest(@limitValues::int[]) AS "limitValue",
                unnest(@windows::text[]) AS "window"
        ) AS subquery
    ORDER BY
        "key"
)
INSERT INTO "RateLimit" (
    "tenantId",
    "key",
    "limitValue",
    "value",
    "window"
)
SELECT
    @tenantId::uuid,
    iv."key",
    iv."limitValue",
    iv."limitValue",
    iv."window"
FROM
    input_values iv
ON CONFLICT ("tenantId", "key") DO UPDATE SET
    "limitValue" = EXCLUDED."limitValue",
    "window" = EXCLUDED."window",
    "value" = CASE WHEN EXCLUDED."limitValue" < "RateLimit"."value" THEN EXCLUDED."limitValue" ELSE "RateLimit"."value" END
-- skip rewriting rows whose definition hasn't changed
WHERE
    ("RateLimit"."limitValue", "RateLimit"."window") IS DISTINCT FROM (EXCLUDED."limitValue", EXCLUDED."window");

-- name: CountRateLimits :one
WITH rate_limits AS (
    SELECT
        rl."key"
    FROM
        "RateLimit" rl
    WHERE
        rl."tenantId" = @tenantId::uuid
        AND (
            sqlc.narg('search')::text IS NULL OR
            rl."key" like concat('%', sqlc.narg('search')::text, '%')
        )
    ORDER BY
        case when @orderBy = 'key ASC' THEN rl."key" END ASC,
        case when @orderBy = 'key DESC' THEN rl."key" END DESC,
        case when @orderBy = 'value ASC' THEN rl."value" END ASC,
        case when @orderBy = 'value DESC' THEN rl."value" END DESC,
        case when @orderBy = 'limitValue ASC' THEN rl."limitValue" END ASC,
        case when @orderBy = 'limitValue DESC' THEN rl."limitValue" END DESC,
        rl."key" ASC
    LIMIT 10000
)
SELECT
    count(rate_limits) AS total
FROM
    rate_limits;

-- name: DeleteRateLimitForTenant :exec
DELETE FROM "RateLimit" rl WHERE
    rl."tenantId" = @tenantId::uuid AND rl."key" = @key;

-- name: ListRateLimitsForTenantNoMutate :many
-- Returns the same results as ListRateLimitsForTenantWithMutate but does not update the rate limit values
SELECT
    rl."tenantId",
    rl."key",
    rl."limitValue",
    (CASE
        WHEN NOW() - rl."lastRefill" >= rl."window"::INTERVAL AND p.key IS NULL THEN
            get_refill_value(rl)
        ELSE
            rl."value"
    END)::int AS "value",
    rl."window",
    (CASE
        WHEN NOW() - rl."lastRefill" >= rl."window"::INTERVAL AND p.key IS NULL THEN
            CURRENT_TIMESTAMP
        ELSE
            rl."lastRefill"
    END)::timestamp AS "lastRefill"
FROM
    "RateLimit" rl
LEFT JOIN
    v1_rate_limit_pause p ON p.tenant_id = rl."tenantId" AND p.key = rl."key" AND p.paused_until > NOW()
WHERE
    "tenantId" = @tenantId::uuid
    AND (
        sqlc.narg('search')::text IS NULL OR
        rl."key" like concat('%', sqlc.narg('search')::text, '%')
    )
ORDER BY
    case when @orderBy = 'key ASC' THEN rl."key" END ASC,
    case when @orderBy = 'key DESC' THEN rl."key" END DESC,
    case when @orderBy = 'value ASC' THEN rl."value" END ASC,
    case when @orderBy = 'value DESC' THEN rl."value" END DESC,
    case when @orderBy = 'limitValue ASC' THEN rl."limitValue" END ASC,
    case when @orderBy = 'limitValue DESC' THEN rl."limitValue" END DESC,
    rl."key" ASC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: ListRateLimitsForTenantWithMutate :many
WITH rls_to_update AS (
    SELECT
        rl.*
    FROM
        "RateLimit" rl
    WHERE
        rl."tenantId" = @tenantId::uuid
        AND NOW() - rl."lastRefill" >= (rl."window"::INTERVAL - INTERVAL '10 milliseconds')
        -- a paused key keeps the units it had left when the pause started
        AND NOT EXISTS (
            SELECT 1
            FROM v1_rate_limit_pause p
            WHERE p.tenant_id = rl."tenantId" AND p.key = rl."key" AND p.paused_until > NOW()
        )
    ORDER BY
        rl."tenantId" ASC, rl."key" ASC
    FOR UPDATE
), refill AS (
    UPDATE
        "RateLimit" rl
    SET
        "value" = get_refill_value(rl),
        "lastRefill" = CURRENT_TIMESTAMP
    FROM
        rls_to_update
    WHERE
        rl."tenantId" = rls_to_update."tenantId"
        AND rl."key" = rls_to_update."key"
    RETURNING rl.*
)
SELECT
    rl.*,
    (rl."lastRefill" + rl."window"::INTERVAL - INTERVAL '10 milliseconds')::timestamp AS "nextRefillAt",
    p.paused_until AS "pausedUntil"
FROM
    "RateLimit" rl
LEFT JOIN
    v1_rate_limit_pause p ON p.tenant_id = rl."tenantId" AND p.key = rl."key" AND p.paused_until > NOW()
WHERE
    rl."tenantId" = @tenantId::uuid
    AND rl."key" NOT IN (SELECT "key" FROM refill)

UNION ALL

SELECT
    refill.*,
    -- return the next refill time
    (refill."lastRefill" + refill."window"::INTERVAL - INTERVAL '10 milliseconds')::timestamp AS "nextRefillAt",
    -- paused keys are never refilled
    NULL::timestamptz AS "pausedUntil"
FROM
    refill;

-- name: SettleExpiredRateLimitPauses :exec
WITH expired AS (
    DELETE FROM v1_rate_limit_pause
    WHERE
        tenant_id = @tenantId::uuid
        AND paused_until <= NOW()
    RETURNING key, paused_at, paused_until
)
UPDATE
    "RateLimit" rl
SET
    -- the window resumes where it was when the pause started
    "lastRefill" = rl."lastRefill" + (e.paused_until - GREATEST(e.paused_at, rl."lastRefill"::timestamptz))
FROM
    expired e
WHERE
    rl."tenantId" = @tenantId::uuid
    AND rl."key" = e.key
    AND e.paused_until > rl."lastRefill"::timestamptz;

-- name: HasActiveRateLimitPause :one
SELECT EXISTS (
    SELECT 1
    FROM v1_rate_limit_pause
    WHERE
        tenant_id = @tenantId::uuid
        AND (sqlc.narg('key')::text IS NULL OR key = sqlc.narg('key')::text)
        AND paused_until > NOW()
)::boolean;

-- name: PauseRateLimit :one
INSERT INTO v1_rate_limit_pause (
    tenant_id,
    key,
    paused_until
) VALUES (
    @tenantId::uuid,
    @key::text,
    @pausedUntil::timestamptz
) ON CONFLICT (tenant_id, key) DO UPDATE SET
    -- an earlier deadline never shortens an active pause
    paused_until = GREATEST(v1_rate_limit_pause.paused_until, EXCLUDED.paused_until)
RETURNING *;

-- name: ListRateLimitsForSteps :many
SELECT
    *
FROM
    "StepRateLimit" srl
WHERE
    srl."stepId" = ANY(@stepIds::uuid[])
    AND srl."tenantId" = @tenantId::uuid;

-- name: ChargeRateLimitUsage :many
WITH input AS (
    SELECT
        "key", "units"
    FROM
        (
            SELECT
                unnest(@keys::text[]) AS "key",
                unnest(@units::int[]) AS "units"
        ) AS subquery
), rls_to_update AS (
    SELECT
        rl.*
    FROM
        "RateLimit" rl
    WHERE
        rl."tenantId" = @tenantId::uuid
        AND rl."key" = ANY(SELECT "key" FROM input)
    ORDER BY
        rl."tenantId" ASC, rl."key" ASC
    FOR UPDATE
)
UPDATE
    "RateLimit" rl
SET
    -- units are charged to the window they were spent in; ListRateLimitsForTenantWithMutate refills afterwards
    "value" = rl."value" - (SELECT "units" FROM input WHERE "key" = rl."key")
FROM
    rls_to_update rl2
WHERE
    rl2."tenantId" = rl."tenantId"
    AND rl2."key" = rl."key"
RETURNING rl.*;
