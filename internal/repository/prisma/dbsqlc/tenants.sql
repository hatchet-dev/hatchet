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