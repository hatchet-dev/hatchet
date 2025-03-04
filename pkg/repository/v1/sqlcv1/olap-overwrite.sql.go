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
    ORDER BY
        inserted_at DESC
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
`

type CountTasksParams struct {
	Tenantid    pgtype.UUID        `json:"tenantid"`
	Since       pgtype.Timestamptz `json:"since"`
	Statuses    []string           `json:"statuses"`
	Until       pgtype.Timestamptz `json:"until"`
	WorkflowIds []pgtype.UUID      `json:"workflowIds"`
	WorkerId    pgtype.UUID        `json:"workerId"`
	Keys        []string           `json:"keys"`
	Values      []string           `json:"values"`
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
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered
`

type CountWorkflowRunsParams struct {
	Tenantid    pgtype.UUID        `json:"tenantid"`
	Statuses    []string           `json:"statuses"`
	WorkflowIds []pgtype.UUID      `json:"workflowIds"`
	Since       pgtype.Timestamptz `json:"since"`
	Until       pgtype.Timestamptz `json:"until"`
	Keys        []string           `json:"keys"`
	Values      []string           `json:"values"`
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

ORDER BY inserted_at DESC, id DESC
LIMIT $9::integer
OFFSET $8::integer
`

type FetchWorkflowRunIdsParams struct {
	Tenantid               pgtype.UUID        `json:"tenantid"`
	Statuses               []string           `json:"statuses"`
	WorkflowIds            []pgtype.UUID      `json:"workflowIds"`
	Since                  pgtype.Timestamptz `json:"since"`
	Until                  pgtype.Timestamptz `json:"until"`
	Keys                   []string           `json:"keys"`
	Values                 []string           `json:"values"`
	Listworkflowrunsoffset int32              `json:"listworkflowrunsoffset"`
	Listworkflowrunslimit  int32              `json:"listworkflowrunslimit"`
	ParentTaskExternalId   pgtype.UUID        `json:"parentTaskExternalId"`
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
ORDER BY
    inserted_at DESC
LIMIT $10::integer
OFFSET $9::integer
`

type ListTasksOlapParams struct {
	Tenantid    pgtype.UUID        `json:"tenantid"`
	Since       pgtype.Timestamptz `json:"since"`
	Statuses    []string           `json:"statuses"`
	Until       pgtype.Timestamptz `json:"until"`
	WorkflowIds []pgtype.UUID      `json:"workflowIds"`
	WorkerId    pgtype.UUID        `json:"workerId"`
	Keys        []string           `json:"keys"`
	Values      []string           `json:"values"`
	Taskoffset  int32              `json:"taskoffset"`
	Tasklimit   int32              `json:"tasklimit"`
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
