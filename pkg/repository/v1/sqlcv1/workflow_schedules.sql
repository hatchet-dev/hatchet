-- name: ListScheduledWorkflows :many
SELECT
    w."name",
    w."id" as "workflowId",
    v."id" as "workflowVersionId",
    w."tenantId",
    t.*,
    wr."createdAt" as "workflowRunCreatedAt",
    wr."status" as "workflowRunStatus",
    wr."id" as "workflowRunId",
    wr."displayName" as "workflowRunName"
FROM "WorkflowTriggerScheduledRef" t
JOIN "WorkflowVersion" v ON t."parentId" = v."id"
JOIN "Workflow" w on v."workflowId" = w."id"
LEFT JOIN "WorkflowRunTriggeredBy" tb ON t."id" = tb."scheduledId"
LEFT JOIN "WorkflowRun" wr ON tb."parentId" = wr."id"
WHERE v."deletedAt" IS NULL
	AND w."tenantId" = @tenantId::uuid
    AND (@scheduleId::uuid IS NULL OR t."id" = @scheduleId::uuid)
    AND (@workflowId::uuid IS NULL OR w."id" = @workflowId::uuid)
    AND (@parentWorkflowRunId::uuid IS NULL OR t."id" = @parentWorkflowRunId::uuid)
    AND (@parentStepRunId::uuid IS NULL OR t."parentStepRunId" = @parentStepRunId::uuid)
    AND (sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        t."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb)
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR
        wr."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
        or (
            @includeScheduled::boolean IS TRUE AND
            wr."status" IS NULL
        )
    )
ORDER BY
    case when @orderBy = 'triggerAt ASC' THEN t."triggerAt" END ASC ,
    case when @orderBy = 'triggerAt DESC' THEN t."triggerAt" END DESC,
    case when @orderBy = 'createdAt ASC' THEN t."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' THEN t."createdAt" END DESC,
    t."id" ASC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: CountScheduledWorkflows :one
SELECT count(*)
FROM "WorkflowTriggerScheduledRef" t
JOIN "WorkflowVersion" v ON t."parentId" = v."id"
JOIN "Workflow" w on v."workflowId" = w."id"
LEFT JOIN "WorkflowRunTriggeredBy" tb ON t."id" = tb."scheduledId"
LEFT JOIN "WorkflowRun" wr ON tb."parentId" = wr."id"
WHERE v."deletedAt" IS NULL
	AND w."tenantId" = @tenantId::uuid
    AND (@scheduleId::uuid IS NULL OR t."id" = @scheduleId::uuid)
    AND (@workflowId::uuid IS NULL OR w."id" = @workflowId::uuid)
    AND (@parentWorkflowRunId::uuid IS NULL OR t."id" = @parentWorkflowRunId::uuid)
    AND (@parentStepRunId::uuid IS NULL OR t."parentStepRunId" = @parentStepRunId::uuid)
    AND (sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        t."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb)
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR
        wr."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
        or (
            @includeScheduled::boolean IS TRUE AND
            wr."status" IS NULL
        )
    );

-- name: UpdateScheduledWorkflow :exec
UPDATE "WorkflowTriggerScheduledRef"
SET "triggerAt" = @triggerAt::timestamp
WHERE
    "id" = @scheduleId::uuid;

-- name: DeleteScheduledWorkflow :exec
DELETE FROM "WorkflowTriggerScheduledRef"
WHERE
    "id" = @scheduleId::uuid;

-- name: GetScheduledWorkflowMetaByIds :many
SELECT
    t."id",
    t."method",
    EXISTS (
        SELECT 1
        FROM "WorkflowRunTriggeredBy" tb
        WHERE tb."scheduledId" = t."id"
    ) AS "hasTriggeredRun"
FROM "WorkflowTriggerScheduledRef" t
JOIN "WorkflowVersion" v ON t."parentId" = v."id"
JOIN "Workflow" w ON v."workflowId" = w."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND t."id" = ANY(@ids::uuid[]);

-- name: BulkDeleteScheduledWorkflows :many
DELETE FROM "WorkflowTriggerScheduledRef" t
USING "WorkflowVersion" v, "Workflow" w
WHERE
    t."parentId" = v."id"
    AND v."workflowId" = w."id"
    AND w."tenantId" = @tenantId::uuid
    AND t."method" = 'API'
    AND t."id" = ANY(@ids::uuid[])
RETURNING t."id";

-- name: BulkUpdateScheduledWorkflows :many
WITH input AS (
    SELECT
        ids.id,
        times."triggerAt"
    FROM unnest(@ids::uuid[]) WITH ORDINALITY AS ids(id, ord)
    JOIN unnest(@triggerAts::timestamp[]) WITH ORDINALITY AS times("triggerAt", ord)
        USING (ord)
)
UPDATE "WorkflowTriggerScheduledRef" t
SET "triggerAt" = i."triggerAt"
FROM input i, "WorkflowVersion" v, "Workflow" w
WHERE
    t."id" = i.id
    AND t."parentId" = v."id"
    AND v."workflowId" = w."id"
    AND w."tenantId" = @tenantId::uuid
    AND t."method" = 'API'
    AND NOT EXISTS (
        SELECT 1
        FROM "WorkflowRunTriggeredBy" tb
        WHERE tb."scheduledId" = t."id"
    )
RETURNING t."id";

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
