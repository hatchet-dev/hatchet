-- name: UpsertQueues :exec
WITH ordered_names AS (
    SELECT unnest(@names::text[]) AS name
    ORDER BY name ASC
), existing_queues AS (
    SELECT tenant_id, name, last_active
    FROM v1_queue
    WHERE tenant_id = $1
      AND name = ANY(@names::text[])
), locked_existing_queues AS (
    SELECT *
    FROM v1_queue
    WHERE
        tenant_id = $1
        AND name IN (SELECT name FROM existing_queues)
    ORDER BY name ASC
    FOR UPDATE SKIP LOCKED
), names_to_insert AS (
    SELECT on1.name
    FROM ordered_names on1
    LEFT JOIN existing_queues eq ON eq.name = on1.name
    WHERE eq.name IS NULL
), updated_queues AS (
    UPDATE v1_queue
    SET last_active = NOW()
    WHERE tenant_id = $1
      AND name IN (SELECT name FROM locked_existing_queues)
)
-- Insert new queues
INSERT INTO v1_queue (tenant_id, name, last_active)
SELECT $1, name, NOW()
FROM names_to_insert
ON CONFLICT (tenant_id, name) DO NOTHING;

-- name: ListActionsForWorkers :many
SELECT
    w."id" as "workerId",
    a."actionId"
FROM
    "Worker" w
LEFT JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
LEFT JOIN
    "Action" a ON atw."A" = a."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."id" = ANY(@workerIds::uuid[])
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false;

-- name: ListAvailableSlotsForWorkers :many
WITH worker_max_runs AS (
    SELECT
        "id",
        "maxRuns"
    FROM
        "Worker"
    WHERE
        "tenantId" = @tenantId::uuid
        AND "id" = ANY(@workerIds::uuid[])
), worker_filled_slots AS (
    SELECT
        worker_id,
        (
            COALESCE(SUM(CASE WHEN batch_id IS NULL THEN 1 ELSE 0 END), 0)::integer
            + COUNT(DISTINCT batch_id)::integer
        ) AS "filledSlots"
    FROM
        v1_task_runtime
    WHERE
        tenant_id = @tenantId::uuid
        AND worker_id = ANY(@workerIds::uuid[])
    GROUP BY
        worker_id
)
-- subtract the filled slots from the max runs to get the available slots
SELECT
    wmr."id",
    wmr."maxRuns" - COALESCE(wfs."filledSlots", 0) AS "availableSlots"
FROM
    worker_max_runs wmr
LEFT JOIN
    worker_filled_slots wfs ON wmr."id" = wfs.worker_id;

-- name: ListQueues :many
SELECT
    *
FROM
    v1_queue
WHERE
    tenant_id = @tenantId::uuid
    AND last_active > NOW() - INTERVAL '1 day';

-- name: ListQueueItemsForQueue :many
SELECT
    *
FROM
    v1_queue_item qi
WHERE
    qi.tenant_id = @tenantId::uuid
    AND qi.queue = @queue::text
    AND (
        sqlc.narg('gtId')::bigint IS NULL OR
        qi.id >= sqlc.narg('gtId')::bigint
    )
    -- Added to ensure that the index is used
    AND qi.priority >= 1 AND qi.priority <= 4
ORDER BY
    qi.priority DESC,
    qi.id ASC
LIMIT
    COALESCE(sqlc.narg('limit')::integer, 100);

-- name: GetMinUnprocessedQueueItemId :one
WITH priority_1 AS (
    SELECT
        id
    FROM
        v1_queue_item
    WHERE
        tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 1
    ORDER BY
        id ASC
    LIMIT 1
),
priority_2 AS (
    SELECT
        id
    FROM
        v1_queue_item
    WHERE
        tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 2
    ORDER BY
        id ASC
    LIMIT 1
),
priority_3 AS (
    SELECT
        id
    FROM
        v1_queue_item
    WHERE
        tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 3
    ORDER BY
        id ASC
    LIMIT 1
),
priority_4 AS (
    SELECT
        id
    FROM
        v1_queue_item
    WHERE
        tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 4
    ORDER BY
        id ASC
    LIMIT 1
)
SELECT
    COALESCE(MIN(id), 0)::bigint AS "minId"
FROM (
    SELECT id FROM priority_1
    UNION ALL
    SELECT id FROM priority_2
    UNION ALL
    SELECT id FROM priority_3
    UNION ALL
    SELECT id FROM priority_4
) AS combined_priorities;

-- name: BulkQueueItems :many
WITH locked_qis AS (
    SELECT
        id
    FROM
        v1_queue_item
    WHERE
        id = ANY(@ids::bigint[])
    ORDER BY
        id ASC
    FOR UPDATE
)
DELETE FROM
    v1_queue_item
