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
    workflows."tenantId" = $1 AND
    workflows."deletedAt" IS NULL
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
    "scheduleTimeout"
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
    coalesce(sqlc.narg('scheduleTimeout')::text, '5m')
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
    "tenantId"
) VALUES (
    @units::integer,
    @stepId::uuid,
    @rateLimitKey::text,
    @tenantId::uuid
) RETURNING *;

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
    "input"
) VALUES (
    @workflowTriggersId::uuid,
    @cronTrigger::text,
    sqlc.narg('input')::jsonb
) RETURNING *;

-- name: CreateWorkflowTriggerScheduledRef :one
INSERT INTO "WorkflowTriggerScheduledRef" (
    "id",
    "parentId",
    "triggerAt",
    "tickerId",
    "input"
) VALUES (
    gen_random_uuid(),
    @workflowVersionId::uuid,
    @scheduledTrigger::timestamp,
    NULL, -- or provide a tickerId if applicable
    NULL -- or provide input if applicable
) RETURNING *;

-- name: ListWorkflowsForEvent :many
SELECT DISTINCT ON ("WorkflowVersion"."workflowId") "WorkflowVersion".id
FROM "WorkflowVersion"
LEFT JOIN "Workflow" AS j1 ON j1.id = "WorkflowVersion"."workflowId"
LEFT JOIN "WorkflowTriggers" AS j2 ON j2."workflowVersionId" = "WorkflowVersion"."id"
WHERE
    (j1."tenantId"::uuid = @tenantId AND j1.id IS NOT NULL)
    AND j1."deletedAt" IS NULL
    AND "WorkflowVersion"."deletedAt" IS NULL
    AND
    (j2.id IN (
        SELECT t3."parentId"
        FROM "WorkflowTriggerEventRef" AS t3
        WHERE t3."eventKey" = @eventKey AND t3."parentId" IS NOT NULL
    ) AND j2.id IS NOT NULL)
    AND "WorkflowVersion".id = (
        -- confirm that the workflow version is the latest
        SELECT wv2.id
        FROM "WorkflowVersion" wv2
        WHERE wv2."workflowId" = "WorkflowVersion"."workflowId"
        ORDER BY wv2."order" DESC
        LIMIT 1
    )
ORDER BY "WorkflowVersion"."workflowId", "WorkflowVersion"."order" DESC;

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

-- name: GetWorkflowByName :one
SELECT
    *
FROM
    "Workflow" as workflows
WHERE
    workflows."tenantId" = @tenantId::uuid AND
    workflows."name" = @name::text AND
    workflows."deletedAt" IS NULL;

-- name: CreateSchedules :many
INSERT INTO "WorkflowTriggerScheduledRef" (
    "id",
    "parentId",
    "triggerAt",
    "input"
) VALUES (
    gen_random_uuid(),
    @workflowRunId::uuid,
    unnest(@triggerTimes::timestamp[]),
    @input::jsonb
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
