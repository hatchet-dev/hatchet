-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_insert_function()
RETURNS trigger AS $$
BEGIN
    WITH parent_slot AS (
        SELECT
            *
        FROM
            new_table cs
        WHERE
            cs.parent_strategy_id IS NOT NULL
    ), parent_to_child_strategy_ids AS (
        SELECT
            wc.id AS parent_strategy_id,
            wc.tenant_id,
            ps.workflow_id,
            ps.workflow_version_id,
            ps.workflow_run_id,
            MAX(ps.sort_id) AS sort_id,
            MAX(ps.priority) AS priority,
            MAX(ps.key) AS key,
            ARRAY_AGG(DISTINCT wc.child_strategy_ids) AS child_strategy_ids
        FROM
            parent_slot ps
        JOIN v1_workflow_concurrency wc ON wc.workflow_id = ps.workflow_id AND wc.workflow_version_id = ps.workflow_version_id AND wc.id = ps.parent_strategy_id
        GROUP BY
            wc.id,
            wc.tenant_id,
            ps.workflow_id,
            ps.workflow_version_id,
            ps.workflow_run_id
    )
    INSERT INTO v1_workflow_concurrency_slot (
        sort_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        strategy_id,
        child_strategy_ids,
        priority,
        key
    )
    SELECT
        pcs.sort_id,
        pcs.tenant_id,
        pcs.workflow_id,
        pcs.workflow_version_id,
        pcs.workflow_run_id,
        pcs.parent_strategy_id,
        pcs.child_strategy_ids,
        pcs.priority,
        pcs.key
    FROM
        parent_to_child_strategy_ids pcs
    ON CONFLICT (strategy_id, workflow_version_id, workflow_run_id) DO NOTHING;

    -- If the v1_step_concurrency strategy is not active, we set it to active.
    WITH inactive_strategies AS (
        SELECT
            strategy.*
        FROM
            new_table cs
        JOIN
            v1_step_concurrency strategy ON strategy.workflow_id = cs.workflow_id AND strategy.workflow_version_id = cs.workflow_version_id AND strategy.id = cs.strategy_id
        WHERE
            strategy.is_active = FALSE
        ORDER BY
            strategy.id
        FOR UPDATE
    )
    UPDATE v1_step_concurrency strategy
    SET is_active = TRUE
    FROM inactive_strategies
    WHERE
        strategy.workflow_id = inactive_strategies.workflow_id AND
        strategy.workflow_version_id = inactive_strategies.workflow_version_id AND
        strategy.step_id = inactive_strategies.step_id AND
        strategy.id = inactive_strategies.id;

    RETURN NULL;
END;

