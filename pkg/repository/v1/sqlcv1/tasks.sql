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

-- name: CreateTaskEventPartition :exec
SELECT create_v1_range_partition(
    'v1_task_event',
    @date::date
);

-- name: ListTaskEventPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v1_partitions_before_date(
        'v1_task_event',
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

-- name: FlattenExternalIds :many
WITH lookup_rows AS (
    SELECT
        *
    FROM
        v1_lookup_table l
    WHERE
        l.external_id = ANY(@externalIds::uuid[])
        AND l.tenant_id = @tenantId::uuid
), tasks_from_dags AS (
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        t.external_id,
        t.workflow_run_id,
        t.additional_metadata,
        t.dag_id,
        t.dag_inserted_at,
        t.parent_task_id,
        t.child_index,
        t.child_key,
        d.external_id AS workflow_run_external_id
    FROM
        lookup_rows l
    JOIN
        v1_dag d ON d.id = l.dag_id AND d.inserted_at = l.inserted_at
    JOIN
        v1_dag_to_task dt ON dt.dag_id = d.id AND dt.dag_inserted_at = d.inserted_at
    JOIN
        v1_task t ON t.id = dt.task_id AND t.inserted_at = dt.task_inserted_at
    WHERE
        l.dag_id IS NOT NULL
)
-- Union the tasks from the lookup table with the tasks from the DAGs
SELECT
    t.id,
    t.inserted_at,
    t.retry_count,
    t.external_id,
    t.workflow_run_id,
    t.additional_metadata,
    t.dag_id,
    t.dag_inserted_at,
    t.parent_task_id,
    t.child_index,
    t.child_key,
    t.external_id AS workflow_run_external_id
FROM
    lookup_rows l
JOIN
    v1_task t ON t.id = l.task_id AND t.inserted_at = l.inserted_at
WHERE
    l.task_id IS NOT NULL

UNION ALL

SELECT
    *
FROM
    tasks_from_dags;

-- name: LookupExternalIds :many
SELECT
    *
FROM
    v1_lookup_table
WHERE
    external_id = ANY(@externalIds::uuid[])
    AND tenant_id = @tenantId::uuid;

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
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at,
                unnest(@retryCounts::integer[]) AS retry_count
        ) AS subquery
), runtimes_to_delete AS (
    SELECT
        task_id,
        task_inserted_at,
        retry_count,
        worker_id
    FROM
        v1_task_runtime
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    ORDER BY
        task_id, task_inserted_at, retry_count
    FOR UPDATE
), deleted_runtimes AS (
    DELETE FROM
        v1_task_runtime
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM runtimes_to_delete)
), retry_queue_items_to_delete AS (
    SELECT
        task_id, task_inserted_at, task_retry_count
    FROM
        v1_retry_queue_item
    WHERE
        (task_id, task_inserted_at, task_retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    ORDER BY
        task_id, task_inserted_at, task_retry_count
    FOR UPDATE
), deleted_rqis AS (
    DELETE FROM
        v1_retry_queue_item r
    WHERE
        (task_id, task_inserted_at, task_retry_count) IN (
            SELECT
                task_id, task_inserted_at, task_retry_count
            FROM
                retry_queue_items_to_delete
        )
)
SELECT
    t.queue,
    t.id,
    t.inserted_at,
    t.external_id,
    t.step_readable_id,
    r.worker_id,
    i.retry_count::int AS retry_count,
    t.concurrency_strategy_ids
FROM
    v1_task t
JOIN
    input i ON i.task_id = t.id AND i.task_inserted_at = t.inserted_at
LEFT JOIN
    runtimes_to_delete r ON r.task_id = t.id AND r.retry_count = t.retry_count;

-- name: FailTaskAppFailure :many
-- Fails a task due to an application-level error
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at
        ) AS subquery
), locked_tasks AS (
    SELECT
        id,
        inserted_at,
        step_id
    FROM
        v1_task
    WHERE
        (id, inserted_at) IN (SELECT task_id, task_inserted_at FROM input)
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
    (v1_task.id, v1_task.inserted_at) IN (SELECT task_id, task_inserted_at FROM input)
    AND tasks_to_steps."retries" > v1_task.app_retry_count
RETURNING
    v1_task.id,
    v1_task.inserted_at,
    v1_task.retry_count,
    v1_task.app_retry_count,
    v1_task.retry_backoff_factor,
    v1_task.retry_max_backoff;

-- name: FailTaskInternalFailure :many
-- Fails a task due to an application-level error
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at
        ) AS subquery
), locked_tasks AS (
    SELECT
        id
    FROM
        v1_task
    WHERE
        (id, inserted_at) IN (SELECT task_id, task_inserted_at FROM input)
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
    (v1_task.id, v1_task.inserted_at) IN (SELECT task_id, task_inserted_at FROM input)
    AND @maxInternalRetries::int > v1_task.internal_retry_count
RETURNING
    v1_task.id,
    v1_task.inserted_at,
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
        task_id, task_inserted_at, retry_count
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
        runtime.task_inserted_at,
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
        v1_task_runtime.task_inserted_at,
        v1_task_runtime.retry_count,
        tasks_on_inactive_workers.worker_id
    FROM
        v1_task_runtime
    JOIN
        tasks_on_inactive_workers USING (task_id, task_inserted_at, retry_count)
    ORDER BY
        task_id
    -- We do a SKIP LOCKED because a lock on v1_task_runtime means its being deleted
    FOR UPDATE SKIP LOCKED
), locked_tasks AS (
    SELECT
        v1_task.id,
        v1_task.inserted_at,
        v1_task.retry_count,
        lrs.worker_id
    FROM
        v1_task
    JOIN
        -- NOTE: we only join when retry count matches
        locked_runtimes lrs ON lrs.task_id = v1_task.id AND lrs.task_inserted_at = v1_task.inserted_at AND lrs.retry_count = v1_task.retry_count
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), deleted_runtimes AS (
    DELETE FROM
        v1_task_runtime
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM locked_runtimes)
), update_tasks AS (
    UPDATE
        v1_task
    SET
        retry_count = v1_task.retry_count + 1,
        internal_retry_count = v1_task.internal_retry_count + 1
    FROM
        locked_tasks
    WHERE
        (v1_task.id, v1_task.inserted_at) IN (SELECT id, inserted_at FROM locked_tasks)
        AND @maxInternalRetries::int > v1_task.internal_retry_count
    RETURNING
        v1_task.id,
        v1_task.inserted_at,
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
    t1.inserted_at,
    t1.retry_count,
    t1.worker_id,
    'REASSIGNED' AS "operation"
FROM
    updated_tasks t1
UNION ALL
SELECT
    t2.id,
    t2.inserted_at,
    t2.retry_count,
    t2.worker_id,
    'FAILED' AS "operation"
FROM
    failed_tasks t2;

-- name: ProcessRetryQueueItems :many
WITH rqis_to_delete AS (
    SELECT
        *
    FROM
        v1_retry_queue_item rqi
    WHERE
        rqi.tenant_id = @tenantId::uuid
        AND rqi.retry_after <= NOW()
    ORDER BY
        rqi.task_id, rqi.task_inserted_at, rqi.task_retry_count
    LIMIT
        COALESCE(sqlc.narg('limit')::integer, 1000)
    FOR UPDATE SKIP LOCKED
)
DELETE FROM
    v1_retry_queue_item
WHERE
    (task_id, task_inserted_at, task_retry_count) IN (SELECT task_id, task_inserted_at, task_retry_count FROM rqis_to_delete)
RETURNING *;

-- name: ListMatchingTaskEvents :many
-- Lists the task events for the **latest** retry of a task, or task events which intentionally
-- aren't associated with a retry count (if the retry_count = -1).
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskExternalIds::uuid[]) AS task_external_id,
                -- can match any of the event types
                unnest_nd_1d(@eventTypes::text[][]) AS event_types
        ) AS subquery
)
SELECT
    t.external_id,
    e.*
