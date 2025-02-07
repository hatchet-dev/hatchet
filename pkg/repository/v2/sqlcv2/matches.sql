-- name: CreateMatchesForDAGTriggers :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@tenantIds::uuid[]) AS tenant_id,
                unnest(cast(@kinds::text[] as v2_match_kind[])) AS kind,
                unnest(@triggerDagIds::bigint[]) AS trigger_dag_id,
                unnest(@triggerDagInsertedAts::timestamptz[]) AS trigger_dag_inserted_at,
                unnest(@triggerStepIds::uuid[]) AS trigger_step_id,
                unnest(@triggerExternalIds::uuid[]) AS trigger_external_id
        ) AS subquery
)
INSERT INTO v2_match (
    tenant_id,
    kind,
    trigger_dag_id,
    trigger_dag_inserted_at,
    trigger_step_id,
    trigger_external_id
)
SELECT
    i.tenant_id,
    i.kind,
    i.trigger_dag_id,
    i.trigger_dag_inserted_at,
    i.trigger_step_id,
    i.trigger_external_id
FROM
    input i
RETURNING
    *;

-- name: CreateMatchConditions :copyfrom
INSERT INTO v2_match_condition (
    v2_match_id,
    tenant_id,
    event_type,
    event_key,
    or_group_id,
    expression
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
);

-- name: ListMatchConditionsForEvent :many
SELECT
    v2_match_id,
    id,
    registered_at,
    event_type,
    event_key,
    expression
FROM
    v2_match_condition m
WHERE
    m.tenant_id = @tenantId::uuid
    AND m.event_type = @eventType::v2_event_type
    AND m.event_key = ANY(@eventKeys::text[])
    AND NOT m.is_satisfied;

-- name: GetSatisfiedMatchConditions :many
-- NOTE: we have to break this into a separate query because CTEs can't see modified rows
-- on the same target table without using RETURNING.
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@matchIds::bigint[]) AS match_id,
                unnest(@conditionIds::bigint[]) AS condition_id,
                unnest(@datas::jsonb[]) AS data
        ) AS subquery
), locked_conditions AS (
    SELECT
        m.v2_match_id,
        m.id,
        i.data
    FROM
        v2_match_condition m
    JOIN
        input i ON i.match_id = m.v2_match_id AND i.condition_id = m.id
    ORDER BY
        m.id
    -- We can afford a SKIP LOCKED because a match condition can only be satisfied by 1 event
    -- at a time
    FOR UPDATE SKIP LOCKED
), updated_conditions AS (
    UPDATE
        v2_match_condition
    SET
        is_satisfied = TRUE,
        data = c.data
    FROM
        locked_conditions c
    WHERE
        (v2_match_condition.v2_match_id, v2_match_condition.id) = (c.v2_match_id, c.id)
    RETURNING
        v2_match_condition.v2_match_id, v2_match_condition.id
), distinct_match_ids AS (
    SELECT
        DISTINCT v2_match_id
    FROM
        updated_conditions
)
SELECT
    m.id
FROM
    v2_match m
JOIN
    distinct_match_ids dm ON dm.v2_match_id = m.id
ORDER BY
    m.id
FOR UPDATE;

-- name: SaveSatisfiedMatchConditions :many
-- NOTE: we have to break this into a separate query because CTEs can't see modified rows
-- on the same target table without using RETURNING.
-- Additionally, since we've placed a FOR UPDATE lock in the previous query, we're guaranteeing
-- that only one transaction can update these rows,so this should be concurrency-safe.
WITH match_counts AS (
    SELECT
        v2_match_id,
        COUNT(DISTINCT or_group_id) AS total_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied THEN or_group_id END) AS satisfied_groups,
        (
            SELECT jsonb_object_agg(event_key, data_array)
            FROM (
                SELECT event_key, jsonb_agg(data) AS data_array
                FROM v2_match_condition sub
                WHERE sub.v2_match_id = ANY(@matchIds::bigint[])
                AND is_satisfied
                GROUP BY event_key
            ) aggregated
        ) AS aggregated_data
    FROM v2_match_condition main
    WHERE v2_match_id = ANY(@matchIds::bigint[])
    GROUP BY v2_match_id
)
UPDATE
    v2_match m
SET
    is_satisfied = TRUE
FROM
    match_counts mc
WHERE
    m.id = mc.v2_match_id
AND
    mc.total_groups = mc.satisfied_groups
RETURNING m.*, mc.aggregated_data::jsonb;
