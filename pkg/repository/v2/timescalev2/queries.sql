-- name: CreateOLAPPartitions :exec
SELECT create_v2_task_events_partitions(
    @partitions::int
);

-- name: CreateOLAPTaskPartition :exec
SELECT create_v2_tasks_olap_partition(
    @date::date
);

-- name: ListOLAPTaskPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v2_tasks_olap_partitions_before(
        @date::date
    ) AS p;

-- name: CreateTasksOLAP :copyfrom
INSERT INTO v2_tasks_olap (
    tenant_id,
    id,
    inserted_at,
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
    additional_metadata
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
    $13,
    $14,
    $15,
    $16
);

-- name: CreateTaskEventsOLAPTmp :copyfrom
INSERT INTO v2_task_events_olap_tmp (
    tenant_id,
    task_id,
    task_inserted_at,
    event_type,
    readable_status,
    retry_count,
    worker_id
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
);

-- name: CreateTaskEventsOLAP :copyfrom
INSERT INTO v2_task_events_olap (
    tenant_id,
    task_id,
    task_inserted_at,
    event_type,
    workflow_id,
    event_timestamp,
    readable_status,
    retry_count,
    error_message,
    output,
    worker_id,
    additional__event_data,
    additional__event_message
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

-- name: GetTenantStatusMetrics :one
SELECT
  COALESCE(SUM(queued_count), 0)::bigint AS total_queued,
  COALESCE(SUM(running_count), 0)::bigint AS total_running,
  COALESCE(SUM(completed_count), 0)::bigint AS total_completed,
  COALESCE(SUM(cancelled_count), 0)::bigint AS total_cancelled,
  COALESCE(SUM(failed_count), 0)::bigint AS total_failed
FROM v2_cagg_status_metrics
WHERE
    tenant_id = @tenantId::uuid
    AND bucket >= time_bucket('5 minutes', @createdAfter::timestamptz)
    AND (
        sqlc.narg('workflowIds')::uuid[] IS NULL OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
    );

-- name: ReadTaskByExternalID :one
WITH lookup_task AS (
    SELECT
        tenant_id,
        task_id,
        inserted_at
    FROM
        v2_task_lookup_table
    WHERE
        external_id = @externalId::uuid
)
SELECT
    t.*
FROM
    v2_tasks_olap t
JOIN
    lookup_task lt ON lt.tenant_id = t.tenant_id AND lt.task_id = t.id AND lt.inserted_at = t.inserted_at;

-- name: ListTaskEvents :many
WITH aggregated_events AS (
  SELECT
    tenant_id,
    task_id,
    task_inserted_at,
    retry_count,
    event_type,
    MIN(event_timestamp) AS time_first_seen,
    MAX(event_timestamp) AS time_last_seen,
    COUNT(*) AS count,
    MIN(id) AS first_id
  FROM v2_task_events_olap
  WHERE
    tenant_id = @tenantId::uuid
    AND task_id = @taskId::bigint
    AND task_inserted_at = @taskInsertedAt::timestamptz
  GROUP BY tenant_id, task_id, task_inserted_at, retry_count, event_type
)
SELECT
  a.tenant_id,
  a.task_id,
  a.task_inserted_at,
  a.retry_count,
  a.event_type,
  a.time_first_seen,
  a.time_last_seen,
  a.count,
  t.id,
  t.event_timestamp,
  t.readable_status,
  t.error_message,
  t.output,
  t.worker_id,
  t.additional__event_data,
  t.additional__event_message
FROM aggregated_events a
JOIN v2_task_events_olap t
  ON t.tenant_id = a.tenant_id
  AND t.task_id = a.task_id
  AND t.task_inserted_at = a.task_inserted_at
  AND t.id = a.first_id
ORDER BY a.time_first_seen DESC, t.event_timestamp DESC;

-- name: ListTasks :many
SELECT
    *
FROM
    v2_tasks_olap
WHERE
    tenant_id = @tenantId::uuid
    AND inserted_at >= @insertedAfter::timestamptz
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR readable_status = ANY(cast(sqlc.narg('statuses')::text[] as v2_readable_status_olap[]))
    )
    AND (
        sqlc.narg('workflowIds')::uuid[] IS NULL OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
    )
    AND (
        sqlc.narg('workerId')::uuid IS NULL OR latest_worker_id = sqlc.narg('workerId')::uuid
    )
ORDER BY
    inserted_at DESC
LIMIT @taskLimit::integer;

