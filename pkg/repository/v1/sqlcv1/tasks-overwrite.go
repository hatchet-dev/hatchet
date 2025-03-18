package sqlcv1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

const createTasks = `-- name: CreateTasks :many
WITH input AS (
    SELECT
        tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, schedule_timeout, step_timeout, priority, sticky, desired_worker_id, external_id, display_name, input, retry_count, additional_metadata, initial_state, dag_id, dag_inserted_at, concurrency_parent_strategy_ids, concurrency_strategy_ids, concurrency_keys, initial_state_reason, parent_task_external_id, parent_task_id, parent_task_inserted_at, child_index, child_key, step_index, retry_backoff_factor, retry_max_backoff, workflow_version_id, workflow_run_id
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
                unnest(cast($10::text[] as v1_sticky_strategy[])) AS sticky,
                unnest($11::uuid[]) AS desired_worker_id,
                unnest($12::uuid[]) AS external_id,
                unnest($13::text[]) AS display_name,
                unnest($14::jsonb[]) AS input,
                unnest($15::integer[]) AS retry_count,
                unnest($16::jsonb[]) AS additional_metadata,
				unnest(cast($17::text[] as v1_task_initial_state[])) AS initial_state,
                -- NOTE: these are nullable, so sqlc doesn't support casting to a type
                unnest($18::bigint[]) AS dag_id,
                unnest($19::timestamptz[]) AS dag_inserted_at,
				unnest_nd_1d($20::bigint[][]) AS concurrency_parent_strategy_ids,
				unnest_nd_1d($21::bigint[][]) AS concurrency_strategy_ids,
				unnest_nd_1d($22::text[][]) AS concurrency_keys,
				unnest($23::text[]) AS initial_state_reason,
				unnest($24::uuid[]) AS parent_task_external_id,
				unnest($25::bigint[]) AS parent_task_id,
				unnest($26::timestamptz[]) AS parent_task_inserted_at,
				unnest($27::integer[]) AS child_index,
				unnest($28::text[]) AS child_key,
				unnest($29::bigint[]) AS step_index,
				unnest($30::double precision[]) AS retry_backoff_factor,
				unnest($31::integer[]) AS retry_max_backoff,
				unnest($32::uuid[]) AS workflow_version_id,
				unnest($33::uuid[]) AS workflow_run_id
        ) AS subquery
)
INSERT INTO v1_task (
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
	concurrency_parent_strategy_ids,
	concurrency_strategy_ids,
	concurrency_keys,
	initial_state_reason,
	parent_task_external_id,
	parent_task_id,
	parent_task_inserted_at,
	child_index,
	child_key,
	step_index,
	retry_backoff_factor,
	retry_max_backoff,
	workflow_version_id,
	workflow_run_id
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
	i.concurrency_parent_strategy_ids,
	i.concurrency_strategy_ids,
	i.concurrency_keys,
	i.initial_state_reason,
	i.parent_task_external_id,
	i.parent_task_id,
	i.parent_task_inserted_at,
	i.child_index,
	i.child_key,
	i.step_index,
	i.retry_backoff_factor,
	i.retry_max_backoff,
	i.workflow_version_id,
	i.workflow_run_id
FROM
    input i
RETURNING
    id, inserted_at, tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, schedule_timeout, step_timeout, priority, sticky, desired_worker_id, external_id, display_name, input, retry_count, internal_retry_count, app_retry_count, additional_metadata, initial_state, dag_id, dag_inserted_at, concurrency_parent_strategy_ids, concurrency_strategy_ids, concurrency_keys, initial_state_reason, parent_task_external_id, parent_task_id, parent_task_inserted_at, child_index, child_key, step_index, retry_backoff_factor, retry_max_backoff, workflow_version_id, workflow_run_id
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
	InitialStates       []string             `json:"initialstates"`
	InitialStateReasons []pgtype.Text        `json:"initialStateReasons"`
	Dagids              []pgtype.Int8        `json:"dagids"`
	Daginsertedats      []pgtype.Timestamptz `json:"daginsertedats"`
	// FIXME: pgx doesn't like multi-dimensional arrays with different lengths, these types
	// probably need to change. Current hack is to group tasks by their step id where these
	// multi-dimensional arrays are the same length.
	Concurrencyparentstrategyids [][]pgtype.Int8      `json:"concurrencyparentstrategyids"`
	ConcurrencyStrategyIds       [][]int64            `json:"concurrencyStrategyIds"`
	ConcurrencyKeys              [][]string           `json:"concurrencyKeys"`
	ParentTaskExternalIds        []pgtype.UUID        `json:"parentTaskExternalIds"`
	ParentTaskIds                []pgtype.Int8        `json:"parentTaskIds"`
	ParentTaskInsertedAts        []pgtype.Timestamptz `json:"parentTaskInsertedAts"`
	ChildIndex                   []pgtype.Int8        `json:"childIndex"`
	ChildKey                     []pgtype.Text        `json:"childKey"`
	StepIndex                    []int64              `json:"stepIndex"`
	RetryBackoffFactor           []pgtype.Float8      `json:"retryBackoffFactor"`
	RetryMaxBackoff              []pgtype.Int4        `json:"retryMaxBackoff"`
	WorkflowVersionIds           []pgtype.UUID        `json:"workflowVersionIds"`
	WorkflowRunIds               []pgtype.UUID        `json:"workflowRunIds"`
}

func (q *Queries) CreateTasks(ctx context.Context, db DBTX, arg CreateTasksParams) ([]*V1Task, error) {
	// panic-recover
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		// log the arg
	// 		argBytes, _ := json.Marshal(arg)
	// 		fmt.Println("ARG BYTES ARE", string(argBytes))

	// 		// also print the panic
	// 		fmt.Println("PANIC", r)
	// 		panic(r)
	// 	}
	// }()

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
		arg.Concurrencyparentstrategyids,
		arg.ConcurrencyStrategyIds,
		arg.ConcurrencyKeys,
		arg.InitialStateReasons,
		arg.ParentTaskExternalIds,
		arg.ParentTaskIds,
		arg.ParentTaskInsertedAts,
		arg.ChildIndex,
		arg.ChildKey,
		arg.StepIndex,
		arg.RetryBackoffFactor,
		arg.RetryMaxBackoff,
		arg.WorkflowVersionIds,
		arg.WorkflowRunIds,
	)
	if err != nil {
		argBytes, _ := json.Marshal(arg)
		fmt.Println("FAILED ARG BYTES ARE", string(argBytes))

		return nil, err
	}
	defer rows.Close()
	var items []*V1Task
	for rows.Next() {
		var i V1Task
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
			&i.ConcurrencyParentStrategyIds,
			&i.ConcurrencyStrategyIds,
			&i.ConcurrencyKeys,
			&i.InitialStateReason,
			&i.ParentTaskExternalID,
			&i.ParentTaskID,
			&i.ParentTaskInsertedAt,
			&i.ChildIndex,
			&i.ChildKey,
			&i.StepIndex,
			&i.RetryBackoffFactor,
			&i.RetryMaxBackoff,
			&i.WorkflowVersionID,
			&i.WorkflowRunID,
		); err != nil {
			argBytes, _ := json.Marshal(arg)
			fmt.Println("FAILED ARG BYTES ARE", string(argBytes))

			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		argBytes, _ := json.Marshal(arg)
		fmt.Println("FAILED ARG BYTES ARE", string(argBytes))

		return nil, err
	}
	return items, nil
}

const createTaskEvents = `-- name: CreateTaskEvents :exec
WITH locked_tasks AS (
    SELECT
        id
    FROM
        v1_task
    WHERE
        id = ANY($2::bigint[])
        AND tenant_id = $1::uuid
    -- order by the task id to get a stable lock order
    ORDER BY
        id
    FOR UPDATE
), input AS (
    SELECT
        task_id, task_inserted_at, retry_count, event_type, event_key, data
    FROM
        (
            SELECT
                unnest($2::bigint[]) AS task_id,
				unnest($3::timestamptz[]) AS task_inserted_at,
                unnest($4::integer[]) AS retry_count,
                unnest(cast($5::text[] as v1_task_event_type[])) AS event_type,
                unnest($6::text[]) AS event_key,
                unnest($7::jsonb[]) AS data
        ) AS subquery
)
INSERT INTO v1_task_event (
    tenant_id,
    task_id,
	task_inserted_at,
    retry_count,
    event_type,
    event_key,
    data
)
SELECT
    $1::uuid,
    i.task_id,
	i.task_inserted_at,
    i.retry_count,
    i.event_type,
    i.event_key,
    i.data
