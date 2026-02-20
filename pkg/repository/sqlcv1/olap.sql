-- name: CreateOLAPPartitions :exec
SELECT
    create_v1_hash_partitions('v1_task_events_olap_tmp'::text, @partitions::int),
    create_v1_hash_partitions('v1_task_status_updates_tmp'::text, @partitions::int),
    create_v1_olap_partition_with_date_and_status('v1_tasks_olap'::text, @date::date),
    create_v1_olap_partition_with_date_and_status('v1_runs_olap'::text, @date::date),
    create_v1_olap_partition_with_date_and_status('v1_dags_olap'::text, @date::date),
    create_v1_range_partition('v1_payloads_olap'::text, @date::date)
;

-- name: CreateOLAPEventPartitions :exec
SELECT
    create_v1_range_partition('v1_events_olap'::text, @date::date),
    create_v1_range_partition('v1_event_to_run_olap'::text, @date::date),
    create_v1_weekly_range_partition('v1_event_lookup_table_olap'::text, @date::date),
    create_v1_range_partition('v1_incoming_webhook_validation_failures_olap'::text, @date::date),
    create_v1_range_partition('v1_cel_evaluation_failures_olap'::text, @date::date)
;

-- name: AnalyzeV1RunsOLAP :exec
ANALYZE v1_runs_olap;

-- name: AnalyzeV1TasksOLAP :exec
ANALYZE v1_tasks_olap;

-- name: AnalyzeV1DAGsOLAP :exec
ANALYZE v1_dags_olap;

-- name: AnalyzeV1DAGToTaskOLAP :exec
ANALYZE v1_dag_to_task_olap;

-- name: AnalyzeV1PayloadsOLAP :exec
ANALYZE v1_payloads_olap;

-- name: AnalyzeV1LookupTableOLAP :exec
ANALYZE v1_lookup_table_olap;

-- name: ListOLAPPartitionsBeforeDate :many
WITH task_partitions AS (
    SELECT 'v1_tasks_olap' AS parent_table, p::text as partition_name FROM get_v1_partitions_before_date('v1_tasks_olap'::text, @date::date) AS p
), dag_partitions AS (
    SELECT 'v1_dags_olap' AS parent_table, p::text as partition_name FROM get_v1_partitions_before_date('v1_dags_olap', @date::date) AS p
), runs_partitions AS (
    SELECT 'v1_runs_olap' AS parent_table, p::text as partition_name FROM get_v1_partitions_before_date('v1_runs_olap', @date::date) AS p
), events_partitions AS (
    SELECT 'v1_events_olap' AS parent_table, p::TEXT AS partition_name FROM get_v1_partitions_before_date('v1_events_olap', @date::date) AS p
), event_trigger_partitions AS (
    SELECT 'v1_event_to_run_olap' AS parent_table, p::TEXT AS partition_name FROM get_v1_partitions_before_date('v1_event_to_run_olap', @date::date) AS p
), events_lookup_table_partitions AS (
    SELECT 'v1_event_lookup_table_olap' AS parent_table, p::TEXT AS partition_name FROM get_v1_weekly_partitions_before_date('v1_event_lookup_table_olap', @date::date) AS p
), incoming_webhook_validation_failure_partitions AS (
    SELECT 'v1_incoming_webhook_validation_failures_olap' AS parent_table, p::TEXT AS partition_name FROM get_v1_partitions_before_date('v1_incoming_webhook_validation_failures_olap', @date::date) AS p
), cel_evaluation_failures_partitions AS (
    SELECT 'v1_cel_evaluation_failures_olap' AS parent_table, p::TEXT AS partition_name FROM get_v1_partitions_before_date('v1_cel_evaluation_failures_olap', @date::date) AS p
), payloads_partitions AS (
    SELECT 'v1_payloads_olap' AS parent_table, p::TEXT AS partition_name FROM get_v1_partitions_before_date('v1_payloads_olap', @date::date) AS p
), candidates AS (
    SELECT
        *
    FROM
        task_partitions

    UNION ALL

    SELECT
        *
    FROM
        dag_partitions

    UNION ALL

    SELECT
        *
    FROM
        runs_partitions

    UNION ALL

    SELECT
        *
    FROM
        events_partitions

    UNION ALL

    SELECT
        *
    FROM
        event_trigger_partitions

    UNION ALL

    SELECT
        *
    FROM
        events_lookup_table_partitions

    UNION ALL

    SELECT
        *
    FROM
        incoming_webhook_validation_failure_partitions

    UNION ALL

    SELECT
        *
    FROM
        cel_evaluation_failures_partitions

    UNION ALL

    SELECT
        *
    FROM
        payloads_partitions
)

SELECT *
FROM candidates
WHERE
    CASE
        WHEN @shouldPartitionEventsTables::BOOLEAN THEN TRUE
        -- this is a list of all of the tables which are hypertables in timescale, so we should not manually drop their
        -- partitions if @shouldPartitionEventsTables is false
        ELSE parent_table NOT IN ('v1_events_olap', 'v1_event_to_run_olap', 'v1_cel_evaluation_failures_olap', 'v1_incoming_webhook_validation_failures_olap')
    END
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
    parent_task_external_id,
    total_tasks
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
    $11
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
    additional__event_message,
    external_id
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
    $14
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
    e.external_id AS event_external_id,
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
    v1_dag_to_task_olap dt ON (lt.dag_id, lt.inserted_at)= (dt.dag_id, dt.dag_inserted_at)
JOIN
    v1_tasks_olap t ON (t.id, t.inserted_at) = (dt.task_id, dt.task_inserted_at)
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
  t.external_id AS event_external_id,
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
    SELECT dt.task_id, dt.task_inserted_at
    FROM v1_lookup_table_olap lt
    JOIN v1_dag_to_task_olap dt ON lt.dag_id = dt.dag_id AND lt.inserted_at = dt.dag_inserted_at
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
    AND (task_id, task_inserted_at) IN (SELECT task_id, task_inserted_at FROM tasks)
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
  t.external_id AS event_external_id,
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
WITH selected_retry_count AS (
    SELECT
        CASE
            WHEN sqlc.narg('retry_count')::int IS NOT NULL THEN sqlc.narg('retry_count')::int
            ELSE MAX(retry_count)::int
        END AS retry_count
    FROM
        v1_task_events_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND task_id = @taskId::bigint
        AND task_inserted_at = @taskInsertedAt::timestamptz
    LIMIT 1
), relevant_events AS (
    SELECT
        *
    FROM
        v1_task_events_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND task_id = @taskId::bigint
        AND task_inserted_at = @taskInsertedAt::timestamptz
        AND retry_count = (SELECT retry_count FROM selected_retry_count)
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
), queued_at AS (
    SELECT
        MAX(event_timestamp) AS queued_at
    FROM
        relevant_events
    WHERE
        event_type = 'QUEUED'
), task_output AS (
    SELECT
        external_id,
        output
    FROM
        relevant_events
    WHERE
        event_type = 'FINISHED'
    LIMIT 1
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
        WHERE
            tenant_id = @tenantId::uuid
            AND id = @taskId::bigint
            AND inserted_at = @taskInsertedAt::timestamptz
        LIMIT 1
    )
)
SELECT
    t.*,
    st.readable_status::v1_readable_status_olap as status,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at,
    q.queued_at::timestamptz as queued_at,
    o.external_id AS output_event_external_id,
    o.output as output,
    e.error_message as error_message,
    sc.spawned_children,
    (SELECT retry_count FROM selected_retry_count) as retry_count
