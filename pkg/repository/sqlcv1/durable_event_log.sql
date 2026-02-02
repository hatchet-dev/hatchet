-- name: GetDurableEventLog :one
SELECT *
FROM v1_durable_event_log
WHERE
    tenant_id = @tenant_id::uuid
    AND task_id = @task_id::bigint
    AND task_inserted_at = @task_inserted_at::timestamptz
    AND retry_count = @retry_count::integer
    AND key = @key::text
LIMIT 1;

-- name: CreateDurableEventLog :one
INSERT INTO v1_durable_event_log (
    tenant_id,
    task_id,
    task_inserted_at,
    retry_count,
    event_type,
    key,
    data
) VALUES (
    @tenant_id::uuid,
    @task_id::bigint,
    @task_inserted_at::timestamptz,
    @retry_count::integer,
    @event_type::text,
    @key::text,
    @data::jsonb
)
RETURNING *;
