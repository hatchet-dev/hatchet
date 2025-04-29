-- name: CountWorkflows :one
SELECT
    count(workflows) OVER() AS total
FROM
    "Workflow" as workflows
WHERE
    workflows."tenantId" = $1 AND
    workflows."deletedAt" IS NULL AND
    (
        sqlc.narg('eventKey')::text IS NULL OR
        workflows."id" IN (
            SELECT
                DISTINCT ON(t1."workflowId") t1."workflowId"
            FROM
                "WorkflowVersion" AS t1
                LEFT JOIN "WorkflowTriggers" AS j2 ON j2."workflowVersionId" = t1."id"
            WHERE
                (
                    j2."id" IN (
                        SELECT
                            t3."parentId"
                        FROM
                            "public"."WorkflowTriggerEventRef" AS t3
                        WHERE
                            t3."eventKey" = sqlc.narg('eventKey')::text
                            AND t3."parentId" IS NOT NULL
                    )
                    AND j2."id" IS NOT NULL
                    AND t1."workflowId" IS NOT NULL
                )
            ORDER BY
                t1."workflowId" DESC, t1."order" DESC
        )
    );

-- name: ListWorkflowsLatestRuns :many
SELECT
    DISTINCT ON (workflow."id") sqlc.embed(runs), workflow."id" as "workflowId"
FROM
    "WorkflowRun" as runs
LEFT JOIN
    "WorkflowVersion" as workflowVersion ON runs."workflowVersionId" = workflowVersion."id"
LEFT JOIN
    "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
WHERE
    runs."tenantId" = $1 AND
    runs."deletedAt" IS NULL AND
    workflow."deletedAt" IS NULL AND
    workflowVersion."deletedAt" IS NULL AND
    (
        sqlc.narg('eventKey')::text IS NULL OR
        workflow."id" IN (
            SELECT
                DISTINCT ON(t1."workflowId") t1."workflowId"
            FROM
                "WorkflowVersion" AS t1
                LEFT JOIN "WorkflowTriggers" AS j2 ON j2."workflowVersionId" = t1."id"
            WHERE
                (
                    j2."id" IN (
                        SELECT
                            t3."parentId"
                        FROM
                            "public"."WorkflowTriggerEventRef" AS t3
                        WHERE
                            t3."eventKey" = sqlc.narg('eventKey')::text
                            AND t3."parentId" IS NOT NULL
                    )
                    AND j2."id" IS NOT NULL
                    AND t1."workflowId" IS NOT NULL
                )
            ORDER BY
                t1."workflowId" DESC, t1."order" DESC
        )
    )
ORDER BY
    workflow."id" DESC, runs."createdAt" DESC;

-- name: ListWorkflows :many
SELECT
    sqlc.embed(workflows)
FROM
    "Workflow" as workflows
WHERE
    workflows."tenantId" = @tenantId::uuid AND
    workflows."deletedAt" IS NULL AND
    (
        sqlc.narg('search')::text IS NULL OR
        workflows.name like concat('%', sqlc.narg('search')::text, '%')
    )
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN workflows."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' then workflows."createdAt" END DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: CreateWorkflow :one
INSERT INTO "Workflow" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "tenantId",
    "name",
    "description"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @tenantId::uuid,
    @name::text,
    @description::text
) RETURNING *;

-- name: CreateWorkflowVersion :one
INSERT INTO "WorkflowVersion" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "checksum",
    "version",
    "workflowId",
    "scheduleTimeout",
    "sticky",
    "kind",
    "defaultPriority"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @checksum::text,
    sqlc.narg('version')::text,
    @workflowId::uuid,
    coalesce(sqlc.narg('scheduleTimeout')::text, '5m'),
    sqlc.narg('sticky')::"StickyStrategy",
    coalesce(sqlc.narg('kind')::"WorkflowKind", 'DAG'),
    sqlc.narg('defaultPriority')::integer
) RETURNING *;

