package sqlcv1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createTasks = `-- name: CreateTasks :many
WITH input AS (
    SELECT
        tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, schedule_timeout, step_timeout, priority, sticky, desired_worker_id, external_id, display_name, input, retry_count, additional_metadata, initial_state, dag_id, dag_inserted_at, concurrency_strategy_ids, concurrency_keys, initial_state_reason
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
				unnest(cast($17::text[] as v2_task_initial_state[])) AS initial_state,
                -- NOTE: these are nullable, so sqlc doesn't support casting to a type
                unnest($18::bigint[]) AS dag_id,
                unnest($19::timestamptz[]) AS dag_inserted_at,
				unnest_nd_1d($20::bigint[][]) AS concurrency_strategy_ids,
				unnest_nd_1d($21::text[][]) AS concurrency_keys,
				unnest($22::text[]) AS initial_state_reason
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
	initial_state,
    dag_id,
    dag_inserted_at,
	concurrency_strategy_ids,
	concurrency_keys,
	initial_state_reason
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
	i.initial_state,
    i.dag_id,
    i.dag_inserted_at,
	i.concurrency_strategy_ids,
	i.concurrency_keys,
	i.initial_state_reason
FROM
    input i
RETURNING
    id, inserted_at, tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, schedule_timeout, step_timeout, priority, sticky, desired_worker_id, external_id, display_name, input, retry_count, internal_retry_count, app_retry_count, additional_metadata, initial_state, dag_id, dag_inserted_at, concurrency_strategy_ids, concurrency_keys, initial_state_reason
`

type CreateTasksParams struct {
	Tenantids              []pgtype.UUID        `json:"tenantids"`
	Queues                 []string             `json:"queues"`
	Actionids              []string             `json:"actionids"`
	Stepids                []pgtype.UUID        `json:"stepids"`
	Stepreadableids        []string             `json:"stepreadableids"`
	Workflowids            []pgtype.UUID        `json:"workflowids"`
	Scheduletimeouts       []string             `json:"scheduletimeouts"`
	Steptimeouts           []string             `json:"steptimeouts"`
	Priorities             []int32              `json:"priorities"`
	Stickies               []string             `json:"stickies"`
	Desiredworkerids       []pgtype.UUID        `json:"desiredworkerids"`
	Externalids            []pgtype.UUID        `json:"externalids"`
	Displaynames           []string             `json:"displaynames"`
	Inputs                 [][]byte             `json:"inputs"`
	Retrycounts            []int32              `json:"retrycounts"`
	Additionalmetadatas    [][]byte             `json:"additionalmetadatas"`
	InitialStates          []string             `json:"initialstates"`
	InitialStateReasons    []pgtype.Text        `json:"initialStateReasons"`
	Dagids                 []pgtype.Int8        `json:"dagids"`
	Daginsertedats         []pgtype.Timestamptz `json:"daginsertedats"`
	ConcurrencyStrategyIds [][]int64            `json:"concurrencyStrategyIds"`
	ConcurrencyKeys        [][]string           `json:"concurrencyKeys"`
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
		arg.InitialStates,
		arg.Dagids,
		arg.Daginsertedats,
		arg.ConcurrencyStrategyIds,
		arg.ConcurrencyKeys,
		arg.InitialStateReasons,
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
			&i.InitialState,
			&i.DagID,
			&i.DagInsertedAt,
			&i.ConcurrencyStrategyIds,
			&i.ConcurrencyKeys,
			&i.InitialStateReason,
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

const createTaskEvents = `-- name: CreateTaskEvents :exec
WITH locked_tasks AS (
    SELECT
        id
    FROM
        v2_task
    WHERE
        id = ANY($2::bigint[])
        AND tenant_id = $1::uuid
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), input AS (
    SELECT
        task_id, retry_count, event_type, event_key, data
    FROM
        (
            SELECT
                unnest($2::bigint[]) AS task_id,
                unnest($3::integer[]) AS retry_count,
                unnest(cast($4::text[] as v2_task_event_type[])) AS event_type,
                unnest($5::text[]) AS event_key,
                unnest($6::jsonb[]) AS data
        ) AS subquery
)
INSERT INTO v2_task_event (
    tenant_id,
    task_id,
    retry_count,
    event_type,
    event_key,
    data
)
SELECT
    $1::uuid,
    i.task_id,
    i.retry_count,
    i.event_type,
    i.event_key,
    i.data
FROM
    input i
ON CONFLICT (tenant_id, task_id, event_type, event_key) WHERE event_key IS NOT NULL DO NOTHING
`

type CreateTaskEventsParams struct {
	Tenantid    pgtype.UUID   `json:"tenantid"`
	Taskids     []int64       `json:"taskids"`
	Retrycounts []int32       `json:"retrycounts"`
	Eventtypes  []string      `json:"eventtypes"`
	Eventkeys   []pgtype.Text `json:"eventkeys"`
	Datas       [][]byte      `json:"datas"`
}

// We get a FOR UPDATE lock on tasks to prevent concurrent writes to the task events
// tables for each task
func (q *Queries) CreateTaskEvents(ctx context.Context, db DBTX, arg CreateTaskEventsParams) error {
	_, err := db.Exec(ctx, createTaskEvents,
		arg.Tenantid,
		arg.Taskids,
		arg.Retrycounts,
		arg.Eventtypes,
		arg.Eventkeys,
		arg.Datas,
	)
	return err
}
