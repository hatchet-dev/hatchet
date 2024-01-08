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