-- name: MoveCronTriggerToNewWorkflowTriggers :exec
WITH triggersToUpdate AS (
    SELECT cronTrigger."id" FROM "WorkflowTriggerCronRef" cronTrigger
    JOIN "WorkflowTriggers" triggers ON triggers."id" = cronTrigger."parentId"
    WHERE triggers."workflowVersionId" = @oldWorkflowVersionId::uuid
    AND cronTrigger."method" = 'API'
)
UPDATE "WorkflowTriggerCronRef"
SET "parentId" = @newWorkflowTriggerId::uuid
WHERE "id" IN (SELECT "id" FROM triggersToUpdate);

-- name: MoveScheduledTriggerToNewWorkflowTriggers :exec
WITH triggersToUpdate AS (
    SELECT scheduledTrigger."id" FROM "WorkflowTriggerScheduledRef" scheduledTrigger
    JOIN "WorkflowTriggers" triggers ON triggers."id" = scheduledTrigger."parentId"
    WHERE triggers."workflowVersionId" = @oldWorkflowVersionId::uuid
    AND scheduledTrigger."method" = 'API'
)
UPDATE "WorkflowTriggerScheduledRef"
SET "parentId" = @newWorkflowTriggerId::uuid
WHERE "id" IN (SELECT "id" FROM triggersToUpdate);

-- name: CreateWorkflowConcurrency :one
INSERT INTO "WorkflowConcurrency" (
    "id",
    "createdAt",
    "updatedAt",
    "workflowVersionId",
    "getConcurrencyGroupId",
    "maxRuns",
    "limitStrategy",
    "concurrencyGroupExpression"
) VALUES (
    gen_random_uuid(),
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @workflowVersionId::uuid,
    sqlc.narg('getConcurrencyGroupId')::uuid,
    coalesce(sqlc.narg('maxRuns')::integer, 1),
    coalesce(sqlc.narg('limitStrategy')::"ConcurrencyLimitStrategy", 'CANCEL_IN_PROGRESS'),
    sqlc.narg('concurrencyGroupExpression')::text
) RETURNING *;

-- name: CreateJob :one
INSERT INTO "Job" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "tenantId",
    "workflowVersionId",
    "name",
    "description",
    "timeout",
    "kind"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @tenantId::uuid,
    @workflowVersionId::uuid,
    @name::text,
    @description::text,
    @timeout::text,
    coalesce(sqlc.narg('kind')::"JobKind", 'DEFAULT')
) RETURNING *;

-- name: LinkOnFailureJob :one
UPDATE "WorkflowVersion"
SET "onFailureJobId" = @jobId::uuid
WHERE "id" = @workflowVersionId::uuid
RETURNING *;

-- name: CreateStep :one
INSERT INTO "Step" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "readableId",
    "tenantId",
    "jobId",
    "actionId",
    "timeout",
    "customUserData",
    "retries",
    "scheduleTimeout",
    "retryBackoffFactor",
    "retryMaxBackoff"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @readableId::text,
    @tenantId::uuid,
    @jobId::uuid,
    @actionId::text,
    sqlc.narg('timeout')::text,
    coalesce(sqlc.narg('customUserData')::jsonb, '{}'),
    coalesce(sqlc.narg('retries')::integer, 0),
    coalesce(sqlc.narg('scheduleTimeout')::text, '5m'),
    sqlc.narg('retryBackoffFactor'),
    sqlc.narg('retryMaxBackoff')
) RETURNING *;

-- name: AddStepParents :exec
INSERT INTO "_StepOrder" ("A", "B")
SELECT
    step."id",
    @id::uuid
FROM
    unnest(@parents::text[]) AS parent_readable_id
JOIN
    "Step" AS step ON step."readableId" = parent_readable_id AND step."jobId" = @jobId::uuid;

-- name: CreateStepRateLimit :one
INSERT INTO "StepRateLimit" (
    "units",
    "stepId",
    "rateLimitKey",
    "tenantId",
    "kind"
) VALUES (
    @units::integer,
    @stepId::uuid,
    @rateLimitKey::text,
    @tenantId::uuid,
    @kind
) RETURNING *;

