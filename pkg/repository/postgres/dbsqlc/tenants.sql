-- name: CreateTenant :one
WITH active_controller_partitions AS (
    SELECT
        "id"
    FROM
        "ControllerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
)
INSERT INTO "Tenant" ("id", "name", "slug", "controllerPartitionId", "dataRetentionPeriod")
VALUES (
    sqlc.arg('id')::uuid,
    sqlc.arg('name')::text,
    sqlc.arg('slug')::text,
    (
        SELECT
            "id"
        FROM
            active_controller_partitions
        ORDER BY
            random()
        LIMIT 1
    ),
    COALESCE(sqlc.narg('dataRetentionPeriod')::text, '720h')
)
RETURNING *;

-- name: UpdateTenant :one
UPDATE
    "Tenant"
SET
    "name" = COALESCE(sqlc.narg('name')::text, "name"),
    "analyticsOptOut" = COALESCE(sqlc.narg('analyticsOptOut')::boolean, "analyticsOptOut"),
    "alertMemberEmails" = COALESCE(sqlc.narg('alertMemberEmails')::boolean, "alertMemberEmails")
WHERE
    "id" = sqlc.arg('id')::uuid
RETURNING *;

-- name: CreateTenantAlertingSettings :one
INSERT INTO "TenantAlertingSettings" ("id", "tenantId")
VALUES (gen_random_uuid(), sqlc.arg('tenantId')::uuid)
RETURNING *;

-- name: ListTenants :many
SELECT
    *
FROM
    "Tenant" as tenants;

-- name: ControllerPartitionHeartbeat :one
UPDATE
    "ControllerPartition" p
SET
    "lastHeartbeat" = NOW()
WHERE
    p."id" = sqlc.arg('controllerPartitionId')::text
RETURNING *;

-- name: WorkerPartitionHeartbeat :one
UPDATE
    "TenantWorkerPartition" p
SET
    "lastHeartbeat" = NOW()
WHERE
    p."id" = sqlc.arg('workerPartitionId')::text
RETURNING *;

-- name: ListTenantsByControllerPartitionId :many
SELECT
    *
FROM
    "Tenant" as tenants
WHERE
    "controllerPartitionId" = sqlc.arg('controllerPartitionId')::text;

-- name: ListTenantsByTenantWorkerPartitionId :many
WITH update_partition AS (
    UPDATE
        "TenantWorkerPartition"
    SET
        "lastHeartbeat" = NOW()
    WHERE
        "id" = sqlc.arg('workerPartitionId')::text
)
SELECT
    *
FROM
    "Tenant" as tenants
WHERE
    "workerPartitionId" = sqlc.arg('workerPartitionId')::text;

-- name: GetTenantByID :one
SELECT
    *
FROM
    "Tenant" as tenants
WHERE
    "id" = sqlc.arg('id')::uuid;

-- name: GetTenantBySlug :one
SELECT
    *
FROM
    "Tenant" as tenants
WHERE
    "slug" = sqlc.arg('slug')::text;

-- name: GetTenantAlertingSettings :one
SELECT
    *
FROM
    "TenantAlertingSettings" as tenantAlertingSettings
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid;

-- name: GetSlackWebhooks :many
SELECT
    *
FROM
    "SlackAppWebhook" as slackWebhooks
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid;

-- name: GetEmailGroups :many
SELECT
    *
FROM
    "TenantAlertEmailGroup" as emailGroups
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid;

-- name: GetMemberEmailGroup :many
SELECT u."email"
FROM "User" u
JOIN "TenantMember" tm ON u."id" = tm."userId"
WHERE u."emailVerified" = true
AND tm."tenantId" = sqlc.arg('tenantId')::uuid;

-- name: UpdateTenantAlertingSettings :one
UPDATE
    "TenantAlertingSettings" as tenantAlertingSettings
SET
    "lastAlertedAt" = COALESCE(sqlc.narg('lastAlertedAt')::timestamp, "lastAlertedAt")
WHERE
    "tenantId" = sqlc.arg('tenantId')::uuid
RETURNING *;

-- name: GetTenantTotalQueueMetrics :one
WITH valid_workflow_runs AS (
    SELECT
        runs."id", workflow."id" as "workflowId", workflow."name" as "workflowName"
    FROM
        "WorkflowRun" as runs
    LEFT JOIN
        "WorkflowVersion" as workflowVersion ON runs."workflowVersionId" = workflowVersion."id"
    LEFT JOIN
        "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
    WHERE
        -- status of the workflow run must be pending, queued or running
        runs."status" IN ('PENDING', 'QUEUED', 'RUNNING') AND
        runs."tenantId" = $1 AND
        (
            sqlc.narg('additionalMetadata')::jsonb IS NULL OR
            runs."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb
        ) AND
        (
            sqlc.narg('workflowIds')::uuid[] IS NULL OR
            workflow."id" = ANY(sqlc.narg('workflowIds')::uuid[])
        )
)
SELECT
    -- count of step runs in a PENDING_ASSIGNMENT state
    COUNT(stepRun."id") FILTER (WHERE stepRun."status" = 'PENDING_ASSIGNMENT') as "pendingAssignmentCount",
    -- count of step runs in a PENDING state
    COUNT(stepRun."id") FILTER (WHERE stepRun."status" = 'PENDING') as "pendingCount",
    -- count of step runs in a RUNNING state
    COUNT(stepRun."id") FILTER (WHERE stepRun."status" = 'RUNNING') as "runningCount"
