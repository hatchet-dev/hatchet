-- +goose Up
-- +goose StatementBegin
WITH all_crons AS (
    -- Get all API cron triggers for a specific tenant
    SELECT 
        t."id" AS trigger_parent_id,
        t."workflowVersionId",
        c."id" AS cron_id,
        c."cron",
        c."name" AS cron_name,
        c."method"
    FROM "WorkflowTriggers" t
    JOIN "WorkflowTriggerCronRef" c ON c."parentId" = t."id"
    WHERE c."method" = 'API'
),
workflow_info AS (
    -- Get workflow information for each version
    SELECT
        v."id" AS version_id,
        v."workflowId",
        v."order",
        v."version"
    FROM "WorkflowVersion" v
    WHERE v."deletedAt" IS NULL
),
latest_versions AS (
    -- Find the latest version for each workflow
    SELECT DISTINCT ON ("workflowId")
        "workflowId",
        "id" AS latest_version_id,
        "order" AS latest_order
    FROM "WorkflowVersion"
    WHERE "deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
),
latest_triggers AS (
    -- Find the latest trigger parent ID for each workflow
    SELECT
        lv."workflowId",
        lv.latest_version_id,
        t."id" AS latest_trigger_id,
        t."tenantId"
    FROM latest_versions lv
    JOIN "WorkflowTriggers" t ON t."workflowVersionId" = lv.latest_version_id
    WHERE t."deletedAt" IS NULL
),
crons_to_update AS (
    -- Identify cron triggers that need to be updated (not on latest version)
    SELECT 
        ac.cron_id,
        ac.trigger_parent_id AS current_parent_id,
        lt.latest_trigger_id AS target_parent_id,
        ac.cron_name,
        ac.cron,
        wi."workflowId",
        wi."order" AS current_version_order,
        lv.latest_order,
        lt."tenantId"
    FROM all_crons ac
    JOIN workflow_info wi ON wi.version_id = ac."workflowVersionId"
    JOIN latest_versions lv ON lv."workflowId" = wi."workflowId"
    JOIN latest_triggers lt ON lt."workflowId" = wi."workflowId"
    WHERE wi."order" < lv.latest_order -- Only include crons not on the latest version
),
conflicting_records AS (
    SELECT 
        ctu.cron_id
    FROM crons_to_update ctu
    JOIN "WorkflowTriggerCronRef" wcr ON 
        wcr."parentId" = ctu.target_parent_id AND
        wcr."cron" = ctu.cron AND
        wcr."name" = ctu.cron_name
    WHERE wcr."id" != ctu.cron_id
),
to_update AS (
    -- Final list of records to update (excluding conflicting ones)
    SELECT *
    FROM crons_to_update
    WHERE cron_id NOT IN (SELECT cron_id FROM conflicting_records)
)
UPDATE "WorkflowTriggerCronRef"
SET "parentId" = ctu.target_parent_id
FROM to_update ctu
WHERE "WorkflowTriggerCronRef"."id" = ctu.cron_id;


WITH all_scheduled AS (
    -- Get all scheduled triggers
    SELECT 
        t."id" AS trigger_parent_id,
        t."workflowVersionId",
        s."id" AS scheduled_id,
        s."triggerAt",
        s."method",
        s."tickerId",
        s."input",
        s."childIndex",
        s."childKey",
        s."parentStepRunId",
        s."parentWorkflowRunId",
        s."additionalMetadata",
        s."priority"
    FROM "WorkflowTriggers" t
    JOIN "WorkflowTriggerScheduledRef" s ON s."parentId" = t."id"
    WHERE s."method" = 'API'
),
workflow_info AS (
    -- Get workflow information for each version
    SELECT
        v."id" AS version_id,
        v."workflowId",
        v."order",
        v."version"
    FROM "WorkflowVersion" v
    WHERE v."deletedAt" IS NULL
),
latest_versions AS (
    -- Find the latest version for each workflow
    SELECT DISTINCT ON ("workflowId")
        "workflowId",
        "id" AS latest_version_id,
        "order" AS latest_order
    FROM "WorkflowVersion"
    WHERE "deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
),
latest_triggers AS (
    -- Find the latest trigger parent ID for each workflow
    SELECT
        lv."workflowId",
        lv.latest_version_id,
        t."id" AS latest_trigger_id,
        t."tenantId"
    FROM latest_versions lv
    JOIN "WorkflowTriggers" t ON t."workflowVersionId" = lv.latest_version_id
    WHERE t."deletedAt" IS NULL
),
scheduled_to_update AS (
    -- Identify scheduled triggers that need to be updated (not on latest version)
    SELECT 
        als.scheduled_id,
        als.trigger_parent_id AS current_parent_id,
        lt.latest_trigger_id AS target_parent_id,
        als."triggerAt",
        als."method",
        als."tickerId",
        als."input",
        als."childIndex",
        als."childKey",
        als."parentStepRunId",
        als."parentWorkflowRunId",
        als."additionalMetadata",
        als."priority",
        wi."workflowId",
        wi."order" AS current_version_order,
        lv.latest_order,
        lt."tenantId"
    FROM all_scheduled als
    JOIN workflow_info wi ON wi.version_id = als."workflowVersionId"
    JOIN latest_versions lv ON lv."workflowId" = wi."workflowId"
    JOIN latest_triggers lt ON lt."workflowId" = wi."workflowId"
    WHERE wi."order" < lv.latest_order -- Only include scheduled triggers not on the latest version
),
conflicting_records AS (
    -- Identify records that would cause conflicts with the unique constraint
    SELECT 
        stu.scheduled_id
    FROM scheduled_to_update stu
    JOIN "WorkflowTriggerScheduledRef" wsr ON 
        wsr."parentId" = stu.target_parent_id AND
        wsr."parentStepRunId" = stu."parentStepRunId" AND
        wsr."childKey" = stu."childKey"
    WHERE wsr."id" != stu.scheduled_id
),
to_update AS (
    -- Final list of records to update (excluding conflicting ones)
    SELECT *
    FROM scheduled_to_update
    WHERE scheduled_id NOT IN (SELECT scheduled_id FROM conflicting_records)
)

UPDATE "WorkflowTriggerScheduledRef"
SET "parentId" = tu.target_parent_id
FROM to_update tu
WHERE "WorkflowTriggerScheduledRef"."id" = tu.scheduled_id;
-- +goose StatementEnd
