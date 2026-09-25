-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    current_month_start DATE := date_trunc('month', NOW())::DATE;
    next_month_start DATE := (date_trunc('month', NOW()) + INTERVAL '1 month')::DATE;
    legacy_partition_name TEXT := 'v1_lookup_table_olap_' || to_char(current_month_start, 'YYYYMMDD');
BEGIN
    EXECUTE format('ALTER TABLE v1_lookup_table_olap RENAME CONSTRAINT v1_lookup_table_olap_external_id_inserted_at_uq TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER TABLE v1_lookup_table_olap RENAME CONSTRAINT v1_lookup_table_olap_pkey TO %I', legacy_partition_name || '_external_id_uq');
    EXECUTE format('ALTER TABLE v1_lookup_table_olap RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_lookup_table_olap_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, next_month_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_lookup_table_olap_attach_bound', legacy_partition_name);
END $$;

ALTER TABLE v1_lookup_table_olap_partitioned RENAME TO v1_lookup_table_olap;
ALTER INDEX v1_lookup_table_olap_partitioned_pkey RENAME TO v1_lookup_table_olap_pkey;

DO $$
DECLARE
    next_month_start DATE := (date_trunc('month', NOW()) + INTERVAL '1 month')::DATE;
    next_month_partition_name TEXT := 'v1_lookup_table_olap_' || to_char(next_month_start, 'YYYYMMDD');
BEGIN
    PERFORM create_v1_monthly_range_partition('v1_lookup_table_olap', next_month_start);
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I UNIQUE (external_id)', next_month_partition_name, next_month_partition_name || '_external_id_uq');
END $$;

