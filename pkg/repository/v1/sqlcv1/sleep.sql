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
    v1_durable_sleep (tenant_id, sleep_until)
SELECT
    @tenant_id::uuid,
    CURRENT_TIMESTAMP + convert_duration_to_interval(sleep_duration)
FROM
    input
RETURNING *;
