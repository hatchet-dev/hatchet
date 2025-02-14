-- name: CreateOLAPTaskEventTmpPartitions :exec
SELECT create_v2_hash_partitions(
    'v2_task_events_olap_tmp'::text,
    @partitions::int
);

-- name: CreateOLAPTaskStatusUpdateTmpPartitions :exec
SELECT create_v2_hash_partitions(
    'v2_task_status_updates_tmp'::text,
    @partitions::int
);

-- name: CreateOLAPTaskPartition :exec
SELECT create_v2_olap_partition_with_date_and_status(
    'v2_tasks_olap'::text,
    @date::date
);

-- name: ListOLAPTaskPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v2_partitions_before_date(
        'v2_tasks_olap'::text,
        @date::date
    ) AS p;

-- name: CreateOLAPDAGPartition :exec
SELECT create_v2_olap_partition_with_date_and_status(
    'v2_dags_olap'::text,
    @date::date
);

-- name: ListOLAPDAGPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v2_partitions_before_date(
        'v2_dags_olap'::text,
        @date::date
    ) AS p;

-- name: CreateOLAPRunsPartition :exec
SELECT create_v2_olap_partition_with_date_and_status(
    'v2_runs_olap'::text,
    @date::date
);

-- name: ListOLAPRunsPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v2_partitions_before_date(
        'v2_runs_olap'::text,
        @date::date
    ) AS p
;

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
    additional_metadata,
    dag_id,
    dag_inserted_at
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
    $16,
    $17,
    $18
);

-- name: CreateDAGsOLAP :copyfrom
INSERT INTO v2_dags_olap (
    tenant_id,
    id,
    inserted_at,
    external_id,
    display_name,
    workflow_id,
    workflow_version_id,
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
    $9
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
        v2_lookup_table
    WHERE
        external_id = @externalId::uuid
)
SELECT
    t.*,
    e.output,
    e.error_message
FROM
    v2_tasks_olap t
JOIN
    lookup_task lt ON lt.tenant_id = t.tenant_id AND lt.task_id = t.id AND lt.inserted_at = t.inserted_at
JOIN
    v2_task_events_olap e ON (e.tenant_id, e.task_id, e.readable_status, e.retry_count) = (t.tenant_id, t.id, t.readable_status, t.latest_retry_count)
;

-- name: ListTasksByExternalIds :many
SELECT
    tenant_id,
    task_id,
    inserted_at
FROM
    v2_lookup_table
WHERE
    external_id = ANY(@externalIds::uuid[])
    AND tenant_id = @tenantId::uuid;

-- name: ListTasksByDAGIds :many
SELECT
    dt.*
FROM
    v2_lookup_table lt
JOIN
    v2_dag_to_task_olap dt ON lt.dag_id = dt.dag_id
WHERE
    lt.external_id = ANY(@dagIds::uuid[])
    AND tenant_id = @tenantId::uuid
;

-- name: ReadDAGByExternalID :one
WITH lookup_task AS (
    SELECT
        tenant_id,
        dag_id,
        inserted_at
    FROM
        v2_lookup_table
    WHERE
        external_id = @externalId::uuid
)
SELECT
    d.*
FROM
    v2_dags_olap d
JOIN
    lookup_task lt ON lt.tenant_id = d.tenant_id AND lt.dag_id = d.id AND lt.inserted_at = d.inserted_at;

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
    id,
    inserted_at
FROM
    v2_tasks_olap
WHERE
    tenant_id = @tenantId::uuid
    AND inserted_at >= @since::timestamptz
    AND (
        sqlc.narg('until')::timestamptz IS NULL
        OR inserted_at <= sqlc.narg('until')::timestamptz
    )
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR readable_status = ANY(cast(sqlc.narg('statuses')::text[] as v2_readable_status_olap[]))
    )
    AND (
        sqlc.narg('workflowIds')::uuid[] IS NULL OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
    )
    AND (
        sqlc.narg('workerId')::uuid IS NULL OR latest_worker_id = sqlc.narg('workerId')::uuid
    )
    AND (
        sqlc.narg('keys')::text[] IS NULL
        OR sqlc.narg('values')::text[] IS NULL
        OR EXISTS (
            SELECT 1 FROM jsonb_each_text(additional_metadata) kv
            JOIN LATERAL (
                SELECT unnest(sqlc.narg('keys')::text[]) AS k,
                    unnest(sqlc.narg('values')::text[]) AS v
            ) AS u ON kv.key = u.k AND kv.value = u.v
        )
    )
ORDER BY
    inserted_at DESC
LIMIT @taskLimit::integer
OFFSET @taskOffset::integer;

