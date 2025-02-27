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
