package sqlcv2

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createTasks = `-- name: CreateTasks :many
WITH input AS (
    SELECT
        tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, schedule_timeout, step_timeout, priority, sticky, desired_worker_id, external_id, display_name, input, retry_count, additional_metadata, dag_id, dag_inserted_at
    FROM
        (
            SELECT
                unnest($1::uuid[]) AS tenant_id,
                unnest($2::text[]) AS queue,
                unnest($3::text[]) AS action_id,
                unnest($4::uuid[]) AS step_id,
                unnest($5::text[]) AS step_readable_id,
                unnest($6::uuid[]) AS workflow_id,
                unnest($7::text[]) AS schedule_timeout,
                unnest($8::text[]) AS step_timeout,
                unnest($9::integer[]) AS priority,
                unnest(cast($10::text[] as v2_sticky_strategy[])) AS sticky,
                unnest($11::uuid[]) AS desired_worker_id,
                unnest($12::uuid[]) AS external_id,
                unnest($13::text[]) AS display_name,
                unnest($14::jsonb[]) AS input,
                unnest($15::integer[]) AS retry_count,
                unnest($16::jsonb[]) AS additional_metadata,
                -- NOTE: these are nullable, so sqlc doesn't support casting to a type
                unnest($17::bigint[]) AS dag_id,
                unnest($18::timestamptz[]) AS dag_inserted_at
        ) AS subquery
)
INSERT INTO v2_task (
    tenant_id,
    queue,
    action_id,
    step_id,
    step_readable_id,
    workflow_id,
    schedule_timeout,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    external_id,
    display_name,
    input,
    retry_count,
    additional_metadata,
    dag_id,
    dag_inserted_at
)
SELECT
    i.tenant_id,
    i.queue,
    i.action_id,
    i.step_id,
    i.step_readable_id,
    i.workflow_id,
    i.schedule_timeout,
    i.step_timeout,
    i.priority,
    i.sticky,
    i.desired_worker_id,
    i.external_id,
    i.display_name,
    i.input,
    i.retry_count,
    i.additional_metadata,
    i.dag_id,
    i.dag_inserted_at
FROM
    input i
RETURNING
    id, inserted_at, tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, schedule_timeout, step_timeout, priority, sticky, desired_worker_id, external_id, display_name, input, retry_count, internal_retry_count, app_retry_count, additional_metadata, dag_id, dag_inserted_at
`

type CreateTasksParams struct {
	Tenantids           []pgtype.UUID        `json:"tenantids"`
	Queues              []string             `json:"queues"`
	Actionids           []string             `json:"actionids"`
	Stepids             []pgtype.UUID        `json:"stepids"`
	Stepreadableids     []string             `json:"stepreadableids"`
	Workflowids         []pgtype.UUID        `json:"workflowids"`
	Scheduletimeouts    []string             `json:"scheduletimeouts"`
	Steptimeouts        []string             `json:"steptimeouts"`
	Priorities          []int32              `json:"priorities"`
	Stickies            []string             `json:"stickies"`
	Desiredworkerids    []pgtype.UUID        `json:"desiredworkerids"`
	Externalids         []pgtype.UUID        `json:"externalids"`
	Displaynames        []string             `json:"displaynames"`
	Inputs              [][]byte             `json:"inputs"`
	Retrycounts         []int32              `json:"retrycounts"`
	Additionalmetadatas [][]byte             `json:"additionalmetadatas"`
	Dagids              []pgtype.Int8        `json:"dagids"`
	Daginsertedats      []pgtype.Timestamptz `json:"daginsertedats"`
}

func (q *Queries) CreateTasks(ctx context.Context, db DBTX, arg CreateTasksParams) ([]*V2Task, error) {
	rows, err := db.Query(ctx, createTasks,
		arg.Tenantids,
		arg.Queues,
		arg.Actionids,
		arg.Stepids,
		arg.Stepreadableids,
		arg.Workflowids,
		arg.Scheduletimeouts,
		arg.Steptimeouts,
		arg.Priorities,
		arg.Stickies,
		arg.Desiredworkerids,
		arg.Externalids,
		arg.Displaynames,
		arg.Inputs,
		arg.Retrycounts,
		arg.Additionalmetadatas,
		arg.Dagids,
		arg.Daginsertedats,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*V2Task
	for rows.Next() {
		var i V2Task
		if err := rows.Scan(
			&i.ID,
			&i.InsertedAt,
			&i.TenantID,
			&i.Queue,
			&i.ActionID,
			&i.StepID,
			&i.StepReadableID,
			&i.WorkflowID,
			&i.ScheduleTimeout,
			&i.StepTimeout,
			&i.Priority,
			&i.Sticky,
			&i.DesiredWorkerID,
			&i.ExternalID,
			&i.DisplayName,
			&i.Input,
			&i.RetryCount,
			&i.InternalRetryCount,
			&i.AppRetryCount,
			&i.AdditionalMetadata,
			&i.DagID,
			&i.DagInsertedAt,
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