-- name: CountTasks :one
WITH filtered AS (
    SELECT
        *
    FROM
        v2_tasks_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND inserted_at >= @since::timestamptz
        AND (
            sqlc.narg('until')::timestamptz IS NULL
            OR inserted_at <= sqlc.narg('until')::timestamptz
        )
        AND (
            sqlc.narg('statuses')::text[] IS NULL OR readable_status = ANY(cast(sqlc.narg('statuses')::text[] as v2_readable_status_olap[]))
        )
        AND (
            sqlc.narg('workflowIds')::uuid[] IS NULL OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
        )
        AND (
            sqlc.narg('workerId')::uuid IS NULL OR latest_worker_id = sqlc.narg('workerId')::uuid
        )
        AND (
            sqlc.narg('keys')::text[] IS NULL
            OR sqlc.narg('values')::text[] IS NULL
            OR EXISTS (
                SELECT 1 FROM jsonb_each_text(additional_metadata) kv
                JOIN LATERAL (
                    SELECT unnest(sqlc.narg('keys')::text[]) AS k,
                        unnest(sqlc.narg('values')::text[]) AS v
                ) AS u ON kv.key = u.k AND kv.value = u.v
            )
        )
    ORDER BY
        inserted_at DESC
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
;

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
), error_message AS (
    SELECT
        error_message
    FROM
        relevant_events
    WHERE
        readable_status = 'FAILED'
    ORDER BY
        event_timestamp DESC
    LIMIT 1
)
SELECT
    t.*,
    st.readable_status::v2_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at,
    o.output::jsonb as output,
    e.error_message as error_message
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
LEFT JOIN
    error_message e ON true
WHERE
    (t.tenant_id, t.id, t.inserted_at) = (@tenantId::uuid, @taskId::bigint, @taskInsertedAt::timestamptz);

-- name: PopulateTaskRunData :many
WITH input AS (
    SELECT
        UNNEST(@taskIds::bigint[]) AS id,
        UNNEST(@taskInsertedAts::timestamptz[]) AS inserted_at
), tasks AS (
    SELECT
        DISTINCT ON(t.tenant_id, t.id, t.inserted_at)
        t.tenant_id,
        t.id,
        d.external_id AS dag_external_id,
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
        t.readable_status
    FROM
        v2_tasks_olap t
    JOIN
        input i ON i.id = t.id AND i.inserted_at = t.inserted_at
    LEFT JOIN
        v2_dag_to_task_olap dtt ON dtt.task_id = t.id
    LEFT JOIN
        v2_dags_olap d ON d.id = dtt.dag_id AND d.tenant_id = t.tenant_id
    WHERE
        t.tenant_id = @tenantId::uuid
), relevant_events AS (
    SELECT
        e.*
    FROM
        v2_task_events_olap e
    JOIN
        tasks t ON t.id = e.task_id AND t.tenant_id = e.tenant_id AND t.inserted_at = e.task_inserted_at
), max_retry_counts AS (
    SELECT
        e.tenant_id,
        e.task_id,
        e.task_inserted_at,
        MAX(e.retry_count) AS max_retry_count
    FROM
        relevant_events e
    GROUP BY
        e.tenant_id, e.task_id, e.task_inserted_at
), finished_ats AS (
    SELECT
        e.task_id::bigint,
        MAX(e.event_timestamp) AS finished_at
    FROM
        relevant_events e
    JOIN
        max_retry_counts mrc ON
            e.tenant_id = mrc.tenant_id
            AND e.task_id = mrc.task_id
            AND e.task_inserted_at = mrc.task_inserted_at
            AND e.retry_count = mrc.max_retry_count
    WHERE
        e.readable_status = ANY(ARRAY['COMPLETED', 'FAILED', 'CANCELLED']::v2_readable_status_olap[])
    GROUP BY e.task_id
), started_ats AS (
    SELECT
        e.task_id::bigint,
        MAX(e.event_timestamp) AS started_at
    FROM
        relevant_events e
    JOIN
        max_retry_counts mrc ON
            e.tenant_id = mrc.tenant_id
            AND e.task_id = mrc.task_id
            AND e.task_inserted_at = mrc.task_inserted_at
            AND e.retry_count = mrc.max_retry_count
    WHERE
        e.event_type = 'STARTED'
    GROUP BY e.task_id
), error_message AS (
    SELECT
        DISTINCT ON (e.task_id) e.task_id::bigint,
        e.error_message
    FROM
        relevant_events e
    JOIN
        max_retry_counts mrc ON
            e.tenant_id = mrc.tenant_id
            AND e.task_id = mrc.task_id
            AND e.task_inserted_at = mrc.task_inserted_at
            AND e.retry_count = mrc.max_retry_count
    WHERE
        e.readable_status = 'FAILED'
    ORDER BY
        e.task_id, e.retry_count DESC
)
SELECT
    t.tenant_id,
    t.id,
    t.dag_external_id,
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
    t.additional_metadata,
    t.readable_status::v2_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at,
    e.error_message as error_message