WHERE
    id = ANY(@ids::bigint[])
RETURNING
    id;

-- name: UpdateTasksToAssigned :many
WITH input AS (
    SELECT
        id,
        inserted_at,
        worker_id
    FROM
        (
            SELECT
                UNNEST(@taskIds::bigint[]) AS id,
                UNNEST(@taskInsertedAts::timestamptz[]) AS inserted_at,
                UNNEST(@workerIds::uuid[]) AS worker_id
        ) AS subquery
    ORDER BY id
), updated_tasks AS (
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        i.worker_id,
        t.tenant_id,
        t.batch_key,
        CURRENT_TIMESTAMP + convert_duration_to_interval(t.step_timeout) AS timeout_at
    FROM
        v1_task t
    JOIN
        input i ON (t.id, t.inserted_at) = (i.id, i.inserted_at)
    WHERE
        t.inserted_at >= @minTaskInsertedAt::timestamptz
        AND NOT EXISTS (
            SELECT 1
            FROM v1_task_event e
            WHERE
                e.task_id = t.id
                AND e.task_inserted_at = t.inserted_at
                AND e.retry_count = t.retry_count
                AND e.event_type = 'CANCELLED'::v1_task_event_type
        )
    ORDER BY t.id
), assigned_tasks AS (
    INSERT INTO v1_task_runtime (
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        tenant_id,
        batch_key,
        timeout_at
    )
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        t.worker_id,
        @tenantId::uuid,
        t.batch_key,
        t.timeout_at
    FROM
        updated_tasks t
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO UPDATE
    SET
        worker_id = EXCLUDED.worker_id,
        tenant_id = EXCLUDED.tenant_id,
        batch_key = EXCLUDED.batch_key,
        timeout_at = EXCLUDED.timeout_at
    -- only return the task ids that were successfully assigned
    RETURNING task_id, worker_id
)
SELECT
    asr.task_id,
    asr.worker_id
FROM
    assigned_tasks asr;

-- name: InsertBufferedTaskRuntimes :exec
WITH input AS (
    SELECT
        UNNEST(@taskIds::bigint[]) AS task_id,
        UNNEST(@taskInsertedAts::timestamptz[]) AS task_inserted_at,
        UNNEST(@taskRetryCounts::integer[]) AS task_retry_count
)
INSERT INTO v1_task_runtime (
    task_id,
    task_inserted_at,
    retry_count,
    worker_id,
    tenant_id,
    timeout_at,
    batch_key
)
SELECT
    input.task_id,
    input.task_inserted_at,
    input.task_retry_count,
    NULL,
    @tenantId::uuid,
    CURRENT_TIMESTAMP + convert_duration_to_interval(t.step_timeout),
    t.batch_key
FROM
    input
JOIN
    v1_task t ON t.id = input.task_id AND t.inserted_at = input.task_inserted_at
ON CONFLICT (task_id, task_inserted_at, retry_count) DO UPDATE
SET
    timeout_at = EXCLUDED.timeout_at,
    batch_key = EXCLUDED.batch_key
WHERE
    v1_task_runtime.worker_id IS NULL;

-- name: GetDesiredLabels :many
SELECT
    "key",
    "strValue",
    "intValue",
    "required",
    "weight",
    "comparator",
    "stepId"
FROM
    "StepDesiredWorkerLabel"
WHERE
    "stepId" = ANY(@stepIds::uuid[]);

-- name: ListStepsWithBatchConfig :many
SELECT
    "id" AS step_id
FROM
    "Step"
WHERE
    "id" = ANY(@stepIds::uuid[])
    AND "batch_size" IS NOT NULL
    AND "batch_size" >= 1;

-- name: GetQueuedCounts :many
SELECT
    queue,
    COUNT(*) AS count
FROM
    v1_queue_item qi
WHERE
    qi.tenant_id = @tenantId::uuid
GROUP BY
    qi.queue;

-- name: DeleteTasksFromQueue :exec
WITH input AS (
    SELECT
        *
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS task_id,
                unnest(@retryCounts::integer[]) AS retry_count
        ) AS subquery
), locked_qis AS (
    SELECT
        id
    FROM
        v1_queue_item
    WHERE
        (task_id, retry_count) IN (SELECT task_id, retry_count FROM input)
    ORDER BY
        id ASC
    FOR UPDATE
)
DELETE FROM
    v1_queue_item
WHERE
    id = ANY(SELECT id FROM locked_qis);

