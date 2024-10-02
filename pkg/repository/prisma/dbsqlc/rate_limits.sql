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

-- name: UpsertRateLimitsBulk :exec
WITH input_values AS (
    SELECT
        unnest(@keys::text[]) AS "key",
        unnest(@limitValues::int[]) AS "limitValue",
        unnest(@windows::text[]) AS "window"
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
    "value" = CASE WHEN EXCLUDED."limitValue" < "RateLimit"."value" THEN EXCLUDED."limitValue" ELSE "RateLimit"."value" END;

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

-- name: ListRateLimitsForTenantNoMutate :many
-- Returns the same results as ListRateLimitsForTenantWithMutate but does not update the rate limit values
SELECT
    "tenantId",
    "key",
    "limitValue",
    (CASE
        WHEN NOW() - rl."lastRefill" >= rl."window"::INTERVAL THEN
            get_refill_value(rl)
        ELSE
            rl."value"
    END)::int AS "value",
    "window",
    (CASE
        WHEN NOW() - rl."lastRefill" >= rl."window"::INTERVAL THEN
            CURRENT_TIMESTAMP
        ELSE
            rl."lastRefill"
    END)::timestamp AS "lastRefill"
FROM
    "RateLimit" rl
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
WITH refill AS (
    UPDATE
        "RateLimit" rl
    SET
        "value" = CASE
            WHEN NOW() - rl."lastRefill" >= rl."window"::INTERVAL THEN
                get_refill_value(rl)
            ELSE
                rl."value"
        END,
        "lastRefill" = CASE
            WHEN NOW() - rl."lastRefill" >= rl."window"::INTERVAL THEN
                CURRENT_TIMESTAMP
            ELSE
                rl."lastRefill"
        END
    WHERE
        rl."tenantId" = @tenantId::uuid
    RETURNING *
)
SELECT
    refill.*,
    -- return the next refill time
    (refill."lastRefill" + refill."window"::INTERVAL)::timestamp AS "nextRefillAt"
FROM
    refill;

-- name: ListRateLimitsForSteps :many
SELECT
    *
FROM
    "StepRateLimit" srl
WHERE
    srl."stepId" = ANY(@stepIds::uuid[])
    AND srl."tenantId" = @tenantId::uuid;

-- name: BulkUpdateRateLimits :many
UPDATE
    "RateLimit" rl
SET
    "value" = get_refill_value(rl) - input."units",
    "lastRefill" = CASE
        WHEN NOW() - rl."lastRefill" >= rl."window"::INTERVAL THEN
            CURRENT_TIMESTAMP
        ELSE
            rl."lastRefill"
    END
FROM
    (
        SELECT
            unnest(@keys::text[]) AS "key",
            unnest(@units::int[]) AS "units"
    ) AS input
WHERE
    rl."key" = input."key"
    AND rl."tenantId" = @tenantId::uuid
RETURNING rl.*;
