-- name: UpdateJobRunStatus :one
UPDATE "JobRun"
SET "status" = @status::"JobRunStatus"
WHERE "id" = @id::uuid AND "tenantId" = @tenantId::uuid
RETURNING *;

-- name: ResolveJobRunStatus :many
WITH stepRuns AS (
    SELECT
        runs."jobRunId",
        sum(case when runs."status" IN ('PENDING', 'PENDING_ASSIGNMENT') then 1 else 0 end) AS pendingRuns,
        sum(case when runs."status" = 'BACKOFF' then 1 else 0 end) AS backoffRuns,
        sum(case when runs."status" IN ('RUNNING', 'ASSIGNED') then 1 else 0 end) AS runningRuns,
        sum(case when runs."status" = 'SUCCEEDED' then 1 else 0 end) AS succeededRuns,
        sum(case when runs."status" = 'FAILED' then 1 else 0 end) AS failedRuns,
        sum(case when runs."status" = 'CANCELLED' then 1 else 0 end) AS cancelledRuns
    FROM "StepRun" as runs
    WHERE
        "jobRunId" = ANY(
            SELECT "jobRunId"
            FROM "StepRun"
            WHERE "id" = ANY(@stepRunIds::uuid[])
        )
    GROUP BY runs."jobRunId"
)
UPDATE "JobRun"
SET "status" = CASE
    -- Final states are final, cannot be updated
    WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
    -- NOTE: Order of the following conditions is important
    -- When one step run is backoff AND no other step runs are running, then the job is backoff
    WHEN s.backoffRuns > 0 AND s.runningRuns = 0 THEN 'BACKOFF'
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
WHERE
    "id" = s."jobRunId"
RETURNING "JobRun"."id";

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

-- name: ListJobRunsForWorkflowRunFull :many
WITH steps AS (
    SELECT
        "id",
        "jobId",
        "status"
    FROM
        "JobRun" jr
    WHERE
        jr."workflowRunId" = @workflowRunId::uuid
)
SELECT
    jr.*,
    sqlc.embed(j)
FROM "JobRun" jr
JOIN "Job" j
    ON jr."jobId" = j."id"
WHERE jr."workflowRunId" = @workflowRunId::uuid
    AND jr."tenantId" = @tenantId::uuid;

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

-- name: GetJobRunsByWorkflowRunId :many

SELECT
    "id",
    "jobId",
    "status"
FROM
    "JobRun" jr
WHERE
    jr."workflowRunId" = @workflowRunId::uuid
    AND jr."tenantId" = @tenantId::uuid;


-- name: ClearJobRunLookupData :one
WITH for_delete AS (
    SELECT
        jrld2."id" as "id"
    FROM "JobRun" jr2
    LEFT JOIN "JobRunLookupData" jrld2 ON jr2."id" = jrld2."jobRunId"
    WHERE
        jr2."tenantId" = @tenantId::uuid AND
        jr2."deletedAt" IS NOT NULL AND
        jrld2."data" IS NOT NULL
    ORDER BY jr2."deletedAt" ASC
    LIMIT sqlc.arg('limit') + 1
),
deleted_with_limit AS (
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
    "JobRunLookupData"
SET
    "data" = NULL
WHERE
    "id" IN (SELECT "id" FROM deleted_with_limit)
RETURNING
    (SELECT has_more FROM has_more) as has_more;
