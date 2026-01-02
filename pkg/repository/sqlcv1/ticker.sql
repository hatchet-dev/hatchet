-- name: IsTenantAlertActive :one
WITH active_setting AS (
    SELECT
        1
    FROM
        "TenantAlertingSettings" as alerts
    WHERE
        "tenantId" = @tenantId::uuid AND
        (
            "lastAlertedAt" IS NULL OR
            "lastAlertedAt" <= NOW() - convert_duration_to_interval(alerts."maxFrequency")
        ) AND
        "enableWorkflowRunFailureAlerts" = true
)
SELECT
    EXISTS (
        select 1 from active_setting
    ) as "isActive",
    "lastAlertedAt" as "lastAlertedAt"
FROM
    "TenantAlertingSettings" as alerts
WHERE
    "tenantId" = @tenantId::uuid;

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

-- name: DeactivateTicker :one
UPDATE
    "Ticker" t
SET
    "isActive" = false
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

-- name: PollCronSchedules :many
WITH latest_workflow_versions AS (
    SELECT
        "workflowId",
        MAX("order") as max_order
    FROM
        "WorkflowVersion"
    WHERE
        "deletedAt" IS NULL
    GROUP BY "workflowId"
),
eligible_cron_with_versions AS (
    SELECT
        cronSchedule."parentId",
        cronSchedule."cron",
        cronSchedule."name",
        triggers."workflowVersionId",
        triggers."tenantId",
        versions."workflowId",
        versions."order"
    FROM
        "WorkflowTriggerCronRef" as cronSchedule
    JOIN
        "WorkflowTriggers" as triggers ON triggers."id" = cronSchedule."parentId"
    JOIN
        "WorkflowVersion" as versions ON versions."id" = triggers."workflowVersionId"
    WHERE cronSchedule."enabled" = TRUE
        AND versions."deletedAt" IS NULL
        AND (
            cronSchedule."tickerId" IS NULL
            OR NOT EXISTS (
                SELECT 1 FROM "Ticker" WHERE "id" = cronSchedule."tickerId" AND "isActive" = true AND "lastHeartbeatAt" >= NOW() - INTERVAL '10 seconds'
            )
            OR cronSchedule."tickerId" = @tickerId::uuid
        )
    FOR UPDATE OF cronSchedule SKIP LOCKED
),
eligible_cron_schedules AS (
    SELECT
        ecv."parentId",
        ecv."cron",
        ecv."name",
        ecv."workflowVersionId",
        ecv."tenantId"
    FROM
        eligible_cron_with_versions as ecv
    JOIN
        latest_workflow_versions as l ON ecv."workflowId" = l."workflowId" AND ecv."order" = l.max_order
)
UPDATE
    "WorkflowTriggerCronRef" as cronSchedules
SET
    "tickerId" = @tickerId::uuid
FROM
    eligible_cron_schedules
WHERE
    cronSchedules."parentId" = eligible_cron_schedules."parentId"
    AND cronSchedules."cron" = eligible_cron_schedules."cron"
    AND cronSchedules."name" = eligible_cron_schedules."name"

RETURNING cronSchedules.*, eligible_cron_schedules."workflowVersionId", eligible_cron_schedules."tenantId";

