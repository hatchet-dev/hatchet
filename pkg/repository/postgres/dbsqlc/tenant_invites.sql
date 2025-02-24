-- name: CreateTenantInvite :one
INSERT INTO "TenantInviteLink" (
    "id",
    "tenantId",
    "inviterEmail",
    "inviteeEmail",
    "expires",
    "role"
) VALUES (
    gen_random_uuid(),
    @tenantId::uuid,
    @inviterEmail::text,
    @inviteeEmail::text,
    @expires::timestamp,
    @role::"TenantMemberRole"
) RETURNING *;

-- name: CountActiveInvites :one
SELECT
    COUNT(*)
FROM
    "TenantInviteLink"
WHERE
    "tenantId" = @tenantId::uuid
    AND "status" = 'PENDING'
    AND "expires" > now();

-- name: GetExistingInvite :one
SELECT
    *
FROM
    "TenantInviteLink"
WHERE
    "inviteeEmail" = @inviteeEmail::text
    AND "tenantId" = @tenantId::uuid
    AND "status" = 'PENDING'
    AND "expires" > now();

-- name: GetInviteById :one
SELECT
    *
FROM
    "TenantInviteLink"
WHERE
    "id" = @id::uuid;

-- name: ListTenantInvitesByEmail :many
SELECT
    *
FROM
    "TenantInviteLink"
WHERE
    "inviteeEmail" = @inviteeEmail::text
    AND "status" = 'PENDING'
    AND "expires" > now();

-- name: ListInvitesByTenantId :many
SELECT
    *
FROM
    "TenantInviteLink"
WHERE
    "tenantId" = @tenantId::uuid
    AND (
        sqlc.narg('status')::"InviteLinkStatus" IS NULL
        OR "status" = sqlc.narg('status')::"InviteLinkStatus"
    )
    AND (
        CASE WHEN sqlc.narg('expired')::boolean IS NULL THEN TRUE
        -- otherwise, if expired is true, return only expired invites, and vice versa
        ELSE sqlc.narg('expired')::boolean = ("expires" < now())
        END
    )
    AND (
        sqlc.narg('expired')::boolean IS NULL
        OR "expires" < now()
    );

-- name: UpdateTenantInvite :one
UPDATE
    "TenantInviteLink"
SET
    "status" = COALESCE(sqlc.narg('status')::"InviteLinkStatus", "status"),
    "role" = COALESCE(sqlc.narg('role')::"TenantMemberRole", "role")
WHERE
    "id" = @id::uuid
RETURNING *;

-- name: DeleteTenantInvite :exec
DELETE FROM
    "TenantInviteLink"
WHERE
    "id" = @id::uuid;
