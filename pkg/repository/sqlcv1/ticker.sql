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
-- Returns every cron schedule this ticker should have scheduled: crons it already owns plus crons it claims
-- on this poll (unassigned, or owned by a ticker that stopped heartbeating). Only the claimed rows are
-- written; already-owned rows are read without an UPDATE so a steady-state poll generates no row churn.
WITH eligible_crons AS (
    SELECT
        cronSchedule."id",
        triggers."workflowVersionId",
        triggers."tenantId"
    FROM
        "WorkflowTriggerCronRef" as cronSchedule
    JOIN
        "WorkflowTriggers" as triggers ON triggers."id" = cronSchedule."parentId"
    JOIN
        "WorkflowVersion" as versions ON versions."id" = triggers."workflowVersionId"
    JOIN
        "Tenant" as tenant ON tenant."id" = triggers."tenantId"
    JOIN
        "Workflow" as workflow ON workflow."id" = versions."workflowId"
    WHERE cronSchedule."enabled" = TRUE
        AND versions."deletedAt" IS NULL
        AND tenant."deletedAt" IS NULL
        AND versions."id" = (
            SELECT latestVersion."id"
            FROM "WorkflowVersion" as latestVersion
            WHERE
                latestVersion."workflowId" = versions."workflowId"
                AND latestVersion."deletedAt" IS NULL
            ORDER BY latestVersion."order" DESC
            LIMIT 1
        )
        AND (
            workflow."isPaused" IS NULL
            OR workflow."isPaused" = FALSE
            OR (
                workflow."isPaused" = TRUE AND workflow."pausedWorkflowCronRunQueueBehavior" = 'QUEUE'
            )
        )
),
crons_to_claim AS MATERIALIZED (
    SELECT
        cronSchedule."id"
    FROM
        "WorkflowTriggerCronRef" as cronSchedule
    JOIN
        eligible_crons ON eligible_crons."id" = cronSchedule."id"
    WHERE
        cronSchedule."tickerId" IS DISTINCT FROM @tickerId::uuid
        AND (
            cronSchedule."tickerId" IS NULL
            OR NOT EXISTS (
                SELECT 1 FROM "Ticker" WHERE "id" = cronSchedule."tickerId" AND "isActive" = true AND "lastHeartbeatAt" >= NOW() - INTERVAL '10 seconds'
            )
        )
    FOR UPDATE OF cronSchedule SKIP LOCKED
),
claimed_crons AS (
    UPDATE
        "WorkflowTriggerCronRef" as cronSchedule
    SET
        "tickerId" = @tickerId::uuid
    FROM
        crons_to_claim
    JOIN
        eligible_crons ON eligible_crons."id" = crons_to_claim."id"
    WHERE
        cronSchedule."id" = crons_to_claim."id"
    RETURNING cronSchedule.*, eligible_crons."workflowVersionId", eligible_crons."tenantId"
)
SELECT * FROM claimed_crons
UNION ALL
SELECT
    cronSchedule.*,
    eligible_crons."workflowVersionId",
    eligible_crons."tenantId"
FROM
    "WorkflowTriggerCronRef" as cronSchedule
JOIN
    eligible_crons ON eligible_crons."id" = cronSchedule."id"
WHERE
    cronSchedule."tickerId" = @tickerId::uuid;

