-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Step"
    ADD COLUMN IF NOT EXISTS "isDurable" BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS v1_worker_slot_config (
    tenant_id UUID NOT NULL,
    worker_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    max_units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, worker_id, slot_type)
);

CREATE TABLE IF NOT EXISTS v1_step_slot_request (
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, step_id, slot_type)
);

CREATE TABLE IF NOT EXISTS v1_task_runtime_slot (
    tenant_id UUID NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID NOT NULL,
    -- slot_type is user defined, we use default and durable internally as defaults
    slot_type TEXT NOT NULL,
    units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, task_inserted_at, retry_count, slot_type)
);

-- Compatibility triggers for blue/green: keep new slot tables updated from the old write paths.

CREATE OR REPLACE FUNCTION v1_worker_slot_config_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_worker_slot_config (tenant_id, worker_id, slot_type, max_units)
    SELECT
        "tenantId",
        "id",
        'default'::text,
        "maxRuns"
    FROM new_rows
    WHERE "maxRuns" IS NOT NULL
    ON CONFLICT (tenant_id, worker_id, slot_type) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS v1_worker_slot_config_insert_trigger ON "Worker";

CREATE TRIGGER v1_worker_slot_config_insert_trigger
AFTER INSERT ON "Worker"
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_worker_slot_config_insert_function();

CREATE OR REPLACE FUNCTION v1_step_slot_request_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_step_slot_request (tenant_id, step_id, slot_type, units)
    SELECT
        "tenantId",
        "id",
        CASE WHEN "isDurable" THEN 'durable'::text ELSE 'default'::text END,
        1
    FROM new_rows
    ON CONFLICT (tenant_id, step_id, slot_type) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS v1_step_slot_request_insert_trigger ON "Step";

CREATE TRIGGER v1_step_slot_request_insert_trigger
AFTER INSERT ON "Step"
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_step_slot_request_insert_function();

CREATE OR REPLACE FUNCTION v1_task_runtime_slot_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_task_runtime_slot (
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        slot_type,
        units
    )
    SELECT
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        'default'::text,
        1
    FROM new_rows nr
    WHERE nr.worker_id IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM v1_task_runtime_slot s
        WHERE s.task_id = nr.task_id
          AND s.task_inserted_at = nr.task_inserted_at
          AND s.retry_count = nr.retry_count
    )
    ON CONFLICT (task_id, task_inserted_at, retry_count, slot_type) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS v1_task_runtime_slot_insert_trigger ON v1_task_runtime;

CREATE TRIGGER v1_task_runtime_slot_insert_trigger
AFTER INSERT ON v1_task_runtime
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_task_runtime_slot_insert_function();

CREATE OR REPLACE FUNCTION v1_task_runtime_slot_delete_function()
RETURNS TRIGGER AS
$$
BEGIN
    DELETE FROM v1_task_runtime_slot s
    USING deleted_rows d
    WHERE s.task_id = d.task_id
      AND s.task_inserted_at = d.task_inserted_at
      AND s.retry_count = d.retry_count;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS v1_task_runtime_slot_delete_trigger ON v1_task_runtime;

CREATE TRIGGER v1_task_runtime_slot_delete_trigger
AFTER DELETE ON v1_task_runtime
REFERENCING OLD TABLE AS deleted_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_task_runtime_slot_delete_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS v1_worker_slot_config_insert_trigger ON "Worker";
DROP FUNCTION IF EXISTS v1_worker_slot_config_insert_function();

DROP TRIGGER IF EXISTS v1_step_slot_request_insert_trigger ON "Step";
DROP FUNCTION IF EXISTS v1_step_slot_request_insert_function();

DROP TRIGGER IF EXISTS v1_task_runtime_slot_insert_trigger ON v1_task_runtime;
DROP FUNCTION IF EXISTS v1_task_runtime_slot_insert_function();

DROP TRIGGER IF EXISTS v1_task_runtime_slot_delete_trigger ON v1_task_runtime;
DROP FUNCTION IF EXISTS v1_task_runtime_slot_delete_function();

DROP TABLE IF EXISTS v1_task_runtime_slot;
DROP TABLE IF EXISTS v1_step_slot_request;
DROP TABLE IF EXISTS v1_worker_slot_config;

ALTER TABLE "Step"
    DROP COLUMN IF EXISTS "isDurable";
-- +goose StatementEnd