$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_workflow_concurrency_slots(
    p_strategy_id BIGINT,
    p_workflow_version_id UUID,
    p_workflow_run_id UUID
) RETURNS VOID AS $$
DECLARE
    v_sort_id INTEGER;
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

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_function()
RETURNS trigger AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT * FROM deleted_rows ORDER BY parent_strategy_id, workflow_version_id, workflow_run_id LOOP
        IF rec.parent_strategy_id IS NOT NULL THEN
            PERFORM cleanup_workflow_concurrency_slots(
                rec.parent_strategy_id,
                rec.workflow_version_id,
                rec.workflow_run_id
            );
        END IF;
    END LOOP;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
    WITH new_slot_rows AS (
        SELECT
            id,
            inserted_at,
            retry_count,
            tenant_id,
            priority,
            concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(concurrency_parent_strategy_ids, 1) > 1 THEN concurrency_parent_strategy_ids[2:array_length(concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            concurrency_strategy_ids[1] AS strategy_id,
            external_id,
            workflow_run_id,
            CASE
                WHEN array_length(concurrency_strategy_ids, 1) > 1 THEN concurrency_strategy_ids[2:array_length(concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            concurrency_keys[1] AS key,
            CASE
                WHEN array_length(concurrency_keys, 1) > 1 THEN concurrency_keys[2:array_length(concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            workflow_id,
            workflow_version_id,
            queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout) AS schedule_timeout_at
        FROM new_table
        WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        queue_to_notify,
        schedule_timeout_at
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        COALESCE(priority, 1),
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM new_slot_rows;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count
    )
    SELECT
        tenant_id,
        queue,
        id,
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout),
        step_timeout,
        COALESCE(priority, 1),
        sticky,
        desired_worker_id,
        retry_count
    FROM new_table
    WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NULL;

    INSERT INTO v1_dag_to_task (
        dag_id,
        dag_inserted_at,
        task_id,
        task_inserted_at
    )
    SELECT
        dag_id,
        dag_inserted_at,
        id,
        inserted_at
    FROM new_table
    WHERE dag_id IS NOT NULL AND dag_inserted_at IS NOT NULL;

    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        task_id,
        inserted_at
    )
    SELECT
        external_id,
        tenant_id,
        id,
        inserted_at
    FROM new_table
    ON CONFLICT (external_id) DO NOTHING;

    -- NOTE: this comes after the insert into v1_dag_to_task and v1_lookup_table, because we case on these tables for cleanup
    FOR rec IN SELECT UNNEST(concurrency_parent_strategy_ids) AS parent_strategy_id, workflow_version_id, workflow_run_id FROM new_table WHERE initial_state != 'QUEUED' AND concurrency_parent_strategy_ids[1] IS NOT NULL ORDER BY parent_strategy_id, workflow_version_id, workflow_run_id LOOP
        IF rec.parent_strategy_id IS NOT NULL THEN
            PERFORM cleanup_workflow_concurrency_slots(
                rec.parent_strategy_id,
                rec.workflow_version_id,
                rec.workflow_run_id
            );
        END IF;
    END LOOP;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_insert_function()
RETURNS trigger AS $$
BEGIN
    WITH parent_slot AS (
        SELECT
            *
        FROM
            new_table cs
        WHERE
            cs.parent_strategy_id IS NOT NULL
    ), parent_to_child_strategy_ids AS (
        SELECT
            wc.id AS parent_strategy_id,
            wc.tenant_id,
            ps.workflow_id,
            ps.workflow_version_id,
            ps.workflow_run_id,
            MAX(ps.sort_id) AS sort_id,
            MAX(ps.priority) AS priority,
            MAX(ps.key) AS key,
            ARRAY_AGG(DISTINCT wc.child_strategy_ids) AS child_strategy_ids
        FROM
            parent_slot ps
        JOIN v1_workflow_concurrency wc ON wc.workflow_id = ps.workflow_id AND wc.workflow_version_id = ps.workflow_version_id AND wc.id = ps.parent_strategy_id
        GROUP BY
            wc.id,
            wc.tenant_id,
            ps.workflow_id,
            ps.workflow_version_id,
            ps.workflow_run_id
    )
    INSERT INTO v1_workflow_concurrency_slot (
        sort_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        strategy_id,
        child_strategy_ids,
        priority,
        key
    )
    SELECT
        pcs.sort_id,
        pcs.tenant_id,
        pcs.workflow_id,
        pcs.workflow_version_id,
        pcs.workflow_run_id,
        pcs.parent_strategy_id,
        pcs.child_strategy_ids,
        pcs.priority,
        pcs.key
    FROM
        parent_to_child_strategy_ids pcs
    ON CONFLICT (strategy_id, workflow_version_id, workflow_run_id) DO UPDATE
        -- When concurrency slots are INSERTED:
        -- We need to REMOVE their strategy_ids from the parent's completed_child_strategy_ids
        -- because these child slots are now active again, not completed
        -- This correctly handles bulk inserts by removing ALL strategy_ids in the current batch
        SET completed_child_strategy_ids = ARRAY(
            SELECT DISTINCT e
            FROM UNNEST(v1_workflow_concurrency_slot.completed_child_strategy_ids) AS e
            WHERE e NOT IN (
                SELECT strategy_id
                FROM new_table
                WHERE parent_strategy_id = EXCLUDED.strategy_id
            )
        );

    -- If the v1_step_concurrency strategy is not active, we set it to active.
    WITH inactive_strategies AS (
        SELECT
            strategy.*
        FROM
            new_table cs
        JOIN
            v1_step_concurrency strategy ON strategy.workflow_id = cs.workflow_id AND strategy.workflow_version_id = cs.workflow_version_id AND strategy.id = cs.strategy_id
        WHERE
            strategy.is_active = FALSE
        ORDER BY
            strategy.id
        FOR UPDATE
    )
    UPDATE v1_step_concurrency strategy
    SET is_active = TRUE
    FROM inactive_strategies
    WHERE
        strategy.workflow_id = inactive_strategies.workflow_id AND
        strategy.workflow_version_id = inactive_strategies.workflow_version_id AND
        strategy.step_id = inactive_strategies.step_id AND
        strategy.id = inactive_strategies.id;

    RETURN NULL;
END;

$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_function()
RETURNS trigger AS $$
BEGIN
    -- When v1_concurrency_slot is DELETED, we add it to the completed_child_strategy_ids on the parent, but only
    -- when it's NOT a backoff retry. Backoff retries will continue to consume a workflow concurrency slot.
    WITH parent_slot AS (
        SELECT
            cs.workflow_id,
            cs.workflow_version_id,
            cs.workflow_run_id,
            cs.strategy_id,
            cs.parent_strategy_id
        FROM
            deleted_rows cs
        JOIN
            v1_task t ON t.id = cs.task_id AND t.inserted_at = cs.task_inserted_at
        LEFT JOIN
            v1_retry_queue_item rqi ON rqi.task_id = t.id AND rqi.task_inserted_at = t.inserted_at
        WHERE
            cs.parent_strategy_id IS NOT NULL
            AND rqi.task_id IS NULL
    ), parent_slots_grouped AS (
        SELECT
            cs.parent_strategy_id,
            cs.workflow_version_id,
            cs.workflow_run_id,
            ARRAY_AGG(cs.strategy_id) AS child_strategy_ids
        FROM
            parent_slot cs
        GROUP BY
            cs.parent_strategy_id,
            cs.workflow_version_id,
            cs.workflow_run_id
    ), locked_parent_slots AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id,
            psg.child_strategy_ids
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            parent_slots_grouped psg ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) =
                                       (psg.parent_strategy_id, psg.workflow_version_id, psg.workflow_run_id)
        ORDER BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FOR UPDATE
    )
    UPDATE v1_workflow_concurrency_slot wcs
    SET completed_child_strategy_ids = ARRAY(
        -- When concurrency slots are DELETED:
        -- We need to ADD their strategy_ids to the parent's completed_child_strategy_ids
        -- This correctly handles bulk deletes by adding ALL strategy_ids in the current batch
        SELECT DISTINCT e
        FROM (
            SELECT UNNEST(wcs.completed_child_strategy_ids) AS e
            UNION
            SELECT UNNEST(lps.child_strategy_ids)
            FROM locked_parent_slots lps
            WHERE lps.strategy_id = wcs.strategy_id
              AND lps.workflow_version_id = wcs.workflow_version_id
              AND lps.workflow_run_id = wcs.workflow_run_id
        ) AS subquery
    )
    FROM locked_parent_slots cs
    WHERE
        wcs.strategy_id = cs.strategy_id
        AND wcs.workflow_version_id = cs.workflow_version_id
        AND wcs.workflow_run_id = cs.workflow_run_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT * FROM new_table WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL AND concurrency_keys[1] IS NULL LOOP
        RAISE WARNING 'New table row: %', row_to_json(rec);
    END LOOP;

    -- When a task is inserted in a non-queued state, we should add all relevant completed_child_strategy_ids to the parent
    -- concurrency slots.
    WITH parent_slots AS (
        SELECT
            nt.workflow_id,
            nt.workflow_version_id,
            nt.workflow_run_id,
            UNNEST(nt.concurrency_strategy_ids) AS strategy_id,
            UNNEST(nt.concurrency_parent_strategy_ids) AS parent_strategy_id
        FROM
            new_table nt
        WHERE
            cardinality(nt.concurrency_parent_strategy_ids) > 0
            AND nt.initial_state != 'QUEUED'
    ), locked_parent_slots AS (
        SELECT
            wcs.workflow_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id,
            wcs.strategy_id,
            cs.strategy_id AS child_strategy_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            parent_slots cs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
        ORDER BY
            wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
        FOR UPDATE
    )
    UPDATE
        v1_workflow_concurrency_slot wcs
    SET
        -- get unique completed_child_strategy_ids after append with cs.strategy_id
        completed_child_strategy_ids = ARRAY(
            SELECT
                DISTINCT UNNEST(ARRAY_APPEND(wcs.completed_child_strategy_ids, cs.child_strategy_id))
        )
    FROM
        locked_parent_slots cs
    WHERE
        wcs.strategy_id = cs.strategy_id
        AND wcs.workflow_version_id = cs.workflow_version_id
        AND wcs.workflow_run_id = cs.workflow_run_id;

    WITH new_slot_rows AS (
        SELECT
            id,
            inserted_at,
            retry_count,
            tenant_id,
            priority,
            concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(concurrency_parent_strategy_ids, 1) > 1 THEN concurrency_parent_strategy_ids[2:array_length(concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            concurrency_strategy_ids[1] AS strategy_id,
            external_id,
            workflow_run_id,
            CASE
                WHEN array_length(concurrency_strategy_ids, 1) > 1 THEN concurrency_strategy_ids[2:array_length(concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            concurrency_keys[1] AS key,
            CASE
                WHEN array_length(concurrency_keys, 1) > 1 THEN concurrency_keys[2:array_length(concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            workflow_id,
            workflow_version_id,
            queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout) AS schedule_timeout_at
        FROM new_table
        WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        queue_to_notify,
        schedule_timeout_at
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        COALESCE(priority, 1),
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM new_slot_rows;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count
    )
    SELECT
        tenant_id,
        queue,
        id,
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout),
        step_timeout,
        COALESCE(priority, 1),
        sticky,
        desired_worker_id,
        retry_count
    FROM new_table
    WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NULL;

    INSERT INTO v1_dag_to_task (
        dag_id,
        dag_inserted_at,
        task_id,
        task_inserted_at
    )
    SELECT
        dag_id,
        dag_inserted_at,
        id,
        inserted_at
    FROM new_table
    WHERE dag_id IS NOT NULL AND dag_inserted_at IS NOT NULL;

    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        task_id,
        inserted_at
    )
    SELECT
        external_id,
        tenant_id,
        id,
        inserted_at
    FROM new_table
    ON CONFLICT (external_id) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP FUNCTION IF EXISTS cleanup_workflow_concurrency_slots(
    p_strategy_id BIGINT,
    p_workflow_version_id UUID,
    p_workflow_run_id UUID
);

-- +goose StatementEnd
