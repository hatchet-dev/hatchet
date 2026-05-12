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
	Tenantid                  uuid.UUID          `json:"tenantid"`
	Since                     pgtype.Timestamptz `json:"since"`
	Statuses                  []string           `json:"statuses"`
	Until                     pgtype.Timestamptz `json:"until"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	WorkerId                  *uuid.UUID         `json:"workerId"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
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
	Tenantid                  uuid.UUID          `json:"tenantid"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	ParentTaskExternalId      *uuid.UUID         `json:"parentTaskExternalId"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
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
	Tenantid                  uuid.UUID          `json:"tenantid"`
	Statuses                  []string           `json:"statuses"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	Since                     pgtype.Timestamptz `json:"since"`
	Until                     pgtype.Timestamptz `json:"until"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Listworkflowrunsoffset    int32              `json:"listworkflowrunsoffset"`
	Listworkflowrunslimit     int32              `json:"listworkflowrunslimit"`
	ParentTaskExternalId      *uuid.UUID         `json:"parentTaskExternalId"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
}

type FetchWorkflowRunIdsRow struct {
	ID         int64              `json:"id"`
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`
	Kind       V1RunKind          `json:"kind"`
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
	Tenantid                  uuid.UUID          `json:"tenantid"`
	Since                     pgtype.Timestamptz `json:"since"`
	Statuses                  []string           `json:"statuses"`
	Until                     pgtype.Timestamptz `json:"until"`
	WorkflowIds               []uuid.UUID        `json:"workflowIds"`
	WorkerId                  *uuid.UUID         `json:"workerId"`
	Keys                      []string           `json:"keys"`
	Values                    []string           `json:"values"`
	Taskoffset                int32              `json:"taskoffset"`
	Tasklimit                 int32              `json:"tasklimit"`
	TriggeringEventExternalId *uuid.UUID         `json:"triggeringEventExternalId"`
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
	TenantID            uuid.UUID
	InsertedAt          pgtype.Timestamptz
	ExternalID          uuid.UUID
	ExternalLocationKey string
	InlineContent       []byte
	Location            V1PayloadLocationOlap
}

