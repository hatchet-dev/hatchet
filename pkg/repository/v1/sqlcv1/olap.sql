-- name: CreateOLAPTaskEventTmpPartitions :exec
SELECT create_v1_hash_partitions(
    'v1_task_events_olap_tmp'::text,
    @partitions::int
);

-- name: CreateOLAPTaskStatusUpdateTmpPartitions :exec
SELECT create_v1_hash_partitions(
    'v1_task_status_updates_tmp'::text,
    @partitions::int
);

-- name: CreateOLAPTaskPartition :exec
SELECT create_v1_olap_partition_with_date_and_status(
    'v1_tasks_olap'::text,
    @date::date
);

-- name: ListOLAPTaskPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v1_partitions_before_date(
        'v1_tasks_olap'::text,
        @date::date
    ) AS p;

-- name: CreateOLAPDAGPartition :exec
SELECT create_v1_olap_partition_with_date_and_status(
    'v1_dags_olap'::text,
    @date::date
);

-- name: ListOLAPDAGPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v1_partitions_before_date(
        'v1_dags_olap'::text,
        @date::date
    ) AS p;

-- name: CreateOLAPRunsPartition :exec
SELECT create_v1_olap_partition_with_date_and_status(
    'v1_runs_olap'::text,
    @date::date
);

-- name: ListOLAPRunsPartitionsBeforeDate :many
SELECT
    p::text AS partition_name
FROM
    get_v1_partitions_before_date(
        'v1_runs_olap'::text,
        @date::date
    ) AS p
;

-- name: CreateTasksOLAP :copyfrom
INSERT INTO v1_tasks_olap (
    tenant_id,
    id,
    inserted_at,
    queue,
    action_id,
    step_id,
    workflow_id,
    workflow_version_id,
    workflow_run_id,
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
    dag_inserted_at,
    parent_task_external_id
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
    $18,
    $19,
    $20,
    $21
);

-- name: CreateDAGsOLAP :copyfrom
INSERT INTO v1_dags_olap (
    tenant_id,
    id,
    inserted_at,
    external_id,
    display_name,
    workflow_id,
    workflow_version_id,
    input,
    additional_metadata,
    parent_task_external_id
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
    $10
);

