-- name: CreateTenant :one
WITH active_controller_partitions AS (
    SELECT
        "id"
    FROM
        "ControllerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
)
INSERT INTO "Tenant" ("id", "name", "slug", "controllerPartitionId")
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
    )
)
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

-- name: ListTenantsByControllerPartitionId :many
WITH update_partition AS (
    UPDATE
        "ControllerPartition"
    SET
        "lastHeartbeat" = NOW()
    WHERE
        "id" = sqlc.arg('controllerPartitionId')::text
)
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
INSERT INTO "ControllerPartition" ("id", "createdAt", "lastHeartbeat")
VALUES (sqlc.arg('id')::text, NOW(), NOW())
ON CONFLICT DO NOTHING
RETURNING *;

-- name: DeleteControllerPartition :one
DELETE FROM "ControllerPartition"
WHERE "id" = sqlc.arg('id')::text
RETURNING *;

-- name: RebalanceAllControllerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id"
    FROM
        "ControllerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
)
UPDATE
    "Tenant" as tenants
SET
    "controllerPartitionId" = (
        SELECT
            "id"
        FROM
            active_partitions
        ORDER BY
            random()
        LIMIT 1
    )
WHERE
    "slug" != 'internal' AND
    (
        "controllerPartitionId" IS NULL OR
        "controllerPartitionId" NOT IN (SELECT "id" FROM active_partitions)
    )
RETURNING *;

-- name: RebalanceInactiveControllerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id"
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
), update_tenants AS (
    UPDATE
        "Tenant" as tenants
    SET
        "controllerPartitionId" = (
            SELECT
                "id"
            FROM
                active_partitions
            ORDER BY
                random()
            LIMIT 1
        )
    WHERE
        "slug" != 'internal' AND
        (
            "controllerPartitionId" IS NULL OR
            "controllerPartitionId" IN (SELECT "id" FROM inactive_partitions)
        )
)
DELETE FROM "ControllerPartition"
WHERE "id" IN (SELECT "id" FROM inactive_partitions);

-- name: CreateTenantWorkerPartition :one
INSERT INTO "TenantWorkerPartition" ("id", "createdAt", "lastHeartbeat")
VALUES (sqlc.arg('id')::text, NOW(), NOW())
ON CONFLICT DO NOTHING
RETURNING *;

-- name: DeleteTenantWorkerPartition :one
DELETE FROM "TenantWorkerPartition"
WHERE "id" = sqlc.arg('id')::text
RETURNING *;

-- name: RebalanceAllTenantWorkerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id"
    FROM
        "TenantWorkerPartition"
    WHERE
        "lastHeartbeat" > NOW() - INTERVAL '1 minute'
)
UPDATE
    "Tenant" as tenants
SET
    "workerPartitionId" = (
        SELECT
            "id"
        FROM
            active_partitions
        ORDER BY
            random()
        LIMIT 1
    )
WHERE
    "slug" != 'internal' AND
    (
        "workerPartitionId" IS NULL OR
        "workerPartitionId" NOT IN (SELECT "id" FROM active_partitions)
    )
RETURNING *;

-- name: RebalanceInactiveTenantWorkerPartitions :exec
WITH active_partitions AS (
    SELECT
        "id"
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
), update_tenants AS (
    UPDATE
        "Tenant" as tenants
    SET
        "workerPartitionId" = (
            SELECT
                "id"
            FROM
                active_partitions
            ORDER BY
                random()
            LIMIT 1
        )
    WHERE
        "slug" != 'internal' AND
        (
            "workerPartitionId" IS NULL OR
            "workerPartitionId" IN (SELECT "id" FROM inactive_partitions)
        )
)
DELETE FROM "TenantWorkerPartition"
WHERE "id" IN (SELECT "id" FROM inactive_partitions);
