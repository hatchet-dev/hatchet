package sqlcv1

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
	WorkerId                  *uuid.UUID         `json:"workerId"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Tenantid                  uuid.UUID          `json:"tenantid"`
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
			OR parent_task_external_id = $8::UUID
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
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
`

type CountWorkflowRunsParams struct {
	ParentTaskExternalId      *uuid.UUID         `json:"parentTaskExternalId"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Tenantid                  uuid.UUID          `json:"tenantid"`
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
		arg.ParentTaskExternalId,
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
	ParentTaskExternalId      *uuid.UUID         `json:"parentTaskExternalId"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Listworkflowrunsoffset    int32              `json:"listworkflowrunsoffset"`
	Listworkflowrunslimit     int32              `json:"listworkflowrunslimit"`
	Tenantid                  uuid.UUID          `json:"tenantid"`
}

type FetchWorkflowRunIdsRow struct {
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`
	Kind       V1RunKind          `json:"kind"`
	ID         int64              `json:"id"`
	ExternalID uuid.UUID          `json:"external_id"`
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
	WorkerId                  *uuid.UUID         `json:"workerId"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Taskoffset                int32              `json:"taskoffset"`
	Tasklimit                 int32              `json:"tasklimit"`
	Tenantid                  uuid.UUID          `json:"tenantid"`
}

