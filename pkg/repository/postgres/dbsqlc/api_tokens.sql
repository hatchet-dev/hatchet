-- name: GetAPITokenById :one
SELECT
    *
FROM
    "APIToken"
WHERE
    "id" = @id::uuid;

-- name: CreateAPIToken :one
INSERT INTO "APIToken" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "name",
    "expiresAt",
    "internal"
) VALUES (
    coalesce(@id::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    sqlc.narg('tenantId')::uuid,
    sqlc.narg('name')::text,
    @expiresAt::timestamp,
    COALESCE(sqlc.narg('internal')::boolean, FALSE)
) RETURNING *;


-- name: RevokeAPIToken :exec
UPDATE
    "APIToken"
SET
    "expiresAt" = CURRENT_TIMESTAMP - INTERVAL '1 second',
    "revoked" = TRUE
WHERE
    "id" = @id::uuid;

-- name: ListAPITokensByTenant :many
SELECT
    *
FROM
    "APIToken"
WHERE
    "tenantId" = @tenantId::uuid
    AND "revoked" = FALSE
    AND "internal" = FALSE;
