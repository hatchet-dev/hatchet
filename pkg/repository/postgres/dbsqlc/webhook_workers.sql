-- name: ListWebhookWorkersByPartitionId :many
WITH tenants AS (
    SELECT
        "id"
    FROM
        "Tenant"
    WHERE
        "workerPartitionId" = sqlc.arg('workerPartitionId')::text OR
        "workerPartitionId" IS NULL
), update_partition AS (
    UPDATE
        "TenantWorkerPartition"
    SET
        "lastHeartbeat" = NOW()
    WHERE
        "id" = sqlc.arg('workerPartitionId')::text
)
SELECT *
FROM "WebhookWorker"
WHERE "tenantId" IN (SELECT "id" FROM tenants);

-- name: ListActiveWebhookWorkers :many
SELECT *
FROM "WebhookWorker"
WHERE "tenantId" = @tenantId::uuid AND "deleted" = false;

-- name: ListWebhookWorkerRequests :many
SELECT *
FROM "WebhookWorkerRequest"
WHERE "webhookWorkerId" = @webhookWorkerId::uuid
ORDER BY "createdAt" DESC
LIMIT 50;

-- name: InsertWebhookWorkerRequest :exec
WITH delete_old AS (
    -- Delete old requests
    DELETE FROM "WebhookWorkerRequest"
    WHERE "webhookWorkerId" = @webhookWorkerId::uuid
    AND "createdAt" < NOW() - INTERVAL '15 minutes'
)
INSERT INTO "WebhookWorkerRequest" (
    "id",
    "createdAt",
    "webhookWorkerId",
    "method",
    "statusCode"
) VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    @webhookWorkerId::uuid,
    @method::"WebhookWorkerRequestMethod",
    @statusCode::integer
);

-- name: UpdateWebhookWorkerToken :one
UPDATE "WebhookWorker"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "tokenValue" = COALESCE(sqlc.narg('tokenValue')::text, "tokenValue"),
    "tokenId" = COALESCE(sqlc.narg('tokenId')::uuid, "tokenId")
WHERE
    "id" = @id::uuid
    AND "tenantId" = @tenantId::uuid
RETURNING *;

-- name: CreateWebhookWorker :one
INSERT INTO "WebhookWorker" (
    "id",
    "createdAt",
    "updatedAt",
    "name",
    "secret",
    "url",
    "tenantId",
    "tokenId",
    "tokenValue",
    "deleted"
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
    sqlc.narg('tokenValue')::text,
    coalesce(sqlc.narg('deleted')::boolean, false)
)
RETURNING *;

-- name: GetWebhookWorkerByID :one
SELECT *
FROM "WebhookWorker"
WHERE "id" = @id::uuid;

-- name: SoftDeleteWebhookWorker :exec
UPDATE "WebhookWorker"
SET
  "deleted" = true,
  "updatedAt" = CURRENT_TIMESTAMP
WHERE
  "id" = @id::uuid
  AND "tenantId" = @tenantId::uuid;

-- name: HardDeleteWebhookWorker :exec
DELETE FROM "WebhookWorker"
WHERE
  "id" = @id::uuid
  AND "tenantId" = @tenantId::uuid;
