-- name: GetAllWebhookWorkers :many
SELECT
    *
FROM
    "WebhookWorker"
WHERE
    "tenantId" = @tenantId::uuid;
