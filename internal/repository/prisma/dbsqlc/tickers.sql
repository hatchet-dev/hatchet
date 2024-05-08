-- name: CreateTicker :one
INSERT INTO
    "Ticker" ("id", "lastHeartbeatAt", "isActive")
VALUES
    (sqlc.arg('id')::uuid, CURRENT_TIMESTAMP, 't')
RETURNING *;

-- name: ListNewlyStaleTickers :many
SELECT
    sqlc.embed(tickers)
FROM "Ticker" as tickers
WHERE
    -- last heartbeat older than 15 seconds
    "lastHeartbeatAt" < NOW () - INTERVAL '15 seconds'
    -- active
    AND "isActive" = true;

-- name: ListActiveTickers :many
SELECT
    sqlc.embed(tickers)
FROM "Ticker" as tickers
WHERE
    -- last heartbeat greater than 15 seconds
    "lastHeartbeatAt" > NOW () - INTERVAL '15 seconds'
    -- active
    AND "isActive" = true;

-- name: SetTickersInactive :many
UPDATE
    "Ticker" as tickers
SET
    "isActive" = false
WHERE
    "id" = ANY (sqlc.arg('ids')::uuid[])
RETURNING
    sqlc.embed(tickers);

-- name: ListTickers :many
SELECT
    *
FROM
    "Ticker" as tickers
WHERE
    (
        sqlc.arg('isActive')::boolean IS NULL OR
        "isActive" = sqlc.arg('isActive')::boolean
    )
    AND
    (
        sqlc.arg('lastHeartbeatAfter')::timestamp IS NULL OR
        tickers."lastHeartbeatAt" > sqlc.narg('lastHeartbeatAfter')::timestamp
    );

-- name: DeleteTicker :one
DELETE FROM
    "Ticker" as tickers
WHERE
    "id" = sqlc.arg('id')::uuid
RETURNING *;

-- name: UpdateTicker :one
UPDATE
    "Ticker" as tickers
SET
    "lastHeartbeatAt" = sqlc.arg('lastHeartbeatAt')::timestamp
WHERE
    "id" = sqlc.arg('id')::uuid
RETURNING *;

