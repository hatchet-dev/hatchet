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
                unnest(@triggerExternalIds::uuid[]) AS trigger_external_id
        ) AS subquery
)
INSERT INTO v1_match (
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

-- name: CreateMatchesForSignalTriggers :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@tenantIds::uuid[]) AS tenant_id,
                unnest(cast(@kinds::text[] as v1_match_kind[])) AS kind,
                unnest(@signalTargetIds::bigint[]) AS signal_target_id,
                unnest(@signalKeys::text[]) AS signal_key
        ) AS subquery
)
INSERT INTO v1_match (
    tenant_id,
    kind,
    signal_target_id,
    signal_key
)
SELECT
    i.tenant_id,
    i.kind,
    i.signal_target_id,
    i.signal_key
FROM
    input i
RETURNING
    *;

-- name: CreateMatchConditions :copyfrom
INSERT INTO v1_match_condition (
    v1_match_id,
    tenant_id,
    event_type,
    event_key,
    or_group_id,
    expression,
    action
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
);

-- name: ListMatchConditionsForEvent :many
SELECT
    v1_match_id,
    id,
    registered_at,
    event_type,
    event_key,
    expression
FROM
    v1_match_condition m
WHERE
    m.tenant_id = @tenantId::uuid
    AND m.event_type = @eventType::v1_event_type
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
        m.v1_match_id,
        m.id,
        i.data
    FROM
        v1_match_condition m
    JOIN
        input i ON i.match_id = m.v1_match_id AND i.condition_id = m.id
    ORDER BY
        m.id
    -- We can afford a SKIP LOCKED because a match condition can only be satisfied by 1 event
    -- at a time
    FOR UPDATE SKIP LOCKED
), updated_conditions AS (
    UPDATE
        v1_match_condition
    SET
        is_satisfied = TRUE,
        data = c.data
    FROM
        locked_conditions c
    WHERE
        (v1_match_condition.v1_match_id, v1_match_condition.id) = (c.v1_match_id, c.id)
    RETURNING
        v1_match_condition.v1_match_id, v1_match_condition.id
), distinct_match_ids AS (
    SELECT
        DISTINCT v1_match_id
    FROM
        updated_conditions
)
SELECT
    m.id
FROM
    v1_match m
JOIN
    distinct_match_ids dm ON dm.v1_match_id = m.id
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
        v1_match_id,
        COUNT(DISTINCT CASE WHEN action = 'CREATE' THEN or_group_id END) AS total_create_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'CREATE' THEN or_group_id END) AS satisfied_create_groups,
        COUNT(DISTINCT CASE WHEN action = 'CANCEL' THEN or_group_id END) AS total_cancel_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'CANCEL' THEN or_group_id END) AS satisfied_cancel_groups,
        (
            SELECT jsonb_object_agg(action, aggregated_1)
            FROM (
                SELECT action, jsonb_object_agg(event_key, data_array) AS aggregated_1
                FROM (
                    SELECT action, event_key, jsonb_agg(data) AS data_array
                    FROM v1_match_condition sub
                    WHERE sub.v1_match_id = ANY(@matchIds::bigint[])
                    AND is_satisfied
                    GROUP BY action, event_key
                ) t
                GROUP BY action
            ) s
        ) AS aggregated_data
    FROM v1_match_condition main
    WHERE v1_match_id = ANY(@matchIds::bigint[])
    GROUP BY v1_match_id
), result_matches AS (
    SELECT
        m.*,
        mc.aggregated_data::jsonb as mc_aggregated_data
    FROM
        v1_match m
    JOIN
        match_counts mc ON m.id = mc.v1_match_id
    WHERE
        (
            mc.total_create_groups = mc.satisfied_create_groups
            OR mc.total_cancel_groups = mc.satisfied_cancel_groups
        )
), deleted_matches AS (
    DELETE FROM
        v1_match
    WHERE
        id IN (SELECT id FROM result_matches)
), locked_conditions AS (
    SELECT
        m.v1_match_id,
        m.id
    FROM
        v1_match_condition m
    JOIN
        result_matches r ON r.id = m.v1_match_id
    ORDER BY
        m.id
    FOR UPDATE
), deleted_conditions AS (
    DELETE FROM
        v1_match_condition
    WHERE
        (v1_match_id, id) IN (SELECT v1_match_id, id FROM locked_conditions)
)
SELECT
    *
FROM
    result_matches;
