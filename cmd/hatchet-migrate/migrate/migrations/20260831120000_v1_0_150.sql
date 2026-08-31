-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_step_concurrency ADD COLUMN tenant_strategy_id BIGINT;

-- Tenant-scoped concurrency strategies, registered independently of any workflow and
-- referenced by steps via v1_step_concurrency.tenant_strategy_id so tasks across different
-- workflows consume the same concurrency limit.
CREATE TABLE v1_tenant_concurrency (
    -- Draws from v1_step_concurrency's identity sequence so strategy ids are unique across
    -- both tables: concurrency slots, outbox topics, leases, and advisory locks all key on
    -- the bare strategy id. The borrow is a runtime name lookup with no recorded catalog
    -- dependency; see the warning on v1_step_concurrency.id before changing either side.
    id bigint NOT NULL DEFAULT nextval(pg_get_serial_sequence('v1_step_concurrency', 'id')),
    tenant_id UUID NOT NULL,
    -- The strategy name, unique per tenant; steps reference the strategy by name at
    -- registration time.
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    -- last_active_at is refreshed at most once per hour when a new slot is inserted for this strategy.
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    strategy v1_concurrency_strategy NOT NULL,
    expression TEXT NOT NULL,
    max_concurrency INTEGER NOT NULL,
    CONSTRAINT v1_tenant_concurrency_pkey PRIMARY KEY (id),
    CONSTRAINT v1_tenant_concurrency_tenant_name_ux UNIQUE (tenant_id, name)
);
-- +goose StatementEnd

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
            OR strategy.last_active_at < NOW() - INTERVAL '1 hour'
        ORDER BY
            strategy.id
        FOR UPDATE
    )
    UPDATE v1_step_concurrency strategy
    SET is_active = TRUE, last_active_at = NOW()
    FROM inactive_strategies
    WHERE
        strategy.workflow_id = inactive_strategies.workflow_id AND
        strategy.workflow_version_id = inactive_strategies.workflow_version_id AND
        strategy.step_id = inactive_strategies.step_id AND
        strategy.id = inactive_strategies.id;

    -- Same reactivation for tenant-scoped strategies: their slots carry the tenant
    -- strategy's id, which (by the shared id sequence) never matches a step strategy.
    WITH inactive_tenant_strategies AS (
        SELECT
            strategy.*
        FROM
            new_table cs
        JOIN
            v1_tenant_concurrency strategy ON strategy.id = cs.strategy_id
        WHERE
            strategy.is_active = FALSE
            OR strategy.last_active_at < NOW() - INTERVAL '1 hour'
        ORDER BY
            strategy.id
        FOR UPDATE
    )
    UPDATE v1_tenant_concurrency strategy
    SET is_active = TRUE, last_active_at = NOW()
    FROM inactive_tenant_strategies
    WHERE
        strategy.id = inactive_tenant_strategies.id;

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
            OR strategy.last_active_at < NOW() - INTERVAL '1 hour'
        ORDER BY
            strategy.id
        FOR UPDATE
    )
    UPDATE v1_step_concurrency strategy
    SET is_active = TRUE, last_active_at = NOW()
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

-- +goose StatementBegin
DROP TABLE IF EXISTS v1_tenant_concurrency;

ALTER TABLE v1_step_concurrency DROP COLUMN tenant_strategy_id;
-- +goose StatementEnd
