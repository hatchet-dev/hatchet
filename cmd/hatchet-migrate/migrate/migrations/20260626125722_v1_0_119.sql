-- +goose Up
-- +goose StatementBegin

ALTER TABLE v1_step_concurrency
    ADD COLUMN last_active TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

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

    WITH inactive_strategies AS (
        SELECT
            strategy.*
        FROM
            new_table cs
        JOIN
            v1_step_concurrency strategy ON strategy.workflow_id = cs.workflow_id AND strategy.workflow_version_id = cs.workflow_version_id AND strategy.id = cs.strategy_id
        WHERE
            strategy.is_active = FALSE
            OR strategy.last_active < NOW() - INTERVAL '1 hour'
        ORDER BY
            strategy.id
        FOR UPDATE
    )
    UPDATE v1_step_concurrency strategy
    SET is_active = TRUE, last_active = NOW()
    FROM inactive_strategies
    WHERE
        strategy.workflow_id = inactive_strategies.workflow_id AND
        strategy.workflow_version_id = inactive_strategies.workflow_version_id AND
        strategy.step_id = inactive_strategies.step_id AND
        strategy.id = inactive_strategies.id;

    RETURN NULL;
END;

$$ LANGUAGE plpgsql;

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
    ON CONFLICT (strategy_id, workflow_version_id, workflow_run_id) DO NOTHING;

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

ALTER TABLE v1_step_concurrency DROP COLUMN last_active;

-- +goose StatementEnd
