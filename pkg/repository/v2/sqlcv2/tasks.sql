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

-- name: ListTaskMetas :many
SELECT
    id,
    external_id,
    retry_count
FROM
    v2_task
WHERE
    tenant_id = $1
    AND id = ANY(@ids::bigint[]);

-- name: ReleaseQueueItems :exec
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@retryCounts::integer[]) AS retry_count
        ) AS subquery
), sqis_to_delete AS (
    SELECT
        task_id,
        retry_count
    FROM
        v2_semaphore_queue_item
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM input)
    ORDER BY
        task_id
    FOR UPDATE
), tqis_to_delete AS (
    SELECT
        task_id,
        retry_count
    FROM
        v2_timeout_queue_item
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM input)
    ORDER BY
        task_id
    FOR UPDATE
), deleted_sqis AS (
    DELETE FROM
        v2_semaphore_queue_item sqi
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM sqis_to_delete)
)
DELETE FROM
    v2_timeout_queue_item tqi
WHERE
    (task_id, retry_count) IN (SELECT task_id, retry_count FROM tqis_to_delete);

-- name: CreateTaskEvents :exec
-- We get a FOR UPDATE lock on tasks to prevent concurrent writes to the task events
-- tables for each task
WITH locked_tasks AS (
    SELECT
        id
    FROM
        v2_task
    WHERE
        id = ANY(@taskIds::bigint[])
        AND tenant_id = @tenantId::uuid
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@retryCounts::integer[]) AS retry_count,
                unnest(cast(@eventTypes::text[] as v2_task_event_type[])) AS event_type,
                unnest(@datas::jsonb[]) AS data
        ) AS subquery
)
INSERT INTO v2_task_event (
    tenant_id,
    task_id,
    retry_count,
    event_type,
    data
)
SELECT
    @tenantId::uuid,
    i.task_id,
    i.retry_count,
    i.event_type,
    i.data
FROM
    input i;

-- name: FailTaskAppFailure :exec
-- Fails a task due to an application-level error
WITH locked_tasks AS (
    SELECT
        id, 
        step_id
    FROM
        v2_task
    WHERE
        id = ANY(@taskIds::bigint[])
        AND tenant_id = @tenantId::uuid
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), tasks_to_steps AS (
    SELECT
        t.id,
        t.step_id,
        s."retries"
    FROM
        locked_tasks t
    JOIN
        "Step" s ON s."id" = t.step_id
    WHERE
        id = ANY(@taskIds::bigint[])
        AND tenant_id = @tenantId::uuid
)
UPDATE
    v2_task
SET
    retry_count = retry_count + 1,
    app_retry_count = app_retry_count + 1
FROM
    tasks_to_steps
WHERE
    v2_task.id = tasks_to_steps.id
    AND tasks_to_steps."retries" > v2_task.app_retry_count;

-- name: FailTaskInternalFailure :exec
-- Fails a task due to an application-level error
WITH locked_tasks AS (
    SELECT
        id
    FROM
        v2_task
    WHERE
        id = ANY(@taskIds::bigint[])
        AND tenant_id = @tenantId::uuid
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
)
UPDATE
    v2_task
SET
    retry_count = retry_count + 1,
    internal_retry_count = internal_retry_count + 1
FROM
    locked_tasks
WHERE
    v2_task.id = locked_tasks.id
    AND @maxInternalRetries::int > v2_task.internal_retry_count;