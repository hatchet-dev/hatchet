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
    input
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
    $15
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
    AND bucket_2 >= @createdAfter::timestamptz;

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
ORDER BY a.time_first_seen;

-- name: LastSucceededStatusAggregate :one
SELECT 
    js.last_successful_finish::timestamptz as last_succeeded
FROM
    timescaledb_information.continuous_aggregates ca
JOIN
    timescaledb_information.jobs j ON j.hypertable_name = ca.materialization_hypertable_name
JOIN
    timescaledb_information.job_stats js ON j.job_id = js.job_id
WHERE
    ca.view_name = 'v2_cagg_task_status'
LIMIT 1;

-- name: ListTasksRealTime :many
WITH relevant_events AS (
    SELECT
        *
    FROM   
        v2_task_events_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND inserted_at >= @insertedAfter::timestamptz
        AND readable_status = ANY(cast(@statuses::text[] as v2_readable_status_olap[]))
        AND (
            sqlc.narg('workflowIds')::uuid[] IS NULL OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
        )
        AND (
            sqlc.narg('eventType')::v2_event_type_olap IS NULL OR event_type = sqlc.narg('eventType')::v2_event_type_olap
        )
    ORDER BY
        task_inserted_at DESC
    -- NOTE: we can't always limit this CTE. We can limit when we're just querying for created tasks,
    -- but we can't limit when we're querying for i.e. failed tasks, because we need to get the latest
    -- event for each task.
    LIMIT sqlc.narg('eventLimit')::integer
), unique_tasks AS (
    SELECT
        tenant_id,
        task_id,
        task_inserted_at
    FROM
        relevant_events
    GROUP BY tenant_id, task_id, task_inserted_at
), all_task_events AS (
    SELECT
        e.id,
        e.tenant_id,
        e.task_id,
        e.task_inserted_at,
        e.readable_status,
        e.retry_count
    FROM
        v2_task_events_olap e
    JOIN
        unique_tasks t ON t.tenant_id = e.tenant_id AND t.task_id = e.task_id AND t.task_inserted_at = e.task_inserted_at
)
SELECT
  tenant_id,
  task_id,
  task_inserted_at,
  (array_agg(readable_status ORDER BY retry_count DESC, readable_status DESC))[1]::v2_readable_status_olap AS status,
  max(retry_count)::integer AS max_retry_count
FROM all_task_events
GROUP BY tenant_id, task_id, task_inserted_at
ORDER BY task_inserted_at DESC, task_id DESC
LIMIT @taskLimit::integer;

-- name: ListTasksFromAggregate :many
SELECT 
    s.tenant_id::uuid as tenant_id,
    s.task_id::bigint as task_id,
    s.task_inserted_at::timestamptz as inserted_at,
    s.status::v2_readable_status_olap as status,
    s.max_retry_count::int as max_retry_count
FROM 
    v2_cagg_task_status s
JOIN
    v2_tasks_olap t ON t.tenant_id = s.tenant_id AND t.id = s.task_id AND t.inserted_at = s.task_inserted_at
WHERE
    s.tenant_id = @tenantId::uuid
    AND bucket >= @createdAfter::timestamptz
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR status = ANY(cast(@statuses::text[] as v2_readable_status_olap[]))
    )
    AND (
        sqlc.narg('workflowIds')::uuid[] IS NULL OR s.workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
    )
ORDER BY bucket DESC, s.task_inserted_at DESC, s.task_id DESC
LIMIT @taskLimit::integer;

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