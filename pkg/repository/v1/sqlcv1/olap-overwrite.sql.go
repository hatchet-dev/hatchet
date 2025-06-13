package sqlcv1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const countTasks = `-- name: CountTasks :one
WITH filtered AS (
    SELECT
        tenant_id, id, inserted_at, external_id, queue, action_id, step_id, workflow_id, schedule_timeout, step_timeout, priority, sticky, desired_worker_id, display_name, input, additional_metadata, readable_status, latest_retry_count, latest_worker_id, dag_id, dag_inserted_at
    FROM
        v1_tasks_olap
    WHERE
        tenant_id = $1::uuid
        AND inserted_at >= $2::timestamptz
        AND readable_status = ANY($3::v1_readable_status_olap[])
        AND (
            $4::timestamptz IS NULL
            OR inserted_at <= $4::timestamptz
        )
        AND (
            $5::uuid[] IS NULL OR workflow_id = ANY($5::uuid[])
        )
        AND (
            $6::uuid IS NULL OR latest_worker_id = $6::uuid
        )
        AND (
            $7::text[] IS NULL
            OR $8::text[] IS NULL
            OR EXISTS (
                SELECT 1 FROM jsonb_each_text(additional_metadata) kv
                JOIN LATERAL (
                    SELECT unnest($7::text[]) AS k,
                        unnest($8::text[]) AS v
                ) AS u ON kv.key = u.k AND kv.value = u.v
            )
        )
		AND (
			$9::UUID IS NULL
			OR (id, inserted_at) IN (
                SELECT etr.run_id, etr.run_inserted_at
                FROM v1_event_lookup_table_olap lt
                JOIN v1_events_olap e ON (lt.tenant_id, lt.event_id, lt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
                JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
    			WHERE
					lt.tenant_id = $1::uuid
					AND lt.external_id = $9::UUID
            )
		)
    ORDER BY
        inserted_at DESC
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
`

