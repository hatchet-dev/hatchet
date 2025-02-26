-- name: GetStreamEventMeta :one
SELECT
    jr."workflowRunId" AS "workflowRunId",
    sr."retryCount" AS "retryCount",
    s."retries" as "retries"
FROM "StepRun" sr
JOIN "Step" s ON sr."stepId" = s."id"
JOIN "JobRun" jr ON sr."jobRunId" = jr."id"
WHERE sr."id" = @stepRunId::uuid
AND sr."tenantId" = @tenantId::uuid;

-- name: CreateStreamEvent :one
INSERT INTO "StreamEvent" (
    "createdAt",
    "tenantId",
    "stepRunId",
    "message",
    "metadata"
)
SELECT
    coalesce(sqlc.narg('createdAt')::timestamp, now()),
    @tenantId::uuid,
    @stepRunId::uuid,
    @message::bytea,
    coalesce(sqlc.narg('metadata')::jsonb, '{}'::jsonb)
FROM "StepRun"
WHERE "StepRun"."id" = @stepRunId::uuid
AND "StepRun"."tenantId" = @tenantId::uuid
RETURNING *;

-- name: GetStreamEvent :one
SELECT * FROM "StreamEvent"
WHERE
  "tenantId" = @tenantId::uuid AND
  "id" = @id::bigint;

-- name: CleanupStreamEvents :exec
DELETE FROM "StreamEvent"
WHERE
  -- older than than 5 minutes ago
  "createdAt" < NOW() - INTERVAL '5 minutes';