-- name: PollScheduledWorkflows :many
-- Returns every scheduled workflow this ticker should have scheduled: ones due within the next 5 seconds that
-- it already owns, plus ones it claims on this poll (unassigned, or owned by a ticker that stopped
-- heartbeating). Only the claimed rows are written; already-owned rows are read without an UPDATE.
WITH eligible_schedules AS (
    SELECT
        scheduledWorkflow."id",
        versions."id" AS "workflowVersionId",
        workflow."tenantId"
    FROM
        "WorkflowTriggerScheduledRef" AS scheduledWorkflow
    JOIN
        "WorkflowVersion" AS versions ON versions."id" = scheduledWorkflow."parentId"
    JOIN
        "Workflow" AS workflow ON workflow."id" = versions."workflowId"
    JOIN
        "Tenant" AS tenant ON tenant."id" = workflow."tenantId"
    WHERE
        scheduledWorkflow."triggerAt" <= NOW() + INTERVAL '5 seconds'
        AND versions."deletedAt" IS NULL
        AND workflow."deletedAt" IS NULL
        AND tenant."deletedAt" IS NULL
        AND NOT EXISTS (
            SELECT 1
            FROM "WorkflowRunTriggeredBy" AS runTriggeredBy
            WHERE runTriggeredBy."scheduledId" = scheduledWorkflow."id"
        )
        AND (
            workflow."isPaused" IS NULL
            OR workflow."isPaused" = FALSE
            OR (
                workflow."isPaused" = TRUE AND workflow."pausedWorkflowScheduledRunQueueBehavior" = 'QUEUE'
            )
        )
),
schedules_to_claim AS MATERIALIZED (
    SELECT
        scheduledWorkflow."id"
    FROM
        "WorkflowTriggerScheduledRef" AS scheduledWorkflow
    JOIN
        eligible_schedules ON eligible_schedules."id" = scheduledWorkflow."id"
    WHERE
        scheduledWorkflow."tickerId" IS DISTINCT FROM @tickerId::uuid
        AND (
            scheduledWorkflow."tickerId" IS NULL
            OR NOT EXISTS (
                SELECT 1 FROM "Ticker" WHERE "id" = scheduledWorkflow."tickerId" AND "isActive" = true AND "lastHeartbeatAt" >= NOW() - INTERVAL '10 seconds'
            )
        )
    FOR UPDATE OF scheduledWorkflow SKIP LOCKED
),
claimed_schedules AS (
    UPDATE
        "WorkflowTriggerScheduledRef" AS scheduledWorkflow
    SET
        "tickerId" = @tickerId::uuid
    FROM
        schedules_to_claim
    JOIN
        eligible_schedules ON eligible_schedules."id" = schedules_to_claim."id"
    WHERE
        scheduledWorkflow."id" = schedules_to_claim."id"
    RETURNING scheduledWorkflow.*, eligible_schedules."workflowVersionId", eligible_schedules."tenantId"
)
SELECT * FROM claimed_schedules
UNION ALL
SELECT
    scheduledWorkflow.*,
    eligible_schedules."workflowVersionId",
    eligible_schedules."tenantId"
FROM
    "WorkflowTriggerScheduledRef" AS scheduledWorkflow
JOIN
    eligible_schedules ON eligible_schedules."id" = scheduledWorkflow."id"
WHERE
    scheduledWorkflow."tickerId" = @tickerId::uuid;

-- name: PollTenantAlerts :many
-- Finds tenant alerts which haven't alerted since their frequency and assigns them to a ticker
WITH active_tenant_alerts AS (
    SELECT
        alerts.*
    FROM
        "TenantAlertingSettings" as alerts
    JOIN
        "Tenant" as tenant ON tenant."id" = alerts."tenantId"
    WHERE
        tenant."deletedAt" IS NULL
        AND (
            "lastAlertedAt" IS NULL OR
            "lastAlertedAt" <= NOW() - convert_duration_to_interval(alerts."maxFrequency")
        )
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
    JOIN
        "Tenant" as tenant ON tenant."id" = t0."tenantId"
    WHERE
        tenant."deletedAt" IS NULL
        AND t0."revoked" = false
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
        "Tenant" AS tenant ON tenant."id" = rl."tenantId"
    WHERE
        tenant."deletedAt" IS NULL
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
            -- Gauge resources (WORKER / WORKER_SLOT) have a null window; default to
            -- 24h so Exhausted/Alarm alerts cannot re-fire on every ticker poll.
            AND trla."createdAt" >= NOW() - COALESCE(NULLIF(arl."window", '')::INTERVAL, INTERVAL '24 hours')
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