-- name: CreateStepExpressions :exec
INSERT INTO "StepExpression" (
    "key",
    "stepId",
    "expression",
    "kind"
) VALUES (
    unnest(@keys::text[]),
    @stepId::uuid,
    unnest(@expressions::text[]),
    unnest(cast(@kinds::text[] as"StepExpressionKind"[]))
) ON CONFLICT ("key", "stepId", "kind") DO UPDATE
SET
    "expression" = EXCLUDED."expression";

-- name: UpsertAction :one
INSERT INTO "Action" (
    "id",
    "actionId",
    "tenantId"
)
VALUES (
    gen_random_uuid(),
    LOWER(@action::text),
    @tenantId::uuid
)
ON CONFLICT ("tenantId", "actionId") DO UPDATE
SET
    "tenantId" = EXCLUDED."tenantId"
WHERE
    "Action"."tenantId" = @tenantId AND "Action"."actionId" = LOWER(@action::text)
RETURNING *;

-- name: UpsertWorkflowTag :exec
INSERT INTO "WorkflowTag" (
    "id",
    "tenantId",
    "name",
    "color"
)
VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    @tenantId::uuid,
    @tagName::text,
    COALESCE(sqlc.narg('tagColor')::text, '#93C5FD')
)
ON CONFLICT ("tenantId", "name") DO UPDATE
SET
    "color" = COALESCE(EXCLUDED."color", "WorkflowTag"."color")
WHERE
    "WorkflowTag"."tenantId" = @tenantId AND "WorkflowTag"."name" = @tagName;

-- name: AddWorkflowTag :exec
INSERT INTO "_WorkflowToWorkflowTag" ("A", "B")
SELECT @id::uuid, @tags::uuid
ON CONFLICT DO NOTHING;

-- name: CreateWorkflowTriggers :one
INSERT INTO "WorkflowTriggers" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "workflowVersionId",
    "tenantId"
) VALUES (
    @id::uuid,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    @workflowVersionId::uuid,
    @tenantId::uuid
) RETURNING *;

-- name: CreateWorkflowTriggerEventRef :one
INSERT INTO "WorkflowTriggerEventRef" (
    "parentId",
    "eventKey"
) VALUES (
    @workflowTriggersId::uuid,
    @eventTrigger::text
) RETURNING *;

-- name: CreateWorkflowTriggerCronRef :one
INSERT INTO "WorkflowTriggerCronRef" (
    "parentId",
    "cron",
    "name",
    "input",
    "additionalMetadata",
    "id",
    "method",
    "priority"
) VALUES (
    @workflowTriggersId::uuid,
    @cronTrigger::text,
    sqlc.narg('name')::text,
    sqlc.narg('input')::jsonb,
    sqlc.narg('additionalMetadata')::jsonb,
    gen_random_uuid(),
    COALESCE(sqlc.narg('method')::"WorkflowTriggerCronRefMethods", 'DEFAULT'),
    COALESCE(sqlc.narg('priority')::integer, 1)
) RETURNING *;


-- name: CreateWorkflowTriggerCronRefForWorkflow :one
WITH latest_version AS (
    SELECT "id" FROM "WorkflowVersion"
    WHERE "workflowId" = @workflowId::uuid
    ORDER BY "order" DESC
    LIMIT 1
),
latest_trigger AS (
    SELECT "id" FROM "WorkflowTriggers"
    WHERE "workflowVersionId" = (SELECT "id" FROM latest_version)
    ORDER BY "createdAt" DESC
    LIMIT 1
)
INSERT INTO "WorkflowTriggerCronRef" (
    "parentId",
    "cron",
    "name",
    "input",
    "additionalMetadata",
    "id",
    "method",
    "priority"
) VALUES (
    (SELECT "id" FROM latest_trigger),
    @cronTrigger::text,
    sqlc.narg('name')::text,
    sqlc.narg('input')::jsonb,
    sqlc.narg('additionalMetadata')::jsonb,
    gen_random_uuid(),
    COALESCE(sqlc.narg('method')::"WorkflowTriggerCronRefMethods", 'DEFAULT'),
    COALESCE(sqlc.narg('priority')::integer, 1)
) RETURNING *;

