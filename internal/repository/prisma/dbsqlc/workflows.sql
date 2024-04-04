-- name: CountWorkflows :one
SELECT
    count(workflows) OVER() AS total
FROM
    "Workflow" as workflows 
WHERE
    workflows."tenantId" = $1 AND
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
FROM (
    SELECT
        DISTINCT ON(workflows."id") workflows.*
    FROM
        "Workflow" as workflows 
    LEFT JOIN
        (
            SELECT * FROM "WorkflowVersion" as workflowVersion ORDER BY workflowVersion."order" DESC LIMIT 1
        ) as workflowVersion ON workflows."id" = workflowVersion."workflowId"
    LEFT JOIN
        "WorkflowTriggers" as workflowTrigger ON workflowVersion."id" = workflowTrigger."workflowVersionId"
    LEFT JOIN
        "WorkflowTriggerEventRef" as workflowTriggerEventRef ON workflowTrigger."id" = workflowTriggerEventRef."parentId"
    WHERE
        workflows."tenantId" = $1 
        AND
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
                    t1."workflowId" DESC
            )
        )
    ORDER BY workflows."id" DESC
) as workflows
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
    "scheduleTimeout"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @checksum::text,
    sqlc.narg('version')::text,
    @workflowId::uuid,
    coalesce(sqlc.narg('scheduleTimeout')::text, '5m')
) RETURNING *;

-- name: CreateWorkflowConcurrency :one
INSERT INTO "WorkflowConcurrency" (
    "id",
    "createdAt",
    "updatedAt",
    "workflowVersionId",
    "getConcurrencyGroupId",
    "maxRuns",
    "limitStrategy"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @workflowVersionId::uuid,
    @getConcurrencyGroupId::uuid,
    coalesce(sqlc.narg('maxRuns')::integer, 1),
    coalesce(sqlc.narg('limitStrategy')::"ConcurrencyLimitStrategy", 'CANCEL_IN_PROGRESS')
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
    "timeout"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @tenantId::uuid,
    @workflowVersionId::uuid,
    @name::text,
    @description::text,
    @timeout::text
) RETURNING *;

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
    AND 
    (j2.id IN (
        SELECT t3."parentId"
        FROM "WorkflowTriggerEventRef" AS t3
        WHERE t3."eventKey" = @eventKey AND t3."parentId" IS NOT NULL
    ) AND j2.id IS NOT NULL)
ORDER BY "WorkflowVersion"."workflowId", "WorkflowVersion"."order" DESC;

-- name: GetWorkflowVersionForEngine :many
SELECT
    sqlc.embed(workflowVersions),
    w."name" as "workflowName",
    wc."limitStrategy" as "concurrencyLimitStrategy",
    wc."maxRuns" as "concurrencyMaxRuns"
FROM
    "WorkflowVersion" as workflowVersions
JOIN
    "Workflow" as w ON w."id" = workflowVersions."workflowId"
LEFT JOIN
    "WorkflowConcurrency" as wc ON wc."workflowVersionId" = workflowVersions."id"
WHERE
    workflowVersions."id" = ANY(@ids::uuid[]) AND
    w."tenantId" = @tenantId::uuid;

-- name: GetWorkflowByName :one
SELECT
    *
FROM
    "Workflow" as workflows
WHERE
    workflows."tenantId" = @tenantId::uuid AND
    workflows."name" = @name::text;

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
    workflowVersions."workflowId" = @workflowId::uuid
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
    (
        sqlc.narg('status')::"WorkflowRunStatus" IS NULL OR
        r1."status" = sqlc.narg('status')::"WorkflowRunStatus"
    ) AND
    workflowVersion."workflowId" = @workflowId::uuid;