-- name: UpsertMessageQueue :one
INSERT INTO
    "MessageQueue" (
        "name",
        "durable",
        "autoDeleted",
        "exclusive",
        "exclusiveConsumerId"
    )
VALUES (
    @name::text,
    @durable::boolean,
    @autoDeleted::boolean,
    @exclusive::boolean,
    CASE WHEN sqlc.narg('exclusiveConsumerId')::uuid IS NOT NULL THEN sqlc.narg('exclusiveConsumerId')::uuid ELSE NULL END
) ON CONFLICT ("name") DO UPDATE
SET
    "durable" = @durable::boolean,
    "autoDeleted" = @autoDeleted::boolean,
    "exclusive" = @exclusive::boolean,
    "exclusiveConsumerId" = CASE WHEN sqlc.narg('exclusiveConsumerId')::uuid IS NOT NULL THEN sqlc.narg('exclusiveConsumerId')::uuid ELSE NULL END
RETURNING *;

-- name: UpdateMessageQueueActive :exec
UPDATE
    "MessageQueue"
SET
    "lastActive" = NOW()
WHERE
    "name" = @name::text;

-- name: CleanupMessageQueue :exec
DELETE FROM
    "MessageQueue"
WHERE
    "lastActive" < NOW() - INTERVAL '1 hour'
    AND "autoDeleted" = true;

-- name: SendMessage :exec
INSERT INTO
    "MessageQueueItem" (
        "payload",
        "queueId",
        "readAfter",
        "expiresAt"
    )
VALUES
    (
        @payload::jsonb,
        @queueId::text,
        NOW(),
        NOW() + INTERVAL '5 minutes'
    );

-- name: BulkSendMessage :copyfrom
INSERT INTO
    "MessageQueueItem" (
        "payload",
        "queueId",
        "readAfter",
        "expiresAt"
    )
VALUES (
    $1,
    $2,
    $3,
    $4
);

-- name: ReadMessages :many
WITH messages AS (
    SELECT
        *
    FROM
        "MessageQueueItem"
    WHERE
        "expiresAt" > NOW()
        AND "queueId" = @queueId::text
        AND "readAfter" <= NOW()
        AND "status" = 'PENDING'
    ORDER BY
        "id" ASC
    LIMIT
        COALESCE(sqlc.narg('limit')::integer, 1000)
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "MessageQueueItem"
SET
    "status" = 'ASSIGNED'
FROM
    messages
WHERE
    "MessageQueueItem"."id" = messages."id"
RETURNING messages.*;

-- name: BulkAckMessages :exec
DELETE FROM
    "MessageQueueItem"
WHERE
    "id" = ANY(@ids::bigint[])
    AND "status" = 'ASSIGNED';

-- name: DeleteExpiredMessages :exec
DELETE FROM
    "MessageQueueItem"
WHERE
    "expiresAt" < NOW();

-- name: GetMinMaxExpiredMessageQueueItems :one
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId",
    COALESCE(MAX("id"), 0)::bigint AS "maxId"
FROM
    "MessageQueueItem"
WHERE
    "expiresAt" < NOW();

-- name: CleanupMessageQueueItems :exec
DELETE FROM "MessageQueueItem"
WHERE "expiresAt" < NOW()
AND
    "id" >= @minId::bigint
    AND "id" <= @maxId::bigint;
