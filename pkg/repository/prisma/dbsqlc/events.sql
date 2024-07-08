-- name: GetEventForEngine :one
SELECT
    *
FROM
    "Event"
WHERE
    "id" = @id::uuid;

-- name: CountEvents :one
SELECT
    count(*) OVER() AS total
FROM
    "Event" as events
LEFT JOIN
  "WorkflowRunTriggeredBy" as runTriggers ON events."id" = runTriggers."eventId"
LEFT JOIN
  "WorkflowRun" as runs ON runTriggers."parentId" = runs."id"
LEFT JOIN
  "WorkflowVersion" as workflowVersion ON workflowVersion."id" = runs."workflowVersionId"
LEFT JOIN
  "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
WHERE
  events."tenantId" = $1 AND
  (
    sqlc.narg('keys')::text[] IS NULL OR
    events."key" = ANY(sqlc.narg('keys')::text[])
    ) AND
  (
    (sqlc.narg('workflows')::text[])::uuid[] IS NULL OR
    (workflow."id" = ANY(sqlc.narg('workflows')::text[]::uuid[]))
    ) AND
  (
    sqlc.narg('search')::text IS NULL OR
    jsonb_path_exists(events."data", cast(concat('$.** ? (@.type() == "string" && @ like_regex "', sqlc.narg('search')::text, '")') as jsonpath))
  ) AND
    (
        sqlc.narg('statuses')::text[] IS NULL OR
        "status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
    );

-- name: CreateEvent :one
INSERT INTO "Event" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "key",
    "tenantId",
    "replayedFromId",
    "data",
    "additionalMetadata"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @key::text,
    @tenantId::uuid,
    sqlc.narg('replayedFromId')::uuid,
    @data::jsonb,
    @additionalMetadata::jsonb
) RETURNING *;

-- name: ListEvents :many
SELECT
    sqlc.embed(events),
    sum(case when runs."status" = 'PENDING' then 1 else 0 end) AS pendingRuns,
    sum(case when runs."status" = 'QUEUED' then 1 else 0 end) AS queuedRuns,
    sum(case when runs."status" = 'RUNNING' then 1 else 0 end) AS runningRuns,
    sum(case when runs."status" = 'SUCCEEDED' then 1 else 0 end) AS succeededRuns,
    sum(case when runs."status" = 'FAILED' then 1 else 0 end) AS failedRuns
FROM
    "Event" as events
LEFT JOIN
    "WorkflowRunTriggeredBy" as runTriggers ON events."id" = runTriggers."eventId"
LEFT JOIN
    "WorkflowRun" as runs ON runTriggers."parentId" = runs."id"
LEFT JOIN
    "WorkflowVersion" as workflowVersion ON workflowVersion."id" = runs."workflowVersionId"
LEFT JOIN
    "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
WHERE
    events."tenantId" = $1 AND
    (
        sqlc.narg('keys')::text[] IS NULL OR
        events."key" = ANY(sqlc.narg('keys')::text[])
    ) AND
        (
        sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        events."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb
    ) AND
    (
        (sqlc.narg('workflows')::text[])::uuid[] IS NULL OR
        (workflow."id" = ANY(sqlc.narg('workflows')::text[]::uuid[]))
    ) AND
    (
        sqlc.narg('search')::text IS NULL OR
        workflow.name like concat('%', sqlc.narg('search')::text, '%') OR
        jsonb_path_exists(events."data", cast(concat('$.** ? (@.type() == "string" && @ like_regex "', sqlc.narg('search')::text, '")') as jsonpath))
    ) AND
    (
        sqlc.narg('statuses')::text[] IS NULL OR
        "status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
    )
GROUP BY
    events."id"
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN events."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' then events."createdAt" END DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: GetEventsForRange :many
SELECT
    date_trunc('hour', "createdAt") AS event_hour,
    COUNT(*) AS event_count
FROM
    "Event"
WHERE
    "createdAt" >= NOW() - INTERVAL '1 week'
GROUP BY
    event_hour
ORDER BY
    event_hour;

-- name: ListEventsByIDs :many
SELECT
    *
FROM
    "Event" as events
WHERE
    "tenantId" = @tenantId::uuid AND
    "id" = ANY (sqlc.arg('ids')::uuid[]);

-- name: DeleteExpiredEvents :one
WITH expired_events_count AS (
    SELECT COUNT(*) as count
    FROM "Event" e1
    WHERE
        e1."tenantId" = @tenantId::uuid AND
        e1."createdAt" < @createdBefore::timestamp
), expired_events_with_limit AS (
    SELECT
        "id"
    FROM "Event" e2
    WHERE
        e2."tenantId" = @tenantId::uuid AND
        e2."createdAt" < @createdBefore::timestamp
    ORDER BY "createdAt" ASC
    LIMIT sqlc.arg('limit')
)
DELETE FROM "Event"
WHERE
    "id" IN (SELECT "id" FROM expired_events_with_limit)
RETURNING (SELECT count FROM expired_events_count) as total, (SELECT count FROM expired_events_count) - (SELECT COUNT(*) FROM expired_events_with_limit) as remaining, (SELECT COUNT(*) FROM expired_events_with_limit) as deleted;