-- name: PollScheduledWorkflows :many
-- Finds workflows that are either past their execution time or will be in the next 5 seconds and assigns them
-- to a ticker, or finds workflows that were assigned to a ticker that is no longer active
WITH latest_workflow_versions AS (
    SELECT
        DISTINCT ON("workflowId")
        "workflowId",
        "id"
    FROM
        "WorkflowVersion"
    WHERE
        "deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
), not_run_scheduled_workflows AS (
    SELECT
        scheduledWorkflow."id",
        latestVersions."id" AS "workflowVersionId",
        workflow."tenantId" AS "tenantId",
        scheduledWorkflow."additionalMetadata" AS "additionalMetadata"
    FROM
        "WorkflowTriggerScheduledRef" AS scheduledWorkflow
    JOIN
        "WorkflowVersion" AS versions ON versions."id" = scheduledWorkflow."parentId"
    JOIN
        "Workflow" AS workflow ON workflow."id" = versions."workflowId"
    JOIN
        latest_workflow_versions AS latestVersions ON latestVersions."workflowId" = workflow."id"
    LEFT JOIN
        "WorkflowRunTriggeredBy" AS runTriggeredBy ON runTriggeredBy."scheduledId" = scheduledWorkflow."id"
    WHERE
        "triggerAt" <= NOW() + INTERVAL '5 seconds'
        AND runTriggeredBy IS NULL
        AND versions."deletedAt" IS NULL
        AND workflow."deletedAt" IS NULL
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
        AND workflowRun."deletedAt" IS NULL
        AND (
            (
                "lastAlertedAt" IS NULL AND
                workflowRun."finishedAt" >= NOW() - convert_duration_to_interval(active_tenant_alerts."maxFrequency")
            ) OR
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


-- name: PollExpiringTokens :many
WITH expiring_tokens AS (
    SELECT
        t0."id", t0."name", t0."expiresAt"
    FROM
        "APIToken" as t0
    WHERE
        t0."revoked" = false
        AND t0."expiresAt" <= NOW() + INTERVAL '7 days'
        AND t0."expiresAt" >= NOW()
        AND (
            t0."nextAlertAt" IS NULL OR
            t0."nextAlertAt" <= NOW()
        )
        AND t0."internal" = false
    FOR UPDATE SKIP LOCKED
    LIMIT 100
)
UPDATE
    "APIToken" as t1
SET
    "nextAlertAt" = NOW() + INTERVAL '1 day'
FROM
    expiring_tokens
WHERE
    t1."id" = expiring_tokens."id"
RETURNING
    t1."id",
    t1."name",
    t1."tenantId",
    t1."expiresAt";

-- name: PollTenantResourceLimitAlerts :many
WITH alerting_resource_limits AS (
    SELECT
        rl."id" AS "resourceLimitId",
        rl."tenantId",
        rl."resource",
        rl."limitValue",
        rl."alarmValue",
        rl."value",
        rl."window",
        rl."lastRefill",
        CASE
            WHEN rl."value" >= rl."limitValue" THEN 'Exhausted'
            WHEN rl."alarmValue" IS NOT NULL AND rl."value" >= rl."alarmValue" THEN 'Alarm'
        END AS "alertType"
    FROM
        "TenantResourceLimit" AS rl
    JOIN
        "TenantAlertingSettings" AS ta
    ON
        ta."tenantId" = rl."tenantId"::uuid
    WHERE
        ta."enableTenantResourceLimitAlerts" = true
        AND (
            (rl."alarmValue" IS NOT NULL AND rl."value" >= rl."alarmValue")
            OR rl."value" >= rl."limitValue"
        )
    FOR UPDATE SKIP LOCKED
),
new_alerts AS (
    SELECT
        arl."resourceLimitId",
        arl."tenantId",
        arl."resource",
        arl."alertType",
        arl."value",
        arl."limitValue" AS "limit",
        EXISTS (
            SELECT 1
            FROM "TenantResourceLimitAlert" AS trla
            WHERE trla."resourceLimitId" = arl."resourceLimitId"
            AND trla."alertType" = arl."alertType"::"TenantResourceLimitAlertType"
            AND trla."createdAt" >= NOW() - arl."window"::INTERVAL
        ) AS "existingAlert"
    FROM
        alerting_resource_limits AS arl
)
INSERT INTO "TenantResourceLimitAlert" (
    "id",
    "createdAt",
    "updatedAt",
    "resourceLimitId",
    "resource",
    "alertType",
    "value",
    "limit",
    "tenantId"
)
SELECT
    gen_random_uuid(),
    NOW(),
    NOW(),
    na."resourceLimitId",
    na."resource",
    na."alertType"::"TenantResourceLimitAlertType",
    na."value",
    na."limit",
    na."tenantId"
FROM
    new_alerts AS na
WHERE
    na."existingAlert" = false
RETURNING *;
