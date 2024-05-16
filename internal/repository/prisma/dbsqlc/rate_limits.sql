-- name: UpsertRateLimit :one
INSERT INTO "RateLimit" (
    "tenantId",
    "key",
    "limitValue",
    "value",
    "window"
) VALUES (
    @tenantId::uuid,
    @key::text,
    sqlc.arg('limit')::int,
    sqlc.arg('limit')::int,
    COALESCE(sqlc.narg('window')::text, '1 minute')
) ON CONFLICT ("tenantId", "key") DO UPDATE SET
    "limitValue" = sqlc.arg('limit')::int,
    "window" = COALESCE(sqlc.narg('window')::text, '1 minute')
RETURNING *;
