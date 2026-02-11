-- name: CreateMatchesForSignalTriggers :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@tenantIds::uuid[]) AS tenant_id,
                unnest(cast(@kinds::text[] as v1_match_kind[])) AS kind,
                unnest(@signalTaskIds::bigint[]) AS signal_task_id,
                unnest(@signalTaskInsertedAts::timestamptz[]) AS signal_task_inserted_at,
                unnest(@signalExternalIds::uuid[]) AS signal_external_id,
                unnest(@signalKeys::text[]) AS signal_key,
                unnest(@callbackDurableTaskIds::bigint[]) AS callback_durable_task_id,
                unnest(@callbackDurableTaskInsertedAts::timestamptz[]) AS callback_durable_task_inserted_at,
                unnest(@callbackNodeIds::bigint[]) AS callback_node_id,
                unnest(@callbackDurableTaskExternalIds::uuid[]) AS callback_durable_task_external_id
        ) AS subquery
)
INSERT INTO v1_match (
    tenant_id,
    kind,
    signal_task_id,
    signal_task_inserted_at,
    signal_external_id,
    signal_key,
    durable_event_log_callback_durable_task_id,
    durable_event_log_callback_durable_task_inserted_at,
    durable_event_log_callback_node_id,
    durable_event_log_callback_durable_task_external_id
)
SELECT
    i.tenant_id,
    i.kind,
    i.signal_task_id,
    i.signal_task_inserted_at,
    i.signal_external_id,
    i.signal_key,
    i.callback_durable_task_id,
    i.callback_durable_task_inserted_at,
    i.callback_node_id,
    i.callback_durable_task_external_id
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
    event_resource_hint,
    readable_data_key,
    or_group_id,
    expression,
    action,
    is_satisfied,
    data
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11
);

-- name: GetSatisfiedMatchConditions :many
-- NOTE: we have to break this into a separate query because CTEs can't see modified rows
-- on the same target table without using RETURNING.
WITH input AS (
    SELECT
        UNNEST(@matchIds::BIGINT[]) AS match_id,
        UNNEST(@conditionIds::BIGINT[]) AS condition_id,
        UNNEST(@datas::JSONB[]) AS data
), locked_matches AS (
    SELECT
        m.id
    FROM
        v1_match m
    WHERE
        m.id = ANY(@matchIds::BIGINT[])
    ORDER BY
        m.id
    FOR UPDATE
), locked_conditions AS (
    SELECT
        m.v1_match_id,
        m.id,
        i.data
    FROM
        v1_match_condition m
    JOIN
        input i ON i.match_id = m.v1_match_id AND i.condition_id = m.id
    JOIN
        locked_matches lm ON lm.id = m.v1_match_id
    ORDER BY
        m.id
    FOR UPDATE
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
)
SELECT
    m.id
FROM
    locked_matches m;

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
        COUNT(DISTINCT CASE WHEN action = 'QUEUE' THEN or_group_id END) AS total_queue_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'QUEUE' THEN or_group_id END) AS satisfied_queue_groups,
        COUNT(DISTINCT CASE WHEN action = 'CANCEL' THEN or_group_id END) AS total_cancel_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'CANCEL' THEN or_group_id END) AS satisfied_cancel_groups,
        COUNT(DISTINCT CASE WHEN action = 'SKIP' THEN or_group_id END) AS total_skip_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'SKIP' THEN or_group_id END) AS satisfied_skip_groups,
        COUNT(DISTINCT CASE WHEN action = 'CREATE_MATCH' THEN or_group_id END) AS total_create_match_groups,
        COUNT(DISTINCT CASE WHEN is_satisfied AND action = 'CREATE_MATCH' THEN or_group_id END) AS satisfied_create_match_groups
    FROM v1_match_condition main
    WHERE v1_match_id = ANY(@matchIds::bigint[])
    GROUP BY v1_match_id
), result_matches AS (
    SELECT
        m.*,
        CASE WHEN
            (mc.total_skip_groups > 0 AND mc.total_skip_groups = mc.satisfied_skip_groups) THEN 'SKIP'
            WHEN (mc.total_cancel_groups > 0 AND mc.total_cancel_groups = mc.satisfied_cancel_groups) THEN 'CANCEL'
            WHEN (mc.total_create_groups > 0 AND mc.total_create_groups = mc.satisfied_create_groups) THEN 'CREATE'
            WHEN (mc.total_queue_groups > 0 AND mc.total_queue_groups = mc.satisfied_queue_groups) THEN 'QUEUE'
            WHEN (mc.total_create_match_groups > 0 AND mc.total_create_match_groups = mc.satisfied_create_match_groups) THEN 'CREATE_MATCH'
        END::v1_match_condition_action AS action
    FROM
        v1_match m
    JOIN
        match_counts mc ON m.id = mc.v1_match_id
    WHERE
        (
            (mc.total_create_groups > 0 AND mc.total_create_groups = mc.satisfied_create_groups)
            OR (mc.total_queue_groups > 0 AND mc.total_queue_groups = mc.satisfied_queue_groups)
            OR (mc.total_cancel_groups > 0 AND mc.total_cancel_groups = mc.satisfied_cancel_groups)
            OR (mc.total_skip_groups > 0 AND mc.total_skip_groups = mc.satisfied_skip_groups)
            OR (mc.total_create_match_groups > 0 AND mc.total_create_match_groups = mc.satisfied_create_match_groups)
        )
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
    RETURNING
        v1_match_id AS id
), matches_with_data AS (
    SELECT
        m.id,
        m.action,
        (
            SELECT jsonb_object_agg(action, aggregated_1)
            FROM (
                SELECT action, jsonb_object_agg(readable_data_key, data_array) AS aggregated_1
                FROM (
                    SELECT mc.action, readable_data_key, jsonb_agg(data) AS data_array
                    FROM v1_match_condition mc
                    WHERE mc.v1_match_id = m.id AND mc.is_satisfied AND mc.action = m.action
                    GROUP BY mc.action, readable_data_key
                ) t
                GROUP BY action
            ) s
        )::jsonb AS mc_aggregated_data
    FROM
        result_matches m
    GROUP BY
        m.id, m.action
), deleted_matches AS (
    DELETE FROM
        v1_match
    WHERE
        id IN (SELECT id FROM deleted_conditions)
)
SELECT
    rm.*,
    COALESCE(rm.existing_data || d.mc_aggregated_data, d.mc_aggregated_data)::jsonb AS mc_aggregated_data
FROM
    result_matches rm
LEFT JOIN
    matches_with_data d ON rm.id = d.id;

-- name: ResetMatchConditions :many
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

-- name: CleanupMatchWithMatchConditions :exec
WITH deleted_match_ids AS (
    DELETE FROM
        v1_match
    WHERE
        signal_task_inserted_at < @date::date
        OR trigger_dag_inserted_at < @date::date
        OR trigger_parent_task_inserted_at < @date::date
        OR trigger_existing_task_inserted_at < @date::date
    RETURNING
        id
)
DELETE FROM
    v1_match_condition
WHERE
    v1_match_id IN (SELECT id FROM deleted_match_ids);
