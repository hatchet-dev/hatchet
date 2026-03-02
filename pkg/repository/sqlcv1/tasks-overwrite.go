package sqlcv1

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	Tenantids           []uuid.UUID          `json:"tenantids"`
	Queues              []string             `json:"queues"`
	Actionids           []string             `json:"actionids"`
	Stepids             []uuid.UUID          `json:"stepids"`
	Stepreadableids     []string             `json:"stepreadableids"`
	Workflowids         []uuid.UUID          `json:"workflowids"`
	Scheduletimeouts    []string             `json:"scheduletimeouts"`
	Steptimeouts        []string             `json:"steptimeouts"`
	Priorities          []int32              `json:"priorities"`
	Stickies            []string             `json:"stickies"`
	Desiredworkerids    []*uuid.UUID         `json:"desiredworkerids"`
	Externalids         []uuid.UUID          `json:"externalids"`
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
	ParentTaskExternalIds        []*uuid.UUID         `json:"parentTaskExternalIds"`
	ParentTaskIds                []pgtype.Int8        `json:"parentTaskIds"`
	ParentTaskInsertedAts        []pgtype.Timestamptz `json:"parentTaskInsertedAts"`
	ChildIndex                   []pgtype.Int8        `json:"childIndex"`
	ChildKey                     []pgtype.Text        `json:"childKey"`
	StepIndex                    []int64              `json:"stepIndex"`
	RetryBackoffFactor           []pgtype.Float8      `json:"retryBackoffFactor"`
	RetryMaxBackoff              []pgtype.Int4        `json:"retryMaxBackoff"`
	WorkflowVersionIds           []uuid.UUID          `json:"workflowVersionIds"`
	WorkflowRunIds               []uuid.UUID          `json:"workflowRunIds"`
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
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const createTaskEvents = `-- name: CreateTaskEvents :many
-- We get a FOR UPDATE lock on tasks to prevent concurrent writes to the task events
-- tables for each task
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
		UNNEST($2::BIGINT[]) AS task_id,
		UNNEST($3::TIMESTAMPTZ[]) AS task_inserted_at,
		UNNEST($4::INTEGER[]) AS retry_count,
		UNNEST(CAST($5::TEXT[] as v1_task_event_type[])) AS event_type,
		UNNEST($6::TEXT[]) AS event_key,
		UNNEST($7::JSONB[]) AS data,
		UNNEST($8::UUID[]) as external_id
)
INSERT INTO v1_task_event (
    tenant_id,
    task_id,
	task_inserted_at,
    retry_count,
    event_type,
    event_key,
    data,
	external_id
)
SELECT
    $1::uuid,
    i.task_id,
	i.task_inserted_at,
    i.retry_count,
    i.event_type,
    i.event_key,
    i.data,
	i.external_id
FROM
    input i
ON CONFLICT (tenant_id, task_id, task_inserted_at, event_type, event_key) WHERE event_key IS NOT NULL DO NOTHING
RETURNING
    v1_task_event.id,
	v1_task_event.inserted_at,
	v1_task_event.tenant_id,
	v1_task_event.task_id,
	v1_task_event.task_inserted_at,
	v1_task_event.retry_count,
	v1_task_event.event_type,
	v1_task_event.event_key,
	v1_task_event.created_at,
	v1_task_event.data,
	v1_task_event.external_id
;
`

type CreateTaskEventsParams struct {
	Tenantid        uuid.UUID            `json:"tenantid"`
	Taskids         []int64              `json:"taskids"`
	Taskinsertedats []pgtype.Timestamptz `json:"taskinsertedats"`
	Retrycounts     []int32              `json:"retrycounts"`
	Eventtypes      []string             `json:"eventtypes"`
	Eventkeys       []pgtype.Text        `json:"eventkeys"`
	Datas           [][]byte             `json:"datas"`
	Externalids     []uuid.UUID          `json:"externalids"`
}

