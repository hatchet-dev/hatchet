-- name: ListTenants :many
SELECT
    *
FROM
    "Tenant" as tenants;

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
