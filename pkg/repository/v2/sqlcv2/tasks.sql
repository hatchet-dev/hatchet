-- name: CreateTablePartition :exec
SELECT create_v2_task_partition(
    @date::date
);

-- name: ListTablePartitionsBeforeDate :many
SELECT 
    p::text AS partition_name
FROM 
    get_v2_task_partitions_before(
        @date::date
    ) AS p;

-- name: CreateTasks :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@tenantIds::uuid[]) AS tenant_id,
                unnest(@queues::text[]) AS queue,
                unnest(@actionIds::text[]) AS action_id,
                unnest(@stepIds::uuid[]) AS step_id,
                unnest(@workflowIds::uuid[]) AS workflow_id,
                unnest(@scheduleTimeouts::text[]) AS schedule_timeout,
                unnest(@stepTimeouts::text[]) AS step_timeout,
                unnest(@priorities::integer[]) AS priority,
                unnest(cast(@stickies::text[] as v2_sticky_strategy[])) AS sticky,
                unnest(@desiredWorkerIds::uuid[]) AS desired_worker_id,
                unnest(@externalIds::uuid[]) AS external_id,
                unnest(@displayNames::text[]) AS display_name,
                unnest(@inputs::jsonb[]) AS input,
                unnest(@retryCounts::integer[]) AS retry_count
        ) AS subquery
)
INSERT INTO v2_task (
    tenant_id,
    queue,
    action_id,
    step_id,
    workflow_id,
    schedule_timeout,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    external_id,
    display_name,
    input,
    retry_count
) 
SELECT
    i.tenant_id,
    i.queue,
    i.action_id,
    i.step_id,
    i.workflow_id,
    i.schedule_timeout,
    i.step_timeout,
    i.priority,
    i.sticky,
    i.desired_worker_id,
    i.external_id,
    i.display_name,
    i.input,
    i.retry_count
FROM
    input i 
RETURNING
    *;

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
    inserted_at,
    external_id,
    retry_count,
    workflow_id
FROM
    v2_task
WHERE
    tenant_id = $1
    AND id = ANY(@ids::bigint[]);

-- name: ReleaseTasks :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@retryCounts::integer[]) AS retry_count
        ) AS subquery
), runtimes_to_delete AS (
    SELECT
        task_id,
        retry_count
    FROM
        v2_task_runtime
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM input)
    ORDER BY
        task_id
    FOR UPDATE
), deleted_runtimes AS (
    DELETE FROM
        v2_task_runtime
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM runtimes_to_delete)
)
SELECT
    t.queue
FROM
    v2_task t
JOIN
    runtimes_to_delete r ON r.task_id = t.id AND r.retry_count = t.retry_count;


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

-- name: FailTaskAppFailure :many
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
    AND tasks_to_steps."retries" > v2_task.app_retry_count
RETURNING 
    v2_task.id,
    v2_task.retry_count;

-- name: FailTaskInternalFailure :many
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
    AND @maxInternalRetries::int > v2_task.internal_retry_count
RETURNING
    v2_task.id,
    v2_task.retry_count;

-- name: ProcessTaskTimeouts :many
WITH expired_runtimes AS (
    SELECT
        task_id,
        retry_count
    FROM
        v2_task_runtime
    WHERE
        tenant_id = @tenantId::uuid
        AND timeout_at <= NOW()
    ORDER BY
        task_id
    LIMIT
        COALESCE(sqlc.narg('limit')::integer, 1000)
    FOR UPDATE SKIP LOCKED
), locked_tasks AS (
    SELECT
        v2_task.id,
        v2_task.retry_count,
        v2_task.step_id
    FROM
        v2_task
    JOIN
        -- NOTE: we only join when retry count matches
        expired_runtimes ON expired_runtimes.task_id = v2_task.id AND expired_runtimes.retry_count = v2_task.retry_count
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), deleted_tqis AS (
    DELETE FROM
        v2_task_runtime
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM expired_runtimes)
), tasks_to_steps AS (
    SELECT
        t.id,
        t.step_id,
        s."retries"
    FROM
        locked_tasks t
    JOIN
        "Step" s ON s."id" = t.step_id
), updated_tasks AS (
    UPDATE
        v2_task
    SET
        retry_count = retry_count + 1,
        app_retry_count = app_retry_count + 1
    FROM
        tasks_to_steps
    WHERE
        v2_task.id = tasks_to_steps.id
        AND tasks_to_steps."retries" > v2_task.app_retry_count
)
SELECT
    *   
FROM
    locked_tasks;

-- name: ProcessTaskReassignments :many
WITH tasks_on_inactive_workers AS (
    SELECT
        runtime.task_id,
        runtime.retry_count,
        runtime.worker_id
    FROM
        "Worker" w
    JOIN
        v2_task_runtime runtime ON w."id" = runtime.worker_id
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '30 seconds'
    LIMIT
        COALESCE(sqlc.narg('limit')::integer, 1000)
), locked_runtimes AS (
    SELECT
        v2_task_runtime.task_id,
        v2_task_runtime.retry_count,
        tasks_on_inactive_workers.worker_id
    FROM
        v2_task_runtime
    JOIN
        tasks_on_inactive_workers ON tasks_on_inactive_workers.task_id = v2_task_runtime.task_id AND tasks_on_inactive_workers.retry_count = v2_task_runtime.retry_count
    ORDER BY
        task_id
    -- We do a SKIP LOCKED because a lock on v2_task_runtime means its being deleted
    FOR UPDATE SKIP LOCKED
), locked_tasks AS (
    SELECT
        v2_task.id,
        v2_task.retry_count,
        locked_runtimes.worker_id
    FROM
        v2_task
    JOIN
        -- NOTE: we only join when retry count matches
        locked_runtimes ON locked_runtimes.task_id = v2_task.id AND locked_runtimes.retry_count = v2_task.retry_count
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), deleted_runtimes AS (
    DELETE FROM
        v2_task_runtime
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM locked_runtimes)
), update_tasks AS (
    UPDATE
        v2_task
    SET
        retry_count = v2_task.retry_count + 1,
        internal_retry_count = v2_task.internal_retry_count + 1
    FROM
        locked_tasks
    WHERE
        v2_task.id = locked_tasks.id
        AND @maxInternalRetries::int > v2_task.internal_retry_count
    RETURNING 
        v2_task.id,
        v2_task.retry_count
), updated_tasks AS (
    SELECT
        *
    FROM
        locked_tasks
    WHERE
        id IN (SELECT id FROM update_tasks)
), failed_tasks AS (
    SELECT
        *
    FROM
        locked_tasks
    WHERE
        id NOT IN (SELECT id FROM update_tasks)
)
SELECT
    t1.id,
    t1.retry_count,
    t1.worker_id,
    'REASSIGNED' AS "operation"
FROM
    updated_tasks t1
UNION ALL
SELECT
    t2.id,
    t2.retry_count,
    t2.worker_id,
    'FAILED' AS "operation"
FROM    
    failed_tasks t2;