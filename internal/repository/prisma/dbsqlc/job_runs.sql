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
    -- When one step run has failed, then the job is failed
    WHEN s.failedRuns > 0 THEN 'FAILED'
    -- When one step run is running, then the job is running
    WHEN s.runningRuns > 0 THEN 'RUNNING'
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
    -- If a step is running, then the job has started
    WHEN s.runningRuns > 0 THEN NOW()
    ELSE "startedAt"
END
FROM stepRuns s
WHERE "id" = (
    SELECT "jobRunId"
    FROM "StepRun"
    WHERE "id" = @stepRunId::uuid
) AND "tenantId" = @tenantId::uuid
RETURNING "JobRun".*;

-- name: UpdateJobRun :one
UPDATE
  "JobRun"
SET "status" = CASE 
    -- Final states are final, cannot be updated
    WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
    ELSE "status" = COALESCE(sqlc.narg('status'), "status")
END
WHERE
    "id" = @id::uuid AND
    "tenantId" = @tenantId::uuid
RETURNING "JobRun".*;
