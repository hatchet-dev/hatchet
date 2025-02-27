-- name: CreateTaskPartition :exec
SELECT create_v1_range_partition(
    'v1_task',
    @date::date
);

-- name: ListTaskPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v1_partitions_before_date(
        'v1_task',
        @date::date
    ) AS p;

-- name: CreateConcurrencyPartition :exec
SELECT create_v1_range_partition(
    'v1_concurrency_slot',
    @date::date
);

-- name: ListConcurrencyPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v1_partitions_before_date(
        'v1_concurrency_slot',
        @date::date
    ) AS p;

-- name: ListTasks :many
SELECT
    *
FROM
    v1_task
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
    v1_task
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
        retry_count,
        worker_id
    FROM
        v1_task_runtime
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM input)
    ORDER BY
        task_id
    FOR UPDATE
), deleted_runtimes AS (
    DELETE FROM
        v1_task_runtime
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM runtimes_to_delete)
)
SELECT
    t.queue,
    t.id,
    t.inserted_at,
    t.external_id,
    t.step_readable_id,
    r.worker_id,
    t.retry_count,
    t.concurrency_strategy_ids
FROM
    v1_task t
JOIN
    runtimes_to_delete r ON r.task_id = t.id AND r.retry_count = t.retry_count;

-- name: FailTaskAppFailure :many
-- Fails a task due to an application-level error
WITH locked_tasks AS (
    SELECT
        id,
        step_id
    FROM
        v1_task
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
    v1_task
SET
    retry_count = retry_count + 1,
    app_retry_count = app_retry_count + 1
FROM
    tasks_to_steps
WHERE
    v1_task.id = tasks_to_steps.id
    AND tasks_to_steps."retries" > v1_task.app_retry_count
RETURNING
    v1_task.id,
    v1_task.retry_count;

-- name: FailTaskInternalFailure :many
-- Fails a task due to an application-level error
WITH locked_tasks AS (
    SELECT
        id
    FROM
        v1_task
    WHERE
        id = ANY(@taskIds::bigint[])
        AND tenant_id = @tenantId::uuid
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
)
UPDATE
    v1_task
SET
    retry_count = retry_count + 1,
    internal_retry_count = internal_retry_count + 1
FROM
    locked_tasks
WHERE
    v1_task.id = locked_tasks.id
    AND @maxInternalRetries::int > v1_task.internal_retry_count
RETURNING
    v1_task.id,
    v1_task.retry_count;

-- name: ProcessTaskTimeouts :many
WITH expired_runtimes AS (
    SELECT
        task_id,
        retry_count,
        worker_id
    FROM
        v1_task_runtime
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
        v1_task.id,
        v1_task.retry_count,
        v1_task.step_id,
        expired_runtimes.worker_id
    FROM
        v1_task
    JOIN
        -- NOTE: we only join when retry count matches
        expired_runtimes ON expired_runtimes.task_id = v1_task.id AND expired_runtimes.retry_count = v1_task.retry_count
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), deleted_tqis AS (
    DELETE FROM
        v1_task_runtime
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
        v1_task
    SET
        retry_count = retry_count + 1,
        app_retry_count = app_retry_count + 1
    FROM
        tasks_to_steps
    WHERE
        v1_task.id = tasks_to_steps.id
        AND tasks_to_steps."retries" > v1_task.app_retry_count
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
        v1_task_runtime runtime ON w."id" = runtime.worker_id
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '30 seconds'
    LIMIT
        COALESCE(sqlc.narg('limit')::integer, 1000)
), locked_runtimes AS (
    SELECT
        v1_task_runtime.task_id,
        v1_task_runtime.retry_count,
        tasks_on_inactive_workers.worker_id
    FROM
        v1_task_runtime
    JOIN
        tasks_on_inactive_workers ON tasks_on_inactive_workers.task_id = v1_task_runtime.task_id AND tasks_on_inactive_workers.retry_count = v1_task_runtime.retry_count
    ORDER BY
        task_id
    -- We do a SKIP LOCKED because a lock on v1_task_runtime means its being deleted
    FOR UPDATE SKIP LOCKED
), locked_tasks AS (
    SELECT
        v1_task.id,
        v1_task.retry_count,
        locked_runtimes.worker_id
    FROM
        v1_task
    JOIN
        -- NOTE: we only join when retry count matches
        locked_runtimes ON locked_runtimes.task_id = v1_task.id AND locked_runtimes.retry_count = v1_task.retry_count
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), deleted_runtimes AS (
    DELETE FROM
        v1_task_runtime
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM locked_runtimes)
), update_tasks AS (
    UPDATE
        v1_task
    SET
        retry_count = v1_task.retry_count + 1,
        internal_retry_count = v1_task.internal_retry_count + 1
    FROM
        locked_tasks
    WHERE
        v1_task.id = locked_tasks.id
        AND @maxInternalRetries::int > v1_task.internal_retry_count
    RETURNING
        v1_task.id,
        v1_task.retry_count
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

-- name: ListMatchingSignalEvents :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@signalKeys::text[]) AS signal_key
        ) AS subquery
)
SELECT
    e.*
