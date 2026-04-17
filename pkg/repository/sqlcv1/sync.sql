-- name: SyncUpsertTenant :one
INSERT INTO "Tenant" (
    "id",
    "createdAt",
    "updatedAt",
    "name",
    "slug",
    "dataRetentionPeriod",
    "version",
    "uiVersion",
    "onboardingData",
    "environment",
    "analyticsOptOut",
    "alertMemberEmails",
    "canUpgradeV1"
) VALUES (
    @id::uuid,
    @createdAt::timestamp,
    @updatedAt::timestamp,
    @name::text,
    @slug::text,
    @dataRetentionPeriod::text,
    @version::"TenantMajorEngineVersion",
    @uiVersion::"TenantMajorUIVersion",
    sqlc.narg('onboardingData')::jsonb,
    sqlc.narg('environment')::"TenantEnvironment",
    @analyticsOptOut::boolean,
    @alertMemberEmails::boolean,
    @canUpgradeV1::boolean
) ON CONFLICT ("id") DO UPDATE SET
    "updatedAt" = @updatedAt::timestamp,
    "name" = @name::text,
    "slug" = @slug::text,
    "dataRetentionPeriod" = @dataRetentionPeriod::text,
    "version" = @version::"TenantMajorEngineVersion",
    "uiVersion" = @uiVersion::"TenantMajorUIVersion",
    "onboardingData" = sqlc.narg('onboardingData')::jsonb,
    "environment" = sqlc.narg('environment')::"TenantEnvironment",
    "analyticsOptOut" = @analyticsOptOut::boolean,
    "alertMemberEmails" = @alertMemberEmails::boolean,
    "canUpgradeV1" = @canUpgradeV1::boolean
RETURNING *;

-- name: SyncUpdateTenant :one
UPDATE "Tenant"
SET
    "updatedAt" = @updatedAt::timestamp,
    "name" = COALESCE(sqlc.narg('name')::text, "name"),
    "analyticsOptOut" = COALESCE(sqlc.narg('analyticsOptOut')::boolean, "analyticsOptOut"),
    "alertMemberEmails" = COALESCE(sqlc.narg('alertMemberEmails')::boolean, "alertMemberEmails"),
    "version" = COALESCE(sqlc.narg('version')::"TenantMajorEngineVersion", "version")
WHERE "id" = @id::uuid
RETURNING *;

-- name: SyncSoftDeleteTenant :exec
UPDATE "Tenant"
SET
    "deletedAt" = @deletedAt::timestamp,
    "slug" = @slug::text
WHERE "id" = @id::uuid;

-- name: SyncUpsertUser :one
INSERT INTO "User" (
    "id",
    "createdAt",
    "updatedAt",
    "email",
    "emailVerified",
    "name"
) VALUES (
    @id::uuid,
    @createdAt::timestamp,
    @updatedAt::timestamp,
    @email::text,
    @emailVerified::boolean,
    sqlc.narg('name')::text
) ON CONFLICT ("id") DO UPDATE SET
    "updatedAt" = @updatedAt::timestamp,
    "email" = @email::text,
    "emailVerified" = @emailVerified::boolean,
    "name" = sqlc.narg('name')::text
RETURNING *;

-- name: SyncUpsertTenantInvite :one
INSERT INTO "TenantInviteLink" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "inviterEmail",
    "inviteeEmail",
    "expires",
    "role",
    "status"
) VALUES (
    @id::uuid,
    @createdAt::timestamp,
    @updatedAt::timestamp,
    @tenantId::uuid,
    @inviterEmail::text,
    @inviteeEmail::text,
    @expires::timestamp,
    @role::"TenantMemberRole",
    @status::"InviteLinkStatus"
) ON CONFLICT ("id") DO UPDATE SET
    "updatedAt" = @updatedAt::timestamp,
    "inviterEmail" = @inviterEmail::text,
    "inviteeEmail" = @inviteeEmail::text,
    "expires" = @expires::timestamp,
    "role" = @role::"TenantMemberRole",
    "status" = @status::"InviteLinkStatus"
RETURNING *;

-- name: SyncUpdateTenantInvite :one
UPDATE "TenantInviteLink"
SET
    "updatedAt" = @updatedAt::timestamp,
    "status" = COALESCE(sqlc.narg('status')::"InviteLinkStatus", "status"),
    "role" = COALESCE(sqlc.narg('role')::"TenantMemberRole", "role")
WHERE "id" = @id::uuid
RETURNING *;

-- name: SyncUpsertTenantMember :one
INSERT INTO "TenantMember" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "userId",
    "role"
) VALUES (
    @id::uuid,
    @createdAt::timestamp,
    @updatedAt::timestamp,
    @tenantId::uuid,
    @userId::uuid,
    @role::"TenantMemberRole"
) ON CONFLICT ("id") DO UPDATE SET
    "updatedAt" = @updatedAt::timestamp,
    "role" = @role::"TenantMemberRole"
RETURNING *;

-- name: SyncUpdateTenantMember :one
UPDATE "TenantMember"
SET
    "updatedAt" = @updatedAt::timestamp,
    "role" = @role::"TenantMemberRole"
WHERE "id" = @id::uuid
RETURNING *;

-- name: SyncUpsertSlackWebhook :one
INSERT INTO "SlackAppWebhook" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "teamId",
    "teamName",
    "channelId",
    "channelName",
    "webhookURL"
) VALUES (
    @id::uuid,
    @createdAt::timestamp,
    @updatedAt::timestamp,
    @tenantId::uuid,
    @teamId::text,
    @teamName::text,
    @channelId::text,
    @channelName::text,
    @webhookURL::bytea
) ON CONFLICT ("id") DO UPDATE SET
    "updatedAt" = @updatedAt::timestamp,
    "teamName" = @teamName::text,
    "channelName" = @channelName::text,
    "webhookURL" = @webhookURL::bytea
RETURNING *;
