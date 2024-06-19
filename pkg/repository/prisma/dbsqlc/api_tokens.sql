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
    "expiresAt"
) VALUES (
    coalesce(@id::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    sqlc.narg('tenantId')::uuid,
    sqlc.narg('name')::text,
    @expiresAt::timestamp
) RETURNING *;

-- name: UpsertAPIToken :one
INSERT INTO "APIToken" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "name",
    "expiresAt"
) VALUES (
    @id::uuid,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    sqlc.narg('tenantId')::uuid,
    sqlc.narg('name')::text,
    @expiresAt::timestamp
)
