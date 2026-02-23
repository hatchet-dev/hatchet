-- name: GetDAGData :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@dagIds::bigint[]) AS dag_id,
                unnest(@dagInsertedAts::timestamptz[]) AS dag_inserted_at
        ) AS subquery
)
SELECT
    *
FROM
    v1_dag_data
JOIN
    input USING (dag_id, dag_inserted_at);

-- name: CreateDAGs :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@tenantIds::uuid[]) AS tenant_id,
                unnest(@externalIds::uuid[]) AS external_id,
                unnest(@displayNames::text[]) AS display_name,
                unnest(@workflowIds::uuid[]) AS workflow_id,
                unnest(@workflowVersionIds::uuid[]) AS workflow_version_id,
                unnest(@parentTaskExternalIds::uuid[]) AS parent_task_external_id
        ) AS subquery
)
INSERT INTO v1_dag (
    tenant_id,
    external_id,
    display_name,
    workflow_id,
    workflow_version_id,
    parent_task_external_id
)
SELECT
    i.tenant_id,
    i.external_id,
    i.display_name,
    i.workflow_id,
    i.workflow_version_id,
    NULLIF(i.parent_task_external_id, '00000000-0000-0000-0000-000000000000'::uuid)
FROM
    input i
RETURNING
    *;

-- name: CreateDAGData :copyfrom
INSERT INTO v1_dag_data (
    dag_id,
    dag_inserted_at,
    input,
    additional_metadata
) VALUES (
    $1,
    $2,
    $3,
    $4
);
