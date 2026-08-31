-- name: CountAllocatedResourcesByTenant :many
-- One row per tenant with live allocated-resource counts on this shard.
-- cron_count: enabled, non-deleted cron refs on the latest non-deleted
-- workflow version (API and DEFAULT).
-- scheduled_run_count: pending scheduled refs only (not deleted, and no
-- WorkflowRunTriggeredBy.scheduledId — fired runs are excluded).
-- webhook_count: row count on v1_incoming_webhook.
-- worker_count: currently connected non-operator workers (5s heartbeat,
-- assigned dispatcher, active, not paused).
-- slot_count: SUM(v1_worker_slot_config.max_units) for those workers.
-- Active workers are collected once and reused for both counts so the
-- tenant-filtered path can use Worker_tenantId_lastHeartbeatAt_idx
-- instead of scanning historical v1_worker_slot_config rows.
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
),
active_workers AS (
    SELECT
        w."id",
        w."tenantId" AS tenant_id
    FROM "Worker" w
    WHERE
        w."dispatcherId" IS NOT NULL
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."isActive" = true
        AND w."isPaused" = false
        AND w."operatorId" IS NULL
        AND (
            sqlc.narg('tenantIds')::uuid[] IS NULL
            OR w."tenantId" = ANY(sqlc.narg('tenantIds')::uuid[])
        )
),
worker_counts AS (
    SELECT
        tenant_id,
        count(*)::bigint AS worker_count
    FROM active_workers
    GROUP BY tenant_id
),
slot_counts AS (
    SELECT
        aw.tenant_id,
        SUM(wc.max_units)::bigint AS slot_count
    FROM active_workers aw
    JOIN v1_worker_slot_config wc
        ON wc.worker_id = aw.id
        AND wc.tenant_id = aw.tenant_id
    GROUP BY aw.tenant_id
),
tenants AS (
    SELECT tenant_id FROM cron_counts
    UNION
    SELECT tenant_id FROM schedule_counts
    UNION
    SELECT tenant_id FROM webhook_counts
    UNION
    SELECT tenant_id FROM worker_counts
    UNION
    SELECT tenant_id FROM slot_counts
)
SELECT
    t.tenant_id,
    COALESCE(c.cron_count, 0)::bigint AS cron_count,
    COALESCE(s.scheduled_run_count, 0)::bigint AS scheduled_run_count,
    COALESCE(wh.webhook_count, 0)::bigint AS webhook_count,
    COALESCE(wk.worker_count, 0)::bigint AS worker_count,
    COALESCE(sl.slot_count, 0)::bigint AS slot_count
FROM tenants t
LEFT JOIN cron_counts c ON c.tenant_id = t.tenant_id
LEFT JOIN schedule_counts s ON s.tenant_id = t.tenant_id
LEFT JOIN webhook_counts wh ON wh.tenant_id = t.tenant_id
LEFT JOIN worker_counts wk ON wk.tenant_id = t.tenant_id
LEFT JOIN slot_counts sl ON sl.tenant_id = t.tenant_id
ORDER BY 1;
