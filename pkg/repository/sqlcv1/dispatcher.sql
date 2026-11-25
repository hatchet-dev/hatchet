-- name: ListStaleDispatchers :many
SELECT
    sqlc.embed(dispatchers)
FROM "Dispatcher" as dispatchers
WHERE
    -- last heartbeat older than 15 seconds
    "lastHeartbeatAt" < NOW () - INTERVAL '15 seconds'
    -- not active
    AND "isActive" = false;

-- name: ListActiveDispatchers :many
SELECT
    sqlc.embed(dispatchers)
FROM "Dispatcher" as dispatchers
WHERE
    -- last heartbeat greater than 15 seconds
    "lastHeartbeatAt" > NOW () - INTERVAL '15 seconds'
    -- active
    AND "isActive" = true;

-- name: SetDispatchersInactive :many
UPDATE
    "Dispatcher" as dispatchers
SET
    "isActive" = false
WHERE
    "id" = ANY (sqlc.arg('ids')::uuid[])
RETURNING
    sqlc.embed(dispatchers);

-- name: ListDispatchers :many
SELECT
    sqlc.embed(dispatchers)
FROM
    "Dispatcher" as dispatchers;

-- name: DeleteDispatcher :one
DELETE FROM
    "Dispatcher" as dispatchers
WHERE
    "id" = sqlc.arg('id')::uuid
RETURNING *;

-- name: CreateDispatcher :one
INSERT INTO
    "Dispatcher" ("id", "lastHeartbeatAt", "isActive")
VALUES
    (sqlc.arg('id')::uuid, CURRENT_TIMESTAMP, 't')
RETURNING *;

-- name: HeartbeatDispatcher :exec
-- Both columns are set from the database clock in one statement, so a reader sees either the
-- stale heartbeat or the restarted "heartbeatingSince", never a fresh heartbeat that hides a gap.
UPDATE
    "Dispatcher" as dispatchers
SET
    "lastHeartbeatAt" = NOW(),
    -- a heartbeat that follows a gap starts a new unbroken run
    "heartbeatingSince" = CASE
        WHEN "lastHeartbeatAt" IS NULL OR "lastHeartbeatAt" < NOW() - INTERVAL '15 seconds' THEN NOW()
        ELSE "heartbeatingSince"
    END
WHERE
    "id" = sqlc.arg('id')::uuid;

-- name: HasDispatcherHeartbeatingThroughout :one
-- Whether some dispatcher is live and has written its own heartbeat without a gap for the last
-- minute. Dispatchers write worker heartbeats through the same process and database, so without
-- one a stale worker heartbeat may be the engine's failure to record it, not the worker's to send it.
-- The minute is the 30 seconds a worker has before it counts as inactive, plus time for workers
-- to get a heartbeat recorded once the engine can record them again.
SELECT EXISTS (
    SELECT 1
    FROM "Dispatcher"
    WHERE
        "isActive" = true
        AND "lastHeartbeatAt" > NOW() - INTERVAL '15 seconds'
        AND "heartbeatingSince" < NOW() - INTERVAL '60 seconds'
);
