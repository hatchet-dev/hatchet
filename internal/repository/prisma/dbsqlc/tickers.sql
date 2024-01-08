-- name: ListNewlyStaleTickers :many
SELECT
    sqlc.embed(tickers)
FROM "Ticker" as tickers
WHERE
    -- last heartbeat older than 15 seconds
    "lastHeartbeatAt" < NOW () - INTERVAL '15 seconds'
    -- active
    AND "isActive" = true;

-- name: ListActiveTickers :many
SELECT
    sqlc.embed(tickers)
FROM "Ticker" as tickers
WHERE
    -- last heartbeat greater than 15 seconds
    "lastHeartbeatAt" > NOW () - INTERVAL '15 seconds'
    -- active
    AND "isActive" = true;

-- name: SetTickersInactive :many
UPDATE
    "Ticker" as tickers
SET
    "isActive" = false
WHERE
    "id" = ANY (sqlc.arg('ids')::uuid[])
RETURNING
    sqlc.embed(tickers);

-- name: ListTickers :many
SELECT
    sqlc.embed(tickers)
FROM
    "Ticker" as tickers;