-- name: MoveRateLimitedQueueItems :many
WITH input AS (
    SELECT
        UNNEST(@ids::bigint[]) AS id,
        UNNEST(@requeueAfter::timestamptz[]) AS requeue_after
), moved_items AS (
    DELETE FROM v1_queue_item
    WHERE id = ANY(SELECT id FROM input)
    RETURNING
        id,
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        batch_key
)
INSERT INTO v1_rate_limited_queue_items (
    requeue_after,
    tenant_id,
    queue,
    task_id,
    task_inserted_at,
    external_id,
    action_id,
    step_id,
    workflow_id,
    workflow_run_id,
    schedule_timeout_at,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    retry_count,
    batch_key
)
SELECT
    i.requeue_after,
    tenant_id,
    queue,
    task_id,
    task_inserted_at,
    external_id,
    action_id,
    step_id,
    workflow_id,
    workflow_run_id,
    schedule_timeout_at,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    retry_count,
    batch_key
FROM moved_items
JOIN input i ON moved_items.id = i.id
ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
RETURNING tenant_id, task_id, task_inserted_at, retry_count;

-- name: RequeueRateLimitedQueueItems :many
WITH ready_items AS (
    SELECT
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        batch_key
    FROM
        v1_rate_limited_queue_items
    WHERE
        tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND requeue_after <= NOW()
    ORDER BY
        task_id, task_inserted_at, retry_count
    FOR UPDATE SKIP LOCKED -- locked are about to be deleted
), deleted_items AS (
    DELETE FROM v1_rate_limited_queue_items
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM ready_items)
    RETURNING
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        batch_key
)
INSERT INTO v1_queue_item (
    tenant_id,
    queue,
    task_id,
    task_inserted_at,
    external_id,
    action_id,
    step_id,
    workflow_id,
    workflow_run_id,
    schedule_timeout_at,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    retry_count,
    batch_key
)
SELECT
    tenant_id,
    queue,
    task_id,
    task_inserted_at,
    external_id,
    action_id,
    step_id,
    workflow_id,
    workflow_run_id,
    schedule_timeout_at,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    retry_count,
    batch_key
FROM ready_items
RETURNING id, tenant_id, task_id, task_inserted_at, retry_count;

-- name: CleanupV1QueueItem :execresult
WITH locked_qis as (
    SELECT qi.task_id, qi.task_inserted_at, qi.retry_count
    FROM v1_queue_item qi
    WHERE NOT EXISTS (
        SELECT 1
        FROM v1_task vt
        WHERE qi.task_id = vt.id
            AND qi.task_inserted_at = vt.inserted_at
    )
    ORDER BY qi.id ASC
    LIMIT @batchSize::int
    FOR UPDATE SKIP LOCKED
)
DELETE FROM v1_queue_item
WHERE (task_id, task_inserted_at, retry_count) IN (
    SELECT task_id, task_inserted_at, retry_count
    FROM locked_qis
);

-- name: CleanupV1RetryQueueItem :execresult
WITH locked_qis as (
    SELECT qi.task_id, qi.task_inserted_at, qi.task_retry_count
    FROM v1_retry_queue_item qi
    WHERE NOT EXISTS (
        SELECT 1
        FROM v1_task vt
        WHERE qi.task_id = vt.id
        AND qi.task_inserted_at = vt.inserted_at
    )
    ORDER BY qi.task_id, qi.task_inserted_at, qi.task_retry_count
    LIMIT @batchSize::int
    FOR UPDATE SKIP LOCKED
)
DELETE FROM v1_retry_queue_item
WHERE (task_id, task_inserted_at) IN (
    SELECT task_id, task_inserted_at
    FROM locked_qis
);

-- name: CleanupV1RateLimitedQueueItem :execresult
WITH locked_qis as (
    SELECT qi.task_id, qi.task_inserted_at, qi.retry_count
    FROM v1_rate_limited_queue_items qi
    WHERE NOT EXISTS (
        SELECT 1
        FROM v1_task vt
        WHERE qi.task_id = vt.id
        AND qi.task_inserted_at = vt.inserted_at
    )
    ORDER BY qi.task_id, qi.task_inserted_at, qi.retry_count
    LIMIT @batchSize::int
    FOR UPDATE SKIP LOCKED
)
DELETE FROM v1_rate_limited_queue_items
WHERE (task_id, task_inserted_at) IN (
    SELECT task_id, task_inserted_at
    FROM locked_qis
);

-- name: ListBatchedQueueItemsForBatch :many
SELECT
    id,
    tenant_id,
    queue,
    task_id,
    task_inserted_at,
    external_id,
    action_id,
    step_id,
    workflow_id,
    workflow_run_id,
    schedule_timeout_at,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    retry_count,
    batch_key,
    inserted_at
