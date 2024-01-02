-- name: CountEvents :one
SELECT
    count(*) OVER() AS total
FROM
    "Event" as events
WHERE
    events."tenantId" = $1 AND
    (
        sqlc.narg('keys')::text[] IS NULL OR
        events."key" = ANY(sqlc.narg('keys')::text[])
    );

-- name: ListEvents :many
SELECT
    sqlc.embed(events),
    sum(case when runs."status" = 'PENDING' then 1 else 0 end) AS pendingRuns,
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
        sqlc.narg('search')::text IS NULL OR
        workflow.name like concat('%', sqlc.narg('search')::text, '%') OR
        jsonb_path_exists(events."data", cast(concat('$.** ? (@.type() == "string" && @ like_regex "', sqlc.narg('search')::text, '")') as jsonpath))
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
