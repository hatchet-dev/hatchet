-- name: CreateUser :one
INSERT INTO "User" (
    "id",
    "email",
    "emailVerified",
    "name"
) VALUES (
    @id::uuid,
    @email::text,
    COALESCE(sqlc.narg('emailVerified')::boolean, FALSE),
    sqlc.narg('name')::text
) RETURNING *;

-- name: CreateUserPassword :one
INSERT INTO "UserPassword" (
    "userId",
    "hash"
) VALUES (
    @userId::uuid,
    @hash::text
) RETURNING *;

-- name: CreateUserOAuth :one
INSERT INTO "UserOAuth" (
    "id",
    "userId",
    "provider",
    "providerUserId",
    "accessToken",
    "refreshToken",
    "expiresAt"
) VALUES (
    gen_random_uuid(),
    @userId::uuid,
    @provider::text,
    @providerUserId::text,
    @accessToken::bytea,
    sqlc.narg('refreshToken')::bytea,
    sqlc.narg('expiresAt')::timestamp
) RETURNING *;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    "User"
WHERE
    "email" = @email::text;

-- name: GetUserByID :one
SELECT
    *
FROM
    "User"
WHERE
    "id" = @id::uuid;

-- name: GetUserPassword :one
SELECT
    *
FROM
    "UserPassword"
WHERE
    "userId" = @userId::uuid;

-- name: UpdateUser :one
UPDATE
    "User"
SET
    "emailVerified" = COALESCE(sqlc.narg('emailVerified')::boolean, "emailVerified"),
    "name" = COALESCE(sqlc.narg('name')::text, "name")
WHERE
    "id" = @id::uuid
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE
    "UserPassword"
SET
    "hash" = @hash::text
WHERE
    "userId" = @userId::uuid
RETURNING *;

-- name: UpsertUserOAuth :one
INSERT INTO "UserOAuth" (
    "id",
    "userId",
    "provider",
    "providerUserId",
    "accessToken",
    "refreshToken",
    "expiresAt"
) VALUES (
    gen_random_uuid(),
    @userId::uuid,
    @provider::text,
    @providerUserId::text,
    @accessToken::bytea,
    sqlc.narg('refreshToken')::bytea,
    sqlc.narg('expiresAt')::timestamp
) ON CONFLICT ("userId", "provider") DO UPDATE SET
    "providerUserId" = @providerUserId::text,
    "accessToken" = @accessToken::bytea,
    "refreshToken" = sqlc.narg('refreshToken')::bytea,
    "expiresAt" = sqlc.narg('expiresAt')::timestamp
RETURNING *;

-- name: ListTenantMemberships :many
SELECT
    *
FROM
    "TenantMember"
WHERE
    "userId" = @userId::uuid;

-- name: CreateUserSession :one
INSERT INTO "UserSession" (
    "id",
    "expiresAt",
    "userId",
    "data"
) VALUES (
    @id::uuid,
    @expiresAt::timestamp,
    sqlc.narg('userId')::uuid,
    @data::jsonb
) RETURNING *;

-- name: UpdateUserSession :one
UPDATE
    "UserSession"
SET
    "userId" = COALESCE(sqlc.narg('userId')::uuid, "userId"),
    "data" = COALESCE(sqlc.narg('data')::jsonb, "data")
WHERE
    "id" = @id::uuid
RETURNING *;

-- name: GetUserSession :one
SELECT
    *
FROM
    "UserSession"
WHERE
    "id" = @id::uuid;

-- name: DeleteUserSession :one
DELETE FROM
    "UserSession"
WHERE
    "id" = @id::uuid
RETURNING *;