FROM
    tasks t
LEFT JOIN
    finished_ats f ON f.task_id = t.id
LEFT JOIN
    started_ats s ON s.task_id = t.id
LEFT JOIN
    error_message e ON e.task_id = t.id
ORDER BY t.inserted_at DESC, t.id DESC;

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
        list_task_events_tmp(
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
        AND
            (
                -- if the retry count is greater than the latest retry count, update the status
                (
                    e.retry_count > t.latest_retry_count
                    AND e.max_readable_status != t.readable_status
                ) OR
                -- if the retry count is equal to the latest retry count, update the status if the status is greater
                (
                    e.retry_count = t.latest_retry_count
                    AND e.max_readable_status > t.readable_status
                )
            )
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
        retry_count
    FROM
        events_to_requeue
    WHERE
        requeue_retries < 10
    RETURNING
        *
)
SELECT
    COUNT(*)
FROM
    locked_events;

-- name: UpdateDAGStatuses :one
WITH locked_events AS (
    SELECT
        *
    FROM
        list_task_status_updates_tmp(
            @partitionNumber::int,
            @tenantId::uuid,
            @eventLimit::int
        )
), distinct_dags AS (
    SELECT
        DISTINCT ON (e.tenant_id, e.dag_id, e.dag_inserted_at)
        e.tenant_id,
        e.dag_id,
        e.dag_inserted_at
    FROM
        locked_events e
), locked_dags AS (
    SELECT
        d.id,
        d.inserted_at,
        d.readable_status,
        d.tenant_id
    FROM
        v2_dags_olap d
    JOIN
        distinct_dags dd ON
            (d.tenant_id, d.id, d.inserted_at) = (dd.tenant_id, dd.dag_id, dd.dag_inserted_at)
    ORDER BY
        d.id, d.inserted_at
    FOR UPDATE
), dag_task_counts AS (
    SELECT
        d.id,
        d.inserted_at,
        COUNT(t.id) AS task_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'COMPLETED') AS completed_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'FAILED') AS failed_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'CANCELLED') AS cancelled_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'QUEUED') AS queued_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'RUNNING') AS running_count
    FROM
        locked_dags d
    LEFT JOIN
        v2_dag_to_task_olap dt ON
            (d.id, d.inserted_at) = (dt.dag_id, dt.dag_inserted_at)
    LEFT JOIN
        v2_tasks_olap t ON
            (dt.task_id, dt.task_inserted_at) = (t.id, t.inserted_at)
    GROUP BY
        d.id, d.inserted_at
), updated_dags AS (
    UPDATE
        v2_dags_olap d
    SET
        readable_status = CASE
            -- If we only have queued events, we should keep the status as is
            WHEN dtc.queued_count = dtc.task_count THEN d.readable_status
            -- If we have any running or queued tasks, we should set the status to running
            WHEN dtc.running_count > 0 OR dtc.queued_count > 0 THEN 'RUNNING'
            WHEN dtc.failed_count > 0 THEN 'FAILED'
            WHEN dtc.cancelled_count > 0 THEN 'CANCELLED'
            WHEN dtc.completed_count = dtc.task_count THEN 'COMPLETED'
            ELSE 'RUNNING'
        END
    FROM
        dag_task_counts dtc
    WHERE
        (d.id, d.inserted_at) = (dtc.id, dtc.inserted_at)
), events_to_requeue AS (
    -- Get events which don't have a corresponding locked_task
    SELECT
        e.tenant_id,
        e.requeue_retries,
        e.dag_id,
        e.dag_inserted_at
    FROM
        locked_events e
    LEFT JOIN
        locked_dags d ON (e.tenant_id, e.dag_id, e.dag_inserted_at) = (d.tenant_id, d.id, d.inserted_at)
    WHERE
        d.id IS NULL
), deleted_events AS (
    DELETE FROM
        v2_task_status_updates_tmp
    WHERE
        (tenant_id, requeue_after, dag_id, id) IN (SELECT tenant_id, requeue_after, dag_id, id FROM locked_events)
), requeued_events AS (
    INSERT INTO
        v2_task_status_updates_tmp (
            tenant_id,
            requeue_after,
            requeue_retries,
            dag_id,
            dag_inserted_at
        )
    SELECT
        tenant_id,
        -- Exponential backoff, we limit to 10 retries which is 2048 seconds/34 minutes
        CURRENT_TIMESTAMP + (2 ^ requeue_retries) * INTERVAL '2 seconds',
        requeue_retries + 1,
        dag_id,
        dag_inserted_at
    FROM
        events_to_requeue
    WHERE
        requeue_retries < 10
    RETURNING
        *
)
SELECT
    COUNT(*)
