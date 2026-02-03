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
WITH worker_capacities AS (
    SELECT
        worker_id,
        max_units
    FROM
        v1_worker_slot_capacity
    WHERE
        tenant_id = @tenantId::uuid
        AND worker_id = ANY(@workerIds::uuid[])
        AND slot_type = @slotType::text
), worker_used_slots AS (
    SELECT
        worker_id,
        SUM(units) AS used_units
    FROM
        v1_task_runtime_slot
    WHERE
        tenant_id = @tenantId::uuid
        AND worker_id = ANY(@workerIds::uuid[])
        AND slot_type = @slotType::text
    GROUP BY
        worker_id
)
SELECT
    wc.worker_id AS "id",
    wc.max_units - COALESCE(wus.used_units, 0) AS "availableSlots"
FROM
    worker_capacities wc
LEFT JOIN
    worker_used_slots wus ON wc.worker_id = wus.worker_id;

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

-- name: ListQueueItemsForTasks :many
WITH input AS (
    SELECT
        UNNEST(@taskIds::bigint[]) AS task_id,
        UNNEST(@taskInsertedAts::timestamptz[]) AS task_inserted_at,
        UNNEST(@retryCounts::integer[]) AS retry_count
)
SELECT
    qi.*
FROM
    v1_queue_item qi
WHERE
    (qi.task_id, qi.task_inserted_at, qi.retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    AND qi.tenant_id = @tenantId::uuid;

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
        t.step_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(t.step_timeout) AS timeout_at
    FROM
        v1_task t
    JOIN
        input i ON (t.id, t.inserted_at) = (i.id, i.inserted_at)
    WHERE
        t.inserted_at >= @minTaskInsertedAt::timestamptz
    ORDER BY t.id
), assigned_tasks AS (
    INSERT INTO v1_task_runtime (
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        tenant_id,
        timeout_at
    )
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        t.worker_id,
        @tenantId::uuid,
        t.timeout_at
    FROM
        updated_tasks t
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    -- only return the task ids that were successfully assigned
    RETURNING task_id, worker_id
), slot_requirements AS (
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        t.worker_id,
        t.tenant_id,
        COALESCE(req.slot_type, 'default'::text) AS slot_type,
        COALESCE(req.units, 1) AS units
    FROM
        updated_tasks t
    LEFT JOIN
        v1_step_slot_requirement req
        ON req.step_id = t.step_id AND req.tenant_id = t.tenant_id
), assigned_slots AS (
    INSERT INTO v1_task_runtime_slot (
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        slot_type,
        units
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        retry_count,
        worker_id,
        slot_type,
        units
    FROM
        slot_requirements
    ON CONFLICT (task_id, task_inserted_at, retry_count, slot_type) DO NOTHING
    RETURNING task_id
)
SELECT
    asr.task_id,
    asr.worker_id
FROM
    assigned_tasks asr;

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

-- name: GetStepSlotRequirements :many
SELECT
    step_id,
    slot_type,
    units
FROM
    v1_step_slot_requirement
WHERE
    step_id = ANY(@stepIds::uuid[]);

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
        retry_count
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
    retry_count
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
    retry_count
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
        retry_count
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
        retry_count
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
    retry_count
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
    retry_count
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

-- name: ReactivateInactiveQueuesWithItems :execresult
-- Reactivates queues that have been marked inactive (last_active > 1 day ago)
-- but still have pending items in v1_queue_item. This is a fallback mechanism
-- to ensure queues don't get stuck inactive while they have work to do.
WITH inactive_queues_with_items AS (
    SELECT q.tenant_id, q.name
    FROM v1_queue q
    WHERE q.last_active <= NOW() - INTERVAL '1 day'
      AND EXISTS (
        SELECT 1
        FROM v1_queue_item qi
        WHERE qi.tenant_id = q.tenant_id
          AND qi.queue = q.name
        LIMIT 1
      )
    FOR UPDATE SKIP LOCKED
)
UPDATE v1_queue q
SET last_active = NOW()
FROM inactive_queues_with_items i
WHERE q.tenant_id = i.tenant_id
  AND q.name = i.name;