-- name: CreateWorkflowTriggerScheduledRefForWorkflow :one
WITH latest_version AS (
    SELECT "id" FROM "WorkflowVersion"
    WHERE "workflowId" = @workflowId::uuid
    ORDER BY "order" DESC
    LIMIT 1
),
latest_trigger AS (
    SELECT "id" FROM "WorkflowTriggers"
    WHERE "workflowVersionId" = (SELECT "id" FROM latest_version)
    ORDER BY "createdAt" DESC
    LIMIT 1
)
INSERT INTO "WorkflowTriggerScheduledRef" (
    "id",
    "parentId",
    "triggerAt",
    "input",
    "additionalMetadata",
    "method",
    "priority"
) VALUES (
    gen_random_uuid(),
    (SELECT "id" FROM latest_version),
    @scheduledTrigger::timestamp,
    @input::jsonb,
    @additionalMetadata::jsonb,
    COALESCE(sqlc.narg('method')::"WorkflowTriggerScheduledRefMethods", 'DEFAULT'),
    COALESCE(sqlc.narg('priority')::integer, 1)
) RETURNING *;

-- name: CreateWorkflowTriggerScheduledRef :one
INSERT INTO "WorkflowTriggerScheduledRef" (
    "id",
    "parentId",
    "triggerAt",
    "input",
    "additionalMetadata",
    "priority"
) VALUES (
    gen_random_uuid(),
    @workflowVersionId::uuid,
    @scheduledTrigger::timestamp,
    @input::jsonb,
    @additionalMetadata::jsonb,
    COALESCE(sqlc.narg('priority')::integer, 1)
) RETURNING *;

-- name: ListWorkflowsForEvent :many
-- Get all of the latest workflow versions for the tenant
WITH latest_versions AS (
    SELECT DISTINCT ON("workflowId")
        workflowVersions."id" AS "workflowVersionId"
    FROM
        "WorkflowVersion" as workflowVersions
    JOIN
        "Workflow" as workflow ON workflow."id" = workflowVersions."workflowId"
    WHERE
        workflow."tenantId" = @tenantId::uuid
        AND workflowVersions."deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
)
-- select the workflow versions that have the event trigger
SELECT
    latest_versions."workflowVersionId"
FROM
    latest_versions
JOIN
    "WorkflowTriggers" as triggers ON triggers."workflowVersionId" = latest_versions."workflowVersionId"
JOIN
    "WorkflowTriggerEventRef" as eventRef ON eventRef."parentId" = triggers."id"
WHERE
    eventRef."eventKey" = @eventKey::text;

-- name: GetWorkflowVersionForEngine :many
SELECT
    sqlc.embed(workflowVersions),
    w."name" as "workflowName",
    wc."limitStrategy" as "concurrencyLimitStrategy",
    wc."maxRuns" as "concurrencyMaxRuns",
    wc."getConcurrencyGroupId" as "concurrencyGroupId",
    wc."concurrencyGroupExpression" as "concurrencyGroupExpression"
FROM
    "WorkflowVersion" as workflowVersions
JOIN
    "Workflow" as w ON w."id" = workflowVersions."workflowId"
LEFT JOIN
    "WorkflowConcurrency" as wc ON wc."workflowVersionId" = workflowVersions."id"
WHERE
    workflowVersions."id" = ANY(@ids::uuid[]) AND
    w."tenantId" = @tenantId::uuid AND
    w."deletedAt" IS NULL AND
    workflowVersions."deletedAt" IS NULL;

-- name: GetLatestWorkflowVersionForWorkflows :many
WITH latest_versions AS (
    SELECT DISTINCT ON (workflowVersions."workflowId")
        workflowVersions."id" AS workflowVersionId,
        workflowVersions."workflowId",
        workflowVersions."order"
    FROM
        "WorkflowVersion" as workflowVersions
    WHERE
        workflowVersions."workflowId" = ANY(@workflowIds::uuid[]) AND
        workflowVersions."deletedAt" IS NULL
    ORDER BY
        workflowVersions."workflowId", workflowVersions."order" DESC
)
SELECT
    workflowVersions."id"
FROM
    latest_versions
JOIN
    "WorkflowVersion" as workflowVersions ON workflowVersions."id" = latest_versions.workflowVersionId