-- name: PopulateSingleTaskRunData :one
WITH latest_retry_count AS (
    SELECT
        MAX(retry_count) AS retry_count
    FROM
        v2_task_events_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND task_id = @taskId::bigint
        AND task_inserted_at = @taskInsertedAt::timestamptz
), relevant_events AS (
    SELECT
        *
    FROM
        v2_task_events_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND task_id = @taskId::bigint
        AND task_inserted_at = @taskInsertedAt::timestamptz
        AND retry_count = (SELECT retry_count FROM latest_retry_count)
    ORDER BY
        event_timestamp DESC
), finished_at AS (
    SELECT
        MAX(event_timestamp) AS finished_at
    FROM
        relevant_events
    WHERE
        readable_status = ANY(ARRAY['COMPLETED', 'FAILED', 'CANCELLED']::v2_readable_status_olap[])
), started_at AS (
    SELECT
        MAX(event_timestamp) AS started_at
    FROM
        relevant_events
    WHERE
        event_type = 'STARTED'
), task_output AS (
    SELECT
        output
    FROM
        relevant_events
    WHERE
        event_type = 'FINISHED'
), status AS (
    SELECT
        readable_status
    FROM
        relevant_events
    ORDER BY
        readable_status DESC
    LIMIT 1
)
SELECT
    t.*,
    st.readable_status::v2_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at,
    o.output::jsonb as output
FROM
    v2_tasks_olap t
LEFT JOIN
    finished_at f ON true
LEFT JOIN
    started_at s ON true
LEFT JOIN
    task_output o ON true
LEFT JOIN
    status st ON true
WHERE
    (t.tenant_id, t.id, t.inserted_at) = (@tenantId::uuid, @taskId::bigint, @taskInsertedAt::timestamptz);

-- name: PopulateTaskRunData :many
WITH input AS (
    SELECT
        UNNEST(@tenantIds::uuid[]) AS tenant_id,
        UNNEST(@taskIds::bigint[]) AS id,
        UNNEST(@taskInsertedAts::timestamptz[]) AS inserted_at,
        UNNEST(@retryCounts::int[]) AS retry_count,
        unnest(cast(@statuses::text[] as v2_readable_status_olap[])) AS status
), tasks AS (
    SELECT
        DISTINCT ON(t.tenant_id, t.id, t.inserted_at)
        t.tenant_id,
        t.id,
        t.inserted_at,
        t.queue,
        t.action_id,
        t.step_id,
        t.workflow_id,
        t.schedule_timeout,
        t.step_timeout,
        t.priority,
        t.sticky,
        t.desired_worker_id,
        t.external_id,
        t.display_name,
        t.input,
        t.additional_metadata,
        i.retry_count,
        i.status
    FROM
        v2_tasks_olap t
    JOIN
        input i ON i.tenant_id = t.tenant_id AND i.id = t.id AND i.inserted_at = t.inserted_at
), finished_ats AS (
    SELECT
        e.task_id::bigint,
        MAX(e.event_timestamp) AS finished_at
    FROM
        v2_task_events_olap e
    JOIN
        tasks t ON t.id = e.task_id AND t.tenant_id = e.tenant_id AND t.inserted_at = e.task_inserted_at AND t.retry_count = e.retry_count
    WHERE
        e.readable_status = ANY(ARRAY['COMPLETED', 'FAILED', 'CANCELLED']::v2_readable_status_olap[])
    GROUP BY e.task_id
), started_ats AS (
    SELECT
        e.task_id::bigint,
        MAX(e.event_timestamp) AS started_at
    FROM
        v2_task_events_olap e
    JOIN
        tasks t ON t.id = e.task_id AND t.tenant_id = e.tenant_id AND t.inserted_at = e.task_inserted_at AND t.retry_count = e.retry_count
    WHERE
        e.event_type = 'STARTED'
    GROUP BY e.task_id
)
SELECT
    t.tenant_id,
    t.id,
    t.inserted_at,
    t.external_id,
    t.queue,
    t.action_id,
    t.step_id,
    t.workflow_id,
    t.schedule_timeout,
    t.step_timeout,
    t.priority,
    t.sticky,
    t.display_name,
    t.retry_count,
    t.additional_metadata,
    t.status::v2_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at
FROM
    tasks t
LEFT JOIN
    finished_ats f ON f.task_id = t.id
LEFT JOIN
    started_ats s ON s.task_id = t.id
ORDER BY t.inserted_at DESC, t.id DESC
LIMIT @taskLimit::int;

