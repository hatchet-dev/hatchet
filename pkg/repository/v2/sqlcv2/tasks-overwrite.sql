-- NOTE: this file doesn't typically get generated, since we need to overwrite the
-- behavior of `@dagIds` and `@dagInsertedAts` to be nullable. It can be generated
-- when we'd like to change the query.

-- name: CreateTasks :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@tenantIds::uuid[]) AS tenant_id,
                unnest(@queues::text[]) AS queue,
                unnest(@actionIds::text[]) AS action_id,
                unnest(@stepIds::uuid[]) AS step_id,
                unnest(@stepReadableIds::text[]) AS step_readable_id,
                unnest(@workflowIds::uuid[]) AS workflow_id,
                unnest(@scheduleTimeouts::text[]) AS schedule_timeout,
                unnest(@stepTimeouts::text[]) AS step_timeout,
                unnest(@priorities::integer[]) AS priority,
                unnest(cast(@stickies::text[] as v2_sticky_strategy[])) AS sticky,
                unnest(@desiredWorkerIds::uuid[]) AS desired_worker_id,
                unnest(@externalIds::uuid[]) AS external_id,
                unnest(@displayNames::text[]) AS display_name,
                unnest(@inputs::jsonb[]) AS input,
                unnest(@retryCounts::integer[]) AS retry_count,
                unnest(@additionalMetadatas::jsonb[]) AS additional_metadata,
                -- NOTE: these are nullable, so sqlc doesn't support casting to a type
                unnest(@dagIds::bigint[]) AS dag_id,
                unnest(@dagInsertedAts::timestamptz[]) AS dag_inserted_at
        ) AS subquery
)
INSERT INTO v2_task (
    tenant_id,
    queue,
    action_id,
    step_id,
    step_readable_id,
    workflow_id,
    schedule_timeout,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    external_id,
    display_name,
    input,
    retry_count,
    additional_metadata,
    dag_id,
    dag_inserted_at
)
SELECT
    i.tenant_id,
    i.queue,
    i.action_id,
    i.step_id,
    i.step_readable_id,
    i.workflow_id,
    i.schedule_timeout,
    i.step_timeout,
    i.priority,
    i.sticky,
    i.desired_worker_id,
    i.external_id,
    i.display_name,
    i.input,
    i.retry_count,
    i.additional_metadata,
    i.dag_id,
    i.dag_inserted_at
FROM
    input i
RETURNING
    *;
