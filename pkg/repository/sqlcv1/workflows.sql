-- name: ListStepsByWorkflowVersionIds :many
WITH steps AS (
    SELECT
        s.*,
        wv."id" as "workflowVersionId",
        w."name" as "workflowName",
        w."id" as "workflowId",
        j."kind" as "jobKind",
        COUNT(mc.id) as "matchConditionCount"
    FROM
        "WorkflowVersion" as wv
    JOIN
        "Workflow" as w ON w."id" = wv."workflowId"
    JOIN
        "Job" j ON j."workflowVersionId" = wv."id"
    JOIN
        "Step" s ON s."jobId" = j."id"
    LEFT JOIN
        v1_step_match_condition mc ON mc.step_id = s."id"
    WHERE
        wv."id" = ANY(@ids::uuid[])
        AND w."tenantId" = @tenantId::uuid
        AND w."deletedAt" IS NULL
        AND wv."deletedAt" IS NULL
    GROUP BY
        s."id", wv."id", w."name", w."id", j."kind"
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
    COALESCE(wv."defaultPriority", 1) AS "defaultPriority",
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
    "kind",
    "defaultPriority",
    "createWorkflowVersionOpts",
    "inputJsonSchema"
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
    coalesce(sqlc.narg('kind')::"WorkflowKind", 'DAG'),
    sqlc.narg('defaultPriority') :: integer,
    sqlc.narg('createWorkflowVersionOpts')::jsonb,
    sqlc.narg('inputJsonSchema')::jsonb
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
WITH previous_trigger AS (
    SELECT "enabled"
    FROM "WorkflowTriggerCronRef" c
    JOIN "WorkflowTriggers" t ON t."id" = c."parentId"
    JOIN "WorkflowVersion" wv ON wv."id" = t."workflowVersionId"
    WHERE
        wv."id" = sqlc.narg('oldWorkflowVersionId')::uuid
        AND "cron" = @cronTrigger::text
        AND "name" = sqlc.narg('name')::text
)

INSERT INTO "WorkflowTriggerCronRef" (
    "parentId",
    "cron",
    "name",
    "input",
    "additionalMetadata",
    "id",
    "method",
    "priority",
    "enabled"
)
SELECT
    @workflowTriggersId::uuid AS "parentId",
    @cronTrigger::text AS "cron",
    sqlc.narg('name')::text AS "name",
    sqlc.narg('input')::jsonb AS "input",
    sqlc.narg('additionalMetadata')::jsonb AS "additionalMetadata",
    gen_random_uuid() AS "id",
    COALESCE(sqlc.narg('method')::"WorkflowTriggerCronRefMethods", 'DEFAULT') AS "method",
    COALESCE(sqlc.narg('priority')::integer, 1) AS "priority",
    COALESCE((SELECT "enabled" FROM previous_trigger), true) AS "enabled"
RETURNING *;

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
    "retryMaxBackoff",
    "isDurable"
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
    sqlc.narg('retryMaxBackoff'),
    coalesce(sqlc.narg('isDurable')::boolean, false)
) RETURNING *;

-- name: CreateStepSlotRequests :exec
INSERT INTO v1_step_slot_request (
    tenant_id,
    step_id,
    slot_type,
    units,
    created_at,
    updated_at
)
SELECT
    @tenantId::uuid,
    @stepId::uuid,
    unnest(@slotTypes::text[]),
    unnest(@units::integer[]),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
;

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
    SELECT
        cronTrigger."id",
        cronTrigger."enabled"
    FROM "WorkflowTriggerCronRef" cronTrigger
    JOIN "WorkflowTriggers" triggers ON triggers."id" = cronTrigger."parentId"
    WHERE triggers."workflowVersionId" = @oldWorkflowVersionId::uuid
    AND cronTrigger."method" = 'API'
)
UPDATE "WorkflowTriggerCronRef"
SET
    "parentId" = @newWorkflowTriggerId::uuid,
    "enabled" = triggersToUpdate."enabled"
FROM triggersToUpdate
WHERE "WorkflowTriggerCronRef"."id" = triggersToUpdate."id"
;

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
SELECT DISTINCT ON (wv."workflowId")
    wv."id"
FROM "WorkflowVersion" wv
INNER JOIN "Workflow" w ON w."id" = wv."workflowId"
WHERE
    w."tenantId" = @tenantId::uuid AND
    wv."workflowId" = ANY(@workflowIds::uuid[]) AND
    w."deletedAt" IS NULL AND
    wv."deletedAt" IS NULL
ORDER BY
    wv."workflowId",
    wv."order" DESC;