// We get a FOR UPDATE lock on tasks to prevent concurrent writes to the task events
// tables for each task
func (q *Queries) CreateTaskEvents(ctx context.Context, db DBTX, arg CreateTaskEventsParams) ([]*V1TaskEvent, error) {
	rows, err := db.Query(ctx, createTaskEvents,
		arg.Tenantid,
		arg.Taskids,
		arg.Taskinsertedats,
		arg.Retrycounts,
		arg.Eventtypes,
		arg.Eventkeys,
		arg.Datas,
		arg.Externalids,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var items []*V1TaskEvent
	for rows.Next() {
		var i V1TaskEvent
		if err := rows.Scan(
			&i.ID,
			&i.InsertedAt,
			&i.TenantID,
			&i.TaskID,
			&i.TaskInsertedAt,
			&i.RetryCount,
			&i.EventType,
			&i.EventKey,
			&i.CreatedAt,
			&i.Data,
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

const lockParentConcurrencySlots = `-- name: LockParentConcurrencySlots :batchexec
WITH input AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
                unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::integer[]) AS retry_count
        ) AS subquery
), concurrency_slots_to_delete AS (
    SELECT
        task_id, task_inserted_at, task_retry_count, parent_strategy_id, workflow_version_id, workflow_run_id
    FROM
        v1_concurrency_slot
    WHERE
        (task_id, task_inserted_at, task_retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
)
SELECT
    sort_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, completed_child_strategy_ids, child_strategy_ids, priority, key, is_filled
FROM
    v1_workflow_concurrency_slot wcs
WHERE
    (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) IN (
        SELECT parent_strategy_id, workflow_version_id, workflow_run_id FROM concurrency_slots_to_delete
    )
ORDER BY
    wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
FOR UPDATE
`

const releaseConcurrencySlots = `-- name: ReleaseConcurrencySlots :batchexec
WITH input AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
                unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::integer[]) AS retry_count
        ) AS subquery
), concurrency_slots_to_delete AS (
    SELECT
        task_id, task_inserted_at, task_retry_count
    FROM
        v1_concurrency_slot
    WHERE
        (task_id, task_inserted_at, task_retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    ORDER BY
        task_id, task_inserted_at, task_retry_count
    FOR UPDATE
)
DELETE FROM
    v1_concurrency_slot
WHERE
    (task_id, task_inserted_at, task_retry_count) IN (SELECT task_id, task_inserted_at, task_retry_count FROM concurrency_slots_to_delete)
`

const releaseQueueItems = `-- name: ReleaseQueueItems :batchexec
WITH input AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
                unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::integer[]) AS retry_count
        ) AS subquery
), queue_items_to_delete AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        v1_queue_item
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    ORDER BY
        task_id, task_inserted_at, retry_count
    FOR UPDATE
)
DELETE FROM
    v1_queue_item
WHERE
    (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM queue_items_to_delete)
`

const releaseRateLimitedQueueItems = `-- name: ReleaseRateLimitedQueueItems :batchexec
WITH input AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
                unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::integer[]) AS retry_count
        ) AS subquery
), rate_limited_items_to_delete AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        v1_rate_limited_queue_items
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    ORDER BY
        task_id, task_inserted_at, retry_count
    FOR UPDATE
)
DELETE FROM
    v1_rate_limited_queue_items
WHERE
    (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM rate_limited_items_to_delete)
`

const releaseRetryQueueItems = `-- name: ReleaseRetryQueueItems :batchexec
WITH input AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
                unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::integer[]) AS retry_count
        ) AS subquery
), retry_queue_items_to_delete AS (
    SELECT
        task_id, task_inserted_at, task_retry_count
    FROM
        v1_retry_queue_item
    WHERE
        (task_id, task_inserted_at, task_retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    ORDER BY
        task_id, task_inserted_at, task_retry_count
    FOR UPDATE
)
DELETE FROM
    v1_retry_queue_item r
WHERE
    (task_id, task_inserted_at, task_retry_count) IN (
        SELECT
            task_id, task_inserted_at, task_retry_count
        FROM
            retry_queue_items_to_delete
    )
`

const releaseTasks = `-- name: ReleaseTasks :batchmany
WITH input AS (
    SELECT
        task_id, task_inserted_at, retry_count
    FROM
        (
            SELECT
                unnest($1::bigint[]) AS task_id,
                unnest($2::timestamptz[]) AS task_inserted_at,
                unnest($3::integer[]) AS retry_count
        ) AS subquery
), runtimes_to_delete AS (
    SELECT
        task_id,
        task_inserted_at,
        retry_count,
        worker_id
    FROM
        v1_task_runtime
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
    ORDER BY
        task_id, task_inserted_at, retry_count
    FOR UPDATE
), deleted_slots AS (
    DELETE FROM
        v1_task_runtime_slot
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
), deleted_runtimes AS (
    DELETE FROM
        v1_task_runtime
    WHERE
        (task_id, task_inserted_at, retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM runtimes_to_delete)
)
SELECT
    t.queue,
    t.id,
    t.inserted_at,
    t.external_id,
    t.step_readable_id,
    t.workflow_run_id,
    r.worker_id,
    i.retry_count::int AS retry_count,
    t.retry_count = i.retry_count AS is_current_retry,
    t.concurrency_strategy_ids
FROM
    v1_task t
JOIN
    input i ON i.task_id = t.id AND i.task_inserted_at = t.inserted_at
LEFT JOIN
    runtimes_to_delete r ON r.task_id = t.id AND r.retry_count = t.retry_count
`

type ReleaseTasksParams struct {
	Taskids         []int64              `json:"taskids"`
	Taskinsertedats []pgtype.Timestamptz `json:"taskinsertedats"`
	Retrycounts     []int32              `json:"retrycounts"`
}

type ReleaseTasksRow struct {
	Queue                  string             `json:"queue"`
	ID                     int64              `json:"id"`
	InsertedAt             pgtype.Timestamptz `json:"inserted_at"`
	ExternalID             uuid.UUID          `json:"external_id"`
	StepReadableID         string             `json:"step_readable_id"`
	WorkflowRunID          uuid.UUID          `json:"workflow_run_id"`
	WorkerID               uuid.UUID          `json:"worker_id"`
	RetryCount             int32              `json:"retry_count"`
	IsCurrentRetry         bool               `json:"is_current_retry"`
	ConcurrencyStrategyIds []int64            `json:"concurrency_strategy_ids"`
}

func (q *Queries) ReleaseTasks(ctx context.Context, db DBTX, arg ReleaseTasksParams) ([]*ReleaseTasksRow, error) {
	batch := &pgx.Batch{}
	vals := []interface{}{
		arg.Taskids,
		arg.Taskinsertedats,
		arg.Retrycounts,
	}

	rowsCh := make(chan *ReleaseTasksRow, len(arg.Taskids))
	errCh := make(chan error, 1)

	res := batch.Queue(releaseTasks, vals...)

	res.Query(func(rows pgx.Rows) error {
		for rows.Next() {
			var i ReleaseTasksRow
			if err := rows.Scan(
				&i.Queue,
				&i.ID,
				&i.InsertedAt,
				&i.ExternalID,
				&i.StepReadableID,
				&i.WorkflowRunID,
				&i.WorkerID,
				&i.RetryCount,
				&i.IsCurrentRetry,
				&i.ConcurrencyStrategyIds,
			); err != nil {
				errCh <- err
				close(rowsCh)
				close(errCh)
				return err
			}
			rowsCh <- &i
		}
		errCh <- rows.Err()
		close(rowsCh)
		close(errCh)
		return nil
	})
	batch.Queue(releaseRetryQueueItems, vals...)
	batch.Queue(releaseQueueItems, vals...)
	batch.Queue(lockParentConcurrencySlots, vals...)
	batch.Queue(releaseConcurrencySlots, vals...)
	batch.Queue(releaseRateLimitedQueueItems, vals...)

	br := db.SendBatch(ctx, batch)
	err := br.Close()

	if err != nil {
		return nil, err
	}

	var items []*ReleaseTasksRow

	for r := range rowsCh {
		items = append(items, r)
	}

	if err := <-errCh; err != nil {
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
        UNNEST($5::JSONB[]) AS additional_metadata,
        -- Scopes are nullable
        UNNEST($6::TEXT[]) AS scope,
        -- Webhook names are nullable
        UNNEST($7::TEXT[]) AS triggering_webhook_name
)
INSERT INTO v1_event (
    tenant_id,
    external_id,
    seen_at,
    key,
    additional_metadata,
    scope,
	triggering_webhook_name
)
SELECT tenant_id, external_id, seen_at, key, additional_metadata, scope, triggering_webhook_name
FROM to_insert
RETURNING tenant_id, id, external_id, seen_at, key, additional_metadata, scope, triggering_webhook_name
`

type BulkCreateEventsParams struct {
	Tenantids              []uuid.UUID          `json:"tenantids"`
	Externalids            []uuid.UUID          `json:"externalids"`
	Seenats                []pgtype.Timestamptz `json:"seenats"`
	Keys                   []string             `json:"keys"`
	Additionalmetadatas    [][]byte             `json:"additionalmetadatas"`
	Scopes                 []pgtype.Text        `json:"scopes"`
	TriggeringWebhookNames []pgtype.Text        `json:"triggeringWebhookName"`
}

func (q *Queries) BulkCreateEvents(ctx context.Context, db DBTX, arg BulkCreateEventsParams) ([]*V1Event, error) {
	rows, err := db.Query(ctx, bulkCreateEvents,
		arg.Tenantids,
		arg.Externalids,
		arg.Seenats,
		arg.Keys,
		arg.Additionalmetadatas,
		arg.Scopes,
		arg.TriggeringWebhookNames,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*V1Event
	for rows.Next() {
		var i V1Event
		if err := rows.Scan(
			&i.TenantID,
			&i.ID,
			&i.ExternalID,
			&i.SeenAt,
			&i.Key,
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
