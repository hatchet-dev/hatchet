-- name: GetEventForEngine :one
SELECT
    *
FROM
    "Event"
WHERE
    "deletedAt" IS NULL AND
    "id" = @id::uuid;

-- name: CountEvents :one
WITH events AS (
    SELECT
        events."id"
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
        events."deletedAt" IS NULL AND
        (
            sqlc.narg('event_ids')::uuid[] IS NULL OR
            events."id" = ANY(sqlc.narg('event_ids')::uuid[])
        ) AND
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
    ORDER BY
        case when @orderBy = 'createdAt ASC' THEN events."createdAt" END ASC ,
        case when @orderBy = 'createdAt DESC' then events."createdAt" END DESC
    LIMIT 10000
)
SELECT
    count(events) AS total
FROM
    events;

-- name: CreateEventKeys :exec
INSERT INTO "EventKey" (
    "key",
    "tenantId"
)
SELECT
    unnest(@keys::text[]) AS "key",
    unnest(@tenantIds::uuid[]) AS "tenantId"
ON CONFLICT ("key", "tenantId") DO NOTHING;

-- name: ListEventKeys :many
SELECT
    "key"
FROM
    "EventKey"
WHERE
    "tenantId" = @tenantId::uuid
ORDER BY "key" ASC;

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

-- name: CreateEvents :copyfrom
INSERT INTO "Event" (
    "id",
    "key",
    "tenantId",
    "replayedFromId",
    "data",
    "additionalMetadata",
    "insertOrder"

) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
);


-- name: GetInsertedEvents :many
SELECT * FROM "Event"
WHERE "id" = ANY(@ids::uuid[])
ORDER BY "insertOrder" ASC;

-- name: ListEvents :many
WITH filtered_events AS (
    SELECT
        events."id"
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
        events."deletedAt" IS NULL AND
        (
            sqlc.narg('event_ids')::uuid[] IS NULL OR
            events."id" = ANY(sqlc.narg('event_ids')::uuid[])
        ) AND
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
            runs."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
        )
    GROUP BY events."id"
    ORDER BY
        case when @orderBy = 'createdAt ASC' THEN MAX(events."createdAt") END ASC,
        case when @orderBy = 'createdAt DESC' then MAX(events."createdAt") END DESC
    OFFSET
        COALESCE(sqlc.narg('offset'), 0)
    LIMIT
        COALESCE(sqlc.narg('limit'), 50)
),
event_run_counts AS (
    SELECT
        events."id" as event_id,
        COUNT(CASE WHEN runs."status" = 'PENDING' THEN 1 END) AS pendingRuns,
        COUNT(CASE WHEN runs."status" = 'QUEUED' THEN 1 END) AS queuedRuns,
        COUNT(CASE WHEN runs."status" = 'RUNNING' THEN 1 END) AS runningRuns,
        COUNT(CASE WHEN runs."status" = 'SUCCEEDED' THEN 1 END) AS succeededRuns,
        COUNT(CASE WHEN runs."status" = 'FAILED' THEN 1 END) AS failedRuns,
        COUNT(CASE WHEN runs."status" = 'CANCELLED' THEN 1 END) AS cancelledRuns
    FROM
        filtered_events
    JOIN
        "Event" as events ON events."id" = filtered_events."id"
    LEFT JOIN
        "WorkflowRunTriggeredBy" as runTriggers ON events."id" = runTriggers."eventId"
    LEFT JOIN
        "WorkflowRun" as runs ON runTriggers."parentId" = runs."id"
    GROUP BY
        events."id"
)
SELECT
    sqlc.embed(events),
    COALESCE(erc.pendingRuns, 0) AS pendingRuns,
    COALESCE(erc.queuedRuns, 0) AS queuedRuns,
    COALESCE(erc.runningRuns, 0) AS runningRuns,
    COALESCE(erc.succeededRuns, 0) AS succeededRuns,
    COALESCE(erc.failedRuns, 0) AS failedRuns,
    COALESCE(erc.cancelledRuns, 0) AS cancelledRuns
FROM
    filtered_events fe
JOIN
    "Event" as events ON events."id" = fe."id"
LEFT JOIN
    event_run_counts erc ON events."id" = erc.event_id
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN events."createdAt" END ASC,
    case when @orderBy = 'createdAt DESC' then events."createdAt" END DESC;

-- name: GetEventsForRange :many
SELECT
    date_trunc('hour', "createdAt") AS event_hour,
    COUNT(*) AS event_count
FROM
    "Event"
WHERE
    events."deletedAt" IS NOT NULL AND
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
    events."deletedAt" IS NULL AND
    "tenantId" = @tenantId::uuid AND
    "id" = ANY (sqlc.arg('ids')::uuid[]);

-- name: SoftDeleteExpiredEvents :one
WITH for_delete AS (
    SELECT
        "id"
    FROM "Event" e
    WHERE
        e."tenantId" = @tenantId::uuid AND
        e."createdAt" < @createdBefore::timestamp AND
        e."deletedAt" IS NULL
    ORDER BY e."createdAt" ASC
    LIMIT sqlc.arg('limit') +1
    FOR UPDATE SKIP LOCKED
),expired_with_limit AS (
    SELECT
        for_delete."id" as "id"
    FROM for_delete
    LIMIT sqlc.arg('limit')
), has_more AS (
    SELECT
        CASE
            WHEN COUNT(*) > sqlc.arg('limit') THEN TRUE
            ELSE FALSE
        END as has_more
    FROM for_delete
)
UPDATE
    "Event"
SET
    "deletedAt" = CURRENT_TIMESTAMP
WHERE
    "id" IN (SELECT "id" FROM expired_with_limit)
RETURNING
    (SELECT has_more FROM has_more) as has_more;


-- name: ClearEventPayloadData :one
WITH for_delete AS (
    SELECT
        e1."id" as "id"
    FROM "Event" e1
    WHERE
        e1."tenantId" = @tenantId::uuid AND
        e1."deletedAt" IS NOT NULL -- TODO change this for all clear queries
        AND e1."data" IS NOT NULL
    LIMIT sqlc.arg('limit') + 1
    FOR UPDATE SKIP LOCKED
), expired_with_limit AS (
    SELECT
        for_delete."id" as "id"
    FROM for_delete
    LIMIT sqlc.arg('limit')
),
has_more AS (
    SELECT
        CASE
            WHEN COUNT(*) > sqlc.arg('limit') THEN TRUE
            ELSE FALSE
        END as has_more
    FROM for_delete
)
UPDATE
    "Event"
SET
    "data" = NULL
WHERE
    "id" IN (SELECT "id" FROM expired_with_limit)
RETURNING
    (SELECT has_more FROM has_more) as has_more;