JOIN
    "Workflow" as w ON w."id" = workflowVersions."workflowId"
LEFT JOIN
    "WorkflowConcurrency" as wc ON wc."workflowVersionId" = workflowVersions."id"
WHERE
    w."tenantId" = @tenantId::uuid AND
    w."deletedAt" IS NULL AND
    workflowVersions."deletedAt" IS NULL;

-- name: GetWorkflowByName :one
SELECT
    *
FROM
    "Workflow" as workflows
WHERE
    workflows."tenantId" = @tenantId::uuid AND
    workflows."name" = @name::text AND
    workflows."deletedAt" IS NULL;

-- name: GetWorkflowsByNames :many
SELECT
    workflows.*
FROM
    "Workflow" as workflows
WHERE
    workflows."tenantId" = @tenantId::uuid AND
    workflows."name" = ANY(@names::text[]) AND
    workflows."deletedAt" IS NULL;



-- name: CreateSchedules :many
INSERT INTO "WorkflowTriggerScheduledRef" (
    "id",
    "parentId",
    "triggerAt",
    "input",
    "additionalMetadata",
    "priority"
) VALUES (
    gen_random_uuid(),
    @workflowRunId::uuid,
    unnest(@triggerTimes::timestamp[]),
    @input::jsonb,
    @additionalMetadata::json,
    COALESCE(sqlc.narg('priority')::integer, 1)
) RETURNING *;

-- name: GetWorkflowLatestVersion :one
SELECT
    "id"
FROM
    "WorkflowVersion" as workflowVersions
WHERE
    workflowVersions."workflowId" = @workflowId::uuid AND
    workflowVersions."deletedAt" IS NULL
ORDER BY
    workflowVersions."order" DESC
LIMIT 1;

-- name: CountWorkflowRunsRoundRobin :one
SELECT COUNT(*) AS total
FROM
    "WorkflowRun" r1
JOIN
    "WorkflowVersion" workflowVersion ON r1."workflowVersionId" = workflowVersion."id"
WHERE
    r1."tenantId" = @tenantId::uuid AND
    workflowVersion."deletedAt" IS NULL AND
    r1."deletedAt" IS NULL AND
    (
        sqlc.narg('status')::"WorkflowRunStatus" IS NULL OR
        r1."status" = sqlc.narg('status')::"WorkflowRunStatus"
    ) AND
    workflowVersion."workflowId" = @workflowId::uuid AND
    r1."concurrencyGroupId" IS NOT NULL AND
    (
        sqlc.narg('groupKey')::text IS NULL OR
        r1."concurrencyGroupId" = sqlc.narg('groupKey')::text
    );

-- name: CountRoundRobinGroupKeys :one
SELECT
    COUNT(DISTINCT "concurrencyGroupId") AS total
FROM
    "WorkflowRun" r1
JOIN
    "WorkflowVersion" workflowVersion ON r1."workflowVersionId" = workflowVersion."id"
WHERE
    r1."tenantId" = @tenantId::uuid AND
    workflowVersion."deletedAt" IS NULL AND
    r1."deletedAt" IS NULL AND
    (
        sqlc.narg('status')::"WorkflowRunStatus" IS NULL OR
        r1."status" = sqlc.narg('status')::"WorkflowRunStatus"
    ) AND
    workflowVersion."workflowId" = @workflowId::uuid;


-- name: SoftDeleteWorkflow :one
WITH versions AS (
    UPDATE "WorkflowVersion"
    SET "deletedAt" = CURRENT_TIMESTAMP
    WHERE "workflowId" = @id::uuid
)
UPDATE "Workflow"
SET
    -- set name to the current name plus a random suffix to avoid conflicts
    "name" = "name" || '-' || gen_random_uuid(),
    "deletedAt" = CURRENT_TIMESTAMP
WHERE "id" = @id::uuid
RETURNING *;

-- name: ListPausedWorkflows :many
SELECT
    "id"
FROM
    "Workflow"
WHERE
    "tenantId" = @tenantId::uuid AND
    "isPaused" = true AND
    "deletedAt" IS NULL;