FROM
    v1_lookup_table l
JOIN
    v1_task t ON t.id = l.task_id AND t.inserted_at = l.inserted_at
JOIN
    v1_task_event e ON e.tenant_id = @tenantId::uuid AND e.task_id = t.id AND e.task_inserted_at = t.inserted_at
JOIN
    input i ON i.task_external_id = l.external_id AND e.event_type::text = ANY(i.event_types)
WHERE
    l.tenant_id = @tenantId::uuid
    AND l.external_id = ANY(@taskExternalIds::uuid[])
    AND (e.retry_count = -1 OR e.retry_count = t.retry_count);

-- name: LockSignalCreatedEvents :many
-- Places a lock on the SIGNAL_CREATED events to make sure concurrent operations don't
-- modify the events.
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at,
                unnest(@eventKeys::text[]) AS event_key
        ) AS subquery
)
SELECT
    e.id,
    e.event_key,
    e.data
FROM
    v1_task_event e
JOIN
    input i ON i.task_id = e.task_id AND i.task_inserted_at = e.task_inserted_at AND i.event_key = e.event_key
WHERE
    e.tenant_id = @tenantId::uuid
    AND e.event_type = 'SIGNAL_CREATED'
ORDER BY
    e.id
FOR UPDATE;

-- name: ListMatchingSignalEvents :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at,
                unnest(@eventKeys::text[]) AS event_key
        ) AS subquery
)
SELECT
    e.*
