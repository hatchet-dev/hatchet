-- name: UpsertSlackWebhook :one
INSERT INTO
    "SlackAppWebhook" (
        "id",
        "tenantId",
        "teamId",
        "teamName",
        "channelId",
        "channelName",
        "webhookURL"
    ) VALUES (
        gen_random_uuid(),
        @tenantId::uuid,
        @teamId::text,
        @teamName::text,
        @channelId::text,
        @channelName::text,
        @webhookURL::bytea
    ) ON CONFLICT ("tenantId", "teamId", "channelId") DO UPDATE SET
        "updatedAt" = CURRENT_TIMESTAMP,
        "teamName" = @teamName::text,
        "channelName" = @channelName::text,
        "webhookURL" = @webhookURL::bytea
    RETURNING *;

-- name: ListSlackWebhooks :many
SELECT
    *
FROM
    "SlackAppWebhook"
WHERE
    "tenantId" = @tenantId::uuid;

-- name: GetSlackWebhookById :one
SELECT
    *
FROM
    "SlackAppWebhook"
WHERE
    "id" = @id::uuid;

-- name: DeleteSlackWebhook :exec
DELETE FROM
    "SlackAppWebhook"
WHERE
    "tenantId" = @tenantId::uuid
    AND "id" = @id::uuid;