FROM
    valid_workflow_runs as runs
LEFT JOIN
    "JobRun" as jobRun ON runs."id" = jobRun."workflowRunId"
LEFT JOIN
    "StepRun" as stepRun ON jobRun."id" = stepRun."jobRunId";

-- name: GetTenantWorkflowQueueMetrics :many
WITH valid_workflow_runs AS (
    SELECT
        runs."id", workflow."id" as "workflowId", workflow."name" as "workflowName"
    FROM
        "WorkflowRun" as runs
    LEFT JOIN
        "WorkflowVersion" as workflowVersion ON runs."workflowVersionId" = workflowVersion."id"
    LEFT JOIN
        "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
    WHERE
        -- status of the workflow run must be pending, queued or running
        runs."status" IN ('PENDING', 'QUEUED', 'RUNNING') AND
        runs."tenantId" = $1 AND
        (
            sqlc.narg('additionalMetadata')::jsonb IS NULL OR
            runs."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb
        ) AND
        (
            sqlc.narg('workflowIds')::uuid[] IS NULL OR
            workflow."id" = ANY(sqlc.narg('workflowIds')::uuid[])
        )
)
SELECT
    runs."workflowId",
    -- count of step runs in a PENDING_ASSIGNMENT state
    COUNT(stepRun."id") FILTER (WHERE stepRun."status" = 'PENDING_ASSIGNMENT') as "pendingAssignmentCount",
    -- count of step runs in a PENDING state
    COUNT(stepRun."id") FILTER (WHERE stepRun."status" = 'PENDING') as "pendingCount",
    -- count of step runs in a RUNNING state
    COUNT(stepRun."id") FILTER (WHERE stepRun."status" = 'RUNNING') as "runningCount"
FROM
    valid_workflow_runs as runs
LEFT JOIN
    "JobRun" as jobRun ON runs."id" = jobRun."workflowRunId"
LEFT JOIN
    "StepRun" as stepRun ON jobRun."id" = stepRun."jobRunId"
GROUP BY
    runs."workflowId";

-- name: CreateControllerPartition :one
INSERT INTO "ControllerPartition" ("id", "createdAt", "lastHeartbeat", "name")
VALUES (gen_random_uuid()::text, NOW(), NOW(), sqlc.narg('name')::text)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: DeleteControllerPartition :one
DELETE FROM "ControllerPartition"
WHERE "id" = sqlc.arg('id')::text
RETURNING *;

-- name: RebalanceAllControllerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "ControllerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
),
tenants_to_update AS (
    SELECT
        tenants."id" AS "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "Tenant" AS tenants
    -- For the controller partition, we DO use the internal tenant as well
)
UPDATE
    "Tenant" AS tenants
SET
    "controllerPartitionId" = partitions."id"
FROM
    tenants_to_update,
    active_partitions AS partitions
WHERE
    tenants."id" = tenants_to_update."id" AND
    partitions.row_number = (tenants_to_update.row_number - 1) % (SELECT COUNT(*) FROM active_partitions) + 1;

-- name: RebalanceInactiveControllerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "ControllerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
), inactive_partitions AS (
    SELECT
        "id"
    FROM
        "ControllerPartition"
    WHERE
        "lastHeartbeat" <= NOW() - INTERVAL '1 minute'
), tenants_to_update AS (
    SELECT
        tenants."id" AS "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "Tenant" AS tenants
    WHERE
        "controllerPartitionId" IS NULL OR
        "controllerPartitionId" IN (SELECT "id" FROM inactive_partitions)
), update_tenants AS (
    UPDATE "Tenant" AS tenants
    SET "controllerPartitionId" = partitions."id"
    FROM
        tenants_to_update,
        active_partitions AS partitions
    WHERE
    tenants."id" = tenants_to_update."id" AND
    partitions.row_number = (tenants_to_update.row_number - 1) % (SELECT COUNT(*) FROM active_partitions) + 1
)
DELETE FROM "ControllerPartition"
WHERE "id" IN (SELECT "id" FROM inactive_partitions);

