-- NOTE: This file mirrors the SQL embedded in `tasks-overwrite.go`.
-- The Go file is the source of truth; keep this file in sync with it.

-- name: CreateTasks :many
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

-- name: ReplayTasks :many
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

-- name: CreateTaskExpressionEvals :exec
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
