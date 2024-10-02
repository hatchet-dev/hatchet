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

-- name: SelectOrInsertTenantResourceLimit :one
WITH existing AS (
  SELECT *
  FROM "TenantResourceLimit"
  WHERE "tenantId" = @tenantId::uuid AND "resource" = sqlc.narg('resource')::"LimitResource"
)
, insert_row AS (
  INSERT INTO "TenantResourceLimit" ("id", "tenantId", "resource", "value", "limitValue", "alarmValue", "window", "lastRefill", "customValueMeter")
  SELECT gen_random_uuid(), @tenantId::uuid, sqlc.narg('resource')::"LimitResource", 0, sqlc.narg('limitValue')::int, sqlc.narg('alarmValue')::int, sqlc.narg('window')::text, CURRENT_TIMESTAMP, COALESCE(sqlc.narg('customValueMeter')::boolean, false)
  WHERE NOT EXISTS (SELECT 1 FROM existing)
  RETURNING *
)
SELECT * FROM insert_row
UNION ALL
SELECT * FROM existing
LIMIT 1;

-- name: UpsertTenantResourceLimit :one
INSERT INTO "TenantResourceLimit" ("id", "tenantId", "resource", "value", "limitValue", "alarmValue", "window", "lastRefill", "customValueMeter")
VALUES (
  gen_random_uuid(),
  @tenantId::uuid,
  sqlc.narg('resource')::"LimitResource",
  0,
  sqlc.narg('limitValue')::int,
  sqlc.narg('alarmValue')::int,
  sqlc.narg('window')::text,
  CURRENT_TIMESTAMP,
  COALESCE(sqlc.narg('customValueMeter')::boolean, false)
)
ON CONFLICT ("tenantId", "resource") DO UPDATE SET
  "limitValue" = sqlc.narg('limitValue')::int,
  "alarmValue" = sqlc.narg('alarmValue')::int,
  "window" = sqlc.narg('window')::text,
  "customValueMeter" = COALESCE(sqlc.narg('customValueMeter')::boolean, false)
RETURNING *;

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

-- name: CountTenantWorkers :one
SELECT COUNT(distinct id) AS "count"
FROM "Worker"
WHERE "tenantId" = @tenantId::uuid
AND "lastHeartbeatAt" >= NOW() - '30 seconds'::INTERVAL
AND "isActive" = true;