type ListTasksOlapRow struct {
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`
	ID         int64              `json:"id"`
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

const bulkCreateEventsOLAP = `-- name: BulkCreateEventsOLAP :many
WITH to_insert AS (
    SELECT
        UNNEST($1::UUID[]) AS tenant_id,
        UNNEST($2::UUID[]) AS external_id,
        UNNEST($3::TIMESTAMPTZ[]) AS seen_at,
        UNNEST($4::TEXT[]) AS key,
        UNNEST($5::JSONB[]) AS payload,
        UNNEST($6::JSONB[]) AS additional_metadata,
        -- Scopes are nullable
        UNNEST($7::TEXT[]) AS scope,
        -- Webhook names are nullable
        UNNEST($8::TEXT[]) AS triggering_webhook_name

)
INSERT INTO v1_events_olap (
    tenant_id,
    external_id,
    seen_at,
    key,
    payload,
    additional_metadata,
    scope,
	triggering_webhook_name
)
SELECT tenant_id, external_id, seen_at, key, payload, additional_metadata, scope, triggering_webhook_name
FROM to_insert
RETURNING tenant_id, id, external_id, seen_at, key, payload, additional_metadata, scope, triggering_webhook_name
`

type BulkCreateEventsOLAPParams struct {
	Tenantids              []uuid.UUID          `json:"tenantids"`
	Externalids            []uuid.UUID          `json:"externalids"`
	Seenats                []pgtype.Timestamptz `json:"seenats"`
	Keys                   []string             `json:"keys"`
	Payloads               [][]byte             `json:"payloads"`
	Additionalmetadatas    [][]byte             `json:"additionalmetadatas"`
	Scopes                 []pgtype.Text        `json:"scopes"`
	TriggeringWebhookNames []pgtype.Text        `json:"triggeringWebhookName"`
}

func (q *Queries) BulkCreateEventsOLAP(ctx context.Context, db DBTX, arg BulkCreateEventsOLAPParams) ([]*V1EventsOlap, error) {
	rows, err := db.Query(ctx, bulkCreateEventsOLAP,
		arg.Tenantids,
		arg.Externalids,
		arg.Seenats,
		arg.Keys,
		arg.Payloads,
		arg.Additionalmetadatas,
		arg.Scopes,
		arg.TriggeringWebhookNames,
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
			&i.TriggeringWebhookName,
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

type CutoverOLAPPayloadToInsert struct {
	InsertedAt          pgtype.Timestamptz
	ExternalLocationKey string
	Location            V1PayloadLocationOlap
	InlineContent       []byte
	TenantID            uuid.UUID
	ExternalID          uuid.UUID
}

type InsertCutOverOLAPPayloadsIntoTempTableRow struct {
	InsertedAt pgtype.Timestamptz
	TenantId   uuid.UUID
	ExternalId uuid.UUID
}

func InsertCutOverOLAPPayloadsIntoTempTable(ctx context.Context, tx DBTX, tableName string, payloads []CutoverOLAPPayloadToInsert) (*InsertCutOverOLAPPayloadsIntoTempTableRow, error) {
	tenantIds := make([]uuid.UUID, 0, len(payloads))
	insertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	externalIds := make([]uuid.UUID, 0, len(payloads))
	locations := make([]string, 0, len(payloads))
	externalLocationKeys := make([]string, 0, len(payloads))
	inlineContents := make([][]byte, 0, len(payloads))

	for _, payload := range payloads {
		externalIds = append(externalIds, payload.ExternalID)
		tenantIds = append(tenantIds, payload.TenantID)
		insertedAts = append(insertedAts, payload.InsertedAt)
		locations = append(locations, string(payload.Location))
		externalLocationKeys = append(externalLocationKeys, string(payload.ExternalLocationKey))
		inlineContents = append(inlineContents, payload.InlineContent)
	}

	row := tx.QueryRow(
		ctx,
		fmt.Sprintf(
			// we unfortunately need to use `INSERT INTO` instead of `COPY` here
			// because we can't have conflict resolution with `COPY`.
			`
				WITH inputs AS (
					SELECT
						UNNEST($1::UUID[]) AS tenant_id,
						UNNEST($2::TIMESTAMPTZ[]) AS inserted_at,
						UNNEST($3::UUID[]) AS external_id,
						UNNEST($4::TEXT[]) AS location,
						UNNEST($5::TEXT[]) AS external_location_key,
						UNNEST($6::JSONB[]) AS inline_content
				), inserts AS (
					INSERT INTO %s (tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at)
					SELECT
						tenant_id,
						external_id,
						location::v1_payload_location_olap,
						external_location_key,
						inline_content,
						inserted_at,
						NOW()
					FROM inputs
					ORDER BY tenant_id, external_id, inserted_at
					ON CONFLICT(tenant_id, external_id, inserted_at) DO NOTHING
				)

				SELECT tenant_id, external_id, inserted_at
				FROM inputs
				ORDER BY tenant_id DESC, external_id DESC, inserted_at DESC
				LIMIT 1
				`,
			tableName,
		),
		tenantIds,
		insertedAts,
		externalIds,
		locations,
		externalLocationKeys,
		inlineContents,
	)

	var insertRow InsertCutOverOLAPPayloadsIntoTempTableRow
	err := row.Scan(&insertRow.TenantId, &insertRow.ExternalId, &insertRow.InsertedAt)

	return &insertRow, err
}

const findV1OLAPPayloadPartitionsBeforeDate = `-- name: findV1OLAPPayloadPartitionsBeforeDate :many
WITH partitions AS (
    SELECT
        child.relname::text AS partition_name,
        SUBSTRING(pg_get_expr(child.relpartbound, child.oid) FROM 'FROM \(''([^'']+)')::DATE AS lower_bound,
        SUBSTRING(pg_get_expr(child.relpartbound, child.oid) FROM 'TO \(''([^'']+)')::DATE AS upper_bound
    FROM pg_inherits
    JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
    JOIN pg_class child ON pg_inherits.inhrelid = child.oid
    WHERE parent.relname = 'v1_payloads_olap'
    ORDER BY child.relname DESC
	LIMIT $1::INTEGER
)

SELECT partition_name, lower_bound AS partition_date
FROM partitions
WHERE lower_bound <= $2::DATE
`

type FindV1OLAPPayloadPartitionsBeforeDateRow struct {
	PartitionName string      `json:"partition_name"`
	PartitionDate pgtype.Date `json:"partition_date"`
}

func (q *Queries) FindV1OLAPPayloadPartitionsBeforeDate(ctx context.Context, db DBTX, maxPartitionsToProcess int32, date pgtype.Date) ([]*FindV1OLAPPayloadPartitionsBeforeDateRow, error) {
	rows, err := db.Query(ctx, findV1OLAPPayloadPartitionsBeforeDate,
		maxPartitionsToProcess,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*FindV1OLAPPayloadPartitionsBeforeDateRow
	for rows.Next() {
		var i FindV1OLAPPayloadPartitionsBeforeDateRow
		if err := rows.Scan(
			&i.PartitionName,
			&i.PartitionDate,
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
