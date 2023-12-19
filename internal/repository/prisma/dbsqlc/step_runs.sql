-- name: UpdateStepRun :one
UPDATE
    "StepRun"
SET
    "requeueAfter" = COALESCE(sqlc.narg('requeueAfter')::timestamp, "requeueAfter"),
    "startedAt" = COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt"),
    "finishedAt" = COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt"),
    "status" = COALESCE(sqlc.narg('status'), "status"),
    "input" = COALESCE(sqlc.narg('input')::jsonb, "input"),
    "output" = COALESCE(sqlc.narg('output')::jsonb, "output"),
    "error" = COALESCE(sqlc.narg('error')::text, "error"),
    "cancelledAt" = COALESCE(sqlc.narg('cancelledAt')::timestamp, "cancelledAt"),
    "cancelledReason" = COALESCE(sqlc.narg('cancelledReason')::text, "cancelledReason")
WHERE 
  "id" = @id::uuid AND
  "tenantId" = @tenantId::uuid
RETURNING "StepRun".*;

-- name: ResolveLaterStepRuns :many
WITH currStepRun AS (
  SELECT *
  FROM "StepRun"
  WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid
)
UPDATE
    "StepRun" as sr
SET "status" = CASE
    -- When the given step run has failed or been cancelled, then all later step runs are cancelled
    WHEN (cs."status" = 'FAILED' OR cs."status" = 'CANCELLED') THEN 'CANCELLED'
    ELSE sr."status"
    END,
    -- When the previous step run timed out, the cancelled reason is set
    "cancelledReason" = CASE
    WHEN (cs."status" = 'CANCELLED' AND cs."cancelledReason" = 'TIMED_OUT'::text) THEN 'PREVIOUS_STEP_TIMED_OUT'
    WHEN (cs."status" = 'CANCELLED') THEN 'PREVIOUS_STEP_CANCELLED'
    ELSE NULL
    END
FROM
    currStepRun cs
WHERE
    sr."jobRunId" = (
        SELECT "jobRunId"
        FROM "StepRun"
        WHERE "id" = @stepRunId::uuid
    ) AND
    sr."order" > (
        SELECT "order"
        FROM "StepRun"
        WHERE "id" = @stepRunId::uuid
    ) AND
    sr."tenantId" = @tenantId::uuid
RETURNING sr.*;