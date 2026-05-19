-- name: CreateDurableSleep :many
WITH input AS (
    SELECT
        sleep_duration
    FROM (
        SELECT
            unnest(@sleep_durations::text[]) as sleep_duration
    ) as subquery
)
INSERT INTO
    v1_durable_sleep (tenant_id, sleep_until, sleep_duration)
SELECT
    @tenant_id::uuid,
    CURRENT_TIMESTAMP + convert_duration_to_interval(sleep_duration),
    sleep_duration
FROM
    input
RETURNING *;

-- name: PopDurableSleep :many
WITH to_delete AS (
    SELECT
        *
    FROM
        v1_durable_sleep
    WHERE
        tenant_id = @tenant_id::uuid
        AND sleep_until <= CURRENT_TIMESTAMP
    ORDER BY
        id ASC
    LIMIT
        COALESCE(sqlc.narg('limit')::integer, 1000)
    FOR UPDATE
)
DELETE FROM
    v1_durable_sleep
WHERE
    (tenant_id, sleep_until, id) IN (SELECT tenant_id, sleep_until, id FROM to_delete)
RETURNING *;
