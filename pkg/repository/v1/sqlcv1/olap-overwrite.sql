-- NOTE: this file doesn't typically get generated, since we need to overwrite the queries
-- to filter by statuses

-- name: ListTasksOlap :many
SELECT
    id,
    inserted_at
FROM
    v1_tasks_olap
WHERE
    tenant_id = @tenantId::uuid
    AND inserted_at >= @since::timestamptz
    AND readable_status = ANY(@statuses::v1_readable_status_olap[])
    AND (
        sqlc.narg('until')::timestamptz IS NULL
        OR inserted_at <= sqlc.narg('until')::timestamptz
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
    AND (
        sqlc.narg('triggeringEventId')::UUID IS NULL
        OR (id, inserted_at) IN (
            SELECT (r.id, r.inserted_at)
            FROM v1_events_olap e
            JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
            JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
            WHERE
                e.tenant_id = @tenantId::uuid
                AND e.id = sqlc.narg('triggeringEventId')::UUID
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
        v1_tasks_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND inserted_at >= @since::timestamptz
        AND readable_status = ANY(@statuses::v1_readable_status_olap[])
        AND (
            sqlc.narg('until')::timestamptz IS NULL
            OR inserted_at <= sqlc.narg('until')::timestamptz
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
		AND (
			sqlc.narg('triggeringEventId')::UUID IS NULL
			OR (id, inserted_at) IN (
				SELECT (r.id, r.inserted_at)
				FROM v1_events_olap e
				JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
				JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
				WHERE
					e.tenant_id = @tenantId::uuid
					AND e.id = sqlc.narg('triggeringEventId')::UUID
			)
		)
    ORDER BY
        inserted_at DESC
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
;

-- name: FetchWorkflowRunIds :many
SELECT id, inserted_at, kind, external_id
FROM v1_runs_olap
WHERE
    tenant_id = @tenantId::uuid
    AND readable_status = ANY(@statuses::v1_readable_status_olap[])
    AND (
        sqlc.narg('workflowIds')::uuid[] IS NULL
        OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
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
    AND (
        sqlc.narg('parentTaskExternalId')::UUID IS NULL
        OR parent_task_external_id = sqlc.narg('parentTaskExternalId')::UUID
    )
    AND (
        sqlc.narg('triggeringEventId')::UUID IS NULL
        OR (id, inserted_at) IN (
            SELECT (r.id, r.inserted_at)
            FROM v1_events_olap e
            JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
            JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
            WHERE
                e.tenant_id = @tenantId::uuid
                AND e.id = sqlc.narg('triggeringEventId')::UUID
        )
    )
ORDER BY inserted_at DESC, id DESC
LIMIT @listWorkflowRunsLimit::integer
OFFSET @listWorkflowRunsOffset::integer
;

-- name: CountWorkflowRuns :one
WITH filtered AS (
    SELECT *
    FROM v1_runs_olap
    WHERE
        tenant_id = @tenantId::uuid
        AND readable_status = ANY(@statuses::v1_readable_status_olap[])
        AND (
            sqlc.narg('workflowIds')::uuid[] IS NULL
            OR workflow_id = ANY(sqlc.narg('workflowIds')::uuid[])
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
		AND (
			sqlc.narg('triggeringEventId')::UUID IS NULL
			OR (id, inserted_at) IN (
				SELECT (r.id, r.inserted_at)
				FROM v1_events_olap e
				JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
				JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
				WHERE
					e.tenant_id = @tenantId::uuid
					AND e.id = sqlc.narg('triggeringEventId')::UUID
			)
		)
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
;