FROM
    v1_task_event e
JOIN
    input i ON i.task_id = e.task_id AND i.signal_key = e.event_key
WHERE
    e.tenant_id = @tenantId::uuid
    AND e.event_type = @eventType::v1_task_event_type;

-- name: LockTasksForReplay :many
WITH locked_tasks AS (
    SELECT
        id,
        inserted_at,
        retry_count,
        dag_id,
        dag_inserted_at,
        step_readable_id,
        step_id,
        workflow_id,
        external_id,
        additional_metadata
    FROM
        v1_task
    WHERE
        id = ANY(@taskIds::bigint[])
        AND tenant_id = @tenantId::uuid
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), step_orders AS (
    SELECT
        t.step_id,
        array_agg(so."A")::uuid[] as "parents"
    FROM
        locked_tasks t
    JOIN
        "Step" s ON s."id" = t.step_id
    JOIN
        "_StepOrder" so ON so."B" = s."id"
    GROUP BY
        t.step_id
)
SELECT
    t.id,
    t.inserted_at,
    t.retry_count,
    t.dag_id,
    t.dag_inserted_at,
    t.step_readable_id,
    t.step_id,
    t.workflow_id,
    t.external_id,
    t.additional_metadata,
    j."kind" as "jobKind",
    COALESCE(so."parents", '{}'::uuid[]) as "parents"
FROM
    locked_tasks t
JOIN
    "Step" s ON s."id" = t.step_id
JOIN
    "Job" j ON j."id" = s."jobId"
LEFT JOIN
    step_orders so ON so.step_id = t.step_id;

-- name: ListAllTasksInDags :many
SELECT
    t.id,
    t.inserted_at,
    t.retry_count,
    t.dag_id,
    t.dag_inserted_at,
    t.step_readable_id,
    t.step_id,
    t.workflow_id,
    t.external_id
FROM
    v1_task t
JOIN
    v1_dag_to_task dt ON dt.task_id = t.id
WHERE
    t.tenant_id = @tenantId::uuid
    AND dt.dag_id = ANY(@dagIds::bigint[]);

-- name: ListMatchingTaskEvents :many
-- Lists matching task events for a set of (dag, task, event_key) tuples
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@dagIds::bigint[]) AS dag_id,
                unnest(@taskExternalIds::uuid[]) AS task_external_id,
                unnest(@eventKeys::text[]) AS event_key
        ) AS subquery
)
SELECT
    DISTINCT ON(e.tenant_id, e.task_id, e.event_key)
    e.*,
    t.external_id AS task_external_id
FROM
    v1_dag d
JOIN
    input i ON i.dag_id = d.id
JOIN
    v1_dag_to_task dt ON dt.dag_id = d.id AND dt.dag_inserted_at = d.inserted_at
JOIN
    v1_task t ON t.id = dt.task_id AND t.tenant_id = d.tenant_id AND t.external_id = i.task_external_id
JOIN
    v1_task_event e ON e.tenant_id = @tenantId::uuid AND e.task_id = dt.task_id AND e.event_key = i.event_key
ORDER BY
    e.tenant_id,
    e.task_id,
    e.event_key,
    e.retry_count DESC;