FROM
    v1_tasks_olap t
LEFT JOIN
    finished_at f ON true
LEFT JOIN
    started_at s ON true
LEFT JOIN
    queued_at q ON true
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
        t.workflow_version_id,
        t.schedule_timeout,
        t.step_timeout,
        t.priority,
        t.sticky,
        t.desired_worker_id,
        t.external_id,
        t.display_name,
        t.input,
        t.additional_metadata,
        t.readable_status,
        t.parent_task_external_id,
        t.workflow_run_id,
        t.latest_retry_count
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
), queued_ats AS (
    SELECT
        e.task_id::bigint,
        MAX(e.event_timestamp) AS queued_at
    FROM
        relevant_events e
    JOIN
        max_retry_counts mrc ON
            e.tenant_id = mrc.tenant_id
            AND e.task_id = mrc.task_id
            AND e.task_inserted_at = mrc.task_inserted_at
            AND e.retry_count = mrc.max_retry_count
    WHERE
        e.event_type = 'QUEUED'
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
        DISTINCT ON (task_id)
        task_id,
        output,
        external_id AS output_event_external_id
    FROM
        relevant_events
    WHERE
        readable_status = 'COMPLETED'
        AND event_type = 'FINISHED'
    ORDER BY
        task_id, event_timestamp DESC
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
    t.workflow_version_id,
    t.schedule_timeout,
    t.step_timeout,
    t.priority,
    t.sticky,
    t.display_name,
    t.additional_metadata,
    t.parent_task_external_id,
    CASE
        WHEN @includePayloads::BOOLEAN THEN t.input
        ELSE '{}'::JSONB
    END::JSONB AS input,
    t.readable_status::v1_readable_status_olap as status,
    t.workflow_run_id,
    f.finished_at::timestamptz as finished_at,
    s.started_at::timestamptz as started_at,
    q.queued_at::timestamptz as queued_at,
    e.error_message as error_message,
    COALESCE(t.latest_retry_count, 0)::int as retry_count,
    CASE
        WHEN @includePayloads::BOOLEAN THEN o.output::JSONB
        ELSE '{}'::JSONB
    END::JSONB as output,
    o.output_event_external_id AS output_event_external_id
FROM
    tasks t
LEFT JOIN
    finished_ats f ON f.task_id = t.id
LEFT JOIN
    started_ats s ON s.task_id = t.id
LEFT JOIN
    queued_ats q ON q.task_id = t.id
LEFT JOIN
    error_message e ON e.task_id = t.id
LEFT JOIN
    task_output o ON o.task_id = t.id
ORDER BY t.inserted_at DESC, t.id DESC;

-- name: FindMinInsertedAtForTaskStatusUpdates :one
WITH tenants AS (
    SELECT UNNEST(
        find_matching_tenants_in_task_events_tmp_partition(
            @partitionNumber::int,
            @tenantIds::UUID[]
        )
    ) AS tenant_id
)

SELECT
    MIN(e.task_inserted_at)::TIMESTAMPTZ AS min_inserted_at
FROM tenants t,
    LATERAL list_task_events_tmp(
        @partitionNumber::int,
        t.tenant_id,
        @eventLimit::int
    ) e
;

-- name: UpdateTaskStatuses :many
WITH tenants AS (
    SELECT UNNEST(
        find_matching_tenants_in_task_events_tmp_partition(
            @partitionNumber::int,
            @tenantIds::UUID[]
        )
    ) AS tenant_id
), locked_events AS (
    SELECT
        e.*
    FROM tenants t,
        LATERAL list_task_events_tmp(
            @partitionNumber::int,
            t.tenant_id,
            @eventLimit::int
        ) e
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
), distinct_tasks_to_lock AS (
    SELECT DISTINCT ON (t.tenant_id, t.id, t.inserted_at)
        t.tenant_id,
        t.id,
        t.inserted_at,
        t.readable_status,
        e.retry_count,
        e.max_readable_status
    FROM
        v1_tasks_olap t
    JOIN
        updatable_events e ON
            (t.tenant_id, t.id, t.inserted_at) = (e.tenant_id, e.task_id, e.task_inserted_at)
    WHERE
        t.inserted_at >= @minInsertedAt::TIMESTAMPTZ
    ORDER BY
        t.tenant_id, t.id, t.inserted_at, t.readable_status
), locked_tasks AS (
    SELECT
        dt.*
    FROM
        v1_tasks_olap t
    JOIN
        distinct_tasks_to_lock dt ON
            (dt.inserted_at, dt.id) = (t.inserted_at, t.id)
    FOR UPDATE
), already_in_target_partition AS (
    -- Check if rows already exist in the target partition (with the new readable_status)
    -- This is to avoid rare cases of duplicates writes causing unique constraint violations,
    -- when i.e. we write to one partition of QUEUED, that gets moved from QUEUED -> COMPLETED,
    -- and we later insert into QUEUED again but try to update to COMPLETED again.
    SELECT
        lt.tenant_id,
        lt.id,
        lt.inserted_at,
        lt.readable_status AS old_readable_status,
        lt.retry_count,
        lt.max_readable_status,
        t.external_id,
        t.latest_worker_id,
        t.workflow_id,
        (t.dag_id IS NOT NULL)::boolean AS is_dag_task
    FROM
        locked_tasks lt
    JOIN
        v1_tasks_olap t ON
            t.inserted_at = lt.inserted_at
            AND t.id = lt.id
            AND t.readable_status = lt.max_readable_status
    WHERE
        -- Only consider rows where we would actually change the status
        lt.readable_status != lt.max_readable_status
), deleted_duplicate_tasks AS (
    -- Delete the old rows that already have a copy in the target partition
    DELETE FROM
        v1_tasks_olap t
    USING
        already_in_target_partition ap
    WHERE
        (t.inserted_at, t.id, t.readable_status) = (ap.inserted_at, ap.id, ap.old_readable_status)
), tasks_to_update AS (
    -- Tasks that need updating and don't already exist in target partition
    SELECT
        lt.tenant_id,
        lt.id,
        lt.inserted_at,
        lt.readable_status,
        lt.retry_count,
        lt.max_readable_status
    FROM
        locked_tasks lt
    WHERE
        NOT EXISTS (
            SELECT 1
            FROM already_in_target_partition ap
            WHERE (ap.tenant_id, ap.id, ap.inserted_at) = (lt.tenant_id, lt.id, lt.inserted_at)
        )
), updated_tasks AS (
    UPDATE
        v1_tasks_olap t
    SET
        readable_status = tu.max_readable_status,
        latest_retry_count = tu.retry_count,
        latest_worker_id = CASE WHEN lw.worker_id::uuid IS NOT NULL THEN lw.worker_id::uuid ELSE t.latest_worker_id END
    FROM
        tasks_to_update tu
    LEFT JOIN
        latest_worker_id lw ON
            (tu.tenant_id, tu.id, tu.inserted_at, tu.retry_count) = (lw.tenant_id, lw.task_id, lw.task_inserted_at, lw.retry_count)
    WHERE
        (t.inserted_at, t.id, t.readable_status) = (tu.inserted_at, tu.id, tu.readable_status)
        AND
            (
                -- if the retry count is greater than the latest retry count, update the status
                (
                    tu.retry_count > t.latest_retry_count
                    AND tu.max_readable_status != t.readable_status
                ) OR
                -- if the retry count is equal to the latest retry count, update the status if the status is greater
                (
                    tu.retry_count = t.latest_retry_count
                    AND tu.max_readable_status > t.readable_status
                )
            )
    RETURNING
        t.tenant_id, t.id, t.inserted_at, t.readable_status, t.external_id, t.latest_worker_id, t.workflow_id, (t.dag_id IS NOT NULL)::boolean AS is_dag_task
), all_result_tasks AS (
    -- Combine actually updated tasks with those already in target partition
    SELECT tenant_id, id, inserted_at, readable_status, external_id, latest_worker_id, workflow_id, is_dag_task
    FROM updated_tasks
    UNION ALL
    SELECT tenant_id, id, inserted_at, max_readable_status AS readable_status, external_id, latest_worker_id, workflow_id, is_dag_task
    FROM already_in_target_partition
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
    WHERE NOT EXISTS (
        SELECT 1
        FROM locked_tasks t
        WHERE (e.tenant_id, e.task_id, e.task_inserted_at) = (t.tenant_id, t.id, t.inserted_at)
    )
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
)
SELECT
    -- Little wonky, but we return the count of events that were processed in each row. Potential edge case
    -- where there are no tasks updated with a non-zero count, but this should be very rare and we'll get
    -- updates on the next run.
    (SELECT count FROM event_count) AS count,
    t.*
FROM
    all_result_tasks t;

-- name: FindMinInsertedAtForDAGStatusUpdates :one
WITH tenants AS (
    SELECT UNNEST(
        find_matching_tenants_in_task_status_updates_tmp_partition(
            @partitionNumber::int,
            @tenantIds::UUID[]
        )
    ) AS tenant_id
)

SELECT
    MIN(u.dag_inserted_at)::TIMESTAMPTZ AS min_inserted_at
FROM tenants t,
    LATERAL list_task_status_updates_tmp(
        @partitionNumber::int,
        t.tenant_id,
        @eventLimit::int
    ) u
;

-- name: UpdateDAGStatuses :many
WITH tenants AS (
    SELECT UNNEST(
        find_matching_tenants_in_task_status_updates_tmp_partition(
            @partitionNumber::int,
            @tenantIds::UUID[]
        )
    ) AS tenant_id
), locked_events AS (
    SELECT
        u.*
    FROM tenants t,
        LATERAL list_task_status_updates_tmp(
            @partitionNumber::int,
            t.tenant_id,
            @eventLimit::int
        ) u
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
        d.tenant_id,
        d.total_tasks
    FROM
        v1_dags_olap d
    WHERE
        d.inserted_at >= @minInsertedAt::TIMESTAMPTZ
        AND (d.inserted_at, d.id, d.tenant_id) IN (
            SELECT
                dd.dag_inserted_at, dd.dag_id, dd.tenant_id
            FROM
                distinct_dags dd
        )
    ORDER BY
        d.inserted_at, d.id
    FOR UPDATE
), relevant_tasks AS (
    SELECT
        t.tenant_id,
        t.id,
        d.id AS dag_id,
        d.inserted_at AS dag_inserted_at,
        t.readable_status
    FROM
        locked_dags d
    JOIN
        v1_dag_to_task_olap dt ON
            (d.id, d.inserted_at) = (dt.dag_id, dt.dag_inserted_at)
    JOIN
        v1_tasks_olap t ON
            (dt.task_id, dt.task_inserted_at) = (t.id, t.inserted_at)
    WHERE
        t.inserted_at >= @minInsertedAt::TIMESTAMPTZ
    -- Note that the ORDER BY seems to help the query planner by pruning partitions earlier. We
    -- have previously seen Postgres use an index-only scan on partitions older than the minInsertedAt,
    -- each of which can take a long time to scan. This can be very pathological since we partition on
    -- both the status and the date, so 14 days of data with 5 statuses is 70 partitions to index scan.
    ORDER BY t.inserted_at DESC
), dag_task_counts AS (
    SELECT
        d.id,
        d.inserted_at,
        d.readable_status,
        d.tenant_id,
        d.total_tasks,
        COUNT(t.id) AS task_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'COMPLETED') AS completed_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'FAILED') AS failed_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'CANCELLED') AS cancelled_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'QUEUED') AS queued_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'RUNNING') AS running_count
    FROM
        locked_dags d
    LEFT JOIN
        relevant_tasks t ON (d.tenant_id, d.id, d.inserted_at) = (t.tenant_id, t.dag_id, t.dag_inserted_at)
    GROUP BY
        d.id, d.inserted_at, d.readable_status, d.tenant_id, d.total_tasks
), dag_new_statuses AS (
    SELECT
        dtc.id,
        dtc.inserted_at,
        dtc.readable_status AS old_readable_status,
        dtc.tenant_id,
        CASE
            -- If we only have queued events, we should keep the status as is
            WHEN dtc.queued_count = dtc.task_count THEN dtc.readable_status
            -- If the task count is not equal to the total tasks, we should set the status to running
            WHEN dtc.task_count != dtc.total_tasks THEN 'RUNNING'
            -- If we have any running or queued tasks, we should set the status to running
            WHEN dtc.running_count > 0 OR dtc.queued_count > 0 THEN 'RUNNING'
            WHEN dtc.failed_count > 0 THEN 'FAILED'
            WHEN dtc.cancelled_count > 0 THEN 'CANCELLED'
            WHEN dtc.completed_count = dtc.task_count THEN 'COMPLETED'
            ELSE 'RUNNING'
        END::v1_readable_status_olap AS new_readable_status
    FROM
        dag_task_counts dtc
), already_in_target_partition AS (
    -- Check if rows already exist in the target partition (with the new readable_status)
    -- This is to avoid rare cases of duplicates writes causing unique constraint violations,
    -- when i.e. we write to one partition of QUEUED, that gets moved from QUEUED -> COMPLETED,
    -- and we later insert into QUEUED again but try to update to COMPLETED again.
    SELECT
        dns.tenant_id,
        dns.id,
        dns.inserted_at,
        dns.old_readable_status,
        dns.new_readable_status
    FROM
        dag_new_statuses dns
    JOIN
        v1_dags_olap d ON
            d.inserted_at = dns.inserted_at
            AND d.id = dns.id
            AND d.readable_status = dns.new_readable_status
    WHERE
        -- Only consider rows where we would actually change the status
        dns.old_readable_status != dns.new_readable_status
), updated_target_partition_dags AS (
    -- Update the existing rows in the target partition
    UPDATE
        v1_dags_olap d
    SET
        -- Touch the row to ensure it's returned (no-op update)
        readable_status = d.readable_status
    FROM
        already_in_target_partition ap
    WHERE
        (d.inserted_at, d.id, d.readable_status) = (ap.inserted_at, ap.id, ap.new_readable_status)
    RETURNING
        d.tenant_id, d.id, d.inserted_at, d.readable_status, d.external_id, d.workflow_id
), deleted_duplicate_dags AS (
    -- Delete the old rows that already have a copy in the target partition
    DELETE FROM
        v1_dags_olap d
    USING
        already_in_target_partition ap
    WHERE
        (d.inserted_at, d.id, d.readable_status) = (ap.inserted_at, ap.id, ap.old_readable_status)
), dags_to_update AS (
    -- DAGs that need updating and don't already exist in target partition
    SELECT DISTINCT ON (dns.tenant_id, dns.id, dns.inserted_at)
        dns.tenant_id,
        dns.id,
        dns.inserted_at,
        dns.old_readable_status,
        dns.new_readable_status
    FROM
        dag_new_statuses dns
    WHERE
        NOT EXISTS (
            SELECT 1
            FROM already_in_target_partition ap
            WHERE (ap.tenant_id, ap.id, ap.inserted_at) = (dns.tenant_id, dns.id, dns.inserted_at)
        )
    ORDER BY
        dns.tenant_id, dns.id, dns.inserted_at, dns.old_readable_status
), updated_dags AS (
    UPDATE
        v1_dags_olap d
    SET
        readable_status = dtu.new_readable_status
    FROM
        dags_to_update dtu
    WHERE
        (d.inserted_at, d.id, d.readable_status) = (dtu.inserted_at, dtu.id, dtu.old_readable_status)
        AND dtu.old_readable_status != dtu.new_readable_status
    RETURNING
        d.tenant_id, d.id, d.inserted_at, d.readable_status, d.external_id, d.workflow_id
), all_result_dags AS (
    -- Combine updated dags from both paths
    SELECT tenant_id, id, inserted_at, readable_status, external_id, workflow_id
    FROM updated_dags
    UNION ALL
    SELECT tenant_id, id, inserted_at, readable_status, external_id, workflow_id
    FROM updated_target_partition_dags
), events_to_requeue AS (
    -- Get events which don't have a corresponding locked_dag
    SELECT
        e.tenant_id,
        e.requeue_retries,
        e.dag_id,
        e.dag_inserted_at
    FROM
        locked_events e
    WHERE NOT EXISTS (
        SELECT 1
        FROM locked_dags d
        WHERE (e.dag_inserted_at, e.dag_id, e.tenant_id) = (d.inserted_at, d.id, d.tenant_id)
    )
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
)
SELECT
    -- Little wonky, but we return the count of events that were processed in each row. Potential edge case
    -- where there are no tasks updated with a non-zero count, but this should be very rare and we'll get
    -- updates on the next run.
    (SELECT count FROM event_count) AS count,
    d.*
