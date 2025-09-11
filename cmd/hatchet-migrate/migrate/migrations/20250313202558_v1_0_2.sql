-- +goose Up
-- +goose StatementBegin

-- When concurrency slot is CREATED, we should check whether the parent concurrency slot exists; if not, we should create
-- the parent concurrency slot as well.
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
        -- If there's a conflict, and we're inserting a new concurrency_slot, we'd like to remove the strategy_id
        -- from the completed child strategy ids.
        SET completed_child_strategy_ids = ARRAY(
            SELECT DISTINCT UNNEST(ARRAY_REMOVE(v1_workflow_concurrency_slot.completed_child_strategy_ids, cs.strategy_id))
            FROM new_table cs
            WHERE EXCLUDED.strategy_id = cs.parent_strategy_id
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
    ), locked_parent_slots AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id,
            cs.strategy_id AS child_strategy_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            parent_slot cs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
        ORDER BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FOR UPDATE
    )
    UPDATE v1_workflow_concurrency_slot wcs
    SET completed_child_strategy_ids = ARRAY(
        SELECT DISTINCT UNNEST(ARRAY_APPEND(wcs.completed_child_strategy_ids, cs.child_strategy_id))
    )
    FROM locked_parent_slots cs
    WHERE
        wcs.strategy_id = cs.strategy_id
        AND wcs.workflow_version_id = cs.workflow_version_id
        AND wcs.workflow_run_id = cs.workflow_run_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