-- name: CreateWorkflowConcurrencyV1 :one
WITH inserted_wcs AS (
    INSERT INTO v1_workflow_concurrency (
      workflow_id,
      workflow_version_id,
      strategy,
      expression,
      tenant_id,
      max_concurrency
    )
    VALUES (
      @workflowId::uuid,
      @workflowVersionId::uuid,
      @limitStrategy::v1_concurrency_strategy,
      @expression,
      @tenantId::uuid,
      COALESCE(sqlc.narg('maxRuns')::integer, 1)
    )
    RETURNING *
), inserted_scs AS (
    INSERT INTO v1_step_concurrency (
      parent_strategy_id,
      workflow_id,
      workflow_version_id,
      step_id,
      strategy,
      expression,
      tenant_id,
      max_concurrency
    )
    SELECT
      wcs.id,
      s."workflowId",
      s."workflowVersionId",
      s."id",
      @limitStrategy::v1_concurrency_strategy,
      @expression,
      s."tenantId",
      COALESCE(sqlc.narg('maxRuns')::integer, 1)
    FROM (
        SELECT
          s."id",
          wf."id" AS "workflowId",
          wv."id" AS "workflowVersionId",
          wf."tenantId"
        FROM "Step" s
        JOIN "Job" j ON s."jobId" = j."id"
        JOIN "WorkflowVersion" wv ON j."workflowVersionId" = wv."id"
        JOIN "Workflow" wf ON wv."workflowId" = wf."id"
        WHERE
          wv."id" = @workflowVersionId::uuid
          AND j."kind" = 'DEFAULT'
    ) s, inserted_wcs wcs
    RETURNING *
)
SELECT
    wcs.id,
    ARRAY_AGG(scs.id)::bigint[] AS child_strategy_ids
FROM
    inserted_wcs wcs
JOIN
    inserted_scs scs ON scs.parent_strategy_id = wcs.id
GROUP BY
    wcs.id;

-- name: UpdateWorkflowConcurrencyWithChildStrategyIds :exec
-- Update the workflow concurrency row using its primary key.
UPDATE v1_workflow_concurrency
SET child_strategy_ids = @childStrategyIds::bigint[]
WHERE workflow_id = @workflowId::uuid
  AND workflow_version_id = @workflowVersionId::uuid
  AND id = @workflowConcurrencyId::bigint;

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
    @strategy::v1_concurrency_strategy,
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

-- name: GetWorkflowShape :many
SELECT
    s.id AS parentStepId,
    s."readableId" AS stepName,
    array_remove(ARRAY_AGG(so."B"), NULL)::uuid[] AS childrenStepIds
FROM "WorkflowVersion" v
JOIN "Job" j ON v."id" = j."workflowVersionId"
JOIN "Step" s ON j."id" = s."jobId"
LEFT JOIN "_StepOrder" so ON so."A" = s.id
WHERE v.id = @workflowVersionId::uuid
GROUP BY s.id, s."readableId"
;

-- name: GetStepsForJobs :many
SELECT
	j."id" as "jobId",
    sqlc.embed(s),
    (
        SELECT array_agg(so."A")::uuid[]  -- Casting the array_agg result to uuid[]
        FROM "_StepOrder" so
        WHERE so."B" = s."id"
    ) AS "parents"
FROM "Job" j
JOIN "Step" s ON s."jobId" = j."id"
WHERE
    j."id" = ANY(@jobIds::uuid[])
    AND j."tenantId" = @tenantId::uuid
    AND j."deletedAt" IS NULL;

-- name: UpdateCronTrigger :exec
UPDATE "WorkflowTriggerCronRef"
SET
    "enabled" = COALESCE(sqlc.narg('enabled')::BOOLEAN, "enabled")
WHERE "id" = @cronTriggerId::uuid
;

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

-- name: GetWorkflowVersionById :one
SELECT
    sqlc.embed(wv),
    sqlc.embed(w),
    wc.id as "concurrencyId",
    wc.max_concurrency as "concurrencyMaxRuns",
    wc.strategy as "concurrencyLimitStrategy",
    wc.expression as "concurrencyExpression"
FROM
    "WorkflowVersion" as wv
JOIN "Workflow" as w on w."id" = wv."workflowId"
LEFT JOIN v1_workflow_concurrency as wc ON (wc.workflow_version_id, wc.workflow_id) = (wv."id", w."id")
WHERE
    wv."id" = @id::uuid AND
    wv."deletedAt" IS NULL
LIMIT 1;

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
        workflows.name iLIKE concat('%', sqlc.narg('search')::TEXT, '%')
    )
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN workflows."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' then workflows."createdAt" END DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: CountWorkflows :one
SELECT COUNT(w.*)
FROM "Workflow" w
WHERE
    w."tenantId" = $1
    AND w."deletedAt" IS NULL
    AND (
        sqlc.narg('eventKey')::TEXT IS NULL OR
        w."id" IN (
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
                            "WorkflowTriggerEventRef" AS t3
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
    AND (
        sqlc.narg('search')::TEXT IS NULL
        OR w.name ILIKE CONCAT('%', sqlc.narg('search')::TEXT, '%')
    )
;

-- name: UpdateWorkflow :one
UPDATE "Workflow"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "isPaused" = coalesce(sqlc.narg('isPaused')::boolean, "isPaused")
WHERE "id" = @id::uuid
RETURNING *;

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