FROM
    input i
ON CONFLICT (tenant_id, task_id, task_inserted_at, event_type, event_key) WHERE event_key IS NOT NULL DO NOTHING
`

type CreateTaskEventsParams struct {
	Tenantid        pgtype.UUID          `json:"tenantid"`
	Taskids         []int64              `json:"taskids"`
	Taskinsertedats []pgtype.Timestamptz `json:"taskinsertedats"`
	Retrycounts     []int32              `json:"retrycounts"`
	Eventtypes      []string             `json:"eventtypes"`
	Eventkeys       []pgtype.Text        `json:"eventkeys"`
	Datas           [][]byte             `json:"datas"`
}

// We get a FOR UPDATE lock on tasks to prevent concurrent writes to the task events
// tables for each task
func (q *Queries) CreateTaskEvents(ctx context.Context, db DBTX, arg CreateTaskEventsParams) error {
	_, err := db.Exec(ctx, createTaskEvents,
		arg.Tenantid,
		arg.Taskids,
		arg.Taskinsertedats,
		arg.Retrycounts,
		arg.Eventtypes,
		arg.Eventkeys,
		arg.Datas,
	)
	return err
}

const replayTasks = `-- name: ReplayTasks :many
WITH input AS (
    SELECT
        task_id, task_inserted_at, input, initial_state, concurrency_keys, initial_state_reason
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
				unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::jsonb[]) AS input,
                unnest(cast($4::text[] as v1_task_initial_state[])) AS initial_state,
				unnest_nd_1d($5::text[][]) AS concurrency_keys,
				unnest($6::text[]) AS initial_state_reason
        ) AS subquery
)
UPDATE
    v1_task
