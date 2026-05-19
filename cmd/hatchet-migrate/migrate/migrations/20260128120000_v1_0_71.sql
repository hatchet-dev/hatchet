-- +goose Up
-- +goose StatementBegin
-- Fix: v1_workflow_concurrency_slot.sort_id is BIGINT, but the cleanup function used INTEGER
-- which can overflow once sort_id exceeds int32.
CREATE OR REPLACE FUNCTION cleanup_workflow_concurrency_slots(
    p_strategy_id BIGINT,
    p_workflow_version_id UUID,
    p_workflow_run_id UUID
) RETURNS VOID AS $$
DECLARE
    v_sort_id BIGINT;
BEGIN
    -- Get the sort_id for the specific workflow concurrency slot
    SELECT sort_id INTO v_sort_id
    FROM v1_workflow_concurrency_slot
    WHERE strategy_id = p_strategy_id
      AND workflow_version_id = p_workflow_version_id
      AND workflow_run_id = p_workflow_run_id;

    -- Acquire an advisory lock for the strategy ID
    -- There is a small chance of collisions but it's extremely unlikely
    PERFORM pg_advisory_xact_lock(1000000 * p_strategy_id + v_sort_id);

    WITH relevant_tasks_for_dags AS (
        SELECT
            t.id,
            t.inserted_at,
            t.retry_count
        FROM
            v1_task t
        JOIN
            v1_dag_to_task dt ON t.id = dt.task_id AND t.inserted_at = dt.task_inserted_at
        JOIN
            v1_lookup_table lt ON dt.dag_id = lt.dag_id AND dt.dag_inserted_at = lt.inserted_at
        WHERE
            lt.external_id = p_workflow_run_id
            AND lt.dag_id IS NOT NULL
    ), final_concurrency_slots_for_dags AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        WHERE
            wcs.strategy_id = p_strategy_id
            AND wcs.workflow_version_id = p_workflow_version_id
            AND wcs.workflow_run_id = p_workflow_run_id
            AND NOT EXISTS (
                -- Check if any task in this DAG has a v1_concurrency_slot
                SELECT 1
                FROM relevant_tasks_for_dags rt
                WHERE EXISTS (
                    SELECT 1
                    FROM v1_concurrency_slot cs2
                    WHERE cs2.task_id = rt.id
                        AND cs2.task_inserted_at = rt.inserted_at
                        AND cs2.task_retry_count = rt.retry_count
                )
            )
            AND CARDINALITY(wcs.child_strategy_ids) <= (
                SELECT COUNT(*)
                FROM relevant_tasks_for_dags rt
            )
        GROUP BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
    ), final_concurrency_slots_for_tasks AS (
        -- If the workflow run id corresponds to a single task, we can safely delete the workflow concurrency slot
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            v1_lookup_table lt ON wcs.workflow_run_id = lt.external_id AND lt.task_id IS NOT NULL
        WHERE
            wcs.strategy_id = p_strategy_id
            AND wcs.workflow_version_id = p_workflow_version_id
            AND wcs.workflow_run_id = p_workflow_run_id
    ), all_parent_slots_to_delete AS (
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            final_concurrency_slots_for_dags
        UNION ALL
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            final_concurrency_slots_for_tasks
    ), locked_parent_slots AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            all_parent_slots_to_delete ps ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (ps.strategy_id, ps.workflow_version_id, ps.workflow_run_id)
        ORDER BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FOR UPDATE
    )
    DELETE FROM
        v1_workflow_concurrency_slot wcs
    WHERE
        (strategy_id, workflow_version_id, workflow_run_id) IN (
            SELECT
                strategy_id,
                workflow_version_id,
                workflow_run_id
            FROM
                locked_parent_slots
        );
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert to previous definition (keeps BIGINT to avoid reintroducing overflow)
CREATE OR REPLACE FUNCTION cleanup_workflow_concurrency_slots(
    p_strategy_id BIGINT,
    p_workflow_version_id UUID,
    p_workflow_run_id UUID
) RETURNS VOID AS $$
DECLARE
    v_sort_id INTEGER;
BEGIN
    SELECT sort_id INTO v_sort_id
    FROM v1_workflow_concurrency_slot
    WHERE strategy_id = p_strategy_id
      AND workflow_version_id = p_workflow_version_id
      AND workflow_run_id = p_workflow_run_id;

    IF v_sort_id IS NULL THEN
        RETURN;
    END IF;

    PERFORM pg_advisory_xact_lock(1000000 * p_strategy_id + v_sort_id);

    WITH final_concurrency_slots_for_dags AS (
        -- If the workflow run id corresponds to a DAG, we get workflow concurrency slots
        -- where NONE of the tasks in the associated DAG have v1_task_runtimes or v1_concurrency_slots
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            v1_lookup_table lt ON wcs.workflow_run_id = lt.external_id AND lt.dag_id IS NOT NULL
        WHERE
            wcs.strategy_id = p_strategy_id
            AND wcs.workflow_version_id = p_workflow_version_id
            AND wcs.workflow_run_id = p_workflow_run_id
            AND NOT EXISTS (
                -- Check if any task in this DAG has a v1_concurrency_slot
                SELECT 1
                FROM v1_dag_to_task dt
                JOIN v1_task t ON dt.task_id = t.id AND dt.task_inserted_at = t.inserted_at
                JOIN v1_concurrency_slot cs2 ON cs2.task_id = t.id AND cs2.task_inserted_at = t.inserted_at AND cs2.task_retry_count = t.retry_count
                WHERE dt.dag_id = lt.dag_id
                AND dt.dag_inserted_at = lt.inserted_at
            )
            AND CARDINALITY(wcs.child_strategy_ids) <= (
                SELECT COUNT(*)
                FROM v1_dag_to_task dt
                JOIN v1_task t ON dt.task_id = t.id AND dt.task_inserted_at = t.inserted_at
                WHERE
                    dt.dag_id = lt.dag_id
                    AND dt.dag_inserted_at = lt.inserted_at
            )
        GROUP BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
    ), final_concurrency_slots_for_tasks AS (
        -- If the workflow run id corresponds to a single task, we can safely delete the workflow concurrency slot
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            v1_lookup_table lt ON wcs.workflow_run_id = lt.external_id AND lt.task_id IS NOT NULL
        WHERE
            wcs.strategy_id = p_strategy_id
            AND wcs.workflow_version_id = p_workflow_version_id
            AND wcs.workflow_run_id = p_workflow_run_id
    ), all_parent_slots_to_delete AS (
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            final_concurrency_slots_for_dags
        UNION ALL
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            final_concurrency_slots_for_tasks
    ), locked_parent_slots AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            all_parent_slots_to_delete ps ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (ps.strategy_id, ps.workflow_version_id, ps.workflow_run_id)
        ORDER BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FOR UPDATE
    )
    DELETE FROM
        v1_workflow_concurrency_slot wcs
    WHERE
        (strategy_id, workflow_version_id, workflow_run_id) IN (
            SELECT
                strategy_id,
                workflow_version_id,
                workflow_run_id
            FROM
                locked_parent_slots
        );
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
