-- name: GetSNSIntegration :one
SELECT
    *
FROM
    "SNSIntegration"
WHERE
    "tenantId" = @tenantId::uuid
    AND "topicArn" = @topicArn::text;

-- name: GetSNSIntegrationById :one
SELECT
    *
FROM
    "SNSIntegration"
WHERE
    "id" = @id::uuid;

-- name: CreateSNSIntegration :one
INSERT INTO "SNSIntegration" (
    "id",
    "tenantId",
    "topicArn"
) VALUES (
    gen_random_uuid(),
    @tenantId::uuid,
    @topicArn::text
) ON CONFLICT ("tenantId", "topicArn") DO NOTHING 
RETURNING *;

-- name: ListSNSIntegrations :many
SELECT
    *
FROM
    "SNSIntegration"
WHERE
    "tenantId" = @tenantId::uuid;

-- name: DeleteSNSIntegration :exec
DELETE FROM
    "SNSIntegration"
WHERE
    "tenantId" = @tenantId::uuid
    AND "id" = @id::uuid;