-- name: CreateTenantWorkerPartition :one
INSERT INTO "TenantWorkerPartition" ("id", "createdAt", "lastHeartbeat", "name")
VALUES (gen_random_uuid()::text, NOW(), NOW(), sqlc.narg('name')::text)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: DeleteTenantWorkerPartition :one
DELETE FROM "TenantWorkerPartition"
WHERE "id" = sqlc.arg('id')::text
RETURNING *;

-- name: RebalanceAllTenantWorkerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "TenantWorkerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
),
tenants_to_update AS (
    SELECT
        tenants."id" AS "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "Tenant" AS tenants
    WHERE
        tenants."slug" != 'internal'
)
UPDATE
    "Tenant" AS tenants
SET
    "workerPartitionId" = partitions."id"
FROM
    tenants_to_update,
    active_partitions AS partitions
WHERE
    tenants."id" = tenants_to_update."id" AND
    partitions.row_number = (tenants_to_update.row_number - 1) % (SELECT COUNT(*) FROM active_partitions) + 1;

-- name: RebalanceInactiveTenantWorkerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "TenantWorkerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
), inactive_partitions AS (
    SELECT
        "id"
    FROM
        "TenantWorkerPartition"
    WHERE
        "lastHeartbeat" <= NOW() - INTERVAL '1 minute'
), tenants_to_update AS (
    SELECT
        tenants."id" AS "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "Tenant" AS tenants
    WHERE
        tenants."slug" != 'internal' AND
        (
            "workerPartitionId" IS NULL OR
            "workerPartitionId" IN (SELECT "id" FROM inactive_partitions)
        )
), update_tenants AS (
    UPDATE "Tenant" AS tenants
    SET "workerPartitionId" = partitions."id"
    FROM
        tenants_to_update,
        active_partitions AS partitions
    WHERE
    tenants."id" = tenants_to_update."id" AND
    partitions.row_number = (tenants_to_update.row_number - 1) % (SELECT COUNT(*) FROM active_partitions) + 1
)
DELETE FROM "TenantWorkerPartition"
WHERE "id" IN (SELECT "id" FROM inactive_partitions);

-- name: SchedulerPartitionHeartbeat :one
UPDATE
    "SchedulerPartition" p
SET
    "lastHeartbeat" = NOW()
WHERE
    p."id" = sqlc.arg('schedulerPartitionId')::text
RETURNING *;

-- name: CreateSchedulerPartition :one
INSERT INTO "SchedulerPartition" ("id", "createdAt", "lastHeartbeat", "name")
VALUES (gen_random_uuid()::text, NOW(), NOW(), sqlc.narg('name')::text)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: DeleteSchedulerPartition :one
DELETE FROM "SchedulerPartition"
WHERE "id" = sqlc.arg('id')::text
RETURNING *;

-- name: RebalanceAllSchedulerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "SchedulerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
),
tenants_to_update AS (
    SELECT
        tenants."id" AS "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "Tenant" AS tenants
    WHERE
        tenants."slug" != 'internal'
)
UPDATE
    "Tenant" AS tenants
SET
    "schedulerPartitionId" = partitions."id"
FROM
    tenants_to_update,
    active_partitions AS partitions
WHERE
    tenants."id" = tenants_to_update."id" AND
    partitions.row_number = (tenants_to_update.row_number - 1) % (SELECT COUNT(*) FROM active_partitions) + 1;

-- name: RebalanceInactiveSchedulerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "SchedulerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
), inactive_partitions AS (
    SELECT
        "id"
    FROM
        "SchedulerPartition"
    WHERE
        "lastHeartbeat" <= NOW() - INTERVAL '1 minute'
), tenants_to_update AS (
    SELECT
        tenants."id" AS "id",
        ROW_NUMBER() OVER () AS row_number
    FROM
        "Tenant" AS tenants
    WHERE
        tenants."slug" != 'internal' AND
        (
            "schedulerPartitionId" IS NULL OR
            "schedulerPartitionId" IN (SELECT "id" FROM inactive_partitions)
        )
), update_tenants AS (
    UPDATE "Tenant" AS tenants
    SET "schedulerPartitionId" = partitions."id"
    FROM
        tenants_to_update,
        active_partitions AS partitions
    WHERE
    tenants."id" = tenants_to_update."id" AND
    partitions.row_number = (tenants_to_update.row_number - 1) % (SELECT COUNT(*) FROM active_partitions) + 1
)
DELETE FROM "SchedulerPartition"
WHERE "id" IN (SELECT "id" FROM inactive_partitions);

-- name: ListTenantsBySchedulerPartitionId :many
SELECT
    *
FROM
    "Tenant" as tenants
WHERE
    "schedulerPartitionId" = sqlc.arg('schedulerPartitionId')::text;

