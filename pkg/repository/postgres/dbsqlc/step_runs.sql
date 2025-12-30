-- name: ListStepRunExpressionEvals :many
SELECT
    *
FROM
    "StepRunExpressionEval" sre
WHERE
    "stepRunId" = ANY(@stepRunIds::uuid[]);

-- name: CreateStepRunExpressionEvalStrs :exec
INSERT INTO "StepRunExpressionEval" (
    "key",
    "stepRunId",
    "valueStr",
    "kind"
) VALUES (
    unnest(@keys::text[]),
    @stepRunId::uuid,
    unnest(@valuesStr::text[]),
    unnest(cast(@kinds::text[] as"StepExpressionKind"[]))
) ON CONFLICT ("key", "stepRunId", "kind") DO UPDATE
SET
    "valueStr" = EXCLUDED."valueStr",
    "valueInt" = EXCLUDED."valueInt";

-- name: CreateStepRunExpressionEvalInts :exec
INSERT INTO "StepRunExpressionEval" (
    "key",
    "stepRunId",
    "valueInt",
    "kind"
) VALUES (
    unnest(@keys::text[]),
    @stepRunId::uuid,
    unnest(@valuesInt::int[]),
    unnest(cast(@kinds::text[] as"StepExpressionKind"[]))
) ON CONFLICT ("key", "stepRunId", "kind") DO UPDATE
SET
    "valueStr" = EXCLUDED."valueStr",
    "valueInt" = EXCLUDED."valueInt";

-- name: GetStepExpressions :many
SELECT
    *
FROM
    "StepExpression"
WHERE
    "stepId" = @stepId::uuid;

-- name: CheckWorker :one
SELECT
    "id"
FROM
    "Worker"
WHERE
    "tenantId" = @tenantId::uuid
    AND "dispatcherId" IS NOT NULL
    AND "isActive" = true
    AND "isPaused" = false
    AND "lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND "id" = @workerId::uuid;

-- name: GetWorkerDispatcherActions :many
WITH actions AS (
    SELECT
        "id",
        "actionId"
    FROM
        "Action"
    WHERE
        "tenantId" = @tenantId::uuid AND
        "actionId" = ANY(@actionIds::text[])
)
SELECT
    w."id",
    a."actionId",
    w."dispatcherId"
FROM
    "Worker" w
JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
JOIN
    actions a ON atw."A" = a."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false;

-- name: GetDesiredLabels :many
SELECT
    "key",
    "strValue",
    "intValue",
    "required",
    "weight",
    "comparator",
    "stepId"
FROM
    "StepDesiredWorkerLabel"
WHERE
    "stepId" = ANY(@stepIds::uuid[]);

-- name: GetWorkerLabels :many
SELECT
    "key",
    "strValue",
    "intValue"
FROM
    "WorkerLabel"
WHERE
    "workerId" = @workerId::uuid;

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

-- name: GetStepDesiredWorkerLabels :one
SELECT
    jsonb_agg(
        jsonb_build_object(
            'key', dwl."key",
            'strValue', dwl."strValue",
            'intValue', dwl."intValue",
            'required', dwl."required",
            'weight', dwl."weight",
            'comparator', dwl."comparator",
            'is_true', false
        )
    ) AS desired_labels
FROM
    "StepDesiredWorkerLabel" dwl
WHERE
    dwl."stepId" = @stepId::uuid;

-- name: HasActiveWorkersForActionId :one
SELECT
    COUNT(DISTINCT w."id") AS "total"
FROM
    "Worker" w
JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
JOIN
    "Action" a ON atw."A" = a."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND a."actionId" = @actionId::text
    AND w."isActive" = true
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '6 seconds';
