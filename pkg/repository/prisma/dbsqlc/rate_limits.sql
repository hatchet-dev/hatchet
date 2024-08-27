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
    "window" = COALESCE(sqlc.narg('window')::text, '1 minute')
RETURNING *;

-- name: ListRateLimitsForTenant :many
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
