-- name: ListTenantResourceLimits :many
SELECT * FROM "TenantResourceLimit"
WHERE "tenantId" = @tenantId::uuid;

-- name: ResolveAllLimitsIfWindowPassed :many
WITH resolved_limits AS (
    UPDATE "TenantResourceLimit"
    SET
        "value" = 0, -- Reset value to 0
        "lastRefill" = CURRENT_TIMESTAMP -- Update lastRefill timestamp
    WHERE
        ("window" IS NOT NULL AND "window" != '' AND NOW() - "lastRefill" >= "window"::INTERVAL)
    RETURNING *
)
SELECT *
FROM resolved_limits;

-- name: GetTenantResourceLimit :one
WITH updated AS (
    UPDATE "TenantResourceLimit"
    SET
        "value" = 0, -- Reset to 0 if the window has passed
        "lastRefill" = CURRENT_TIMESTAMP -- Update lastRefill if the window has passed
    WHERE "tenantId" = @tenantId::uuid
      AND (("window" IS NOT NULL AND "window" != '' AND NOW() - "lastRefill" >= "window"::INTERVAL))
      AND "resource" = sqlc.narg('resource')::"LimitResource"
      AND "customValueMeter" = false
    RETURNING *
)
SELECT * FROM updated
UNION ALL
SELECT * FROM "TenantResourceLimit"
WHERE "tenantId" = @tenantId::uuid
    AND "resource" = sqlc.narg('resource')::"LimitResource"
    AND NOT EXISTS (SELECT 1 FROM updated);

-- name: UpsertTenantResourceLimits :exec
WITH input_values AS (
    SELECT
        "resource",
        "limitValue",
        "alarmValue",
        "window",
        "customValueMeter"
    FROM (
        SELECT
            unnest(cast(@resources::text[] AS "LimitResource"[])) AS "resource",
            unnest(@limitValues::int[]) AS "limitValue",
            unnest(@alarmValues::int[]) AS "alarmValue",
            unnest(@windows::text[]) AS "window",
            unnest(@customValueMeters::boolean[]) AS "customValueMeter"
    ) AS subquery
)
INSERT INTO "TenantResourceLimit" ("id", "tenantId", "resource", "value", "limitValue", "alarmValue", "window", "customValueMeter", "lastRefill")
SELECT
    gen_random_uuid(),
    @tenantId::uuid,
    iv."resource",
    0,
    iv."limitValue",
    NULLIF(iv."alarmValue", 0),
    NULLIF(iv."window", ''),
    iv."customValueMeter",
    CURRENT_TIMESTAMP
FROM input_values iv
ON CONFLICT ("tenantId", "resource") DO UPDATE SET
    "limitValue" = EXCLUDED."limitValue",
    "alarmValue" = EXCLUDED."alarmValue",
    "updatedAt" = CURRENT_TIMESTAMP;

-- name: MeterTenantResource :one
UPDATE "TenantResourceLimit"
SET
    "value" = CASE
        WHEN ("customValueMeter" = true OR ("window" IS NOT NULL AND "window" != '' AND NOW() - "lastRefill" >= "window"::INTERVAL)) THEN
            0 -- Refill to 0 since the window has passed
        ELSE
            "value" + @numResources::int -- Increment the current value within the window by the number of resources
    END,
    "lastRefill" = CASE
        WHEN ("window" IS NOT NULL AND "window" != '' AND NOW() - "lastRefill" >= "window"::INTERVAL) THEN
            CURRENT_TIMESTAMP -- Update lastRefill if the window has passed
        ELSE
            "lastRefill" -- Keep the lastRefill unchanged if within the window
    END
WHERE "tenantId" = @tenantId::uuid
    AND "resource" = sqlc.narg('resource')::"LimitResource"
RETURNING *;

-- name: BulkMeterTenantResources :many
WITH input AS (
    SELECT
        unnest(@tenantIds::uuid[]) AS tenant_id,
        unnest(cast(@resources::text[] AS "LimitResource"[])) AS resource,
        unnest(@numResources::int[]) AS num_resources
), to_update AS (
    SELECT
        trl.*,
        i.num_resources
    FROM
        "TenantResourceLimit" trl
    JOIN
        input i ON i.tenant_id = trl."tenantId" AND i.resource = trl."resource"
    ORDER BY
        trl."tenantId", trl."resource"
    FOR UPDATE
)
UPDATE "TenantResourceLimit" trl
SET
    "value" = CASE
        WHEN (trl."customValueMeter" = true OR (trl."window" IS NOT NULL AND trl."window" != '' AND NOW() - trl."lastRefill" >= trl."window"::INTERVAL)) THEN
            0
        ELSE
            trl."value" + tu.num_resources
    END,
    "lastRefill" = CASE
        WHEN (trl."window" IS NOT NULL AND trl."window" != '' AND NOW() - trl."lastRefill" >= trl."window"::INTERVAL) THEN
            CURRENT_TIMESTAMP
        ELSE
            trl."lastRefill"
    END
FROM
    to_update tu
WHERE
    trl."tenantId" = tu."tenantId"
    AND trl."resource" = tu."resource"
RETURNING trl.*;

-- name: CountTenantWorkers :one
SELECT COUNT(distinct id) AS "count"
FROM "Worker"
WHERE "tenantId" = @tenantId::uuid
AND "lastHeartbeatAt" >= NOW() - '30 seconds'::INTERVAL
AND "isActive" = true;
