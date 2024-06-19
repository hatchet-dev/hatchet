-- name: UpdateJobRunStatus :one
UPDATE "JobRun"
SET "status" = @status::"JobRunStatus"
WHERE "id" = @id::uuid AND "tenantId" = @tenantId::uuid
RETURNING *;

-- name: ResolveJobRunStatus :one
WITH stepRuns AS (
    SELECT sum(case when runs."status" IN ('PENDING', 'PENDING_ASSIGNMENT') then 1 else 0 end) AS pendingRuns,
        sum(case when runs."status" IN ('RUNNING', 'ASSIGNED') then 1 else 0 end) AS runningRuns,
        sum(case when runs."status" = 'SUCCEEDED' then 1 else 0 end) AS succeededRuns,
        sum(case when runs."status" = 'FAILED' then 1 else 0 end) AS failedRuns,
        sum(case when runs."status" = 'CANCELLED' then 1 else 0 end) AS cancelledRuns
    FROM "StepRun" as runs
    WHERE
        "jobRunId" = (
            SELECT "jobRunId"
            FROM "StepRun"
            WHERE "id" = @stepRunId::uuid
        ) AND
        "tenantId" = @tenantId::uuid
)
UPDATE "JobRun"
SET "status" = CASE
    -- Final states are final, cannot be updated
    WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
    -- NOTE: Order of the following conditions is important
    -- When one step run is running, then the job is running
    WHEN (s.runningRuns > 0 OR s.pendingRuns > 0) THEN 'RUNNING'
    -- When one step run has failed, then the job is failed
    WHEN s.failedRuns > 0 THEN 'FAILED'
    -- When one step run has been cancelled, then the job is cancelled
    WHEN s.cancelledRuns > 0 THEN 'CANCELLED'
    -- When no step runs exist that are not succeeded, then the job is succeeded
    WHEN s.succeededRuns > 0 AND s.pendingRuns = 0 AND s.runningRuns = 0 AND s.failedRuns = 0 AND s.cancelledRuns = 0 THEN 'SUCCEEDED'
    ELSE "status"
END, "finishedAt" = CASE
    -- Final states are final, cannot be updated
    WHEN "finishedAt" IS NOT NULL THEN "finishedAt"
    WHEN s.runningRuns > 0 THEN NULL
    -- When one step run has failed or been cancelled, then the job is finished
    WHEN s.failedRuns > 0 OR s.cancelledRuns > 0 THEN NOW()
    -- When no step runs exist that are not succeeded, then the job is finished
    WHEN s.succeededRuns > 0 AND s.pendingRuns = 0 AND s.runningRuns = 0 AND s.failedRuns = 0 AND s.cancelledRuns = 0 THEN NOW()
    ELSE "finishedAt"
END, "startedAt" = CASE
    -- Started at is final, cannot be changed
    WHEN "startedAt" IS NOT NULL THEN "startedAt"
    -- If steps are running (or have finished), then set the started at time
    WHEN s.runningRuns > 0 OR s.succeededRuns > 0 OR s.failedRuns > 0 AND s.cancelledRuns > 0 THEN NOW()
    ELSE "startedAt"
END
FROM stepRuns s
WHERE "id" = (
    SELECT "jobRunId"
    FROM "StepRun"
    WHERE "id" = @stepRunId::uuid
) AND "tenantId" = @tenantId::uuid
RETURNING "JobRun".*;

-- name: UpsertJobRunLookupData :exec
INSERT INTO "JobRunLookupData" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "jobRunId",
    "tenantId",
    "data"
) VALUES (
    gen_random_uuid(), -- Generates a new UUID for id
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    @jobRunId::uuid,
    @tenantId::uuid,
    jsonb_set('{}', @fieldPath::text[], @jsonData::jsonb, true)
) ON CONFLICT ("jobRunId", "tenantId") DO UPDATE
SET
    "data" = jsonb_set("JobRunLookupData"."data", @fieldPath::text[], @jsonData::jsonb, true),
    "updatedAt" = CURRENT_TIMESTAMP;

-- name: UpdateJobRunLookupDataWithStepRun :exec
WITH readable_id AS (
    SELECT "readableId"
    FROM "Step"
    WHERE "id" = (
        SELECT "stepId"
        FROM "StepRun"
        WHERE "id" = @stepRunId::uuid
    )
)
UPDATE "JobRunLookupData"
SET
    "data" = CASE
        WHEN @jsonData::jsonb IS NULL THEN
            jsonb_set(
                "data",
                '{steps}',
                ("data"->'steps') - (SELECT "readableId" FROM readable_id),
                true
            )
        ELSE
            jsonb_set(
                "data",
                ARRAY['steps', (SELECT "readableId" FROM readable_id)],
                @jsonData::jsonb,
                true
            )
    END,
    "updatedAt" = CURRENT_TIMESTAMP
WHERE
    "jobRunId" = (
        SELECT "jobRunId"
        FROM "StepRun"
        WHERE "id" = @stepRunId::uuid
    )
    AND "tenantId" = @tenantId::uuid;

-- name: ListJobRunsForWorkflowRun :many
SELECT
    "id",
    "jobId"
FROM
    "JobRun" jr
WHERE
    jr."workflowRunId" = @workflowRunId::uuid;

-- name: GetJobRunByWorkflowRunIdAndJobId :one
SELECT
    "id",
    "jobId",
    "status"
FROM
    "JobRun" jr
WHERE
    jr."tenantId" = @tenantId::uuid
    AND jr."workflowRunId" = @workflowRunId::uuid
    AND jr."jobId" = @jobId::uuid;