type InsertCutOverOLAPPayloadsIntoTempTableRow struct {
	TenantId   uuid.UUID
	ExternalId uuid.UUID
	InsertedAt pgtype.Timestamptz
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

const createTasksOLAP = `-- name: CreateTasksOLAP :exec
WITH inputs AS (
    SELECT
        UNNEST($1::UUID[]) AS tenant_id,
        UNNEST($2::BIGINT[]) AS id,
        UNNEST($3::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST($4::TEXT[]) AS queue,
        UNNEST($5::TEXT[]) AS action_id,
        UNNEST($6::UUID[]) AS step_id,
        UNNEST($7::UUID[]) AS workflow_id,
        UNNEST($8::UUID[]) AS workflow_version_id,
        UNNEST($9::UUID[]) AS workflow_run_id,
        UNNEST($10::TEXT[]) AS schedule_timeout,
        UNNEST($11::TEXT[]) AS step_timeout,
        UNNEST($12::INTEGER[]) AS priority,
        UNNEST(CAST($13::TEXT[] AS v1_sticky_strategy_olap[])) AS sticky,
        UNNEST($14::UUID[]) AS desired_worker_id,
        UNNEST($15::UUID[]) AS external_id,
        UNNEST($16::TEXT[]) AS display_name,
        UNNEST($17::JSONB[]) AS input,
        UNNEST($18::JSONB[]) AS additional_metadata,
        UNNEST($19::BIGINT[]) AS dag_id,
        UNNEST($20::TIMESTAMPTZ[]) AS dag_inserted_at,
        UNNEST($21::UUID[]) AS parent_task_external_id,
        UNNEST($22::BOOLEAN[]) AS is_durable
)
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
    parent_task_external_id,
    is_durable
)
SELECT
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
    parent_task_external_id,
    is_durable
FROM inputs
ON CONFLICT (inserted_at, id) DO NOTHING
`

type CreateTasksOLAPParams struct {
	Tenantids             []uuid.UUID          `json:"tenantids"`
	Ids                   []int64              `json:"ids"`
	Insertedats           []pgtype.Timestamptz `json:"insertedats"`
	Queues                []string             `json:"queues"`
	Actionids             []string             `json:"actionids"`
	Stepids               []uuid.UUID          `json:"stepids"`
	Workflowids           []uuid.UUID          `json:"workflowids"`
	Workflowversionids    []uuid.UUID          `json:"workflowversionids"`
	Workflowrunids        []uuid.UUID          `json:"workflowrunids"`
	Scheduletimeouts      []string             `json:"scheduletimeouts"`
	Steptimeouts          []pgtype.Text        `json:"steptimeouts"`
	Priorities            []pgtype.Int4        `json:"priorities"`
	Stickies              []string             `json:"stickies"`
	Desiredworkerids      []*uuid.UUID         `json:"desiredworkerids"`
	Externalids           []uuid.UUID          `json:"externalids"`
	Displaynames          []string             `json:"displaynames"`
	Inputs                [][]byte             `json:"inputs"`
	Additionalmetadatas   [][]byte             `json:"additionalmetadatas"`
	Dagids                []pgtype.Int8        `json:"dagids"`
	Daginsertedats        []pgtype.Timestamptz `json:"daginsertedats"`
	Parenttaskexternalids []*uuid.UUID         `json:"parenttaskexternalids"`
	Isdurables            []bool               `json:"isdurables"`
}

func (q *Queries) CreateTasksOLAP(ctx context.Context, db DBTX, arg CreateTasksOLAPParams) error {
	_, err := db.Exec(ctx, createTasksOLAP,
		arg.Tenantids,
		arg.Ids,
		arg.Insertedats,
		arg.Queues,
		arg.Actionids,
		arg.Stepids,
		arg.Workflowids,
		arg.Workflowversionids,
		arg.Workflowrunids,
		arg.Scheduletimeouts,
		arg.Steptimeouts,
		arg.Priorities,
		arg.Stickies,
		arg.Desiredworkerids,
		arg.Externalids,
		arg.Displaynames,
		arg.Inputs,
		arg.Additionalmetadatas,
		arg.Dagids,
		arg.Daginsertedats,
		arg.Parenttaskexternalids,
		arg.Isdurables,
	)
	return err
}

const createDAGsOLAP = `-- name: CreateDAGsOLAP :exec
WITH inputs AS (
    SELECT
        UNNEST($1::UUID[]) AS tenant_id,
        UNNEST($2::BIGINT[]) AS id,
        UNNEST($3::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST($4::UUID[]) AS external_id,
        UNNEST($5::TEXT[]) AS display_name,
        UNNEST($6::UUID[]) AS workflow_id,
        UNNEST($7::UUID[]) AS workflow_version_id,
        UNNEST($8::JSONB[]) AS input,
        UNNEST($9::JSONB[]) AS additional_metadata,
        UNNEST($10::UUID[]) AS parent_task_external_id,
        UNNEST($11::INTEGER[]) AS total_tasks
), dag_task_counts AS (
    SELECT
        i.id,
        i.inserted_at,
        i.total_tasks,
        COUNT(t.id) AS task_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'COMPLETED') AS completed_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'FAILED') AS failed_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'CANCELLED') AS cancelled_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'QUEUED') AS queued_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'RUNNING') AS running_count,
        COUNT(t.id) FILTER (WHERE t.readable_status = 'EVICTED') AS evicted_count
    FROM inputs i
    JOIN v1_dag_to_task_olap dt ON (i.id, i.inserted_at) = (dt.dag_id, dt.dag_inserted_at)
    JOIN v1_tasks_olap t ON (dt.task_id, dt.task_inserted_at) = (t.id, t.inserted_at)
    GROUP BY i.id, i.inserted_at, i.total_tasks
), dag_statuses AS (
    SELECT
        dtc.id,
        dtc.inserted_at,
        CASE
            WHEN dtc.queued_count = dtc.task_count THEN 'QUEUED'
            WHEN dtc.task_count != dtc.total_tasks THEN 'RUNNING'
            WHEN dtc.running_count > 0 OR dtc.queued_count > 0 THEN 'RUNNING'
            WHEN dtc.evicted_count = dtc.task_count AND dtc.task_count = dtc.total_tasks THEN 'EVICTED'
            WHEN dtc.failed_count > 0 THEN 'FAILED'
            WHEN dtc.cancelled_count > 0 THEN 'CANCELLED'
            WHEN dtc.completed_count = dtc.task_count THEN 'COMPLETED'
            ELSE 'RUNNING'
        END::v1_readable_status_olap AS computed_status
    FROM dag_task_counts dtc
)
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
    total_tasks,
    readable_status
)
SELECT
    i.tenant_id,
    i.id,
    i.inserted_at,
    i.external_id,
    i.display_name,
    i.workflow_id,
    i.workflow_version_id,
    i.input,
    i.additional_metadata,
    i.parent_task_external_id,
    i.total_tasks,
    COALESCE(ds.computed_status, 'QUEUED'::v1_readable_status_olap)
FROM inputs i
LEFT JOIN dag_statuses ds ON (i.id, i.inserted_at) = (ds.id, ds.inserted_at)
ON CONFLICT (inserted_at, id) DO UPDATE SET
    readable_status = CASE
        WHEN v1_status_to_priority(EXCLUDED.readable_status) > v1_status_to_priority(v1_dags_olap.readable_status)
        THEN EXCLUDED.readable_status
        ELSE v1_dags_olap.readable_status
    END
`

type CreateDAGsOLAPOverwriteParams struct {
	Tenantids             []uuid.UUID          `json:"tenantids"`
	Ids                   []int64              `json:"ids"`
	Insertedats           []pgtype.Timestamptz `json:"insertedats"`
	Externalids           []uuid.UUID          `json:"externalids"`
	Displaynames          []string             `json:"displaynames"`
	Workflowids           []uuid.UUID          `json:"workflowids"`
	Workflowversionids    []uuid.UUID          `json:"workflowversionids"`
	Inputs                [][]byte             `json:"inputs"`
	Additionalmetadatas   [][]byte             `json:"additionalmetadatas"`
	Parenttaskexternalids []*uuid.UUID         `json:"parenttaskexternalids"`
	Totaltasks            []int32              `json:"totaltasks"`
}

func (q *Queries) CreateDAGsOLAP(ctx context.Context, db DBTX, arg CreateDAGsOLAPOverwriteParams) error {
	_, err := db.Exec(ctx, createDAGsOLAP,
		arg.Tenantids,
		arg.Ids,
		arg.Insertedats,
		arg.Externalids,
		arg.Displaynames,
		arg.Workflowids,
		arg.Workflowversionids,
		arg.Inputs,
		arg.Additionalmetadatas,
		arg.Parenttaskexternalids,
		arg.Totaltasks,
	)
	return err
}