-- name: PollStepRuns :many
WITH stepRunsToTimeout AS (
    SELECT
        stepRun."id"
    FROM
        "StepRun" as stepRun
    WHERE
        ("status" = 'RUNNING' OR "status" = 'ASSIGNED')
        AND "timeoutAt" < NOW()
        AND (
            NOT EXISTS (
                SELECT 1 FROM "Ticker" WHERE "id" = stepRun."tickerId" AND "isActive" = true AND "lastHeartbeatAt" >= NOW() - INTERVAL '10 seconds'
            )
            OR "tickerId" IS NULL 
        )
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "StepRun" as stepRuns
SET
    "tickerId" = @tickerId::uuid
FROM
    stepRunsToTimeout
WHERE
    stepRuns."id" = stepRunsToTimeout."id"
RETURNING stepRuns.*;

-- name: PollGetGroupKeyRuns :many
WITH getGroupKeyRunsToTimeout AS (
    SELECT
        getGroupKeyRun."id"
    FROM
        "GetGroupKeyRun" as getGroupKeyRun
    WHERE
        ("status" = 'RUNNING' OR "status" = 'ASSIGNED')
        AND "timeoutAt" < NOW()
        AND (
            NOT EXISTS (
                SELECT 1 FROM "Ticker" WHERE "id" = getGroupKeyRun."tickerId" AND "isActive" = true AND "lastHeartbeatAt" >= NOW() - INTERVAL '10 seconds'
            )
            OR "tickerId" IS NULL 
        )
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "GetGroupKeyRun" as getGroupKeyRuns
SET
    "tickerId" = @tickerId::uuid
FROM
    getGroupKeyRunsToTimeout
WHERE
    getGroupKeyRuns."id" = getGroupKeyRunsToTimeout."id"
RETURNING getGroupKeyRuns.*;

-- name: PollCronSchedules :many
WITH latest_workflow_versions AS (
    SELECT
        "workflowId",
        MAX("order") as max_order
    FROM
        "WorkflowVersion"
    GROUP BY "workflowId"
),
active_cron_schedules AS (
    SELECT
        cronSchedule."parentId",
        versions."id" AS "workflowVersionId",
        triggers."tenantId" AS "tenantId"
    FROM
        "WorkflowTriggerCronRef" as cronSchedule
    JOIN 
        "WorkflowTriggers" as triggers ON triggers."id" = cronSchedule."parentId"
    JOIN
        "WorkflowVersion" as versions ON versions."id" = triggers."workflowVersionId"
    JOIN 
        latest_workflow_versions l ON versions."workflowId" = l."workflowId" AND versions."order" = l.max_order
    WHERE
        "enabled" = TRUE AND
        ("tickerId" IS NULL 
        OR NOT EXISTS (
            SELECT 1 FROM "Ticker" WHERE "id" = cronSchedule."tickerId" AND "isActive" = true AND "lastHeartbeatAt" >= NOW() - INTERVAL '10 seconds'
        )
        OR "tickerId" = @tickerId::uuid)
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "WorkflowTriggerCronRef" as cronSchedules
SET
    "tickerId" = @tickerId::uuid
FROM
    active_cron_schedules
WHERE
    cronSchedules."parentId" = active_cron_schedules."parentId"
RETURNING cronSchedules.*, active_cron_schedules."workflowVersionId", active_cron_schedules."tenantId";

-- name: PollScheduledWorkflows :many
-- Finds workflows that are either past their execution time or will be in the next 5 seconds and assigns them
-- to a ticker, or finds workflows that were assigned to a ticker that is no longer active
WITH latest_workflow_versions AS (
    SELECT
        "workflowId",
        MAX("order") as max_order
    FROM
        "WorkflowVersion"
    GROUP BY "workflowId"
),
not_run_scheduled_workflows AS (
    SELECT
        scheduledWorkflow."id",
        versions."id" AS "workflowVersionId",
        workflow."tenantId" AS "tenantId"
    FROM
        "WorkflowTriggerScheduledRef" as scheduledWorkflow
    JOIN
        "WorkflowVersion" as versions ON versions."id" = scheduledWorkflow."parentId"
    JOIN 
        latest_workflow_versions l ON versions."workflowId" = l."workflowId" AND versions."order" = l.max_order
    JOIN
        "Workflow" as workflow ON workflow."id" = versions."workflowId"
    LEFT JOIN
        "WorkflowRunTriggeredBy" as runTriggeredBy ON runTriggeredBy."scheduledId" = scheduledWorkflow."id"
    WHERE
        "triggerAt" <= NOW() + INTERVAL '5 seconds'
        AND runTriggeredBy IS NULL
        AND (
            "tickerId" IS NULL 
            OR NOT EXISTS (
                SELECT 1 FROM "Ticker" WHERE "id" = scheduledWorkflow."tickerId" AND "isActive" = true AND "lastHeartbeatAt" >= NOW() - INTERVAL '10 seconds'
            )
            OR "tickerId" = @tickerId::uuid
        )
),
active_scheduled_workflows AS (
    SELECT
        *
    FROM
        not_run_scheduled_workflows
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "WorkflowTriggerScheduledRef" as scheduledWorkflows
SET
    "tickerId" = @tickerId::uuid
FROM
    active_scheduled_workflows
WHERE
    scheduledWorkflows."id" = active_scheduled_workflows."id"
RETURNING scheduledWorkflows.*, active_scheduled_workflows."workflowVersionId", active_scheduled_workflows."tenantId";

-- name: PollTenantAlerts :many
-- Finds tenant alerts which haven't alerted since their frequency and assigns them to a ticker
WITH active_tenant_alerts AS (
    SELECT
        alerts.*
    FROM
        "TenantAlertingSettings" as alerts
    -- only return alerts which have a slack webhook enabled
    JOIN
        "SlackAppWebhook" as webhooks ON webhooks."tenantId" = alerts."tenantId"
    WHERE
        "lastAlertedAt" IS NULL OR
        "lastAlertedAt" <= NOW() - convert_duration_to_interval(alerts."maxFrequency")
    FOR UPDATE SKIP LOCKED
),
failed_run_count_by_tenant AS (
    SELECT
        workflowRun."tenantId",
        COUNT(*) as "failedWorkflowRunCount"
    FROM
        "WorkflowRun" as workflowRun
    JOIN
        active_tenant_alerts ON active_tenant_alerts."tenantId" = workflowRun."tenantId"
    WHERE
        "status" = 'FAILED'
        AND (
            "lastAlertedAt" IS NULL OR
            workflowRun."finishedAt" >= "lastAlertedAt"
        )
    GROUP BY workflowRun."tenantId"
)
UPDATE
    "TenantAlertingSettings" as alerts
SET
    "tickerId" = @tickerId::uuid,
    "lastAlertedAt" = NOW()
FROM
    active_tenant_alerts
WHERE
    alerts."id" = active_tenant_alerts."id" AND
    alerts."tenantId" IN (SELECT "tenantId" FROM failed_run_count_by_tenant WHERE "failedWorkflowRunCount" > 0)
RETURNING alerts.*, active_tenant_alerts."lastAlertedAt" AS "prevLastAlertedAt";