CREATE OR REPLACE FUNCTION v1_tasks_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'TASK',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    FROM new_rows
    WHERE dag_id IS NULL
    ON CONFLICT (inserted_at, id) DO NOTHING;

    INSERT INTO v1_lookup_table_olap (
        tenant_id,
        external_id,
        task_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    -- If the task has a dag_id and dag_inserted_at, insert into the lookup table
    INSERT INTO v1_dag_to_task_olap (
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
    FROM new_rows
    WHERE dag_id IS NOT NULL
    ON CONFLICT (dag_id, dag_inserted_at, task_id, task_inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION v1_dags_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'DAG',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    FROM new_rows
    ON CONFLICT (inserted_at, id) DO NOTHING;

    INSERT INTO v1_lookup_table_olap (
        tenant_id,
        external_id,
        dag_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    today_start DATE := (NOW() AT TIME ZONE 'UTC')::DATE;
    tomorrow_start DATE := today_start + 1;
    legacy_partition_name TEXT := 'v1_dag_to_task_olap_' || to_char(today_start, 'YYYYMMDD');
BEGIN
    EXECUTE format('ALTER TABLE v1_dag_to_task_olap RENAME CONSTRAINT v1_dag_to_task_olap_pkey TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER TABLE v1_dag_to_task_olap RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_dag_to_task_olap_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, tomorrow_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_dag_to_task_olap_attach_bound', legacy_partition_name);
END $$;

ALTER TABLE v1_dag_to_task_olap_partitioned RENAME TO v1_dag_to_task_olap;
ALTER INDEX v1_dag_to_task_olap_partitioned_pkey RENAME TO v1_dag_to_task_olap_pkey;

SELECT create_v1_range_partition('v1_dag_to_task_olap', ((NOW() AT TIME ZONE 'UTC')::DATE + 1));
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    current_month_start DATE := date_trunc('month', NOW())::DATE;
    next_month_start DATE := (date_trunc('month', NOW()) + INTERVAL '1 month')::DATE;
    legacy_partition_name TEXT := 'v1_statuses_olap_' || to_char(current_month_start, 'YYYYMMDD');
BEGIN
    EXECUTE format('ALTER TABLE v1_statuses_olap RENAME CONSTRAINT v1_statuses_olap_pkey TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER TABLE v1_statuses_olap RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_statuses_olap_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, next_month_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_statuses_olap_attach_bound', legacy_partition_name);
END $$;

ALTER TABLE v1_statuses_olap_partitioned RENAME TO v1_statuses_olap;
ALTER INDEX v1_statuses_olap_partitioned_pkey RENAME TO v1_statuses_olap_pkey;

SELECT create_v1_monthly_range_partition('v1_statuses_olap', (date_trunc('month', NOW()) + INTERVAL '1 month')::DATE);
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    today_start DATE := (NOW() AT TIME ZONE 'UTC')::DATE;
    tomorrow_start DATE := today_start + 1;
    legacy_partition_name TEXT := 'v1_task_events_olap_' || to_char(today_start, 'YYYYMMDD');
    next_id BIGINT;
BEGIN
    SELECT CASE WHEN is_called THEN last_value + 1 ELSE last_value END INTO next_id FROM v1_task_events_olap_id_seq;

    -- the parent's identity assigns ids for every partition, so the legacy table's own identity
    -- (and its sequence, which holds the name the parent's sequence takes below) goes away
    ALTER TABLE v1_task_events_olap ALTER COLUMN id DROP IDENTITY;

    EXECUTE format('ALTER TABLE v1_task_events_olap RENAME CONSTRAINT v1_task_events_olap_pkey TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER INDEX v1_task_events_olap_task_id_idx RENAME TO %I', legacy_partition_name || '_task_id_idx');
    EXECUTE format('ALTER TABLE v1_task_events_olap RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_task_events_olap_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, tomorrow_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_task_events_olap_attach_bound', legacy_partition_name);

    ALTER TABLE v1_task_events_olap_partitioned RENAME TO v1_task_events_olap;
    ALTER INDEX v1_task_events_olap_partitioned_pkey RENAME TO v1_task_events_olap_pkey;
    ALTER INDEX v1_task_events_olap_partitioned_task_id_idx RENAME TO v1_task_events_olap_task_id_idx;

    ALTER SEQUENCE v1_task_events_olap_partitioned_id_seq RENAME TO v1_task_events_olap_id_seq;
    EXECUTE format('ALTER SEQUENCE v1_task_events_olap_id_seq RESTART WITH %s', next_id);
END $$;

SELECT create_v1_range_partition('v1_task_events_olap', ((NOW() AT TIME ZONE 'UTC')::DATE + 1));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    legacy_partition_name TEXT;
    next_id BIGINT;
BEGIN
    IF (SELECT relkind FROM pg_class WHERE oid = 'v1_task_events_olap'::regclass) <> 'p' THEN
        RETURN;
    END IF;

    SELECT CASE WHEN is_called THEN last_value + 1 ELSE last_value END INTO next_id FROM v1_task_events_olap_id_seq;

    SELECT c.relname
    INTO legacy_partition_name
    FROM pg_inherits i
    JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'v1_task_events_olap'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_task_events_olap_original (
            tenant_id UUID NOT NULL,
            id BIGINT NOT NULL,
            inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
            external_id UUID NOT NULL DEFAULT gen_random_uuid(),
            task_id BIGINT NOT NULL,
            task_inserted_at TIMESTAMPTZ NOT NULL,
            event_type v1_event_type_olap NOT NULL,
            workflow_id UUID NOT NULL,
            event_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
            readable_status v1_readable_status_olap NOT NULL,
            retry_count INT NOT NULL DEFAULT 0,
            error_message TEXT,
            output JSONB,
            worker_id UUID,
            additional__event_data TEXT,
            additional__event_message TEXT,
            durable_invocation_count INT NOT NULL DEFAULT 0,
            PRIMARY KEY (task_id, task_inserted_at, id)
        );

        INSERT INTO v1_task_events_olap_original (tenant_id, id, inserted_at, external_id, task_id, task_inserted_at, event_type, workflow_id, event_timestamp, readable_status, retry_count, error_message, output, worker_id, additional__event_data, additional__event_message, durable_invocation_count)
        SELECT tenant_id, id, inserted_at, external_id, task_id, task_inserted_at, event_type, workflow_id, event_timestamp, readable_status, retry_count, error_message, output, worker_id, additional__event_data, additional__event_message, durable_invocation_count FROM v1_task_events_olap;

        DROP TABLE v1_task_events_olap;
        ALTER TABLE v1_task_events_olap_original RENAME TO v1_task_events_olap;
        ALTER INDEX v1_task_events_olap_original_pkey RENAME TO v1_task_events_olap_pkey;
    ELSE
        EXECUTE format('ALTER TABLE v1_task_events_olap DETACH PARTITION %I', legacy_partition_name);
        EXECUTE format('INSERT INTO %I (tenant_id, id, inserted_at, external_id, task_id, task_inserted_at, event_type, workflow_id, event_timestamp, readable_status, retry_count, error_message, output, worker_id, additional__event_data, additional__event_message, durable_invocation_count) SELECT tenant_id, id, inserted_at, external_id, task_id, task_inserted_at, event_type, workflow_id, event_timestamp, readable_status, retry_count, error_message, output, worker_id, additional__event_data, additional__event_message, durable_invocation_count FROM v1_task_events_olap ON CONFLICT DO NOTHING', legacy_partition_name);
        DROP TABLE v1_task_events_olap;
        EXECUTE format('ALTER TABLE %I RENAME TO v1_task_events_olap', legacy_partition_name);
        EXECUTE format('ALTER TABLE v1_task_events_olap RENAME CONSTRAINT %I TO v1_task_events_olap_pkey', legacy_partition_name || '_pkey');
        EXECUTE format('ALTER INDEX %I RENAME TO v1_task_events_olap_task_id_idx', legacy_partition_name || '_task_id_idx');
    END IF;

    EXECUTE format('ALTER TABLE v1_task_events_olap ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (START WITH %s)', next_id);
    CREATE INDEX IF NOT EXISTS v1_task_events_olap_task_id_idx ON v1_task_events_olap (task_id);
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    legacy_partition_name TEXT;
BEGIN
    IF (SELECT relkind FROM pg_class WHERE oid = 'v1_statuses_olap'::regclass) <> 'p' THEN
        RETURN;
    END IF;

    SELECT c.relname
    INTO legacy_partition_name
    FROM pg_inherits i
    JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'v1_statuses_olap'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_statuses_olap_original (
            external_id UUID NOT NULL,
            inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
            tenant_id UUID NOT NULL,
            workflow_id UUID NOT NULL,
            kind v1_run_kind NOT NULL,
            readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
            PRIMARY KEY (external_id, inserted_at)
        );

        INSERT INTO v1_statuses_olap_original (external_id, inserted_at, tenant_id, workflow_id, kind, readable_status)
        SELECT external_id, inserted_at, tenant_id, workflow_id, kind, readable_status FROM v1_statuses_olap;

        DROP TABLE v1_statuses_olap;
        ALTER TABLE v1_statuses_olap_original RENAME TO v1_statuses_olap;
        ALTER INDEX v1_statuses_olap_original_pkey RENAME TO v1_statuses_olap_pkey;
        RETURN;
    END IF;

    EXECUTE format('ALTER TABLE v1_statuses_olap DETACH PARTITION %I', legacy_partition_name);
    EXECUTE format('INSERT INTO %I (external_id, inserted_at, tenant_id, workflow_id, kind, readable_status) SELECT external_id, inserted_at, tenant_id, workflow_id, kind, readable_status FROM v1_statuses_olap ON CONFLICT DO NOTHING', legacy_partition_name);
    DROP TABLE v1_statuses_olap;
    EXECUTE format('ALTER TABLE %I RENAME TO v1_statuses_olap', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_statuses_olap RENAME CONSTRAINT %I TO v1_statuses_olap_pkey', legacy_partition_name || '_pkey');
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    legacy_partition_name TEXT;
BEGIN
    IF (SELECT relkind FROM pg_class WHERE oid = 'v1_dag_to_task_olap'::regclass) <> 'p' THEN
        RETURN;
    END IF;

    SELECT c.relname
    INTO legacy_partition_name
    FROM pg_inherits i
    JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'v1_dag_to_task_olap'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_dag_to_task_olap_original (
            dag_id BIGINT NOT NULL,
            dag_inserted_at TIMESTAMPTZ NOT NULL,
            task_id BIGINT NOT NULL,
            task_inserted_at TIMESTAMPTZ NOT NULL,
            PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
        );

        INSERT INTO v1_dag_to_task_olap_original (dag_id, dag_inserted_at, task_id, task_inserted_at)
        SELECT dag_id, dag_inserted_at, task_id, task_inserted_at FROM v1_dag_to_task_olap;

        DROP TABLE v1_dag_to_task_olap;
        ALTER TABLE v1_dag_to_task_olap_original RENAME TO v1_dag_to_task_olap;
        ALTER INDEX v1_dag_to_task_olap_original_pkey RENAME TO v1_dag_to_task_olap_pkey;
        RETURN;
    END IF;

    EXECUTE format('ALTER TABLE v1_dag_to_task_olap DETACH PARTITION %I', legacy_partition_name);
    EXECUTE format('INSERT INTO %I (dag_id, dag_inserted_at, task_id, task_inserted_at) SELECT dag_id, dag_inserted_at, task_id, task_inserted_at FROM v1_dag_to_task_olap ON CONFLICT DO NOTHING', legacy_partition_name);
    DROP TABLE v1_dag_to_task_olap;
    EXECUTE format('ALTER TABLE %I RENAME TO v1_dag_to_task_olap', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_dag_to_task_olap RENAME CONSTRAINT %I TO v1_dag_to_task_olap_pkey', legacy_partition_name || '_pkey');
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    legacy_partition_name TEXT;
BEGIN
    IF (SELECT relkind FROM pg_class WHERE oid = 'v1_lookup_table_olap'::regclass) <> 'p' THEN
        RETURN;
    END IF;

    SELECT c.relname
    INTO legacy_partition_name
    FROM pg_inherits i
    JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'v1_lookup_table_olap'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_lookup_table_olap_original (
            tenant_id UUID NOT NULL,
            external_id UUID NOT NULL,
            task_id BIGINT,
            dag_id BIGINT,
            inserted_at TIMESTAMPTZ NOT NULL,
            PRIMARY KEY (external_id)
        );

        INSERT INTO v1_lookup_table_olap_original
        SELECT tenant_id, external_id, task_id, dag_id, inserted_at FROM v1_lookup_table_olap
        ON CONFLICT (external_id) DO NOTHING;

        DROP TABLE v1_lookup_table_olap;
        ALTER TABLE v1_lookup_table_olap_original RENAME TO v1_lookup_table_olap;
        ALTER INDEX v1_lookup_table_olap_original_pkey RENAME TO v1_lookup_table_olap_pkey;
        RETURN;
    END IF;

    EXECUTE format('ALTER TABLE v1_lookup_table_olap DETACH PARTITION %I', legacy_partition_name);
    EXECUTE format('INSERT INTO %I SELECT tenant_id, external_id, task_id, dag_id, inserted_at FROM v1_lookup_table_olap ON CONFLICT (external_id) DO NOTHING', legacy_partition_name);
    DROP TABLE v1_lookup_table_olap;
    EXECUTE format('ALTER TABLE %I RENAME TO v1_lookup_table_olap', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_lookup_table_olap RENAME CONSTRAINT %I TO v1_lookup_table_olap_pkey', legacy_partition_name || '_external_id_uq');
    EXECUTE format('ALTER TABLE v1_lookup_table_olap RENAME CONSTRAINT %I TO v1_lookup_table_olap_external_id_inserted_at_uq', legacy_partition_name || '_pkey');
END $$;

CREATE OR REPLACE FUNCTION v1_tasks_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'TASK',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    FROM new_rows
    WHERE dag_id IS NULL
    ON CONFLICT (inserted_at, id) DO NOTHING;

    INSERT INTO v1_lookup_table_olap (
        tenant_id,
        external_id,
        task_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id) DO NOTHING;

    -- If the task has a dag_id and dag_inserted_at, insert into the lookup table
    INSERT INTO v1_dag_to_task_olap (
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
    FROM new_rows
    WHERE dag_id IS NOT NULL
    ON CONFLICT (dag_id, dag_inserted_at, task_id, task_inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION v1_dags_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'DAG',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id,
        idempotency_key
    FROM new_rows
    ON CONFLICT (inserted_at, id) DO NOTHING;

    INSERT INTO v1_lookup_table_olap (
        tenant_id,
        external_id,
        dag_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd
