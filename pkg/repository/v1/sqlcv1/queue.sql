-- name: UpsertQueues :exec
WITH ordered_names AS (
    SELECT unnest(@names::text[]) AS name
    ORDER BY name
)
INSERT INTO
    v1_queue (
        tenant_id,
        name,
        last_active
    )
SELECT
    $1,
    name,
    NOW()
FROM ordered_names
ON CONFLICT (tenant_id, name) DO UPDATE
SET
    last_active = NOW();

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
        COUNT(task_id) AS "filledSlots"
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
        worker_id
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS id,
                unnest(@workerIds::uuid[]) AS worker_id
        ) AS subquery
    ORDER BY id
), updated_tasks AS (
    SELECT
        t.id,
        t.inserted_at,
        t.retry_count,
        input.worker_id,
        t.tenant_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(t.step_timeout) AS timeout_at
    FROM
        input
    JOIN
        v1_task t ON t.id = input.id
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