FROM
    v1_batched_queue_item
WHERE
    tenant_id = @tenantId::uuid
    AND step_id = @stepId::uuid
    AND batch_key = @batchKey::text
    AND (
        sqlc.narg('afterId')::bigint IS NULL
        OR id > sqlc.narg('afterId')::bigint
    )
ORDER BY
    priority DESC,
    id ASC
LIMIT
    COALESCE(sqlc.narg('limit')::integer, 1000);

-- name: ListDistinctBatchResources :many
SELECT
    b.step_id,
    b.batch_key,
    MIN(b.inserted_at)::timestamptz AS oldest_item_at,
    COUNT(*) AS pending_count,
    MAX(s."batch_size")::integer AS batch_size,
    MAX(s."batch_flush_interval_ms")::integer AS batch_flush_interval_ms,
    MAX(s."batch_max_runs") AS batch_max_runs
FROM
    v1_batched_queue_item b
JOIN
    "Step" s ON s."id" = b.step_id
WHERE
    b.tenant_id = @tenantId::uuid
GROUP BY
    b.step_id,
    b.batch_key
ORDER BY
    oldest_item_at ASC;

-- name: MoveQueueItemsToBatchedQueue :many
WITH locked_qis AS (
    SELECT
        qi.*
    FROM
        v1_queue_item qi
    JOIN
        "Step" s ON s."id" = qi.step_id
    WHERE
        qi.id = ANY(@ids::bigint[])
        AND NULLIF(BTRIM(qi.batch_key), '') IS NOT NULL
        AND s."batch_size" IS NOT NULL
        AND s."batch_size" >= 1
    ORDER BY
        qi.id ASC
    FOR UPDATE
), inserted AS (
    INSERT INTO v1_batched_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        batch_key,
        inserted_at
    )
    SELECT
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        BTRIM(batch_key),
        CURRENT_TIMESTAMP
    FROM
        locked_qis
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    RETURNING task_id
), deleted AS (
    DELETE FROM v1_queue_item
    WHERE id IN (SELECT id FROM locked_qis)
    RETURNING id
)
SELECT id FROM deleted;

-- name: DeleteBatchedQueueItems :exec
DELETE FROM
    v1_batched_queue_item
WHERE
    id = ANY(@ids::bigint[]);

-- name: ListBatchedQueueItemsToTimeout :many
SELECT
    bqi.id,
    bqi.tenant_id,
    bqi.queue,
    bqi.task_id,
    bqi.task_inserted_at,
    bqi.external_id,
    bqi.action_id,
    bqi.step_id,
    bqi.workflow_id,
    bqi.workflow_run_id,
    bqi.schedule_timeout_at,
    bqi.step_timeout,
    bqi.priority,
    bqi.sticky,
    bqi.desired_worker_id,
    bqi.retry_count,
    bqi.batch_key
FROM
    v1_batched_queue_item bqi
LEFT JOIN
    v1_task_runtime tr ON (
        tr.task_id = bqi.task_id
        AND tr.task_inserted_at = bqi.task_inserted_at
        AND tr.retry_count = bqi.retry_count
    )
WHERE
    bqi.tenant_id = @tenantId::uuid
    AND bqi.schedule_timeout_at <= NOW()
    AND tr.task_id IS NULL  -- Only timeout tasks that are NOT already running
ORDER BY
    bqi.id ASC
LIMIT
    COALESCE(sqlc.narg('limit')::integer, 1000);

-- name: ListExistingBatchedQueueItemIds :many
SELECT
    id
FROM
    v1_batched_queue_item
WHERE
    tenant_id = @tenantId::uuid
    AND id = ANY(@ids::bigint[]);

-- name: MoveBatchedQueueItems :many
WITH input AS (
    SELECT
        UNNEST(@ids::bigint[]) AS id
), moved_items AS (
    DELETE FROM v1_batched_queue_item
    WHERE id = ANY(SELECT id FROM input)
    RETURNING
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        batch_key
)
INSERT INTO v1_queue_item (
    tenant_id,
    queue,
    task_id,
    task_inserted_at,
    external_id,
    action_id,
    step_id,
    workflow_id,
    workflow_run_id,
    schedule_timeout_at,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    retry_count,
    batch_key
)
SELECT
    tenant_id,
    queue,
    task_id,
    task_inserted_at,
    external_id,
    action_id,
    step_id,
    workflow_id,
    workflow_run_id,
    schedule_timeout_at,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    retry_count,
    batch_key
FROM moved_items
RETURNING id, tenant_id, task_id, task_inserted_at, retry_count;
