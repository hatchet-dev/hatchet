-- name: CreateTasks :copyfrom
INSERT INTO v2_task (
    tenant_id,
    queue,
    action_id,
    step_id,
    schedule_timeout,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    external_id,
    display_name,
    input,
    retry_count
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
    $11,
    $12,
    $13
);

-- name: ListTasks :many
SELECT
    *
FROM
    v2_task
WHERE
    tenant_id = $1
    AND id = ANY(@ids::bigint[]);

-- name: ReleaseQueueItems :exec
-- TODO: PROCESS THIS IN BULK?
WITH delete_sqi AS (
    DELETE FROM
        v2_semaphore_queue_item sqi
    WHERE
        sqi.task_id = @task_id AND
        sqi.retry_count = @retry_count AND
        sqi.tenant_id = @tenant_id
)
DELETE FROM
    v2_timeout_queue_item tqi
WHERE
    tqi.task_id = @task_id AND
    tqi.retry_count = @retry_count AND
    tqi.tenant_id = @tenant_id;