FROM
    v1_task_event e
JOIN
    input i ON i.task_id = e.task_id AND i.task_inserted_at = e.task_inserted_at AND i.event_key = e.event_key
WHERE
    e.tenant_id = @tenantId::uuid
    AND e.event_type = @eventType::v1_task_event_type;

-- name: DeleteMatchingSignalEvents :exec
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at,
                unnest(@eventKeys::text[]) AS event_key
        ) AS subquery
), matching_events AS (
    SELECT
        e.id
    FROM
        v1_task_event e
    JOIN
        input i ON i.task_id = e.task_id AND i.task_inserted_at = e.task_inserted_at AND i.event_key = e.event_key
    WHERE
        e.tenant_id = @tenantId::uuid
        AND e.event_type = @eventType::v1_task_event_type
    ORDER BY
        e.id
    FOR UPDATE
)
DELETE FROM
    v1_task_event
WHERE
    id IN (SELECT id FROM matching_events);

-- name: ListTasksForReplay :many
-- Lists tasks for replay by recursively selecting all tasks that are children of the input tasks,
-- then locks the tasks for replay.
WITH RECURSIVE augmented_tasks AS (
    -- First, select the tasks from the input
    SELECT
        id,
        tenant_id,
        dag_id,
        step_id
    FROM
        v1_task
    WHERE
        (id, inserted_at) IN (
            SELECT
                unnest(@taskIds::bigint[]),
                unnest(@taskInsertedAts::timestamptz[])
        )
        AND tenant_id = @tenantId::uuid

    UNION

    -- Then, select the tasks that are children of the input tasks
    SELECT
        t.id,
        t.tenant_id,
        t.dag_id,
        t.step_id
    FROM
        augmented_tasks at
    JOIN
        "Step" s1 ON s1."id" = at.step_id
    JOIN
        v1_dag_to_task dt ON dt.dag_id = at.dag_id
    JOIN
        v1_task t ON t.id = dt.task_id
    JOIN
        "Step" s2 ON s2."id" = t.step_id
    JOIN
        "_StepOrder" so ON so."B" = s2."id" AND so."A" = s1."id"
), locked_tasks AS (
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
        t.parent_task_external_id,
        t.parent_task_id,
        t.parent_task_inserted_at,
        t.step_index,
        t.child_index,
        t.child_key
    FROM
        v1_task t
    JOIN
        -- TODO: USE INSERTED_AT
        augmented_tasks at ON at.id = t.id AND at.tenant_id = t.tenant_id
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
    t.parent_task_external_id,
    t.parent_task_id,
    t.parent_task_inserted_at,
    t.step_index,
    t.child_index,
    t.child_key,
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

-- name: LockDAGsForReplay :many
-- Locks a list of DAGs for replay. Returns successfully locked DAGs which can be replayed.
SELECT
    id
FROM
    v1_dag
WHERE
    id = ANY(@dagIds::bigint[])
    AND tenant_id = @tenantId::uuid
ORDER BY id
-- We skip locked tasks because replays are the only thing that can lock a DAG for updates
FOR UPDATE SKIP LOCKED;

-- name: PreflightCheckDAGsForReplay :many
-- Checks whether DAGs can be replayed by ensuring that the length of the tasks which have been written
-- match the length of steps in the DAG. This assumes that we have a lock on DAGs so concurrent replays
-- don't interfere with each other. It also does not check for whether the tasks are running, as that's
-- checked in a different query. It returns DAGs which cannot be replayed.
WITH dags_to_step_counts AS (
    SELECT
        d.id,
        d.external_id,
        d.inserted_at,
        COUNT(DISTINCT s."id") as step_count,
        COUNT(DISTINCT dt.task_id) as task_count
    FROM
        v1_dag d
    JOIN
        v1_dag_to_task dt ON dt.dag_id = d.id
    JOIN
        "WorkflowVersion" wv ON wv."id" = d.workflow_version_id
    LEFT JOIN
        "Job" j ON j."workflowVersionId" = wv."id"
    LEFT JOIN
        "Step" s ON s."jobId" = j."id"
    WHERE
        d.id = ANY(@dagIds::bigint[])
        AND d.tenant_id = @tenantId::uuid
    GROUP BY
        d.id,
        d.inserted_at
)
SELECT
    d.id,
    d.external_id,
    d.inserted_at,
    d.step_count,
    d.task_count
FROM
    dags_to_step_counts d;

-- name: PreflightCheckTasksForReplay :many
-- Checks whether tasks can be replayed by ensuring that they don't have any active runtimes,
-- concurrency slots, or retry queue items. Returns the tasks which cannot be replayed.
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at
        ) AS subquery
)
SELECT
    t.id,
    t.dag_id
