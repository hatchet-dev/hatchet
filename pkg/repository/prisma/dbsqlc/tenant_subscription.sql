-- name: GetTenantSubscription :one
SELECT
  *
FROM
  "TenantSubscription"
WHERE
  "tenantId" = @tenantId::uuid
;


-- name: UpsertTenantSubscription :one
INSERT INTO "TenantSubscription" (
  "tenantId",
  "plan",
  "period",
  "status",
  "note"
)
VALUES (
  @tenantId::uuid,
  sqlc.narg('plan')::"TenantSubscriptionPlanCodes",
  sqlc.narg('period')::"TenantSubscriptionPeriod",
  sqlc.narg('status')::"TenantSubscriptionStatus",
  sqlc.narg('note')::text
)
ON CONFLICT ("tenantId") DO UPDATE SET
  "plan" = sqlc.narg('plan')::"TenantSubscriptionPlanCodes",
  "period" = sqlc.narg('period')::"TenantSubscriptionPeriod",
  "status" = sqlc.narg('status')::"TenantSubscriptionStatus",
  "note" = sqlc.narg('note')::text
RETURNING *;
