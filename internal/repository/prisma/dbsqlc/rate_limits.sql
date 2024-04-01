-- name: CreateRateLimit :one
INSERT INTO "RateLimit" (
    "tenantId", 
    "key", 
    "max", 
    "value", 
    "window"
) VALUES (
    @tenantId::uuid,
    @key::text,
    @max::int,
    @max::int,
    COALESCE(sqlc.narg('window')::text, '1 minute')
) RETURNING *;