type CountTasksParams struct {
	Tenantid                  pgtype.UUID        `json:"tenantid"`
	Since                     pgtype.Timestamptz `json:"since"`
	Statuses                  []string           `json:"statuses"`
	Until                     pgtype.Timestamptz `json:"until"`
	WorkflowIds               []pgtype.UUID      `json:"workflowIds"`
	WorkerId                  pgtype.UUID        `json:"workerId"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	TriggeringEventExternalId pgtype.UUID        `json:"triggeringEventExternalId"`
}

func (q *Queries) CountTasks(ctx context.Context, db DBTX, arg CountTasksParams) (int64, error) {
	row := db.QueryRow(ctx, countTasks,
		arg.Tenantid,
		arg.Since,
		arg.Statuses,
		arg.Until,
		arg.WorkflowIds,
		arg.WorkerId,
		arg.Keys,
		arg.Values,
		arg.TriggeringEventExternalId,
	)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countWorkflowRuns = `-- name: CountWorkflowRuns :one
WITH filtered AS (
    SELECT tenant_id, id, inserted_at, external_id, readable_status, kind, workflow_id, additional_metadata
    FROM v1_runs_olap
    WHERE
        tenant_id = $1::uuid
        AND readable_status = ANY($2::v1_readable_status_olap[])
        AND (
            $3::uuid[] IS NULL
            OR workflow_id = ANY($3::uuid[])
        )
        AND inserted_at >= $4::timestamptz
        AND (
            $5::timestamptz IS NULL
            OR inserted_at <= $5::timestamptz
        )
        AND (
            $6::text[] IS NULL
            OR $7::text[] IS NULL
            OR EXISTS (
                SELECT 1 FROM jsonb_each_text(additional_metadata) kv
                JOIN LATERAL (
                    SELECT unnest($6::text[]) AS k,
                        unnest($7::text[]) AS v
                ) AS u ON kv.key = u.k AND kv.value = u.v
            )
        )
		AND (
			$8::UUID IS NULL
			OR (id, inserted_at) IN (
                SELECT etr.run_id, etr.run_inserted_at
                FROM v1_event_lookup_table_olap lt
                JOIN v1_events_olap e ON (lt.tenant_id, lt.event_id, lt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
                JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
    			WHERE
					lt.tenant_id = $1::uuid
					AND lt.external_id = $8::UUID
            )
		)
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
`

type CountWorkflowRunsParams struct {
	Tenantid                  pgtype.UUID        `json:"tenantid"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []pgtype.UUID      `json:"workflowIds"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	TriggeringEventExternalId pgtype.UUID        `json:"triggeringEventExternalId"`
}

func (q *Queries) CountWorkflowRuns(ctx context.Context, db DBTX, arg CountWorkflowRunsParams) (int64, error) {
	row := db.QueryRow(ctx, countWorkflowRuns,
		arg.Tenantid,
		arg.Statuses,
		arg.WorkflowIds,
		arg.Since,
		arg.Until,
		arg.Keys,
		arg.Values,
		arg.TriggeringEventExternalId,
	)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const fetchWorkflowRunIds = `-- name: FetchWorkflowRunIds :many
SELECT id, inserted_at, kind, external_id
FROM v1_runs_olap
WHERE
    tenant_id = $1::uuid
    AND readable_status = ANY($2::v1_readable_status_olap[])
    AND (
        $3::uuid[] IS NULL
        OR workflow_id = ANY($3::uuid[])
    )
    AND inserted_at >= $4::timestamptz
    AND (
        $5::timestamptz IS NULL
        OR inserted_at <= $5::timestamptz
    )
    AND (
        $6::text[] IS NULL
        OR $7::text[] IS NULL
        OR EXISTS (
            SELECT 1 FROM jsonb_each_text(additional_metadata) kv
            JOIN LATERAL (
                SELECT unnest($6::text[]) AS k,
                    unnest($7::text[]) AS v
            ) AS u ON kv.key = u.k AND kv.value = u.v
        )
    )
    AND (
        $10::UUID IS NULL
        OR parent_task_external_id = $10::UUID
    )
    AND (
        $11::UUID IS NULL
		OR (id, inserted_at) IN (
			SELECT etr.run_id, etr.run_inserted_at
			FROM v1_event_lookup_table_olap lt
			JOIN v1_events_olap e ON (lt.tenant_id, lt.event_id, lt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
			JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
			WHERE
				lt.tenant_id = $1::uuid
				AND lt.external_id = $11::UUID
		)
    )
ORDER BY inserted_at DESC, id DESC
LIMIT $9::integer
OFFSET $8::integer
`

type FetchWorkflowRunIdsParams struct {
	Tenantid                  pgtype.UUID        `json:"tenantid"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []pgtype.UUID      `json:"workflowIds"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Listworkflowrunsoffset    int32              `json:"listworkflowrunsoffset"`
	Listworkflowrunslimit     int32              `json:"listworkflowrunslimit"`
	ParentTaskExternalId      pgtype.UUID        `json:"parentTaskExternalId"`
	TriggeringEventExternalId pgtype.UUID        `json:"triggeringEventExternalId"`
}

type FetchWorkflowRunIdsRow struct {
	ID         int64              `json:"id"`
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`
	Kind       V1RunKind          `json:"kind"`
	ExternalID pgtype.UUID        `json:"external_id"`
}

func (q *Queries) FetchWorkflowRunIds(ctx context.Context, db DBTX, arg FetchWorkflowRunIdsParams) ([]*FetchWorkflowRunIdsRow, error) {
	rows, err := db.Query(ctx, fetchWorkflowRunIds,
		arg.Tenantid,
		arg.Statuses,
		arg.WorkflowIds,
		arg.Since,
		arg.Until,
		arg.Keys,
		arg.Values,
		arg.Listworkflowrunsoffset,
		arg.Listworkflowrunslimit,
		arg.ParentTaskExternalId,
		arg.TriggeringEventExternalId,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*FetchWorkflowRunIdsRow
	for rows.Next() {
		var i FetchWorkflowRunIdsRow
		if err := rows.Scan(
			&i.ID,
			&i.InsertedAt,
			&i.Kind,
			&i.ExternalID,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTasksOlap = `-- name: ListTasksOlap :many
SELECT
    id,
    inserted_at
FROM
    v1_tasks_olap
WHERE
    tenant_id = $1::uuid
    AND inserted_at >= $2::timestamptz
    AND readable_status = ANY($3::v1_readable_status_olap[])
    AND (
        $4::timestamptz IS NULL
        OR inserted_at <= $4::timestamptz
    )
    AND (
        $5::uuid[] IS NULL OR workflow_id = ANY($5::uuid[])
    )
    AND (
        $6::uuid IS NULL OR latest_worker_id = $6::uuid
    )
    AND (
        $7::text[] IS NULL
        OR $8::text[] IS NULL
        OR EXISTS (
            SELECT 1 FROM jsonb_each_text(additional_metadata) kv
            JOIN LATERAL (
                SELECT unnest($7::text[]) AS k,
                    unnest($8::text[]) AS v
            ) AS u ON kv.key = u.k AND kv.value = u.v
        )
    )
    AND (
        $11::UUID IS NULL
		OR (id, inserted_at) IN (
			SELECT etr.run_id, etr.run_inserted_at
			FROM v1_event_lookup_table_olap lt
			JOIN v1_events_olap e ON (lt.tenant_id, lt.event_id, lt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
			JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
			WHERE
				lt.tenant_id = $1::uuid
				AND lt.external_id = $11::UUID
		)
    )
ORDER BY
    inserted_at DESC
LIMIT $10::integer
OFFSET $9::integer
`

type ListTasksOlapParams struct {
	Tenantid                  pgtype.UUID        `json:"tenantid"`
	Since                     pgtype.Timestamptz `json:"since"`
	Statuses                  []string           `json:"statuses"`
	Until                     pgtype.Timestamptz `json:"until"`
	WorkflowIds               []pgtype.UUID      `json:"workflowIds"`
	WorkerId                  pgtype.UUID        `json:"workerId"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Taskoffset                int32              `json:"taskoffset"`
	Tasklimit                 int32              `json:"tasklimit"`
	TriggeringEventExternalId pgtype.UUID        `json:"triggeringEventExternalId"`
}

type ListTasksOlapRow struct {
	ID         int64              `json:"id"`
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`
}

func (q *Queries) ListTasksOlap(ctx context.Context, db DBTX, arg ListTasksOlapParams) ([]*ListTasksOlapRow, error) {
	rows, err := db.Query(ctx, listTasksOlap,
		arg.Tenantid,
		arg.Since,
		arg.Statuses,
		arg.Until,
		arg.WorkflowIds,
		arg.WorkerId,
		arg.Keys,
		arg.Values,
		arg.Taskoffset,
		arg.Tasklimit,
		arg.TriggeringEventExternalId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListTasksOlapRow
	for rows.Next() {
		var i ListTasksOlapRow
		if err := rows.Scan(&i.ID, &i.InsertedAt); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const bulkCreateEvents = `-- name: BulkCreateEvents :many
WITH to_insert AS (
    SELECT
        UNNEST($1::UUID[]) AS tenant_id,
        UNNEST($2::UUID[]) AS external_id,
        UNNEST($3::TIMESTAMPTZ[]) AS seen_at,
        UNNEST($4::TEXT[]) AS key,
        UNNEST($5::JSONB[]) AS payload,
        UNNEST($6::JSONB[]) AS additional_metadata,
        -- Scopes are nullable
        UNNEST($7::TEXT[]) AS scope
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
SELECT tenant_id, external_id, seen_at, key, payload, additional_metadata, scope
FROM to_insert
RETURNING tenant_id, id, external_id, seen_at, key, payload, additional_metadata, scope
`

type BulkCreateEventsParams struct {
	Tenantids           []pgtype.UUID        `json:"tenantids"`
	Externalids         []pgtype.UUID        `json:"externalids"`
	Seenats             []pgtype.Timestamptz `json:"seenats"`
	Keys                []string             `json:"keys"`
	Payloads            [][]byte             `json:"payloads"`
	Additionalmetadatas [][]byte             `json:"additionalmetadatas"`
	Scopes              []*string            `json:"scopes"`
}

func (q *Queries) BulkCreateEvents(ctx context.Context, db DBTX, arg BulkCreateEventsParams) ([]*V1EventsOlap, error) {
	rows, err := db.Query(ctx, bulkCreateEvents,
		arg.Tenantids,
		arg.Externalids,
		arg.Seenats,
		arg.Keys,
		arg.Payloads,
		arg.Additionalmetadatas,
		arg.Scopes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*V1EventsOlap
	for rows.Next() {
		var i V1EventsOlap
		if err := rows.Scan(
			&i.TenantID,
			&i.ID,
			&i.ExternalID,
			&i.SeenAt,
			&i.Key,
			&i.Payload,
			&i.AdditionalMetadata,
			&i.Scope,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const countEvents = `-- name: CountEvents :one
WITH included_events AS (
    SELECT DISTINCT e.tenant_id, e.id, e.external_id, e.seen_at, e.key, e.payload, e.additional_metadata, e.scope
    FROM v1_event_lookup_table_olap elt
    JOIN v1_events_olap e ON (elt.tenant_id, elt.event_id, elt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
    LEFT JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
    LEFT JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
    WHERE
        e.tenant_id = $1
        AND (
            $2::TEXT[] IS NULL OR
            "key" = ANY($2::TEXT[])
        )
        AND e.seen_at >= $3::TIMESTAMPTZ
        AND (
            $4::TIMESTAMPTZ IS NULL OR
            e.seen_at <= $4::TIMESTAMPTZ
        )
        AND (
            $5::UUID[] IS NULL OR
            r.workflow_id = ANY($5::UUID[])
        )
        AND (
            $6::UUID[] IS NULL OR
            elt.external_id = ANY($6::UUID[])
        )
        AND (
            $7::JSONB IS NULL OR
            e.additional_metadata @> $7::JSONB
        )
        AND (
            $8::v1_readable_status_olap[] IS NULL OR
            r.readable_status = ANY($8::v1_readable_status_olap[])
        )
        AND (
            $9::TEXT[] IS NULL OR
            e.scope = ANY($9::TEXT[])
        )
        ORDER BY e.seen_at DESC, e.id
    LIMIT 20000
)

SELECT COUNT(*)
FROM included_events e
`

type CountEventsParams struct {
	Tenantid           pgtype.UUID        `json:"tenantid"`
	Keys               []string           `json:"keys"`
	Since              pgtype.Timestamptz `json:"since"`
	Until              pgtype.Timestamptz `json:"until"`
	WorkflowIds        []pgtype.UUID      `json:"workflowIds"`
	EventIds           []pgtype.UUID      `json:"eventIds"`
	AdditionalMetadata []byte             `json:"additionalMetadata"`
	Statuses           []string           `json:"statuses"`
	Scopes             []string           `json:"scopes"`
}

func (q *Queries) CountEvents(ctx context.Context, db DBTX, arg CountEventsParams) (int64, error) {
	row := db.QueryRow(ctx, countEvents,
		arg.Tenantid,
		arg.Keys,
		arg.Since,
		arg.Until,
		arg.WorkflowIds,
		arg.EventIds,
		arg.AdditionalMetadata,
		arg.Statuses,
		arg.Scopes,
	)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const listEvents = `-- name: ListEvents :many
WITH included_events AS (
    SELECT
		e.tenant_id,
		e.id,
		e.external_id,
		e.seen_at,
		e.key,
		e.payload,
		e.additional_metadata,
		e.scope,
		ARRAY_AGG(r.external_id) FILTER (WHERE r.external_id IS NOT NULL)::UUID[] AS triggered_run_external_ids
    FROM v1_event_lookup_table_olap elt
    JOIN v1_events_olap e ON (elt.tenant_id, elt.event_id, elt.event_seen_at) = (e.tenant_id, e.id, e.seen_at)
    LEFT JOIN v1_event_to_run_olap etr ON (e.id, e.seen_at) = (etr.event_id, etr.event_seen_at)
    LEFT JOIN v1_runs_olap r ON (etr.run_id, etr.run_inserted_at) = (r.id, r.inserted_at)
    WHERE
        e.tenant_id = $1
        AND (
            $2::TEXT[] IS NULL OR
            "key" = ANY($2::TEXT[])
        )
        AND e.seen_at >= $3::TIMESTAMPTZ
        AND (
            $4::TIMESTAMPTZ IS NULL OR
            e.seen_at <= $4::TIMESTAMPTZ
        )
        AND (
            $5::UUID[] IS NULL OR
            r.workflow_id = ANY($5::UUID[])
        )
        AND (
            $6::UUID[] IS NULL OR
            elt.external_id = ANY($6::UUID[])
        )
        AND (
            $7::JSONB IS NULL OR
            e.additional_metadata @> $7::JSONB
        )
        AND (
            $8::v1_readable_status_olap[] IS NULL OR
            r.readable_status = ANY($8::v1_readable_status_olap[])
        )
        AND (
            $9::TEXT[] IS NULL OR
            e.scope = ANY($9::TEXT[])
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
        COALESCE($10::BIGINT, 0)
    LIMIT
        COALESCE($11::BIGINT, 50)
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
	e.triggered_run_external_ids
FROM
    included_events e
LEFT JOIN
    status_counts sc ON (e.tenant_id, e.id, e.seen_at) = (sc.tenant_id, sc.id, sc.seen_at)
ORDER BY e.seen_at DESC
`

type ListEventsParams struct {
	Tenantid           pgtype.UUID        `json:"tenantid"`
	Keys               []string           `json:"keys"`
	Since              pgtype.Timestamptz `json:"since"`
	Until              pgtype.Timestamptz `json:"until"`
	WorkflowIds        []pgtype.UUID      `json:"workflowIds"`
	EventIds           []pgtype.UUID      `json:"eventIds"`
	AdditionalMetadata []byte             `json:"additionalMetadata"`
	Statuses           []string           `json:"statuses"`
	Scopes             []string           `json:"scopes"`
	Offset             pgtype.Int8        `json:"offset"`
	Limit              pgtype.Int8        `json:"limit"`
}

type ListEventsRow struct {
	TenantID                pgtype.UUID        `json:"tenant_id"`
	EventID                 int64              `json:"event_id"`
	EventExternalID         pgtype.UUID        `json:"event_external_id"`
	EventSeenAt             pgtype.Timestamptz `json:"event_seen_at"`
	EventKey                string             `json:"event_key"`
	EventPayload            []byte             `json:"event_payload"`
	EventAdditionalMetadata []byte             `json:"event_additional_metadata"`
	EventScope              *string            `json:"event_scope,omitempty"`
	QueuedCount             pgtype.Int8        `json:"queued_count"`
	RunningCount            pgtype.Int8        `json:"running_count"`
	CompletedCount          pgtype.Int8        `json:"completed_count"`
	CancelledCount          pgtype.Int8        `json:"cancelled_count"`
	FailedCount             pgtype.Int8        `json:"failed_count"`
	TriggeredRunExternalIds []pgtype.UUID      `json:"triggered_run_external_ids"`
}

func (q *Queries) ListEvents(ctx context.Context, db DBTX, arg ListEventsParams) ([]*ListEventsRow, error) {
	rows, err := db.Query(ctx, listEvents,
		arg.Tenantid,
		arg.Keys,
		arg.Since,
		arg.Until,
		arg.WorkflowIds,
		arg.EventIds,
		arg.AdditionalMetadata,
		arg.Statuses,
		arg.Scopes,
		arg.Offset,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListEventsRow
	for rows.Next() {
		var i ListEventsRow
		if err := rows.Scan(
			&i.TenantID,
			&i.EventID,
			&i.EventExternalID,
			&i.EventSeenAt,
			&i.EventKey,
			&i.EventPayload,
			&i.EventAdditionalMetadata,
			&i.EventScope,
			&i.QueuedCount,
			&i.RunningCount,
			&i.CompletedCount,
			&i.CancelledCount,
			&i.FailedCount,
			&i.TriggeredRunExternalIds,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