FROM
    all_result_dags d;

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
        CASE
            WHEN @includePayloads::BOOLEAN THEN d.input
            ELSE '{}'::JSONB
        END::JSONB AS input,
        d.additional_metadata,
        d.workflow_version_id,
        d.parent_task_external_id
    FROM input i
    JOIN v1_runs_olap r ON (i.id, i.inserted_at) = (r.id, r.inserted_at)
    JOIN v1_dags_olap d ON (r.id, r.inserted_at) = (d.id, d.inserted_at)
    WHERE r.tenant_id = @tenantId::uuid AND r.kind = 'DAG'
), relevant_events AS (
    SELECT r.run_id, e.*
    FROM runs r
    JOIN v1_dag_to_task_olap dt ON (r.dag_id, r.inserted_at) = (dt.dag_id, dt.dag_inserted_at)
    JOIN v1_task_events_olap e ON (e.task_id, e.task_inserted_at) = (dt.task_id, dt.task_inserted_at)
    WHERE e.tenant_id = @tenantId::uuid
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
        output,
        external_id
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
    CASE
        WHEN @includePayloads::BOOLEAN THEN o.output::JSONB
        ELSE '{}'::JSONB
    END::JSONB AS output,
    o.external_id AS output_event_external_id,
    COALESCE(mrc.max_retry_count, 0)::int as retry_count
FROM runs r
LEFT JOIN metadata m ON r.run_id = m.run_id
LEFT JOIN error_message e ON r.run_id = e.run_id
LEFT JOIN task_output o ON r.run_id = o.run_id
LEFT JOIN max_retry_count mrc ON r.run_id = mrc.run_id
ORDER BY r.inserted_at DESC, r.run_id DESC;

-- name: GetTaskPointMetrics :many
SELECT
    DATE_BIN(
        COALESCE(sqlc.narg('interval')::INTERVAL, '1 minute'),
        inserted_at,
        TIMESTAMPTZ '1970-01-01 00:00:00+00'
    ) :: TIMESTAMPTZ AS minute_bucket,
    COUNT(*) FILTER (WHERE readable_status = 'COMPLETED') AS completed_count,
    COUNT(*) FILTER (WHERE readable_status = 'FAILED') AS failed_count
FROM v1_statuses_olap
WHERE
    tenant_id = @tenantId::UUID
    AND inserted_at BETWEEN @createdAfter::TIMESTAMPTZ AND @createdBefore::TIMESTAMPTZ
GROUP BY minute_bucket
ORDER BY minute_bucket;


-- name: GetTenantStatusMetrics :one
WITH task_external_ids AS (
    SELECT external_id
    FROM v1_runs_olap
    WHERE (
        sqlc.narg('parentTaskExternalId')::UUID IS NULL OR parent_task_external_id = sqlc.narg('parentTaskExternalId')::UUID
    ) AND (
        sqlc.narg('triggeringEventExternalId')::UUID IS NULL
        OR (id, inserted_at) IN (
            SELECT etr.run_id, etr.run_inserted_at
            FROM v1_event_lookup_table_olap lt
            JOIN v1_events_olap e ON (lt.tenant_id, lt.event_id, lt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
            JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
            WHERE
                lt.tenant_id = @tenantId::uuid
                AND lt.external_id = sqlc.narg('triggeringEventExternalId')::UUID
        )
    )
    AND (
        sqlc.narg('additionalMetaKeys')::text[] IS NULL
        OR sqlc.narg('additionalMetaValues')::text[] IS NULL
        OR EXISTS (
            SELECT 1 FROM jsonb_each_text(additional_metadata) kv
            JOIN LATERAL (
                SELECT unnest(sqlc.narg('additionalMetaKeys')::text[]) AS k,
                    unnest(sqlc.narg('additionalMetaValues')::text[]) AS v
            ) AS u ON kv.key = u.k AND kv.value = u.v
        )
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
        sqlc.narg('createdBefore')::TIMESTAMPTZ IS NULL OR inserted_at <= sqlc.narg('createdBefore')::TIMESTAMPTZ
    )
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
        d.workflow_version_id AS workflow_version_id,
        d.parent_task_external_id AS parent_task_external_id
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
        t.workflow_version_id AS workflow_version_id,
        NULL :: UUID AS parent_task_external_id
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
    JOIN v1_task_events_olap e ON (e.task_id, e.task_inserted_at) = (dt.task_id, dt.task_inserted_at)
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
        v1_dag_to_task_olap dt ON l.dag_id = dt.dag_id AND l.inserted_at = dt.dag_inserted_at
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
    t.external_id,
    t.latest_retry_count AS retry_count
FROM
    v1_tasks_olap t
JOIN
    unioned_tasks ut ON (t.inserted_at, t.id) = (ut.task_inserted_at, ut.task_id);


-- name: ListWorkflowRunDisplayNames :many
SELECT
    lt.external_id,
    COALESCE(t.display_name, d.display_name) AS display_name,
    COALESCE(t.inserted_at, d.inserted_at) AS inserted_at
FROM v1_lookup_table_olap lt
LEFT JOIN v1_dags_olap d ON (lt.dag_id, lt.inserted_at) = (d.id, d.inserted_at)
LEFT JOIN v1_tasks_olap t ON (lt.task_id, lt.inserted_at) = (t.id, t.inserted_at)
WHERE
    lt.external_id = ANY(@externalIds::uuid[])
    AND lt.tenant_id = @tenantId::uuid
LIMIT 10000
;

-- name: GetRunsListRecursive :many
WITH RECURSIVE all_runs AS (
  -- seed term
    SELECT
        t.id,
        t.inserted_at,
        t.tenant_id,
        t.external_id,
        t.parent_task_external_id,
        0 AS depth
    FROM
        v1_lookup_table_olap lt
        JOIN v1_tasks_olap t
        ON t.inserted_at = lt.inserted_at
        AND t.id          = lt.task_id
    WHERE
        lt.external_id = ANY(@taskExternalIds::uuid[])

    UNION ALL

    -- single recursive term for both DAG- and TASK-driven children
    SELECT
        t.id,
        t.inserted_at,
        t.tenant_id,
        t.external_id,
        t.parent_task_external_id,
        ar.depth + 1 AS depth
    FROM
        v1_runs_olap r
    JOIN all_runs ar ON ar.external_id = r.parent_task_external_id

    -- only present when r.kind = 'DAG'
    LEFT JOIN v1_dag_to_task_olap dt ON r.kind = 'DAG' AND r.id = dt.dag_id AND r.inserted_at = dt.dag_inserted_at

    -- pick the correct task row for either branch
    JOIN v1_tasks_olap t
      ON (
        r.kind = 'DAG'
        AND t.id = dt.task_id
        AND t.inserted_at = dt.task_inserted_at
         )
      OR (
        r.kind = 'TASK'
        AND t.id = r.id
        AND t.inserted_at = r.inserted_at
      )
  WHERE
    r.tenant_id = @tenantId::uuid
    AND ar.depth < @depth::int
    AND r.inserted_at >= @createdAfter::timestamptz
    AND t.inserted_at >= @createdAfter::timestamptz
)
SELECT
  tenant_id,
  id,
  inserted_at,
  external_id,
  parent_task_external_id,
  depth
FROM
  all_runs
WHERE
  tenant_id = @tenantId::uuid;


-- name: BulkCreateEventTriggers :copyfrom
INSERT INTO v1_event_to_run_olap(
    run_id,
    run_inserted_at,
    event_id,
    event_seen_at,
    filter_id
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
;

-- name: ListEventKeys :many
SELECT DISTINCT key
FROM
    v1_events_olap
WHERE
    tenant_id = @tenantId::uuid
    AND seen_at > NOW() - INTERVAL '1 day'
;


-- name: PopulateEventData :many
SELECT
    elt.external_id,
    COUNT(*) FILTER (WHERE r.readable_status = 'QUEUED') AS queued_count,
    COUNT(*) FILTER (WHERE r.readable_status = 'RUNNING') AS running_count,
    COUNT(*) FILTER (WHERE r.readable_status = 'COMPLETED') AS completed_count,
    COUNT(*) FILTER (WHERE r.readable_status = 'CANCELLED') AS cancelled_count,
    COUNT(*) FILTER (WHERE r.readable_status = 'FAILED') AS failed_count,
    JSON_AGG(JSON_BUILD_OBJECT('run_external_id', r.external_id, 'filter_id', etr.filter_id)) FILTER (WHERE r.external_id IS NOT NULL)::JSONB AS triggered_runs
FROM v1_event_lookup_table_olap elt
JOIN v1_events_olap e ON (elt.tenant_id, elt.event_id, elt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
WHERE
    elt.external_id = ANY(@eventExternalIds::uuid[])
    AND elt.tenant_id = @tenantId::uuid
GROUP BY elt.external_id
;

-- name: GetEventByExternalId :one
SELECT e.*
FROM v1_event_lookup_table_olap elt
JOIN v1_events_olap e ON (elt.event_id, elt.event_seen_at) = (e.id, e.seen_at)
WHERE elt.external_id = @eventExternalId::uuid
;

-- name: GetEventByExternalIdUsingTenantId :one
SELECT e.*
FROM v1_event_lookup_table_olap elt
JOIN v1_events_olap e ON (elt.event_id, elt.event_seen_at) = (e.id, e.seen_at)
WHERE
    elt.external_id = @eventExternalId::uuid
    AND elt.tenant_id = @tenantId::uuid
;

-- name: ListEvents :many
SELECT e.*
FROM v1_event_lookup_table_olap elt
JOIN v1_events_olap e ON (elt.tenant_id, elt.event_id, elt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
WHERE
    e.tenant_id = @tenantId
    AND (
        sqlc.narg('keys')::TEXT[] IS NULL OR
        "key" = ANY(sqlc.narg('keys')::TEXT[])
    )
    AND e.seen_at >= @since::TIMESTAMPTZ
    AND (
        sqlc.narg('until')::TIMESTAMPTZ IS NULL OR
        e.seen_at <= sqlc.narg('until')::TIMESTAMPTZ
    )
    AND (
        sqlc.narg('workflowIds')::UUID[] IS NULL OR
        EXISTS (
            SELECT 1
            FROM v1_event_to_run_olap etr
            JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
            WHERE
                (etr.event_id, etr.event_seen_at) = (e.id, e.seen_at)
                AND r.workflow_id = ANY(sqlc.narg('workflowIds')::UUID[]::UUID[])
        )
    )
    AND (
        sqlc.narg('eventIds')::UUID[] IS NULL OR
        elt.external_id = ANY(sqlc.narg('eventIds')::UUID[])
    )
    AND (
        sqlc.narg('additionalMetadata')::JSONB IS NULL OR
        e.additional_metadata @> sqlc.narg('additionalMetadata')::JSONB
    )
    AND (
        CAST(sqlc.narg('statuses')::TEXT[] AS v1_readable_status_olap[]) IS NULL OR
        EXISTS (
            SELECT 1
            FROM v1_event_to_run_olap etr
            JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
            WHERE
                (etr.event_id, etr.event_seen_at) = (e.id, e.seen_at)
                AND r.readable_status = ANY(CAST(sqlc.narg('statuses')::text[]::TEXT[] AS v1_readable_status_olap[]))
        )
    )
    AND (
        sqlc.narg('scopes')::TEXT[] IS NULL OR
        e.scope = ANY(sqlc.narg('scopes')::TEXT[])
    )
ORDER BY e.seen_at DESC, e.id
OFFSET
    COALESCE(sqlc.narg('offset')::BIGINT, 0)
LIMIT
    COALESCE(sqlc.narg('limit')::BIGINT, 50)
;

-- name: CountEvents :one
WITH included_events AS (
    SELECT e.*
    FROM v1_event_lookup_table_olap elt
    JOIN v1_events_olap e ON (elt.tenant_id, elt.event_id, elt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
    WHERE
        e.tenant_id = @tenantId
        AND (
            sqlc.narg('keys')::TEXT[] IS NULL OR
            "key" = ANY(sqlc.narg('keys')::TEXT[])
        )
        AND e.seen_at >= @since::TIMESTAMPTZ
        AND (
            sqlc.narg('until')::TIMESTAMPTZ IS NULL OR
            e.seen_at <= sqlc.narg('until')::TIMESTAMPTZ
        )
        AND (
            sqlc.narg('workflowIds')::UUID[] IS NULL OR
            EXISTS (
                SELECT 1
                FROM v1_event_to_run_olap etr
                JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
                WHERE
                    (etr.event_id, etr.event_seen_at) = (e.id, e.seen_at)
                    AND r.workflow_id = ANY(sqlc.narg('workflowIds')::UUID[]::UUID[])
            )
        )
        AND (
            sqlc.narg('eventIds')::UUID[] IS NULL OR
            elt.external_id = ANY(sqlc.narg('eventIds')::UUID[])
        )
        AND (
            sqlc.narg('additionalMetadata')::JSONB IS NULL OR
            e.additional_metadata @> sqlc.narg('additionalMetadata')::JSONB
        )
        AND (
            CAST(sqlc.narg('statuses')::TEXT[] AS v1_readable_status_olap[]) IS NULL OR
            EXISTS (
                SELECT 1
                FROM v1_event_to_run_olap etr
                JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
                WHERE
                    (etr.event_id, etr.event_seen_at) = (e.id, e.seen_at)
                    AND r.readable_status = ANY(CAST(sqlc.narg('statuses')::text[]::TEXT[] AS v1_readable_status_olap[]))
            )
        )
        AND (
            sqlc.narg('scopes')::TEXT[] IS NULL OR
            e.scope = ANY(sqlc.narg('scopes')::TEXT[])
        )
        ORDER BY e.seen_at DESC, e.id
    LIMIT 20000
)

SELECT COUNT(*)
FROM included_events e
;

-- name: GetDagDurations :many
SELECT
    lt.external_id,
    MIN(e.event_timestamp) FILTER (WHERE e.readable_status = 'RUNNING')::TIMESTAMPTZ AS started_at,
    MAX(e.event_timestamp) FILTER (WHERE e.readable_status IN ('COMPLETED', 'FAILED', 'CANCELLED'))::TIMESTAMPTZ AS finished_at
FROM
    v1_lookup_table_olap lt
JOIN
    v1_dags_olap d ON (lt.dag_id, lt.inserted_at) = (d.id, d.inserted_at)
JOIN
    v1_dag_to_task_olap dt ON (d.id, d.inserted_at) = (dt.dag_id, dt.dag_inserted_at)
JOIN
    v1_task_events_olap e ON (dt.task_id, dt.task_inserted_at) = (e.task_id, e.task_inserted_at)
WHERE lt.external_id = ANY(@externalIds::UUID[])
    AND lt.tenant_id = @tenantId::UUID
    AND d.inserted_at >= @minInsertedAt::TIMESTAMPTZ
GROUP BY lt.external_id
;

-- name: GetTaskDurationsByTaskIds :many
WITH input AS (
    SELECT
        UNNEST(@taskIds::bigint[]) AS task_id,
        UNNEST(@taskInsertedAts::timestamptz[]) AS inserted_at,
        UNNEST(@readableStatuses::v1_readable_status_olap[]) AS readable_status
), task_data AS (
    SELECT
        i.task_id,
        i.inserted_at,
        t.external_id,
        t.display_name,
        t.readable_status,
        t.latest_retry_count,
        t.tenant_id
    FROM
        input i
    JOIN
        v1_tasks_olap t ON (t.inserted_at, t.id, t.readable_status, t.tenant_id) = (i.inserted_at, i.task_id, i.readable_status, @tenantId::uuid)
), task_events AS (
    SELECT
        td.task_id,
        td.inserted_at,
        e.event_type,
        e.event_timestamp,
        e.readable_status
    FROM
        task_data td
    JOIN
        v1_task_events_olap e ON (e.tenant_id, e.task_id, e.task_inserted_at, e.retry_count) = (td.tenant_id, td.task_id, td.inserted_at, td.latest_retry_count)
), task_times AS (
    SELECT
        task_id,
        inserted_at,
        MIN(CASE WHEN event_type = 'STARTED' THEN event_timestamp END) AS started_at,
        MAX(CASE WHEN readable_status = ANY(ARRAY['COMPLETED', 'FAILED', 'CANCELLED']::v1_readable_status_olap[])
            THEN event_timestamp END) AS finished_at
    FROM task_events
    GROUP BY task_id, inserted_at
)
SELECT
    tt.started_at::timestamptz AS started_at,
    tt.finished_at::timestamptz AS finished_at
FROM
    task_data td
LEFT JOIN
    task_times tt ON (td.task_id, td.inserted_at) = (tt.task_id, tt.inserted_at)
ORDER BY td.task_id, td.inserted_at;

-- name: CreateIncomingWebhookValidationFailureLogs :exec
WITH inputs AS (
    SELECT
        UNNEST(@incomingWebhookNames::TEXT[]) AS incoming_webhook_name,
        UNNEST(@errors::TEXT[]) AS error
)
INSERT INTO v1_incoming_webhook_validation_failures_olap(
    tenant_id,
    incoming_webhook_name,
    error
)
SELECT
    @tenantId::UUID,
    i.incoming_webhook_name,
    i.error
FROM inputs i;

-- name: StoreCELEvaluationFailures :exec
WITH inputs AS (
    SELECT
        UNNEST(CAST(@sources::TEXT[] AS v1_cel_evaluation_failure_source[])) AS source,
        UNNEST(@errors::TEXT[]) AS error
)
INSERT INTO v1_cel_evaluation_failures_olap (
    tenant_id,
    source,
    error
)
SELECT @tenantId::UUID, source, error
FROM inputs
;

-- name: PutPayloads :exec
WITH inputs AS (
    SELECT
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(@payloads::JSONB[]) AS payload,
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(CAST(@locations::TEXT[] AS v1_payload_location_olap[])) AS location,
        UNNEST(@externalLocationKeys::TEXT[]) AS external_location_key
)

INSERT INTO v1_payloads_olap (
    tenant_id,
    external_id,
    inserted_at,
    location,
    external_location_key,
    inline_content
)

SELECT
    i.tenant_id,
    i.external_id,
    i.inserted_at,
    i.location,
    CASE
        WHEN i.location = 'EXTERNAL' THEN i.external_location_key
        ELSE NULL
    END,
    CASE
        WHEN i.location = 'INLINE' THEN i.payload
        ELSE NULL
    END AS inline_content
FROM inputs i
ON CONFLICT (tenant_id, external_id, inserted_at) DO UPDATE
SET
    location = EXCLUDED.location,
    external_location_key = EXCLUDED.external_location_key,
    inline_content = EXCLUDED.inline_content,
    updated_at = NOW()
;

-- name: OffloadPayloads :exec
WITH inputs AS (
    SELECT
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@externalLocationKeys::TEXT[]) AS external_location_key
)

UPDATE v1_payloads_olap
SET
    location = 'EXTERNAL',
    external_location_key = i.external_location_key,
    inline_content = NULL,
    updated_at = NOW()
FROM inputs i
WHERE
    (v1_payloads_olap.tenant_id, v1_payloads_olap.external_id) = (i.tenant_id, i.external_id)
    AND v1_payloads_olap.location = 'INLINE'
    AND v1_payloads_olap.external_location_key IS NULL
;

-- name: ReadPayloadsOLAP :many
SELECT *
FROM v1_payloads_olap
WHERE
    tenant_id = @tenantId::UUID
    AND external_id = ANY(@externalIds::UUID[])
;

-- name: ListWorkflowRunExternalIds :many
SELECT external_id
FROM v1_runs_olap
WHERE
    tenant_id = @tenantId::UUID
    AND inserted_at > @since::TIMESTAMPTZ
    AND (
        sqlc.narg('until')::TIMESTAMPTZ IS NULL
        OR inserted_at <= sqlc.narg('until')::TIMESTAMPTZ
    )
    AND readable_status = ANY(CAST(@statuses::TEXT[] AS v1_readable_status_olap[]))
    AND (
        sqlc.narg('additionalMetaKeys')::text[] IS NULL
        OR sqlc.narg('additionalMetaValues')::text[] IS NULL
        OR EXISTS (
            SELECT 1 FROM jsonb_each_text(additional_metadata) kv
            JOIN LATERAL (
                SELECT unnest(sqlc.narg('additionalMetaKeys')::text[]) AS k,
                    unnest(sqlc.narg('additionalMetaValues')::text[]) AS v
            ) AS u ON kv.key = u.k AND kv.value = u.v
        )
    )
    AND (
        sqlc.narg('workflowIds')::UUID[] IS NULL OR workflow_id = ANY(sqlc.narg('workflowIds')::UUID[])
    )
;

-- name: CountOLAPTempTableSizeForDAGStatusUpdates :one
SELECT COUNT(*) AS total
FROM v1_task_status_updates_tmp
;

-- name: CountOLAPTempTableSizeForTaskStatusUpdates :one
SELECT COUNT(*) AS total
FROM v1_task_events_olap_tmp
;

-- name: ListYesterdayRunCountsByStatus :many
SELECT readable_status, COUNT(*)
FROM v1_runs_olap
WHERE inserted_at::DATE = (NOW() - INTERVAL '1 day')::DATE
GROUP BY readable_status
;

-- name: CheckLastAutovacuumForPartitionedTables :many
SELECT
    s.schemaname,
    s.relname AS tablename,
    s.last_autovacuum,
    EXTRACT(EPOCH FROM (NOW() - s.last_autovacuum)) AS seconds_since_last_autovacuum
FROM pg_stat_user_tables s
JOIN pg_catalog.pg_class c ON c.oid = (quote_ident(s.schemaname)||'.'||quote_ident(s.relname))::regclass
WHERE s.schemaname = 'public'
    AND c.relispartition = true
    AND c.relkind = 'r'
ORDER BY s.last_autovacuum ASC NULLS LAST
;

-- name: ListPaginatedOLAPPayloadsForOffload :many
WITH payloads AS (
    SELECT
        (p).*
    FROM list_paginated_olap_payloads_for_offload(
        @partitionDate::DATE,
        @lastTenantId::UUID,
        @lastExternalId::UUID,
        @lastInsertedAt::TIMESTAMPTZ,
        @nextTenantId::UUID,
        @nextExternalId::UUID,
        @nextInsertedAt::TIMESTAMPTZ,
        @batchSize::INTEGER
    ) p
)
SELECT
    tenant_id::UUID,
    external_id::UUID,
    location::v1_payload_location_olap,
    COALESCE(external_location_key, '')::TEXT AS external_location_key,
    inline_content::JSONB AS inline_content,
    inserted_at::TIMESTAMPTZ,
    updated_at::TIMESTAMPTZ
FROM payloads;

-- name: CreateOLAPPayloadRangeChunks :many
WITH chunks AS (
    SELECT
        (p).*
    FROM create_olap_payload_offload_range_chunks(
        @partitionDate::DATE,
        @windowSize::INTEGER,
        @chunkSize::INTEGER,
        @lastTenantId::UUID,
        @lastExternalId::UUID,
        @lastInsertedAt::TIMESTAMPTZ
    ) p
)

SELECT
    lower_tenant_id::UUID,
    lower_external_id::UUID,
    lower_inserted_at::TIMESTAMPTZ,
    upper_tenant_id::UUID,
    upper_external_id::UUID,
    upper_inserted_at::TIMESTAMPTZ
FROM chunks
;

-- name: CreateV1PayloadOLAPCutoverTemporaryTable :exec
SELECT copy_v1_payloads_olap_partition_structure(@date::DATE);

-- name: SwapV1PayloadOLAPPartitionWithTemp :exec
SELECT swap_v1_payloads_olap_partition_with_temp(@date::DATE);

-- name: AcquireOrExtendOLAPCutoverJobLease :one
WITH inputs AS (
    SELECT
        @key::DATE AS key,
        @leaseProcessId::UUID AS lease_process_id,
        @leaseExpiresAt::TIMESTAMPTZ AS lease_expires_at,
        @lastTenantId::UUID AS last_tenant_id,
        @lastExternalId::UUID AS last_external_id,
        @lastInsertedAt::TIMESTAMPTZ AS last_inserted_at
), any_lease_held_by_other_process AS (
    -- need coalesce here in case there are no rows that don't belong to this process
    SELECT COALESCE(BOOL_OR(lease_expires_at > NOW()), FALSE) AS lease_exists
    FROM v1_payloads_olap_cutover_job_offset
    WHERE lease_process_id != @leaseProcessId::UUID
), to_insert AS (
    SELECT *
    FROM inputs
    -- if a lease is held by another process, we shouldn't try to insert a new row regardless
    -- of which key we're trying to acquire a lease on
    WHERE NOT (SELECT lease_exists FROM any_lease_held_by_other_process)
)

INSERT INTO v1_payloads_olap_cutover_job_offset (key, lease_process_id, lease_expires_at, last_tenant_id, last_external_id, last_inserted_at)
SELECT
    ti.key,
    ti.lease_process_id,
    ti.lease_expires_at,
    ti.last_tenant_id,
    ti.last_external_id,
    ti.last_inserted_at
FROM to_insert ti
ON CONFLICT (key)
DO UPDATE SET
    -- if the lease is held by this process, then we extend the offset to the new value
    -- otherwise it's a new process acquiring the lease, so we should keep the offset where it was before
    last_tenant_id = CASE
        WHEN EXCLUDED.lease_process_id = v1_payloads_olap_cutover_job_offset.lease_process_id THEN EXCLUDED.last_tenant_id
        ELSE v1_payloads_olap_cutover_job_offset.last_tenant_id
    END,
    last_external_id = CASE
        WHEN EXCLUDED.lease_process_id = v1_payloads_olap_cutover_job_offset.lease_process_id THEN EXCLUDED.last_external_id
        ELSE v1_payloads_olap_cutover_job_offset.last_external_id
    END,
    last_inserted_at = CASE
        WHEN EXCLUDED.lease_process_id = v1_payloads_olap_cutover_job_offset.lease_process_id THEN EXCLUDED.last_inserted_at
        ELSE v1_payloads_olap_cutover_job_offset.last_inserted_at
    END,

    lease_process_id = EXCLUDED.lease_process_id,
    lease_expires_at = EXCLUDED.lease_expires_at
WHERE v1_payloads_olap_cutover_job_offset.lease_expires_at < NOW() OR v1_payloads_olap_cutover_job_offset.lease_process_id = @leaseProcessId::UUID
RETURNING *
;

-- name: MarkOLAPCutoverJobAsCompleted :exec
UPDATE v1_payloads_olap_cutover_job_offset
SET is_completed = TRUE
WHERE key = @key::DATE
;

-- name: CleanUpOLAPCutoverJobOffsets :exec
DELETE FROM v1_payloads_olap_cutover_job_offset
WHERE NOT key = ANY(@keysToKeep::DATE[])
;


-- name: DiffOLAPPayloadSourceAndTargetPartitions :many
WITH payloads AS (
    SELECT
        (p).*
    FROM diff_olap_payload_source_and_target_partitions(@partitionDate::DATE) p
)

SELECT
    tenant_id::UUID,
    external_id::UUID,
    inserted_at::TIMESTAMPTZ,
    location::v1_payload_location_olap,
    COALESCE(external_location_key, '')::TEXT AS external_location_key,
    inline_content::JSONB AS inline_content,
    updated_at::TIMESTAMPTZ
FROM payloads
;

-- name: ComputeOLAPPayloadBatchSize :one
SELECT compute_olap_payload_batch_size(
    @partitionDate::DATE,
    @lastTenantId::UUID,
    @lastExternalId::UUID,
    @lastInsertedAt::TIMESTAMPTZ,
    @batchSize::INTEGER
) AS total_size_bytes;
