-- name: CountAllocatedResourcesByTenant :many
-- One row per tenant with a cron, a pending schedule, or a webhook.
-- A tenant whose three counts are all zero is absent.
-- NULL tenantIds returns every such tenant.
SELECT
    tenant_id,
    cron_count,
    scheduled_run_count,
    webhook_count
FROM tenant_allocated_resource_count
WHERE (
    sqlc.narg('tenantIds')::uuid[] IS NULL
    OR tenant_id = ANY(sqlc.narg('tenantIds')::uuid[])
)
AND (
    cron_count > 0
    OR scheduled_run_count > 0
    OR webhook_count > 0
)
ORDER BY 1;

-- name: LockAllocatedResourceCountTable :exec
-- EXCLUSIVE blocks writers for the rest of this transaction and waits for
-- writers already in the table. A plain read is still allowed. The rebuild
-- deletes every row and inserts the replacement, so a writer must not commit
-- a new tenant row between those two statements.
LOCK TABLE tenant_allocated_resource_count IN EXCLUSIVE MODE;

-- name: DeleteAllocatedResourceCounts :exec
DELETE FROM tenant_allocated_resource_count;

-- name: InsertAllocatedResourceCounts :exec
-- Replaces tenant_allocated_resource_count. An enabled cron is counted on the
-- latest live version of a live workflow. A pending schedule is not deleted,
-- is on a live version and workflow, and has not fired. A webhook is every
-- incoming webhook row.
WITH latest_versions AS (
    SELECT DISTINCT ON (v."workflowId")
        v.id,
        v."workflowId"
    FROM "WorkflowVersion" v
    JOIN "Workflow" w ON w."id" = v."workflowId"
    WHERE
        v."deletedAt" IS NULL
        AND w."deletedAt" IS NULL
    ORDER BY v."workflowId", v."order" DESC
),
cron_counts AS (
    SELECT
        w."tenantId" AS tenant_id,
        count(*)::bigint AS cron_count
    FROM "WorkflowTriggerCronRef" c
    JOIN "WorkflowTriggers" t ON t."id" = c."parentId"
    JOIN latest_versions lv ON lv.id = t."workflowVersionId"
    JOIN "Workflow" w ON w."id" = lv."workflowId"
    WHERE
        c."deletedAt" IS NULL
        AND c."enabled" = true
        AND t."deletedAt" IS NULL
        AND w."deletedAt" IS NULL
    GROUP BY w."tenantId"
),
schedule_counts AS (
    SELECT
        w."tenantId" AS tenant_id,
        count(*)::bigint AS scheduled_run_count
    FROM "WorkflowTriggerScheduledRef" s
    JOIN "WorkflowVersion" v ON s."parentId" = v."id"
    JOIN "Workflow" w ON v."workflowId" = w."id"
    WHERE
        v."deletedAt" IS NULL
        AND w."deletedAt" IS NULL
        AND s."deletedAt" IS NULL
        AND NOT EXISTS (
            SELECT 1
            FROM "WorkflowRunTriggeredBy" tb
            WHERE tb."scheduledId" = s."id"
        )
    GROUP BY w."tenantId"
),
webhook_counts AS (
    SELECT
        tenant_id,
        count(*)::bigint AS webhook_count
    FROM v1_incoming_webhook
    GROUP BY tenant_id
),
fresh AS (
    SELECT
        COALESCE(c.tenant_id, s.tenant_id, wh.tenant_id) AS tenant_id,
        COALESCE(c.cron_count, 0)::bigint AS cron_count,
        COALESCE(s.scheduled_run_count, 0)::bigint AS scheduled_run_count,
        COALESCE(wh.webhook_count, 0)::bigint AS webhook_count
    FROM cron_counts c
    FULL OUTER JOIN schedule_counts s ON s.tenant_id = c.tenant_id
    FULL OUTER JOIN webhook_counts wh ON wh.tenant_id = COALESCE(c.tenant_id, s.tenant_id)
)
INSERT INTO tenant_allocated_resource_count (
    tenant_id,
    cron_count,
    scheduled_run_count,
    webhook_count
)
SELECT tenant_id, cron_count, scheduled_run_count, webhook_count
FROM fresh;

-- name: AdjustAllocatedResourceCounts :exec
INSERT INTO tenant_allocated_resource_count AS c (
    tenant_id,
    cron_count,
    scheduled_run_count,
    webhook_count
) VALUES (
    @tenantId::uuid,
    GREATEST(@cronDelta::bigint, 0),
    GREATEST(@scheduleDelta::bigint, 0),
    GREATEST(@webhookDelta::bigint, 0)
)
ON CONFLICT (tenant_id) DO UPDATE SET
    cron_count = GREATEST(c.cron_count + @cronDelta::bigint, 0),
    scheduled_run_count = GREATEST(c.scheduled_run_count + @scheduleDelta::bigint, 0),
    webhook_count = GREATEST(c.webhook_count + @webhookDelta::bigint, 0);

