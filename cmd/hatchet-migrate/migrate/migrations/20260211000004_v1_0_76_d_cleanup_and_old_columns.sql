-- +goose Up
-- NOTE: this must be run after old version is destroyed and new version is deployed
-- +goose StatementBegin
DROP TRIGGER IF EXISTS v1_worker_slot_config_insert_trigger ON "Worker";
DROP FUNCTION IF EXISTS v1_worker_slot_config_insert_function();

DROP TRIGGER IF EXISTS v1_step_slot_request_insert_trigger ON "Step";
DROP FUNCTION IF EXISTS v1_step_slot_request_insert_function();

DROP TRIGGER IF EXISTS v1_task_runtime_slot_insert_trigger ON v1_task_runtime;
DROP FUNCTION IF EXISTS v1_task_runtime_slot_insert_function();

DROP TRIGGER IF EXISTS v1_task_runtime_slot_delete_trigger ON v1_task_runtime;
DROP FUNCTION IF EXISTS v1_task_runtime_slot_delete_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
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
    FROM new_rows
    WHERE worker_id IS NOT NULL
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