-- name: GetTaskPointMetrics :many
SELECT
    time_bucket(COALESCE(sqlc.narg('interval')::interval, '1 minute'), bucket)::timestamptz as bucket_2,
    SUM(completed_count)::int as completed_count,
    SUM(failed_count)::int as failed_count
FROM
    v2_cagg_task_events_minute
WHERE
    tenant_id = @tenantId::uuid AND
    -- timestamptz makes this fast, apparently:
    -- https://www.timescale.com/forum/t/very-slow-query-planning-time-in-postgresql/255/8
    bucket >= time_bucket('1 minute', @createdAfter::timestamptz) AND
    bucket <= time_bucket('1 minute', @createdBefore::timestamptz)
GROUP BY bucket_2
ORDER BY bucket_2;

-- name: UpdateTaskStatuses :one
WITH locked_events AS (
    SELECT
        *
    FROM
        list_task_events(
            @partitionNumber::int,
            @tenantId::uuid,
            @eventLimit::int
        )
), max_retry_counts AS (
    SELECT
        tenant_id,
        task_id,
        task_inserted_at,
        MAX(retry_count) AS max_retry_count
    FROM
        locked_events
    GROUP BY
        tenant_id, task_id, task_inserted_at
), updatable_events AS (
    SELECT
        e.tenant_id,
        e.task_id,
        e.task_inserted_at,
        e.retry_count,
        e.worker_id,
        MAX(e.readable_status) AS max_readable_status
    FROM
        locked_events e
    JOIN
        max_retry_counts mrc ON
            e.tenant_id = mrc.tenant_id
            AND e.task_id = mrc.task_id
            AND e.task_inserted_at = mrc.task_inserted_at
            AND e.retry_count = mrc.max_retry_count
    GROUP BY
        e.tenant_id, e.task_id, e.task_inserted_at, e.retry_count, e.worker_id
), locked_tasks AS (
    SELECT
        t.tenant_id,
        t.id,
        t.inserted_at,
        e.retry_count,
        e.max_readable_status
    FROM
        v2_tasks_olap t
    JOIN
        updatable_events e ON
            (t.tenant_id, t.id, t.inserted_at) = (e.tenant_id, e.task_id, e.task_inserted_at)
    ORDER BY
        t.id
    FOR UPDATE
), updated_tasks AS (
    UPDATE
        v2_tasks_olap t
    SET
        readable_status = e.max_readable_status,
        latest_retry_count = e.retry_count,
        latest_worker_id = CASE WHEN e.worker_id IS NOT NULL THEN e.worker_id ELSE t.latest_worker_id END
    FROM
        updatable_events e
    WHERE
        (t.tenant_id, t.id, t.inserted_at) = (e.tenant_id, e.task_id, e.task_inserted_at)
        AND e.retry_count >= t.latest_retry_count
        AND e.max_readable_status > t.readable_status
    RETURNING
        t.tenant_id, t.id, t.inserted_at
), events_to_requeue AS (
    -- Get events which don't have a corresponding locked_task
    SELECT
        e.tenant_id,
        e.requeue_retries,
        e.task_id,
        e.task_inserted_at,
        e.event_type,
        e.readable_status,
        e.retry_count
    FROM
        locked_events e
    LEFT JOIN
        locked_tasks t ON (e.tenant_id, e.task_id, e.task_inserted_at) = (t.tenant_id, t.id, t.inserted_at)
    WHERE
        t.id IS NULL
), deleted_events AS (
    DELETE FROM
        v2_task_events_olap_tmp
    WHERE
        (tenant_id, requeue_after, task_id, id) IN (SELECT tenant_id, requeue_after, task_id, id FROM locked_events)
), requeued_events AS (
    INSERT INTO
        v2_task_events_olap_tmp (
            tenant_id,
            requeue_after,
            requeue_retries,
            task_id,
            task_inserted_at,
            event_type,
            readable_status,
            retry_count
        )
    SELECT
        tenant_id,
        -- Exponential backoff, we limit to 10 retries which is 2048 seconds/34 minutes
        CURRENT_TIMESTAMP + (2 ^ requeue_retries) * INTERVAL '2 seconds',
        requeue_retries + 1,
        task_id,
        task_inserted_at,
        event_type,
        readable_status,
        retry_count + 1
    FROM
        events_to_requeue
    WHERE
        retry_count < 10
    RETURNING
        *
)
SELECT
    COUNT(*)
FROM
    locked_events;