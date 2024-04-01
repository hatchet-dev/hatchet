-- name: CreateStreamEvent :one
INSERT INTO "StreamEvent" (
    "createdAt",
    "tenantId",
    "stepRunId",
    "message",
    "metadata"
) VALUES (
    coalesce(sqlc.narg('createdAt')::timestamp, now()),
    @tenantId::uuid,
    @stepRunId::uuid,
    @message::bytea,
    coalesce(sqlc.narg('metadata')::jsonb, '{}'::jsonb)
) RETURNING *;

-- name: GetStreamEvent :one
SELECT * FROM "StreamEvent"
WHERE
  "tenantId" = @tenantId::uuid AND
  "id" = @id::bigint;

-- name: DeleteStreamEvent :one
DELETE FROM "StreamEvent"
WHERE
  "tenantId" = @tenantId::uuid AND
  "id" = @id::bigint
RETURNING *;

-- name: ListStreamEvents :many
SELECT * FROM "StreamEvent"
WHERE
  "tenantId" = @tenantId::uuid AND
  (sqlc.narg('stepRunId')::uuid IS NULL OR "stepRunId" = sqlc.narg('stepRunId')::uuid)
ORDER BY
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt ASC' THEN "createdAt" END ASC,
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt DESC' THEN "createdAt" END DESC,
  -- add order by id to make sure the order is deterministic
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt ASC' THEN "id" END ASC,
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt DESC' THEN "id" END DESC
LIMIT COALESCE(sqlc.narg('limit'), 50)
OFFSET COALESCE(sqlc.narg('offset'), 0);

-- name: CountStreamEvents :one
SELECT COUNT(*) AS total
FROM "StreamEvent"
WHERE
  "tenantId" = @tenantId::uuid AND
  (sqlc.narg('stepRunId')::uuid IS NULL OR "stepRunId" = sqlc.narg('stepRunId')::uuid) AND
  (sqlc.narg('levels')::"LogLineLevel"[] IS NULL OR "level" = ANY(sqlc.narg('levels')::"LogLineLevel"[]));


-- name: CleanupStreamEvents :exec
DELETE FROM "StreamEvent"
WHERE
    -- older than than 5 minutes ago
    "createdAt" > NOW () - INTERVAL '5 minutes';