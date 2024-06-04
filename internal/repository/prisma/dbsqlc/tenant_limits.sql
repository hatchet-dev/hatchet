-- name: GetTenantResourceLimit :one
WITH updated AS (
    UPDATE "TenantResourceLimit"
    SET
        "value" = 0, -- Reset to 0 if the window has passed
        "lastRefill" = CURRENT_TIMESTAMP -- Update lastRefill if the window has passed
    WHERE "tenantId" = @tenantId::uuid
      AND NOW() - "lastRefill" >= "window"::INTERVAL
      AND "resource" = sqlc.narg('resource')::"LimitResource"
    RETURNING *
)
SELECT * FROM updated
UNION ALL
SELECT * FROM "TenantResourceLimit"
WHERE "tenantId" = @tenantId::uuid
    AND "resource" = sqlc.narg('resource')::"LimitResource"
    AND NOT EXISTS (SELECT 1 FROM updated);


-- name: CreateTenantResourceLimit :one
INSERT INTO "TenantResourceLimit" ("id", "tenantId", "resource", "value", "limitValue", "alarmValue", "window", "lastRefill")
VALUES (gen_random_uuid(), @tenantId::uuid, sqlc.narg('resource')::"LimitResource", 0, sqlc.narg('limitValue')::int, sqlc.narg('alarmValue')::int, sqlc.narg('window')::text, CURRENT_TIMESTAMP)
RETURNING *;

-- name: MeterTenantResource :one
UPDATE "TenantResourceLimit"
SET
    "value" = CASE
        WHEN NOW() - "lastRefill" >= "window"::INTERVAL THEN
            0 -- Refill to 0 since the window has passed
        ELSE
            "value" + 1 -- Increment the current value within the window
    END,
    "lastRefill" = CASE
        WHEN NOW() - "lastRefill" >= "window"::INTERVAL THEN
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