FROM
    v1_task t
LEFT JOIN
    v1_task_event e ON e.task_id = t.id AND e.task_inserted_at = t.inserted_at AND e.retry_count = t.retry_count AND e.event_type = ANY('{COMPLETED, FAILED, CANCELLED}'::v1_task_event_type[])
LEFT JOIN
    v1_task_runtime tr ON tr.task_id = t.id
LEFT JOIN
    v1_concurrency_slot cs ON cs.task_id = t.id
LEFT JOIN
    v1_retry_queue_item rqi ON rqi.task_id = t.id
WHERE
    (t.id, t.inserted_at) IN (
        SELECT
            task_id,
            task_inserted_at
        FROM
            input
    )
    AND t.tenant_id = @tenantId::uuid
    AND e.id IS NULL
    AND (tr.task_id IS NOT NULL OR cs.task_id IS NOT NULL OR rqi.task_id IS NOT NULL);

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

-- name: ListTaskExpressionEvals :many
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@taskInsertedAts::timestamptz[]) AS task_inserted_at
        ) AS subquery
)
SELECT
    *
FROM
    v1_task_expression_eval te
WHERE
    (task_id, task_inserted_at) IN (
        SELECT
            task_id,
            task_inserted_at
        FROM
            input
    );

-- name: RefreshTimeoutBy :one
WITH task AS (
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        t.tenant_id
    FROM
        v1_lookup_table lt
    JOIN
        v1_task t ON t.id = lt.task_id AND t.inserted_at = lt.inserted_at
    WHERE
        lt.external_id = @externalId::uuid AND
        lt.tenant_id = @tenantId::uuid
), locked_runtime AS (
    SELECT
        tr.task_id,
        tr.task_inserted_at,
        tr.retry_count,
        tr.worker_id
    FROM
        v1_task_runtime tr
    WHERE
        (tr.task_id, tr.task_inserted_at, tr.retry_count) IN (SELECT id, inserted_at, retry_count FROM task)
    ORDER BY
        task_id, task_inserted_at, retry_count
    FOR UPDATE
)
UPDATE
    v1_task_runtime
SET
    timeout_at = timeout_at + convert_duration_to_interval(sqlc.narg('incrementTimeoutBy')::text)
FROM
    task
WHERE
    (v1_task_runtime.task_id, v1_task_runtime.task_inserted_at, v1_task_runtime.retry_count) IN (SELECT id, inserted_at, retry_count FROM task)
RETURNING
    v1_task_runtime.*;

-- name: ManualSlotRelease :one
WITH task AS (
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        t.tenant_id
    FROM
        v1_lookup_table lt
    JOIN
        v1_task t ON t.id = lt.task_id AND t.inserted_at = lt.inserted_at
    WHERE
        lt.external_id = @externalId::uuid AND
        lt.tenant_id = @tenantId::uuid
), locked_runtime AS (
    SELECT
        tr.task_id,
        tr.task_inserted_at,
        tr.retry_count,
        tr.worker_id
    FROM
        v1_task_runtime tr
    WHERE
        (tr.task_id, tr.task_inserted_at, tr.retry_count) IN (SELECT id, inserted_at, retry_count FROM task)
    ORDER BY
        task_id, task_inserted_at, retry_count
    FOR UPDATE
)
UPDATE
    v1_task_runtime
SET
    worker_id = NULL
FROM
    task
WHERE
    (v1_task_runtime.task_id, v1_task_runtime.task_inserted_at, v1_task_runtime.retry_count) IN (SELECT id, inserted_at, retry_count FROM task)
RETURNING
    v1_task_runtime.*;
