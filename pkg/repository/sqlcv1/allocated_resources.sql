-- name: CountAllocatedResourcesByTenant :many
-- One row per tenant with live allocated-resource counts on this shard.
-- cron_count: enabled, non-deleted cron refs on the latest non-deleted
-- workflow version (API and DEFAULT).
-- scheduled_run_count: pending scheduled refs only (not deleted, and no
-- WorkflowRunTriggeredBy.scheduledId — fired runs are excluded).
-- webhook_count: row count on v1_incoming_webhook.
-- NULL tenantIds drops the tenant filter (hourly shard-wide collect).
-- The result is still one row per tenant, not a summed shard total.
WITH cron_counts AS (
    SELECT
        w."tenantId" AS tenant_id,
        count(*)::bigint AS cron_count
    FROM "WorkflowTriggerCronRef" c
    JOIN "WorkflowTriggers" t ON t."id" = c."parentId"
    JOIN "WorkflowVersion" v ON v."id" = t."workflowVersionId"
    JOIN "Workflow" w ON w."id" = v."workflowId"
    WHERE
        c."deletedAt" IS NULL
        AND c."enabled" = true
        AND t."deletedAt" IS NULL
        AND v."deletedAt" IS NULL
        AND w."deletedAt" IS NULL
        AND (
            sqlc.narg('tenantIds')::uuid[] IS NULL
            OR w."tenantId" = ANY(sqlc.narg('tenantIds')::uuid[])
        )
        AND NOT EXISTS (
            SELECT 1
            FROM "WorkflowVersion" newer
            WHERE newer."workflowId" = v."workflowId"
              AND newer."deletedAt" IS NULL
              AND newer."order" > v."order"
        )
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
        AND (
            sqlc.narg('tenantIds')::uuid[] IS NULL
            OR w."tenantId" = ANY(sqlc.narg('tenantIds')::uuid[])
        )
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
    WHERE
        sqlc.narg('tenantIds')::uuid[] IS NULL
        OR tenant_id = ANY(sqlc.narg('tenantIds')::uuid[])
    GROUP BY tenant_id
)
SELECT
    COALESCE(c.tenant_id, s.tenant_id, wh.tenant_id) AS tenant_id,
    COALESCE(c.cron_count, 0)::bigint AS cron_count,
    COALESCE(s.scheduled_run_count, 0)::bigint AS scheduled_run_count,
    COALESCE(wh.webhook_count, 0)::bigint AS webhook_count
FROM cron_counts c
FULL OUTER JOIN schedule_counts s ON s.tenant_id = c.tenant_id
FULL OUTER JOIN webhook_counts wh ON wh.tenant_id = COALESCE(c.tenant_id, s.tenant_id)
ORDER BY 1;
