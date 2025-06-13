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
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
;

-- name: BulkCreateEvents :many
WITH to_insert AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@seenAts::TIMESTAMPTZ[]) AS seen_at,
        UNNEST(@keys::TEXT[]) AS key,
        UNNEST(@payloads::JSONB[]) AS payload,
        UNNEST(@additionalMetadatas::JSONB[]) AS additional_metadata,
        -- Scopes are nullable
        UNNEST(@scopes::TEXT[]) AS scope
)
INSERT INTO v1_events_olap (
    tenant_id,
    external_id,
    seen_at,
    key,
    payload,
    additional_metadata,
    scope
)
SELECT *
FROM to_insert
RETURNING *
;


-- name: ListEvents :many
WITH included_events AS (
    SELECT
        e.*,
        JSON_AGG(JSON_BUILD_OBJECT('run_external_id', r.external_id, 'filter_id', etr.filter_id)) FILTER (WHERE r.external_id IS NOT NULL)::JSONB AS triggered_runs
    FROM v1_event_lookup_table_olap elt
    JOIN v1_events_olap e ON (elt.tenant_id, elt.event_id, elt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
    LEFT JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
    LEFT JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
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
            r.workflow_id = ANY(sqlc.narg('workflowIds')::UUID[])
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
            sqlc.narg('statuses')::v1_readable_status_olap[] IS NULL OR
            r.readable_status = ANY(sqlc.narg('statuses')::v1_readable_status_olap[])
        )
        AND (
            sqlc.narg('scopes')::TEXT[] IS NULL OR
            e.scope = ANY(sqlc.narg('scopes')::TEXT[])
        )
    GROUP BY
        e.tenant_id,
        e.id,
        e.external_id,
        e.seen_at,
        e.key,
        e.payload,
        e.additional_metadata,
        e.scope
    ORDER BY e.seen_at DESC, e.id
    OFFSET
        COALESCE(sqlc.narg('offset')::BIGINT, 0)
    LIMIT
        COALESCE(sqlc.narg('limit')::BIGINT, 50)
), status_counts AS (
    SELECT
        e.tenant_id,
        e.id,
        e.seen_at,
        COUNT(*) FILTER (WHERE r.readable_status = 'QUEUED') AS queued_count,
        COUNT(*) FILTER (WHERE r.readable_status = 'RUNNING') AS running_count,
        COUNT(*) FILTER (WHERE r.readable_status = 'COMPLETED') AS completed_count,
        COUNT(*) FILTER (WHERE r.readable_status = 'CANCELLED') AS cancelled_count,
        COUNT(*) FILTER (WHERE r.readable_status = 'FAILED') AS failed_count
    FROM
        included_events e
    LEFT JOIN
        v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
    LEFT JOIN
        v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
    GROUP BY
        e.tenant_id, e.id, e.seen_at
)

SELECT
    e.tenant_id,
    e.id AS event_id,
    e.external_id AS event_external_id,
    e.seen_at AS event_seen_at,
    e.key AS event_key,
    e.payload AS event_payload,
    e.additional_metadata AS event_additional_metadata,
    e.scope AS event_scope,
    sc.queued_count,
    sc.running_count,
    sc.completed_count,
    sc.cancelled_count,
    sc.failed_count,
    e.triggered_runs
FROM
    included_events e
LEFT JOIN
    status_counts sc ON (e.tenant_id, e.id, e.seen_at) = (sc.tenant_id, sc.id, sc.seen_at)
ORDER BY e.seen_at DESC
;

-- name: CountEvents :one
WITH included_events AS (
    SELECT DISTINCT e.*
    FROM v1_event_lookup_table_olap elt
    JOIN v1_events_olap e ON (elt.tenant_id, elt.event_id, elt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
    LEFT JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
    LEFT JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
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
            r.workflow_id = ANY(sqlc.narg('workflowIds')::UUID[])
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
            sqlc.narg('statuses')::v1_readable_status_olap[] IS NULL OR
            r.readable_status = ANY(sqlc.narg('statuses')::v1_readable_status_olap[])
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