-- name: UpdateWorkflow :one
UPDATE "Workflow"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "isPaused" = coalesce(sqlc.narg('isPaused')::boolean, "isPaused")
WHERE "id" = @id::uuid
RETURNING *;

-- name: HandleWorkflowUnpaused :exec
WITH matching_qis AS (
    -- We know that we're going to need to scan all the queue items in this queue
    -- for the tenant, so we write this query in such a way that the index is used.
    SELECT
        qi."id"
    FROM
        "InternalQueueItem" qi
    WHERE
        qi."isQueued" = true
        AND qi."tenantId" = @tenantId::uuid
        AND qi."queue" = 'WORKFLOW_RUN_PAUSED'
        AND qi."priority" = 1
    ORDER BY
        qi."id" DESC
)
UPDATE "InternalQueueItem"
-- We update all the queue items to have a higher priority so we can unpause them
SET "priority" = 4
FROM
    matching_qis
WHERE
    "InternalQueueItem"."id" = matching_qis."id"
    AND "data"->>'workflow_id' = @workflowId::text;

-- name: GetWorkflowWorkerCount :one
WITH UniqueWorkers AS (
    SELECT DISTINCT w."id" AS workerId
    FROM "Worker" w
    JOIN "_ActionToWorker" atw ON w."id" = atw."B"
    JOIN "Action" a ON atw."A" = a."id"
    JOIN "Step" s ON a."actionId" = s."actionId"
    JOIN "Job" j ON s."jobId" = j."id"
    JOIN "WorkflowVersion" workflowVersion ON j."workflowVersionId" = workflowVersion."id"
    WHERE
        w."tenantId" = @tenantId::uuid
        AND workflowVersion."deletedAt" IS NULL
        AND w."deletedAt" IS NULL
        AND w."dispatcherId" IS NOT NULL
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."isActive" = true
        AND w."isPaused" = false
        AND workflowVersion."workflowId" = @workflowId::uuid
),
workers AS (
    SELECT SUM("maxRuns") AS maxR
    FROM "Worker"
    WHERE "id" IN (SELECT workerId FROM UniqueWorkers)
),
slots AS (
    SELECT COUNT(*) AS usedSlotCount
    FROM "SemaphoreQueueItem" sqi
    WHERE sqi."workerId" IN (SELECT workerId FROM UniqueWorkers)
)
SELECT
    COALESCE(maxR, 0) AS totalSlotCount,
    COALESCE(maxR, 0)  - COALESCE(usedSlotCount, 0) AS freeSlotCount
FROM workers, slots;

-- name: GetWorkflowVersionCronTriggerRefs :many
SELECT
    wtc.*
FROM
    "WorkflowTriggerCronRef" as wtc
JOIN "WorkflowTriggers" as wt ON wt."id" = wtc."parentId"
WHERE
    wt."workflowVersionId" = @workflowVersionId::uuid;

-- name: GetWorkflowVersionEventTriggerRefs :many
SELECT
    wtc.*
FROM
    "WorkflowTriggerEventRef" as wtc
JOIN "WorkflowTriggers" as wt ON wt."id" = wtc."parentId"
WHERE
    wt."workflowVersionId" = @workflowVersionId::uuid;

-- name: GetWorkflowVersionScheduleTriggerRefs :many
SELECT
    wtc.*
FROM
    "WorkflowTriggerScheduledRef" as wtc
JOIN "WorkflowTriggers" as wt ON wt."id" = wtc."parentId"
WHERE
    wt."workflowVersionId" = @workflowVersionId::uuid;

-- name: GetWorkflowVersionById :one
SELECT
    sqlc.embed(wv),
    sqlc.embed(w),
    wc."id" as "concurrencyId",
    wc."maxRuns" as "concurrencyMaxRuns",
    wc."getConcurrencyGroupId" as "concurrencyGroupId",
    wc."limitStrategy" as "concurrencyLimitStrategy"
FROM
    "WorkflowVersion" as wv
JOIN "Workflow" as w on w."id" = wv."workflowId"
LEFT JOIN "WorkflowConcurrency" as wc ON wc."workflowVersionId" = wv."id"
WHERE
    wv."id" = @id::uuid AND
    wv."deletedAt" IS NULL
