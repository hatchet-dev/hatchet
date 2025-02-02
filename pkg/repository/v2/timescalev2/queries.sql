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
    $12
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

-- name: ListTasks :many
WITH task_statuses AS (
    SELECT 
        s.tenant_id::uuid as tenant_id,
        s.task_id::bigint as id,
        s.task_inserted_at::timestamptz as inserted_at,
        s.status::v2_readable_status_olap as status,
        s.max_retry_count::int as max_retry_count,
        t.external_id as external_id,
        t.queue as queue,
        t.action_id as action_id,
        t.step_id as step_id,
        t.workflow_id as workflow_id,
        t.schedule_timeout as schedule_timeout,
        t.step_timeout as step_timeout,
        t.priority as priority,
        t.sticky as sticky,
        t.display_name as display_name
    FROM 
        v2_cagg_task_status s
    JOIN
        v2_tasks_olap t ON t.tenant_id = s.tenant_id AND t.id = s.task_id AND t.inserted_at = s.task_inserted_at
    WHERE
        s.tenant_id = @tenantId::uuid
        AND bucket >= @createdAfter::timestamptz
    ORDER BY bucket DESC, s.task_inserted_at DESC, s.task_id DESC
    LIMIT 50
), finished_ats AS (
    SELECT
        e.task_id::bigint,
        MAX(e.event_timestamp) AS finished_at
    FROM
        v2_task_events_olap e
    JOIN    
        task_statuses ts ON ts.id = e.task_id AND ts.tenant_id = e.tenant_id AND ts.inserted_at = e.task_inserted_at AND ts.max_retry_count = e.retry_count
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
        task_statuses ts ON ts.id = e.task_id AND ts.tenant_id = e.tenant_id AND ts.inserted_at = e.task_inserted_at AND ts.max_retry_count = e.retry_count
    WHERE
        e.event_type = 'STARTED'
    GROUP BY e.task_id
)
SELECT
    ts.tenant_id,
    ts.id,
    ts.inserted_at,
    ts.external_id,
    ts.queue,
    ts.action_id,
    ts.step_id,
    ts.workflow_id,
    ts.schedule_timeout,
    ts.step_timeout,
    ts.priority,
    ts.sticky,
    ts.display_name,
    ts.status::v2_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at
FROM
    task_statuses ts
LEFT JOIN
    finished_ats f ON f.task_id = ts.id
LEFT JOIN
    started_ats s ON s.task_id = ts.id
ORDER BY ts.inserted_at DESC, ts.id DESC;

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