-- name: UpsertTenantAlertingSettings :one
INSERT INTO "TenantAlertingSettings" (
    "id",
    "tenantId", 
    "maxFrequency", 
    "enableExpiringTokenAlerts", 
    "enableWorkflowRunFailureAlerts", 
    "enableTenantResourceLimitAlerts"
) VALUES (
    gen_random_uuid(),
    @tenantId::uuid,
    COALESCE(sqlc.narg('maxFrequency')::text, '1h'),
    COALESCE(sqlc.narg('enableExpiringTokenAlerts')::boolean, TRUE),
    COALESCE(sqlc.narg('enableWorkflowRunFailureAlerts')::boolean, FALSE),
    COALESCE(sqlc.narg('enableTenantResourceLimitAlerts')::boolean, TRUE)
) ON CONFLICT ("tenantId") DO UPDATE SET
    "maxFrequency" = COALESCE(sqlc.narg('maxFrequency')::text, "TenantAlertingSettings"."maxFrequency"),
    "enableExpiringTokenAlerts" = COALESCE(sqlc.narg('enableExpiringTokenAlerts')::boolean, "TenantAlertingSettings"."enableExpiringTokenAlerts"),
    "enableWorkflowRunFailureAlerts" = COALESCE(sqlc.narg('enableWorkflowRunFailureAlerts')::boolean, "TenantAlertingSettings"."enableWorkflowRunFailureAlerts"),
    "enableTenantResourceLimitAlerts" = COALESCE(sqlc.narg('enableTenantResourceLimitAlerts')::boolean, "TenantAlertingSettings"."enableTenantResourceLimitAlerts")
RETURNING *;

-- name: CreateTenantAlertGroup :one
INSERT INTO "TenantAlertEmailGroup" (
    "id",
    "tenantId",
    "emails"
) VALUES (
    gen_random_uuid(),
    @tenantId::uuid,
    @emails::text
) RETURNING *;

-- name: UpdateTenantAlertGroup :one
UPDATE "TenantAlertEmailGroup"
SET "emails" = @emails::text
WHERE "id" = @id::uuid
RETURNING *;

-- name: GetTenantAlertGroupById :one
SELECT
    *
FROM
    "TenantAlertEmailGroup"
WHERE
    "id" = @id::uuid;

-- name: ListTenantAlertGroups :many
SELECT
    *
FROM
    "TenantAlertEmailGroup"
WHERE
    "tenantId" = @tenantId::uuid;

-- name: DeleteTenantAlertGroup :exec
DELETE FROM
    "TenantAlertEmailGroup"
WHERE
    "tenantId" = @tenantId::uuid
    AND "id" = @id::uuid;

-- name: CreateTenantMember :one
INSERT INTO "TenantMember" (
    "id",
    "tenantId",
    "userId",
    "role"
) VALUES (
    gen_random_uuid(),
    @tenantId::uuid,
    @userId::uuid,
    @role::"TenantMemberRole"
) ON CONFLICT ("tenantId", "userId") DO UPDATE SET
    "role" = @role::"TenantMemberRole"
RETURNING *;

-- name: GetTenantMemberByID :one
SELECT
    *
FROM
    "TenantMember"
WHERE
    "id" = @id::uuid;

-- name: GetTenantMemberByUserID :one
SELECT
    *
FROM
    "TenantMember"
WHERE
    "tenantId" = @tenantId::uuid
    AND "userId" = @userId::uuid;

-- name: ListTenantMembers :many
SELECT
    *
FROM
    "TenantMember"
WHERE
    "tenantId" = @tenantId::uuid;

-- name: PopulateTenantMembers :many
SELECT
    tm.*,
    u."email",
    u."name",
    t."id" as "tenantId",
    t."createdAt" as "tenantCreatedAt",
    t."updatedAt" as "tenantUpdatedAt",
    t."name" as "tenantName",
    t."slug" as "tenantSlug",
    t."alertMemberEmails" as "alertMemberEmails",
    t."analyticsOptOut" as "analyticsOptOut"
FROM
    "TenantMember" tm
JOIN
    "User" u ON tm."userId" = u."id"
JOIN
    "Tenant" t ON tm."tenantId" = t."id"
WHERE
    tm."id" = ANY(@ids::uuid[]);

-- name: GetTenantMemberByEmail :one
SELECT
    tm.*
FROM
    "TenantMember" tm
JOIN
    "User" u ON tm."userId" = u."id"
WHERE
    tm."tenantId" = @tenantId::uuid
    AND u."email" = @email::text;

-- name: UpdateTenantMember :one
UPDATE "TenantMember"
SET 
    "role" = COALESCE(sqlc.narg('role')::"TenantMemberRole", "role")
WHERE "id" = @id::uuid
RETURNING *;

-- name: DeleteTenantMember :exec
DELETE FROM "TenantMember"
WHERE "id" = @id::uuid;