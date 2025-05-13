-- NOTE: this file doesn't typically get generated, it's just used for generating boilerplate
-- for queries

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
