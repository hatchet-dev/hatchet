-- +goose Up
-- +goose StatementBegin

CREATE TABLE tenant_allocated_resource_count (
    tenant_id UUID PRIMARY KEY,
    cron_count BIGINT NOT NULL DEFAULT 0,
    scheduled_run_count BIGINT NOT NULL DEFAULT 0,
    webhook_count BIGINT NOT NULL DEFAULT 0
);

CREATE OR REPLACE FUNCTION tenant_allocated_resource_count_on_schedule_fire()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE tenant_allocated_resource_count
    SET scheduled_run_count = GREATEST(scheduled_run_count - 1, 0)
    WHERE tenant_id = NEW."tenantId";

    RETURN NULL;
END;
$$;

CREATE TRIGGER tenant_allocated_resource_count_schedule_fire
AFTER INSERT ON "WorkflowRunTriggeredBy"
FOR EACH ROW
WHEN (NEW."scheduledId" IS NOT NULL)
EXECUTE FUNCTION tenant_allocated_resource_count_on_schedule_fire();

-- One-time backfill. The hourly sync rebuilds this table from the same aggregation.
SET LOCAL statement_timeout = 0;

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

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS tenant_allocated_resource_count_schedule_fire ON "WorkflowRunTriggeredBy";
DROP FUNCTION IF EXISTS tenant_allocated_resource_count_on_schedule_fire();
DROP TABLE IF EXISTS tenant_allocated_resource_count;

-- +goose StatementEnd
