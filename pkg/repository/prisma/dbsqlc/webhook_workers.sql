-- name: ListWebhookWorkers :many
SELECT *
FROM "WebhookWorker"
WHERE "tenantId" = @tenantId::uuid;

-- name: ListActiveWebhookWorkers :many
SELECT *
FROM "WebhookWorker"
WHERE "tenantId" = @tenantId::uuid AND "deleted" = false;


-- name: UpsertWebhookWorker :one
INSERT INTO "WebhookWorker" (
    "id",
    "createdAt",
    "updatedAt",
    "name",
    "secret",
    "url",
    "tenantId",
    "tokenId",
    "tokenValue"
)
VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @name::text,
    @secret::text,
    @url::text,
    @tenantId::uuid,
    sqlc.narg('tokenId')::uuid,
    sqlc.narg('tokenValue')::text
)
ON CONFLICT ("url") DO
UPDATE
SET
    "tokenId" = coalesce(sqlc.narg('tokenId')::uuid, excluded."tokenId"),
    "tokenValue" = coalesce(sqlc.narg('tokenValue')::text, excluded."tokenValue"),
    "name" = coalesce(sqlc.narg('name')::text, excluded."name"),
    "secret" = coalesce(sqlc.narg('secret')::text, excluded."secret"),
    "url" = coalesce(sqlc.narg('url')::text, excluded."url")
RETURNING *;

-- name: DeleteWebhookWorker :exec
UPDATE "WebhookWorker"
SET "deleted" = true
WHERE
  "id" = @id::uuid
  and "tenantId" = @tenantId::uuid;
