-- name: GetTenantSubscription :one
SELECT
  "tenantId",
  "planCode",
  "status"
FROM
  "TenantSubscription"
WHERE
  "tenantId" = @tenantId::uuid
;


-- name: UpsertTenantSubscription :one
INSERT INTO "TenantSubscription" (
  "tenantId",
  "planCode",
  "status"
)
VALUES (
  @tenantId::uuid,
  sqlc.narg('planCode')::text,
  sqlc.narg('status')::"TenantSubscriptionStatus"
)
ON CONFLICT (tenantId) DO UPDATE SET
  "planCode" = EXCLUDED.planCode,
  "status" = EXCLUDED.status
RETURNING *;