SET
    retry_count = retry_count + 1,
    app_retry_count = 0,
    internal_retry_count = 0,
    input = CASE WHEN i.input IS NOT NULL THEN i.input ELSE v1_task.input END,
    initial_state = i.initial_state,
    concurrency_keys = i.concurrency_keys,
    initial_state_reason = i.initial_state_reason
FROM
    input i
WHERE
	(v1_task.id, v1_task.inserted_at) = (i.task_id, i.task_inserted_at)
RETURNING
    v1_task.id, v1_task.inserted_at, v1_task.tenant_id, v1_task.queue, v1_task.action_id, v1_task.step_id, v1_task.step_readable_id, v1_task.workflow_id, v1_task.schedule_timeout, v1_task.step_timeout, v1_task.priority, v1_task.sticky, v1_task.desired_worker_id, v1_task.external_id, v1_task.display_name, v1_task.input, v1_task.retry_count, v1_task.internal_retry_count, v1_task.app_retry_count, v1_task.additional_metadata, v1_task.dag_id, v1_task.dag_inserted_at, v1_task.parent_task_id, v1_task.child_index, v1_task.child_key, v1_task.initial_state, v1_task.initial_state_reason, v1_task.concurrency_parent_strategy_ids, v1_task.concurrency_strategy_ids, v1_task.concurrency_keys, v1_task.retry_backoff_factor, v1_task.retry_max_backoff
`

type ReplayTasksParams struct {
	Taskids         []int64              `json:"taskids"`
	Taskinsertedats []pgtype.Timestamptz `json:"taskinsertedats"`
	Inputs          [][]byte             `json:"inputs"`
	InitialStates   []string             `json:"initialstates"`
	// FIXME: pgx doesn't like multi-dimensional arrays with different lengths, these types
	// probably need to change. Current hack is to group tasks by their step id where these
	// multi-dimensional arrays are the same length.
	Concurrencykeys     [][]string    `json:"concurrencykeys"`
	InitialStateReasons []pgtype.Text `json:"initialStateReasons"`
}

// NOTE: at this point, we assume we have a lock on tasks and therefor we can update the tasks
func (q *Queries) ReplayTasks(ctx context.Context, db DBTX, arg ReplayTasksParams) ([]*V1Task, error) {
	rows, err := db.Query(ctx, replayTasks,
		arg.Taskids,
		arg.Taskinsertedats,
		arg.Inputs,
		arg.InitialStates,
		arg.Concurrencykeys,
		arg.InitialStateReasons,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*V1Task
	for rows.Next() {
		var i V1Task
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
			&i.ParentTaskID,
			&i.ChildIndex,
			&i.ChildKey,
			&i.InitialState,
			&i.InitialStateReason,
			&i.ConcurrencyParentStrategyIds,
			&i.ConcurrencyStrategyIds,
			&i.ConcurrencyKeys,
			&i.RetryBackoffFactor,
			&i.RetryMaxBackoff,
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

const createTaskExpressionEvals = `-- name: CreateTaskExpressionEvals :exec
WITH input AS (
    SELECT
        task_id, task_inserted_at, key, value_str, value_int, kind
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
                unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::text[]) AS key,
                unnest($4::text[]) AS value_str,
				unnest($5::integer[]) AS value_int,
                unnest(cast($6::text[] as "StepExpressionKind"[])) AS kind
        ) AS subquery
)
INSERT INTO v1_task_expression_eval (
    key,
    task_id,
    task_inserted_at,
    value_str,
	value_int,
    kind
)
SELECT
    i.key,
    i.task_id,
    i.task_inserted_at,
    i.value_str,
	i.value_int,
    i.kind
FROM
    input i
ON CONFLICT (task_id, task_inserted_at, kind, key) DO UPDATE
SET
    value_str = EXCLUDED.value_str,
    value_int = EXCLUDED.value_int
`

type CreateTaskExpressionEvalsParams struct {
	Taskids         []int64              `json:"taskids"`
	Taskinsertedats []pgtype.Timestamptz `json:"taskinsertedats"`
	Keys            []string             `json:"keys"`
	Valuesstr       []pgtype.Text        `json:"valuesstr"`
	Valuesint       []pgtype.Int4        `json:"valuesint"`
	Kinds           []string             `json:"kinds"`
}

func (q *Queries) CreateTaskExpressionEvals(ctx context.Context, db DBTX, arg CreateTaskExpressionEvalsParams) error {
	_, err := db.Exec(ctx, createTaskExpressionEvals,
		arg.Taskids,
		arg.Taskinsertedats,
		arg.Keys,
		arg.Valuesstr,
		arg.Valuesint,
		arg.Kinds,
	)
	return err
}
