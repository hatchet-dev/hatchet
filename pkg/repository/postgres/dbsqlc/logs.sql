-- name: CreateLogLine :one
INSERT INTO "LogLine" (
    "createdAt",
    "tenantId",
    "stepRunId",
    "message",
    "level",
    "metadata"
)
SELECT
    coalesce(sqlc.narg('createdAt')::timestamp, now()),
    @tenantId::uuid,
    @stepRunId::uuid,
    @message::text,
    coalesce(sqlc.narg('level')::"LogLineLevel", 'INFO'::"LogLineLevel"),
    coalesce(sqlc.narg('metadata')::jsonb, '{}'::jsonb)
FROM "StepRun"
WHERE "StepRun"."id" = @stepRunId::uuid
AND "StepRun"."tenantId" = @tenantId::uuid
RETURNING *;

-- name: ListLogLines :many
SELECT * FROM "LogLine"
WHERE
  "tenantId" = @tenantId::uuid AND
  (sqlc.narg('stepRunId')::uuid IS NULL OR "stepRunId" = sqlc.narg('stepRunId')::uuid) AND
  (sqlc.narg('search')::text IS NULL OR "message" LIKE concat('%', sqlc.narg('search')::text, '%')) AND
  (sqlc.narg('levels')::"LogLineLevel"[] IS NULL OR "level" = ANY(sqlc.narg('levels')::"LogLineLevel"[]))
ORDER BY
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt ASC' THEN "createdAt" END ASC,
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt DESC' THEN "createdAt" END DESC,
  -- add order by id to make sure the order is deterministic
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt ASC' THEN "id" END ASC,
  CASE WHEN sqlc.narg('orderBy')::text = 'createdAt DESC' THEN "id" END DESC
LIMIT COALESCE(sqlc.narg('limit'), 50)
OFFSET COALESCE(sqlc.narg('offset'), 0);

-- name: CountLogLines :one
SELECT COUNT(*) AS total
FROM "LogLine"
WHERE
  "tenantId" = @tenantId::uuid AND
  (sqlc.narg('stepRunId')::uuid IS NULL OR "stepRunId" = sqlc.narg('stepRunId')::uuid) AND
  (sqlc.narg('search')::text IS NULL OR "message" LIKE concat('%', sqlc.narg('search')::text, '%')) AND
  (sqlc.narg('levels')::"LogLineLevel"[] IS NULL OR "level" = ANY(sqlc.narg('levels')::"LogLineLevel"[]));
