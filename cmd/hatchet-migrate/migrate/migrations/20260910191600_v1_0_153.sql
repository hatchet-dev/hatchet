-- +goose Up
-- +goose StatementBegin
-- Release a parent workflow concurrency slot as soon as no DAG task still
-- holds a child slot. A later step's insert trigger recreates the parent if
-- that step still needs the gate.
CREATE OR REPLACE FUNCTION cleanup_workflow_concurrency_slots(
    p_strategy_id BIGINT,
    p_workflow_version_id UUID,
    p_workflow_run_id UUID
) RETURNS VOID AS $$
DECLARE
    v_sort_id BIGINT;
BEGIN
    SELECT sort_id INTO v_sort_id
    FROM v1_workflow_concurrency_slot
    WHERE strategy_id = p_strategy_id
      AND workflow_version_id = p_workflow_version_id
      AND workflow_run_id = p_workflow_run_id;

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
        GROUP BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
    ), final_concurrency_slots_for_tasks AS (
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
CREATE OR REPLACE FUNCTION cleanup_workflow_concurrency_slots(
    p_strategy_id BIGINT,
    p_workflow_version_id UUID,
    p_workflow_run_id UUID
) RETURNS VOID AS $$
DECLARE
    v_sort_id BIGINT;
BEGIN
    SELECT sort_id INTO v_sort_id
    FROM v1_workflow_concurrency_slot
    WHERE strategy_id = p_strategy_id
      AND workflow_version_id = p_workflow_version_id
      AND workflow_run_id = p_workflow_run_id;

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