-- name: CountEnabledCronsOnVersion :one
SELECT count(*)::bigint
FROM "WorkflowTriggerCronRef" c
JOIN "WorkflowTriggers" t ON t."id" = c."parentId"
JOIN "WorkflowVersion" v ON v."id" = t."workflowVersionId"
JOIN "Workflow" w ON w."id" = v."workflowId"
WHERE
    t."workflowVersionId" = @versionId::uuid
    AND c."deletedAt" IS NULL
    AND c."enabled" = true
    AND t."deletedAt" IS NULL
    AND v."deletedAt" IS NULL
    AND w."deletedAt" IS NULL;

-- name: GetCronAllocationState :one
-- on_latest is true when this cron is enabled-countable: live cron, trigger,
-- version, and workflow, and the version is the latest non-deleted one.
SELECT
    c."enabled" AS enabled,
    CASE
        WHEN
            c."deletedAt" IS NULL
            AND t."deletedAt" IS NULL
            AND v."deletedAt" IS NULL
            AND w."deletedAt" IS NULL
            AND v."id" = (
                SELECT v2."id"
                FROM "WorkflowVersion" v2
                WHERE
                    v2."workflowId" = v."workflowId"
                    AND v2."deletedAt" IS NULL
                ORDER BY v2."order" DESC
                LIMIT 1
            )
        THEN true
        ELSE false
    END AS on_latest
FROM "WorkflowTriggerCronRef" c
JOIN "WorkflowTriggers" t ON t."id" = c."parentId"
JOIN "WorkflowVersion" v ON v."id" = t."workflowVersionId"
JOIN "Workflow" w ON w."id" = v."workflowId"
WHERE
    c."id" = @id::uuid
    AND w."tenantId" = @tenantId::uuid;

-- name: GetCronWorkflowID :one
SELECT w."id" AS workflow_id
FROM "WorkflowTriggerCronRef" c
JOIN "WorkflowTriggers" t ON t."id" = c."parentId"
JOIN "WorkflowVersion" v ON v."id" = t."workflowVersionId"
JOIN "Workflow" w ON w."id" = v."workflowId"
WHERE
    c."id" = @id::uuid
    AND w."tenantId" = @tenantId::uuid;

-- name: GetCronAllocationTarget :one
-- Tenant and workflow for a cron delete that does not already know them.
SELECT
    w."id" AS workflow_id,
    w."tenantId" AS tenant_id
FROM "WorkflowTriggerCronRef" c
JOIN "WorkflowTriggers" t ON t."id" = c."parentId"
JOIN "WorkflowVersion" v ON v."id" = t."workflowVersionId"
JOIN "Workflow" w ON w."id" = v."workflowId"
WHERE c."id" = @id::uuid;

-- name: ListScheduleWorkflowIDs :many
SELECT DISTINCT w."id"
FROM "WorkflowTriggerScheduledRef" s
JOIN "WorkflowVersion" v ON s."parentId" = v."id"
JOIN "Workflow" w ON v."workflowId" = w."id"
WHERE
    s."id" = ANY(@ids::uuid[])
    AND w."tenantId" = @tenantId::uuid
ORDER BY w."id";

-- name: LockSchedulesCountedTowardAllocation :many
-- Locks the schedules that still contribute to the count, in id order, and
-- returns those ids. apiOnly limits the set to API schedules, the set bulk
-- delete removes. The caller holds each workflow's latest version lock first.
SELECT s."id"
FROM "WorkflowTriggerScheduledRef" s
JOIN "WorkflowVersion" v ON s."parentId" = v."id"
JOIN "Workflow" w ON v."workflowId" = w."id"
WHERE
    s."id" = ANY(@ids::uuid[])
    AND w."tenantId" = @tenantId::uuid
    AND s."deletedAt" IS NULL
    AND v."deletedAt" IS NULL
    AND w."deletedAt" IS NULL
    AND (
        NOT @apiOnly::boolean
        OR s."method" = 'API'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM "WorkflowRunTriggeredBy" tb
        WHERE tb."scheduledId" = s."id"
    )
ORDER BY s."id"
FOR UPDATE OF s;

-- name: LockWorkflowSchedulesTowardAllocation :many
-- Locks the pending schedules a workflow delete removes from the count.
-- The caller holds the workflow's latest version lock first.
SELECT s."id"
FROM "WorkflowTriggerScheduledRef" s
JOIN "WorkflowVersion" v ON s."parentId" = v."id"
JOIN "Workflow" w ON v."workflowId" = w."id"
WHERE
    w."id" = @workflowId::uuid
    AND w."tenantId" = @tenantId::uuid
    AND s."deletedAt" IS NULL
    AND v."deletedAt" IS NULL
    AND w."deletedAt" IS NULL
    AND NOT EXISTS (
        SELECT 1
        FROM "WorkflowRunTriggeredBy" tb
        WHERE tb."scheduledId" = s."id"
    )
ORDER BY s."id"
FOR UPDATE OF s;