-- name: CreateTaskEventsOLAPTmp :copyfrom
INSERT INTO v1_task_events_olap_tmp (
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
INSERT INTO v1_task_events_olap (
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

-- name: ReadTaskByExternalID :one
WITH lookup_task AS (
    SELECT
        tenant_id,
        task_id,
        inserted_at
    FROM
        v1_lookup_table_olap
    WHERE
        external_id = @externalId::uuid
)
SELECT
    t.*,
    e.output,
    e.error_message
FROM
    v1_tasks_olap t
JOIN
    lookup_task lt ON lt.tenant_id = t.tenant_id AND lt.task_id = t.id AND lt.inserted_at = t.inserted_at
JOIN
    v1_task_events_olap e ON (e.tenant_id, e.task_id, e.readable_status, e.retry_count) = (t.tenant_id, t.id, t.readable_status, t.latest_retry_count)
;

-- name: ListTasksByExternalIds :many
SELECT
    tenant_id,
    task_id,
    inserted_at
FROM
    v1_lookup_table_olap
WHERE
    external_id = ANY(@externalIds::uuid[])
    AND tenant_id = @tenantId::uuid;

-- name: ListTasksByDAGIds :many
SELECT
    DISTINCT ON (t.external_id)
    dt.*,
    lt.external_id AS dag_external_id
FROM
    v1_lookup_table_olap lt
JOIN
    v1_dag_to_task_olap dt ON lt.dag_id = dt.dag_id
JOIN
    v1_tasks_olap t ON (t.tenant_id, t.id, t.inserted_at) = (lt.tenant_id, dt.task_id, dt.task_inserted_at)
WHERE
    lt.external_id = ANY(@dagIds::uuid[])
    AND lt.tenant_id = @tenantId::uuid
ORDER BY
    t.external_id, t.inserted_at DESC;

-- name: ReadDAGByExternalID :one
WITH lookup_task AS (
    SELECT
        tenant_id,
        dag_id,
        inserted_at
    FROM
        v1_lookup_table_olap
    WHERE
        external_id = @externalId::uuid
)
SELECT
    d.*
FROM
    v1_dags_olap d
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
  FROM v1_task_events_olap
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
JOIN v1_task_events_olap t
  ON t.tenant_id = a.tenant_id
  AND t.task_id = a.task_id
  AND t.task_inserted_at = a.task_inserted_at
  AND t.id = a.first_id
ORDER BY a.time_first_seen DESC, t.event_timestamp DESC;

-- name: ListTaskEventsForWorkflowRun :many
WITH tasks AS (
    SELECT dt.task_id
    FROM v1_lookup_table_olap lt
    JOIN v1_dag_to_task_olap dt ON lt.dag_id = dt.dag_id
    WHERE
        lt.external_id = @workflowRunId::uuid
        AND lt.tenant_id = @tenantId::uuid
), aggregated_events AS (
  SELECT
    tenant_id,
    task_id,
    task_inserted_at,
    retry_count,
    event_type,
    MIN(event_timestamp)::timestamptz AS time_first_seen,
    MAX(event_timestamp)::timestamptz AS time_last_seen,
    COUNT(*) AS count,
    MIN(id) AS first_id
  FROM v1_task_events_olap
  WHERE
    tenant_id = @tenantId::uuid
    AND task_id IN (SELECT task_id FROM tasks)
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
  t.additional__event_message,
  tsk.display_name,
  tsk.external_id AS task_external_id
FROM aggregated_events a
JOIN v1_task_events_olap t
  ON t.tenant_id = a.tenant_id
  AND t.task_id = a.task_id
  AND t.task_inserted_at = a.task_inserted_at
  AND t.id = a.first_id
JOIN v1_tasks_olap tsk
    ON (tsk.tenant_id, tsk.id, tsk.inserted_at) = (t.tenant_id, t.task_id, t.task_inserted_at)
ORDER BY a.time_first_seen DESC, t.event_timestamp DESC;

-- name: PopulateSingleTaskRunData :one
WITH latest_retry_count AS (
    SELECT
        MAX(retry_count) AS retry_count
    FROM
        v1_task_events_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND task_id = @taskId::bigint
        AND task_inserted_at = @taskInsertedAt::timestamptz
), relevant_events AS (
    SELECT
        *
    FROM
        v1_task_events_olap
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
        readable_status = ANY(ARRAY['COMPLETED', 'FAILED', 'CANCELLED']::v1_readable_status_olap[])
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
), spawned_children AS (
    SELECT COUNT(*) AS spawned_children
    FROM v1_runs_olap
    WHERE parent_task_external_id = (
        SELECT external_id
        FROM v1_tasks_olap
        WHERE id = @taskId::bigint
    )
)

SELECT
    t.*,
    st.readable_status::v1_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at,
    o.output::jsonb as output,
    e.error_message as error_message,
    sc.spawned_children
FROM
    v1_tasks_olap t
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
LEFT JOIN
    spawned_children sc ON true
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
        v1_tasks_olap t
    JOIN
        input i ON i.id = t.id AND i.inserted_at = t.inserted_at
    WHERE
        t.tenant_id = @tenantId::uuid
), relevant_events AS (
    SELECT
        e.*
    FROM
        v1_task_events_olap e
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
        e.readable_status = ANY(ARRAY['COMPLETED', 'FAILED', 'CANCELLED']::v1_readable_status_olap[])
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
), task_output AS (
    SELECT
        task_id,
        MAX(output::TEXT)::JSONB AS output
    FROM
        relevant_events
    WHERE
        readable_status = 'COMPLETED'
    GROUP BY
        task_id
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
    t.additional_metadata,
    t.input,
    t.readable_status::v1_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at,
    e.error_message as error_message,
    o.output::jsonb as output
FROM
    tasks t
LEFT JOIN
    finished_ats f ON f.task_id = t.id
LEFT JOIN
    started_ats s ON s.task_id = t.id
LEFT JOIN
    error_message e ON e.task_id = t.id
LEFT JOIN
    task_output o ON o.task_id = t.id
ORDER BY t.inserted_at DESC, t.id DESC;


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
        e.tenant_id, e.task_id, e.task_inserted_at, e.retry_count
), latest_worker_id AS (
    SELECT
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        MAX(worker_id::text) AS worker_id
    FROM
        locked_events
    WHERE
        worker_id IS NOT NULL
    GROUP BY
        tenant_id, task_id, task_inserted_at, retry_count
), locked_tasks AS (
    SELECT
        t.tenant_id,
        t.id,
        t.inserted_at,
        e.retry_count,
        e.max_readable_status
    FROM
        v1_tasks_olap t
    JOIN
        updatable_events e ON
            (t.tenant_id, t.id, t.inserted_at) = (e.tenant_id, e.task_id, e.task_inserted_at)
    ORDER BY
        t.id
    FOR UPDATE
), updated_tasks AS (
    UPDATE
        v1_tasks_olap t
    SET
        readable_status = e.max_readable_status,
        latest_retry_count = e.retry_count,
        latest_worker_id = CASE WHEN lw.worker_id::uuid IS NOT NULL THEN lw.worker_id::uuid ELSE t.latest_worker_id END
    FROM
        updatable_events e
    LEFT JOIN
        latest_worker_id lw ON
            (e.tenant_id, e.task_id, e.task_inserted_at, e.retry_count) = (lw.tenant_id, lw.task_id, lw.task_inserted_at, lw.retry_count)
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
        t.tenant_id, t.id, t.inserted_at, t.readable_status, t.external_id
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
        v1_task_events_olap_tmp
    WHERE
        (tenant_id, requeue_after, task_id, id) IN (SELECT tenant_id, requeue_after, task_id, id FROM locked_events)
), requeued_events AS (
    INSERT INTO
        v1_task_events_olap_tmp (
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
), event_count AS (
    SELECT
        COUNT(*) as count
    FROM
        locked_events
), rows_to_return AS (
    SELECT
        ARRAY_REMOVE(ARRAY_AGG(t.id), NULL)::bigint[] AS task_ids,
        ARRAY_REMOVE(ARRAY_AGG(t.inserted_at), NULL)::timestamptz[] AS task_inserted_ats,
        ARRAY_REMOVE(ARRAY_AGG(t.readable_status), NULL)::text[] AS readable_statuses,
        ARRAY_REMOVE(ARRAY_AGG(t.external_id), NULL)::uuid[] AS external_ids
    FROM
        updated_tasks t
)
SELECT
    (SELECT count FROM event_count) AS count,
    task_ids,
    task_inserted_ats,
    readable_statuses,
    external_ids
FROM
    rows_to_return;

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
        v1_dags_olap d
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
        v1_dag_to_task_olap dt ON
            (d.id, d.inserted_at) = (dt.dag_id, dt.dag_inserted_at)
    LEFT JOIN
        v1_tasks_olap t ON
            (dt.task_id, dt.task_inserted_at) = (t.id, t.inserted_at)
    GROUP BY
        d.id, d.inserted_at
), updated_dags AS (
    UPDATE
        v1_dags_olap d
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
    RETURNING
        d.id, d.inserted_at, d.readable_status, d.external_id
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
        v1_task_status_updates_tmp
    WHERE
        (tenant_id, requeue_after, dag_id, id) IN (SELECT tenant_id, requeue_after, dag_id, id FROM locked_events)
), requeued_events AS (
    INSERT INTO
        v1_task_status_updates_tmp (
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
), event_count AS (
    SELECT
        COUNT(*) as count
    FROM
        locked_events
), rows_to_return AS (
    SELECT
        ARRAY_REMOVE(ARRAY_AGG(d.id), NULL)::bigint[] AS dag_ids,
        ARRAY_REMOVE(ARRAY_AGG(d.inserted_at), NULL)::timestamptz[] AS dag_inserted_ats,
        ARRAY_REMOVE(ARRAY_AGG(d.readable_status), NULL)::text[] AS readable_statuses,
        ARRAY_REMOVE(ARRAY_AGG(d.external_id), NULL)::uuid[] AS external_ids
    FROM
        updated_dags d
)
SELECT
    (SELECT count FROM event_count) AS count,
    dag_ids,
    dag_inserted_ats,
    readable_statuses,
    external_ids
FROM
    rows_to_return;

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
        d.additional_metadata,
        d.workflow_version_id
    FROM v1_runs_olap r
    JOIN v1_dags_olap d ON (r.tenant_id, r.external_id, r.inserted_at) = (d.tenant_id, d.external_id, d.inserted_at)
    WHERE
        (r.inserted_at, r.id) IN (SELECT inserted_at, id FROM input)
        AND r.tenant_id = @tenantId::uuid
        AND r.kind = 'DAG'
), relevant_events AS (
    SELECT
        r.run_id,
        e.*
    FROM runs r
    JOIN v1_dag_to_task_olap dt ON r.dag_id = dt.dag_id  -- Do I need to join by `inserted_at` here too?
    JOIN v1_task_events_olap e ON e.task_id = dt.task_id -- Do I need to join by `inserted_at` here too?
), max_retry_count AS (
    SELECT run_id, MAX(retry_count) AS max_retry_count
    FROM relevant_events
    GROUP BY run_id
), metadata AS (
    SELECT
        e.run_id,
        MIN(e.inserted_at)::timestamptz AS created_at,
        MIN(e.inserted_at) FILTER (WHERE e.readable_status = 'RUNNING')::timestamptz AS started_at,
        MAX(e.inserted_at) FILTER (WHERE e.readable_status IN ('COMPLETED', 'CANCELLED', 'FAILED'))::timestamptz AS finished_at
    FROM
        relevant_events e
    JOIN max_retry_count mrc ON (e.run_id, e.retry_count) = (mrc.run_id, mrc.max_retry_count)
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
), task_output AS (
    SELECT
        run_id,
        output
    FROM
        relevant_events
    WHERE
        event_type = 'FINISHED'
)

SELECT
    r.*,
    m.created_at,
    m.started_at,
    m.finished_at,
    e.error_message,
    o.output,
    w.name AS workflow_name
FROM runs r
JOIN "Workflow" w ON r.workflow_id = w.id
LEFT JOIN metadata m ON r.run_id = m.run_id
LEFT JOIN error_message e ON r.run_id = e.run_id
LEFT JOIN task_output o ON r.run_id = o.run_id
ORDER BY r.inserted_at DESC, r.run_id DESC;


-- name: GetTaskPointMetrics :many
SELECT
    DATE_BIN(
        COALESCE(sqlc.narg('interval')::INTERVAL, '1 minute'),
        task_inserted_at,
        TIMESTAMPTZ '1970-01-01 00:00:00+00'
    ) :: TIMESTAMPTZ AS bucket_2,
    COUNT(*) FILTER (WHERE readable_status = 'COMPLETED') AS completed_count,
    COUNT(*) FILTER (WHERE readable_status = 'FAILED') AS failed_count
FROM
    v1_task_events_olap
WHERE
    tenant_id = @tenantId::UUID
    AND task_inserted_at BETWEEN @createdAfter::TIMESTAMPTZ AND @createdBefore::TIMESTAMPTZ
GROUP BY bucket_2
ORDER BY bucket_2;


-- name: GetTenantStatusMetrics :one
WITH task_external_ids AS (
    SELECT external_id
    FROM v1_runs_olap
    WHERE (
        sqlc.narg('parentTaskExternalId')::UUID IS NULL OR parent_task_external_id = sqlc.narg('parentTaskExternalId')::UUID
    )
)
SELECT
    tenant_id,
    COUNT(*) FILTER (WHERE readable_status = 'QUEUED') AS total_queued,
    COUNT(*) FILTER (WHERE readable_status = 'RUNNING') AS total_running,
    COUNT(*) FILTER (WHERE readable_status = 'COMPLETED') AS total_completed,
    COUNT(*) FILTER (WHERE readable_status = 'CANCELLED') AS total_cancelled,
    COUNT(*) FILTER (WHERE readable_status = 'FAILED') AS total_failed
FROM v1_statuses_olap
WHERE
    tenant_id = @tenantId::UUID
    AND inserted_at >= @createdAfter::TIMESTAMPTZ
    AND (
        sqlc.narg('workflowIds')::UUID[] IS NULL OR workflow_id = ANY(sqlc.narg('workflowIds')::UUID[])
    )
    AND external_id IN (
        SELECT external_id
        FROM task_external_ids
    )
GROUP BY tenant_id
;

-- name: ReadWorkflowRunByExternalId :one
WITH runs AS (
    SELECT
        lt.dag_id AS dag_id,
        lt.task_id AS task_id,
        r.id AS id,
        r.tenant_id,
        r.inserted_at,
        r.external_id,
        r.readable_status,
        r.kind,
        r.workflow_id,
        d.display_name AS display_name,
        d.input AS input,
        d.additional_metadata AS additional_metadata,
        d.workflow_version_id AS workflow_version_id
    FROM
        v1_lookup_table_olap lt
    JOIN
        v1_runs_olap r ON r.inserted_at = lt.inserted_at AND r.id = lt.dag_id
    JOIN
        v1_dags_olap d ON (lt.tenant_id, lt.dag_id, lt.inserted_at) = (d.tenant_id, d.id, d.inserted_at)
    WHERE
        lt.external_id = @workflowRunExternalId::uuid
        AND lt.dag_id IS NOT NULL

    UNION ALL

    SELECT
        lt.dag_id AS dag_id,
        lt.task_id AS task_id,
        r.id AS id,
        r.tenant_id,
        r.inserted_at,
        r.external_id,
        r.readable_status,
        r.kind,
        r.workflow_id,
        t.display_name AS display_name,
        t.input AS input,
        t.additional_metadata AS additional_metadata,
        t.workflow_version_id AS workflow_version_id
    FROM
        v1_lookup_table_olap lt
    JOIN
        v1_runs_olap r ON r.inserted_at = lt.inserted_at AND r.id = lt.task_id
    JOIN
        v1_tasks_olap t ON (lt.tenant_id, lt.task_id, lt.inserted_at) = (t.tenant_id, t.id, t.inserted_at)
    WHERE
        lt.external_id = @workflowRunExternalId::uuid
        AND lt.task_id IS NOT NULL
), relevant_events AS (
    SELECT
        e.*
    FROM runs r
    JOIN v1_dag_to_task_olap dt ON r.dag_id = dt.dag_id AND r.inserted_at = dt.dag_inserted_at
    JOIN v1_task_events_olap e ON e.task_id = dt.task_id AND e.task_inserted_at = dt.task_inserted_at
    WHERE r.dag_id IS NOT NULL

    UNION ALL

    SELECT
        e.*
    FROM runs r
    JOIN v1_task_events_olap e ON e.task_id = r.task_id AND e.task_inserted_at = r.inserted_at
    WHERE r.task_id IS NOT NULL
), max_retry_counts AS (
    SELECT task_id, MAX(retry_count) AS max_retry_count
    FROM relevant_events
    GROUP BY task_id
), metadata AS (
    SELECT
        MIN(e.inserted_at)::timestamptz AS created_at,
        MIN(e.inserted_at) FILTER (WHERE e.readable_status = 'RUNNING')::timestamptz AS started_at,
        MAX(e.inserted_at) FILTER (WHERE e.readable_status IN ('COMPLETED', 'CANCELLED', 'FAILED'))::timestamptz AS finished_at,
        JSON_AGG(JSON_BUILD_OBJECT('task_id', e.task_id,'task_inserted_at', e.task_inserted_at)) AS task_metadata
    FROM
        relevant_events e
    JOIN max_retry_counts mrc ON (e.task_id, e.retry_count) = (mrc.task_id, mrc.max_retry_count)
), error_message AS (
    SELECT
        e.error_message
    FROM
        relevant_events e
    WHERE
        e.readable_status = 'FAILED'
    ORDER BY
        e.retry_count DESC
    LIMIT 1
)
SELECT
    r.*,
    m.created_at,
    m.started_at,
    m.finished_at,
    e.error_message,
    m.task_metadata
FROM runs r
LEFT JOIN metadata m ON true
LEFT JOIN error_message e ON true
ORDER BY r.inserted_at DESC;

-- name: GetWorkflowRunIdFromDagIdInsertedAt :one
SELECT external_id
FROM v1_dags_olap
WHERE
    id = @dagId::bigint
    AND inserted_at = @dagInsertedAt::timestamptz
;

-- name: FlattenTasksByExternalIds :many
WITH lookups AS (
    SELECT
        *
    FROM
        v1_lookup_table_olap
    WHERE
        external_id = ANY(@externalIds::uuid[])
        AND tenant_id = @tenantId::uuid
), tasks_from_dags AS (
    SELECT
        l.tenant_id,
        dt.task_id,
        dt.task_inserted_at
    FROM
        lookups l
    JOIN
        v1_dag_to_task_olap dt ON l.dag_id = dt.dag_id
    WHERE
        l.dag_id IS NOT NULL
), unioned_tasks AS (
    SELECT
        l.tenant_id AS tenant_id,
        l.task_id AS task_id,
        l.inserted_at AS task_inserted_at
    FROM
        lookups l
    UNION ALL
    SELECT
        t.tenant_id AS tenant_id,
        t.task_id AS task_id,
        t.task_inserted_at AS task_inserted_at
    FROM
        tasks_from_dags t
)
-- Get retry counts for each task
SELECT
    t.tenant_id,
    t.id,
    t.inserted_at,
    t.latest_retry_count AS retry_count
FROM
    v1_tasks_olap t
JOIN
    unioned_tasks ut ON (t.inserted_at, t.id) = (ut.task_inserted_at, ut.task_id);
