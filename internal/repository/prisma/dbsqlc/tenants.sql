-- name: ListTenants :many
SELECT
    *
FROM
    "Tenant" as tenants;

-- name: GetTenantByID :one
SELECT
    *
FROM
    "Tenant" as tenants
WHERE
    "id" = sqlc.arg('id')::uuid;

-- name: GetTenantAlertingSettings :one
SELECT
    *
FROM
    "TenantAlertingSettings" as tenantAlertingSettings
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid;

-- name: GetSlackWebhooks :many
SELECT
    *
FROM
    "SlackAppWebhook" as slackWebhooks
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid;

-- name: GetEmailGroups :many
SELECT
    *
FROM
    "TenantAlertEmailGroup" as emailGroups
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid;

-- name: GetMemberEmailGroup :many
SELECT u."email"
FROM "User" u
JOIN "TenantMember" tm ON u."id" = tm."userId"
WHERE u."emailVerified" = true
AND tm."tenantId" = sqlc.arg('tenantId')::uuid;

-- name: UpdateTenantAlertingSettings :one
UPDATE
    "TenantAlertingSettings" as tenantAlertingSettings
SET
    "lastAlertedAt" = COALESCE(sqlc.narg('lastAlertedAt')::timestamp, "lastAlertedAt")
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid
RETURNING *;