FROM
    locked_events;

-- name: FetchWorkflowRunIds :many
SELECT id, inserted_at, kind, external_id
FROM v2_runs_olap
WHERE
    tenant_id = @tenantId::uuid
    AND (
        sqlc.narg('workflowIds')::uuid[] IS NULL
        OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
    )
    AND (
        sqlc.narg('statuses')::text[] IS NULL
        OR readable_status = ANY(cast(sqlc.narg('statuses')::text[] as v2_readable_status_olap[]))
    )
    AND inserted_at >= @since::timestamptz
    AND (
        sqlc.narg('until')::timestamptz IS NULL
        OR inserted_at <= sqlc.narg('until')::timestamptz
    )
    AND (
        sqlc.narg('keys')::text[] IS NULL
        OR sqlc.narg('values')::text[] IS NULL
        OR EXISTS (
            SELECT 1 FROM jsonb_each_text(additional_metadata) kv
            JOIN LATERAL (
                SELECT unnest(sqlc.narg('keys')::text[]) AS k,
                    unnest(sqlc.narg('values')::text[]) AS v
            ) AS u ON kv.key = u.k AND kv.value = u.v
        )
    )
ORDER BY inserted_at DESC, id DESC
LIMIT @listWorkflowRunsLimit::integer
OFFSET @listWorkflowRunsOffset::integer
;

-- name: CountWorkflowRuns :one
WITH filtered AS (
    SELECT *
    FROM v2_runs_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND (
            sqlc.narg('workflowIds')::uuid[] IS NULL
            OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
        )
        AND (
            sqlc.narg('statuses')::text[] IS NULL
            OR readable_status = ANY(cast(sqlc.narg('statuses')::text[] as v2_readable_status_olap[]))
        )
        AND inserted_at >= @since::timestamptz
        AND (
            sqlc.narg('until')::timestamptz IS NULL
            OR inserted_at <= sqlc.narg('until')::timestamptz
        )
        AND (
            sqlc.narg('keys')::text[] IS NULL
            OR sqlc.narg('values')::text[] IS NULL
            OR EXISTS (
                SELECT 1 FROM jsonb_each_text(additional_metadata) kv
                JOIN LATERAL (
                    SELECT unnest(sqlc.narg('keys')::text[]) AS k,
                        unnest(sqlc.narg('values')::text[]) AS v
                ) AS u ON kv.key = u.k AND kv.value = u.v
            )
        )
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
;

-- name: PopulateDAGMetadata :many
WITH input AS (
    SELECT
        UNNEST(@ids::bigint[]) AS id,
        UNNEST(@insertedAts::timestamptz[]) AS inserted_at
), runs AS (
    SELECT
        d.id AS dag_id,
        r.id AS run_id,
        r.tenant_id,
        r.inserted_at,
        r.external_id,
        r.readable_status,
        r.kind,
        r.workflow_id,
        d.display_name,
        d.input,
        d.additional_metadata
    FROM v2_runs_olap r
    JOIN v2_dags_olap d ON (r.tenant_id, r.external_id, r.inserted_at) = (d.tenant_id, d.external_id, d.inserted_at)
    WHERE
        (r.inserted_at, r.id) IN (SELECT inserted_at, id FROM input)
        AND r.tenant_id = @tenantId::uuid
        AND r.kind = 'DAG'
), relevant_events AS (
    SELECT
        r.run_id,
        e.*
    FROM runs r
    JOIN v2_dag_to_task_olap dt ON r.dag_id = dt.dag_id  -- Do I need to join by `inserted_at` here too?
    JOIN v2_task_events_olap e ON e.task_id = dt.task_id -- Do I need to join by `inserted_at` here too?
), metadata AS (
    SELECT
        e.run_id,
        MIN(e.inserted_at)::timestamptz AS created_at,
        MIN(e.inserted_at) FILTER (WHERE e.readable_status = 'RUNNING')::timestamptz AS started_at,
        MAX(e.inserted_at) FILTER (WHERE e.readable_status IN ('COMPLETED', 'CANCELLED', 'FAILED'))::timestamptz AS finished_at
    FROM
        relevant_events e
    GROUP BY e.run_id
), error_message AS (
    SELECT
        DISTINCT ON (e.run_id) e.run_id::bigint,
        e.error_message
    FROM
        relevant_events e
    WHERE
        e.readable_status = 'FAILED'
    ORDER BY
        e.run_id, e.retry_count DESC
)
SELECT
    r.*,
    m.created_at,
    m.started_at,
    m.finished_at,
    e.error_message
FROM runs r
LEFT JOIN metadata m ON r.run_id = m.run_id
LEFT JOIN error_message e ON r.run_id = e.run_id
ORDER BY r.inserted_at DESC, r.run_id DESC;