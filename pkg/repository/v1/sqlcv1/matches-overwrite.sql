-- name: ListMatchConditionsForEvent :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@eventKeys::text[]) AS event_key,
                -- NOTE: nullable field
                unnest(@eventResourceHints::text[]) AS event_resource_hint
        ) AS subquery
)
SELECT
    v1_match_id,
    id,
    registered_at,
    event_type,
    m.event_key,
    m.event_resource_hint,
    readable_data_key,
    expression
FROM
    v1_match_condition m
JOIN
    input i ON (m.tenant_id, m.event_type, m.event_key, m.is_satisfied, m.event_resource_hint) =
        (@tenantId::uuid, @eventType::v1_event_type, i.event_key, FALSE, i.event_resource_hint);

-- name: CreateMatchesForDAGTriggers :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@tenantIds::uuid[]) AS tenant_id,
                unnest(cast(@kinds::text[] as v1_match_kind[])) AS kind,
                unnest(@triggerDagIds::bigint[]) AS trigger_dag_id,
                unnest(@triggerDagInsertedAts::timestamptz[]) AS trigger_dag_inserted_at,
                unnest(@triggerStepIds::uuid[]) AS trigger_step_id,
                unnest(@triggerExternalIds::uuid[]) AS trigger_external_id,
                unnest(@triggerExistingTaskIds::bigint[]) AS trigger_existing_task_id,
                unnest(@triggerExistingTaskInsertedAts::timestamptz[]) AS trigger_existing_task_inserted_at
        ) AS subquery
)
INSERT INTO v1_match (
    tenant_id,
    kind,
    trigger_dag_id,
    trigger_dag_inserted_at,
    trigger_step_id,
    trigger_external_id,
    trigger_existing_task_id,
    trigger_existing_task_inserted_at
)
SELECT
    i.tenant_id,
    i.kind,
    i.trigger_dag_id,
    i.trigger_dag_inserted_at,
    i.trigger_step_id,
    i.trigger_external_id,
    i.trigger_existing_task_id,
    i.trigger_existing_task_inserted_at
FROM
    input i
RETURNING
    *;