LIMIT 1;

-- name: GetWorkflowById :one
SELECT
    sqlc.embed(w),
    wv."id" as "workflowVersionId"
FROM
    "Workflow" as w
LEFT JOIN "WorkflowVersion" as wv ON w."id" = wv."workflowId"
WHERE
    w."id" = @id::uuid AND
    w."deletedAt" IS NULL
ORDER BY
    wv."order" DESC
LIMIT 1;


-- name: ListCronWorkflows :many
-- Get all of the latest workflow versions for the tenant
WITH latest_versions AS (
    SELECT DISTINCT ON("workflowId")
        workflowVersions."id" AS "workflowVersionId",
        workflowVersions."workflowId"
    FROM
        "WorkflowVersion" as workflowVersions
    JOIN
        "Workflow" as workflow ON workflow."id" = workflowVersions."workflowId"
    WHERE
        workflow."tenantId" = @tenantId::uuid
        AND workflowVersions."deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
)
SELECT
    latest_versions."workflowVersionId",
    w."name" as "workflowName",
    w."id" as "workflowId",
    w."tenantId",
    t."id" as "triggerId",
    c."id" as "cronId",
    t.*,
    c.*
FROM
    latest_versions
JOIN
    "WorkflowTriggers" as t ON t."workflowVersionId" = latest_versions."workflowVersionId"
JOIN
    "WorkflowTriggerCronRef" as c ON c."parentId" = t."id"
JOIN
    "Workflow" w on w."id" = latest_versions."workflowId"
WHERE
    t."deletedAt" IS NULL
    AND w."tenantId" = @tenantId::uuid
    AND (@cronTriggerId::uuid IS NULL OR c."id" = @cronTriggerId::uuid)
    AND (@workflowId::uuid IS NULL OR w."id" = @workflowId::uuid)
    AND (sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        c."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb)
    AND (@cronName::TEXT IS NULL OR c."name" = @cronName::TEXT)
    AND (@workflowName::TEXT IS NULL OR w."name" = @workflowName::TEXT)
ORDER BY
    case when @orderBy = 'name ASC' THEN w."name" END ASC,
    case when @orderBy = 'name DESC' THEN w."name" END DESC,
    case when @orderBy = 'createdAt ASC' THEN c."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' THEN c."createdAt" END DESC,
    t."id" ASC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: CountCronWorkflows :one
-- Get all of the latest workflow versions for the tenant
WITH latest_versions AS (
    SELECT DISTINCT ON("workflowId")
        workflowVersions."id" AS "workflowVersionId",
        workflowVersions."workflowId"
    FROM
        "WorkflowVersion" as workflowVersions
    JOIN
        "Workflow" as workflow ON workflow."id" = workflowVersions."workflowId"
    WHERE
        workflow."tenantId" = @tenantId::uuid
        AND workflowVersions."deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
)
SELECT
    count(c.*)
FROM
    latest_versions
JOIN
    "WorkflowTriggers" as t ON t."workflowVersionId" = latest_versions."workflowVersionId"
JOIN
    "WorkflowTriggerCronRef" as c ON c."parentId" = t."id"
JOIN
    "Workflow" w on w."id" = latest_versions."workflowId"
WHERE
    t."deletedAt" IS NULL
    AND w."tenantId" = @tenantId::uuid
    AND (@cronTriggerId::uuid IS NULL OR c."id" = @cronTriggerId::uuid)
    AND (@workflowId::uuid IS NULL OR w."id" = @workflowId::uuid)
    AND (sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        c."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb);

-- name: DeleteWorkflowTriggerCronRef :exec
DELETE FROM "WorkflowTriggerCronRef"
WHERE
    "id" = @id::uuid;

-- name: LockWorkflowVersion :one
SELECT
    "id"
FROM
    "WorkflowVersion"
WHERE
    "workflowId" = @workflowId::uuid AND
    "deletedAt" IS NULL
ORDER BY
    "order" DESC
LIMIT 1
FOR UPDATE;
