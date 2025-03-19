-- name: ListStepsByWorkflowVersionIds :many
WITH steps AS (
    SELECT
        s.*,
        wv."id" as "workflowVersionId",
        w."name" as "workflowName",
        w."id" as "workflowId",
        j."kind" as "jobKind"
    FROM
        "WorkflowVersion" as wv
    JOIN
        "Workflow" as w ON w."id" = wv."workflowId"
    JOIN
        "Job" j ON j."workflowVersionId" = wv."id"
    JOIN
        "Step" s ON s."jobId" = j."id"
    WHERE
        wv."id" = ANY(@ids::uuid[])
        AND w."tenantId" = @tenantId::uuid
        AND w."deletedAt" IS NULL
        AND wv."deletedAt" IS NULL
), step_orders AS (
    SELECT
        so."B" as "stepId",
        array_agg(so."A")::uuid[] as "parents"
    FROM
        steps
    JOIN
        "_StepOrder" so ON so."B" = steps."id"
    GROUP BY
        so."B"
)
SELECT
    s.*,
    COALESCE(so."parents", '{}'::uuid[]) as "parents"
FROM
    steps s
LEFT JOIN
    step_orders so ON so."stepId" = s."id";

-- name: ListStepsByIds :many
SELECT
    s.*,
    wv."id" as "workflowVersionId",
    wv."sticky" as "workflowVersionSticky",
    w."name" as "workflowName",
    w."id" as "workflowId",
    COUNT(se."stepId") as "exprCount",
    COUNT(sc.id) as "concurrencyCount"
FROM
    "Step" s
JOIN
    "Job" j ON j."id" = s."jobId"
JOIN
    "WorkflowVersion" wv ON wv."id" = j."workflowVersionId"
JOIN
    "Workflow" w ON w."id" = wv."workflowId"
LEFT JOIN
    v1_step_concurrency sc ON sc.workflow_id = w."id" AND sc.step_id = s."id"
LEFT JOIN
    "StepExpression" se ON se."stepId" = s."id"
WHERE
    s."id" = ANY(@ids::uuid[])
    AND w."tenantId" = @tenantId::uuid
    AND w."deletedAt" IS NULL
    AND wv."deletedAt" IS NULL
GROUP BY
    s."id", wv."id", w."name", w."id", wv."sticky";

-- name: ListStepExpressions :many
SELECT
    *
FROM
    "StepExpression"
WHERE
    "stepId" = ANY(@stepIds::uuid[]);

-- name: ListWorkflowNamesByIds :many
SELECT id, name
FROM "Workflow"
WHERE id = ANY(@ids::uuid[])
;


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
    "kind"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @checksum::text,
    sqlc.narg('version')::text,
    @workflowId::uuid,
    -- Deprecated: this is set but unused
    '5m',
    sqlc.narg('sticky')::"StickyStrategy",
    coalesce(sqlc.narg('kind')::"WorkflowKind", 'DAG')
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
    -- Deprecated: this is set but unused
    '5m',
    coalesce(sqlc.narg('kind')::"JobKind", 'DEFAULT')
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
    "method"
) VALUES (
    @workflowTriggersId::uuid,
    @cronTrigger::text,
    sqlc.narg('name')::text,
    sqlc.narg('input')::jsonb,
    sqlc.narg('additionalMetadata')::jsonb,
    gen_random_uuid(),
    COALESCE(sqlc.narg('method')::"WorkflowTriggerCronRefMethods", 'DEFAULT')
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

-- name: UpsertDesiredWorkerLabel :one
INSERT INTO "StepDesiredWorkerLabel" (
    "createdAt",
    "updatedAt",
    "stepId",
    "key",
    "intValue",
    "strValue",
    "required",
    "weight",
    "comparator"
) VALUES (
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @stepId::uuid,
    @key::text,
    COALESCE(sqlc.narg('intValue')::int, NULL),
    COALESCE(sqlc.narg('strValue')::text, NULL),
    COALESCE(sqlc.narg('required')::boolean, false),
    COALESCE(sqlc.narg('weight')::int, 100),
    COALESCE(sqlc.narg('comparator')::"WorkerLabelComparator", 'EQUAL')
) ON CONFLICT ("stepId", "key") DO UPDATE
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "intValue" = COALESCE(sqlc.narg('intValue')::int, null),
    "strValue" = COALESCE(sqlc.narg('strValue')::text, null),
    "required" = COALESCE(sqlc.narg('required')::boolean, false),
    "weight" = COALESCE(sqlc.narg('weight')::int, 100),
    "comparator" = COALESCE(sqlc.narg('comparator')::"WorkerLabelComparator", 'EQUAL')
RETURNING *;

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

-- name: CreateStepConcurrency :one
INSERT INTO v1_step_concurrency (
    workflow_id,
    workflow_version_id,
    step_id,
    strategy,
    expression,
    tenant_id,
    max_concurrency
)
VALUES (
    @workflowId::uuid,
    @workflowVersionId::uuid,
    @stepId::uuid,
    @strategy::text,
    @expression::text,
    @tenantId::uuid,
    @maxConcurrency::integer
) RETURNING *;

-- name: CreateStepMatchCondition :one
INSERT INTO v1_step_match_condition (
    tenant_id,
    step_id,
    readable_data_key,
    action,
    or_group_id,
    expression,
    kind,
    sleep_duration,
    event_key,
    parent_readable_id
)
VALUES (
    @tenantId::uuid,
    @stepId::uuid,
    @readableDataKey::text,
    @action::v1_match_condition_action,
    @orGroupId::uuid,
    sqlc.narg('expression')::text,
    @kind::v1_step_match_condition_kind,
    sqlc.narg('sleepDuration')::text,
    sqlc.narg('eventKey')::text,
    sqlc.narg('parentReadableId')::text
) RETURNING *;

-- name: ListStepMatchConditions :many
SELECT
    *
FROM
    v1_step_match_condition
WHERE
    step_id = ANY(@stepIds::uuid[])
    AND tenant_id = @tenantId::uuid;
