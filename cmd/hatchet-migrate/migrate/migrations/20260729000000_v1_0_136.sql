-- +goose NO TRANSACTION
-- +goose Up

-- Converts the four remaining high-volume, unpartitioned OLAP tables to
-- range-partitioned tables so they can participate in partition-based
-- retention (and so downstream consumers don't need TimescaleDB for
-- retention on these tables):
--
--   - v1_statuses_olap      partitioned by inserted_at
--   - v1_task_events_olap   partitioned by task_inserted_at
--   - v1_lookup_table_olap  partitioned by inserted_at (PK widened to (external_id, inserted_at))
--   - v1_dag_to_task_olap   partitioned by dag_inserted_at
--
-- For each table, the existing table becomes the first ("legacy") partition of a
-- new partitioned parent, covering (MINVALUE, tomorrow). The legacy partition is
-- deliberately named {table}_{YYYYMMDD} for the migration day so that:
--
--   1. create_v1_range_partition skips creating a conflicting partition for
--      the migration day (it checks for the table name before creating), and
--   2. get_v1_partitions_before_date picks the legacy partition up for retention
--      once the migration day falls outside the retention window.
--
-- To attach without holding an ACCESS EXCLUSIVE lock during a full-table
-- validation scan, we first add a CHECK constraint matching the partition bound
-- as NOT VALID (brief lock), VALIDATE it (SHARE UPDATE EXCLUSIVE, writes
-- continue), and only then do the rename + attach dance, which is catalog-only.
--
-- This migration runs outside a transaction (required for CREATE INDEX
-- CONCURRENTLY), so every statement is written to be idempotent in case of a
-- partial failure and re-run. Migration-day state (the partition boundary) is
-- persisted in a scratch table that is dropped at the end.
--
-- NOTE: if this migration is run close to midnight UTC on a very large database,
-- there is a small window between adding the CHECK constraint and the swap where
-- inserts with a partition-key timestamp past midnight would be rejected by the
-- constraint. Prefer running this migration away from the UTC day boundary.

-- +goose StatementBegin
DO $mig$
DECLARE
    part_boundary date := (now() AT TIME ZONE 'UTC')::date + 1;
    t record;
BEGIN
    CREATE TABLE IF NOT EXISTS v1_olap_partition_migration_state (
        table_name text PRIMARY KEY,
        boundary date NOT NULL
    );

    FOR t IN
        SELECT * FROM (VALUES
            ('v1_statuses_olap', 'inserted_at'),
            ('v1_task_events_olap', 'task_inserted_at'),
            ('v1_lookup_table_olap', 'inserted_at'),
            ('v1_dag_to_task_olap', 'dag_inserted_at')
        ) AS v(table_name, part_col)
    LOOP
        -- skip tables that are already partitioned (fresh re-run after partial failure)
        IF EXISTS (
            SELECT 1 FROM pg_class
            WHERE relname = t.table_name AND relkind = 'p' AND relnamespace = 'public'::regnamespace
        ) THEN
            CONTINUE;
        END IF;

        INSERT INTO v1_olap_partition_migration_state (table_name, boundary)
        VALUES (t.table_name, part_boundary)
        ON CONFLICT (table_name) DO NOTHING;

        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conrelid = t.table_name::regclass
              AND conname = t.table_name || '_partition_check'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I ADD CONSTRAINT %I CHECK (%I < %L::timestamptz) NOT VALID',
                t.table_name,
                t.table_name || '_partition_check',
                t.part_col,
                (SELECT s.boundary FROM v1_olap_partition_migration_state s WHERE s.table_name = t.table_name)
            );
        END IF;
    END LOOP;
END
$mig$;
-- +goose StatementEnd

-- Validate the CHECK constraints. This scans each table but only takes a
-- SHARE UPDATE EXCLUSIVE lock, so reads and writes continue.
-- +goose StatementBegin
DO $mig$
DECLARE
    c record;
BEGIN
    FOR c IN
        SELECT conrelid::regclass::text AS tbl, conname
        FROM pg_constraint
        WHERE conname IN (
            'v1_statuses_olap_partition_check',
            'v1_task_events_olap_partition_check',
            'v1_lookup_table_olap_partition_check',
            'v1_dag_to_task_olap_partition_check'
        )
        AND NOT convalidated
    LOOP
        EXECUTE format('ALTER TABLE %s VALIDATE CONSTRAINT %I', c.tbl, c.conname);
    END LOOP;
END
$mig$;
-- +goose StatementEnd

-- v1_lookup_table_olap needs a unique index on (external_id, inserted_at) so it
-- can back the new parent's primary key when attached. Drop a leftover invalid
-- index from a previously interrupted run, if any, then build it concurrently.
-- +goose StatementBegin
DO $mig$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_index i
        JOIN pg_class c ON c.oid = i.indexrelid
        WHERE c.relname = 'v1_lookup_table_olap_external_id_inserted_at_key'
          AND NOT i.indisvalid
    ) THEN
        EXECUTE 'DROP INDEX v1_lookup_table_olap_external_id_inserted_at_key';
    END IF;
END
$mig$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS v1_lookup_table_olap_external_id_inserted_at_key
    ON v1_lookup_table_olap (external_id, inserted_at);
-- +goose StatementEnd

-- Swap v1_statuses_olap. Catalog-only: rename old table + indexes out of the way,
-- create the partitioned parent, attach the old table as the legacy partition
-- (no scan thanks to the validated CHECK constraint), create tomorrow's partition.
-- +goose StatementBegin
DO $mig$
DECLARE
    part_boundary date;
    legacy_name text;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class
        WHERE relname = 'v1_statuses_olap' AND relkind = 'p' AND relnamespace = 'public'::regnamespace
    ) THEN
        RETURN;
    END IF;

    SET LOCAL lock_timeout = '30s';

    SELECT s.boundary INTO STRICT part_boundary
    FROM v1_olap_partition_migration_state s
    WHERE s.table_name = 'v1_statuses_olap';

    legacy_name := 'v1_statuses_olap_' || to_char(part_boundary - 1, 'YYYYMMDD');

    EXECUTE format('ALTER TABLE v1_statuses_olap RENAME TO %I', legacy_name);
    EXECUTE format('ALTER INDEX v1_statuses_olap_pkey RENAME TO %I', legacy_name || '_pkey');
    EXECUTE format('ALTER INDEX idx_v1_statuses_olap_query_optim RENAME TO %I', 'idx_' || legacy_name || '_query_optim');

    CREATE TABLE v1_statuses_olap (
        external_id UUID NOT NULL,
        inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        tenant_id UUID NOT NULL,
        workflow_id UUID NOT NULL,
        kind v1_run_kind NOT NULL,
        readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',

        PRIMARY KEY (external_id, inserted_at)
    ) PARTITION BY RANGE(inserted_at);

    CREATE INDEX idx_v1_statuses_olap_query_optim ON v1_statuses_olap (tenant_id, workflow_id);

    EXECUTE format(
        'ALTER TABLE v1_statuses_olap ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)',
        legacy_name, to_char(part_boundary, 'YYYYMMDD')
    );
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_statuses_olap_partition_check', legacy_name);

    PERFORM create_v1_range_partition('v1_statuses_olap', part_boundary);
END
$mig$;
-- +goose StatementEnd

-- Swap v1_task_events_olap. Same dance, plus identity/sequence handling: the
-- legacy table keeps its identity (pointing at the renamed sequence), and the
-- new parent's identity sequence is advanced past the legacy sequence so ids
-- keep increasing.
-- +goose StatementBegin
DO $mig$
DECLARE
    part_boundary date;
    legacy_name text;
    legacy_last_value bigint;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class
        WHERE relname = 'v1_task_events_olap' AND relkind = 'p' AND relnamespace = 'public'::regnamespace
    ) THEN
        RETURN;
    END IF;

    SET LOCAL lock_timeout = '30s';

    SELECT s.boundary INTO STRICT part_boundary
    FROM v1_olap_partition_migration_state s
    WHERE s.table_name = 'v1_task_events_olap';

    legacy_name := 'v1_task_events_olap_' || to_char(part_boundary - 1, 'YYYYMMDD');

    EXECUTE format('ALTER TABLE v1_task_events_olap RENAME TO %I', legacy_name);
    EXECUTE format('ALTER INDEX v1_task_events_olap_pkey RENAME TO %I', legacy_name || '_pkey');
    EXECUTE format('ALTER INDEX v1_task_events_olap_task_id_idx RENAME TO %I', legacy_name || '_task_id_idx');
    EXECUTE format('ALTER SEQUENCE v1_task_events_olap_id_seq RENAME TO %I', legacy_name || '_id_seq');

    CREATE TABLE v1_task_events_olap (
        tenant_id UUID NOT NULL,
        id BIGINT GENERATED ALWAYS AS IDENTITY,
        inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
        external_id UUID NOT NULL DEFAULT gen_random_uuid(),
        durable_invocation_count INT NOT NULL DEFAULT 0,

        PRIMARY KEY (task_id, task_inserted_at, id)
    ) PARTITION BY RANGE(task_inserted_at);

    CREATE INDEX v1_task_events_olap_task_id_idx ON v1_task_events_olap (task_id);

    EXECUTE format('SELECT last_value FROM %I', legacy_name || '_id_seq') INTO legacy_last_value;
    PERFORM setval(pg_get_serial_sequence('v1_task_events_olap', 'id'), legacy_last_value);

    EXECUTE format(
        'ALTER TABLE v1_task_events_olap ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)',
        legacy_name, to_char(part_boundary, 'YYYYMMDD')
    );
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_task_events_olap_partition_check', legacy_name);

    PERFORM create_v1_range_partition('v1_task_events_olap', part_boundary);
END
$mig$;
-- +goose StatementEnd

-- Swap v1_dag_to_task_olap.
-- +goose StatementBegin
DO $mig$
DECLARE
    part_boundary date;
    legacy_name text;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class
        WHERE relname = 'v1_dag_to_task_olap' AND relkind = 'p' AND relnamespace = 'public'::regnamespace
    ) THEN
        RETURN;
    END IF;

    SET LOCAL lock_timeout = '30s';

    SELECT s.boundary INTO STRICT part_boundary
    FROM v1_olap_partition_migration_state s
    WHERE s.table_name = 'v1_dag_to_task_olap';

    legacy_name := 'v1_dag_to_task_olap_' || to_char(part_boundary - 1, 'YYYYMMDD');

    EXECUTE format('ALTER TABLE v1_dag_to_task_olap RENAME TO %I', legacy_name);
    EXECUTE format('ALTER INDEX v1_dag_to_task_olap_pkey RENAME TO %I', legacy_name || '_pkey');

    CREATE TABLE v1_dag_to_task_olap (
        dag_id BIGINT NOT NULL,
        dag_inserted_at TIMESTAMPTZ NOT NULL,
        task_id BIGINT NOT NULL,
        task_inserted_at TIMESTAMPTZ NOT NULL,
        PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
    ) PARTITION BY RANGE(dag_inserted_at);

    EXECUTE format(
        'ALTER TABLE v1_dag_to_task_olap ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)',
        legacy_name, to_char(part_boundary, 'YYYYMMDD')
    );
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_dag_to_task_olap_partition_check', legacy_name);

    PERFORM create_v1_range_partition('v1_dag_to_task_olap', part_boundary);
END
$mig$;
-- +goose StatementEnd

-- Swap v1_lookup_table_olap. The primary key is widened from (external_id) to
-- (external_id, inserted_at) — required because the partition key must be part
-- of any unique constraint. The concurrently-built unique index on the legacy
-- table backs the new parent PK on attach, and the old single-column PK is
-- dropped so replays (same external_id, new inserted_at) aren't rejected while
-- today's rows still route to the legacy partition.
--
-- The two trigger functions whose ON CONFLICT target referenced the old PK are
-- replaced in the same transaction.
-- +goose StatementBegin
DO $mig$
DECLARE
    part_boundary date;
    legacy_name text;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class
        WHERE relname = 'v1_lookup_table_olap' AND relkind = 'p' AND relnamespace = 'public'::regnamespace
    ) THEN
        RETURN;
    END IF;

    SET LOCAL lock_timeout = '30s';

    SELECT s.boundary INTO STRICT part_boundary
    FROM v1_olap_partition_migration_state s
    WHERE s.table_name = 'v1_lookup_table_olap';

    legacy_name := 'v1_lookup_table_olap_' || to_char(part_boundary - 1, 'YYYYMMDD');

    EXECUTE format('ALTER TABLE v1_lookup_table_olap RENAME TO %I', legacy_name);

    -- Drop the legacy single-column PK before attaching: a partition cannot have
    -- its own PK in addition to the parent's. Uniqueness on the legacy partition
    -- is enforced by the concurrently-built (external_id, inserted_at) index,
    -- which backs the parent PK on attach.
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_lookup_table_olap_pkey', legacy_name);

    CREATE TABLE v1_lookup_table_olap (
        tenant_id UUID NOT NULL,
        external_id UUID NOT NULL,
        task_id BIGINT,
        dag_id BIGINT,
        inserted_at TIMESTAMPTZ NOT NULL,

        PRIMARY KEY (external_id, inserted_at)
    ) PARTITION BY RANGE(inserted_at);

    EXECUTE format(
        'ALTER TABLE v1_lookup_table_olap ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)',
        legacy_name, to_char(part_boundary, 'YYYYMMDD')
    );

    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_lookup_table_olap_partition_check', legacy_name);

    PERFORM create_v1_range_partition('v1_lookup_table_olap', part_boundary);

    CREATE OR REPLACE FUNCTION v1_tasks_olap_insert_function()
    RETURNS TRIGGER AS
    $fn$
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
    $fn$
    LANGUAGE plpgsql;

    CREATE OR REPLACE FUNCTION v1_dags_olap_insert_function()
    RETURNS TRIGGER AS
    $fn$
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
    $fn$
    LANGUAGE plpgsql;
END
$mig$;
-- +goose StatementEnd

-- Drop the scratch state table once all four tables are partitioned.
-- +goose StatementBegin
DO $mig$
BEGIN
    IF (
        SELECT count(*) FROM pg_class
        WHERE relname IN ('v1_statuses_olap', 'v1_task_events_olap', 'v1_lookup_table_olap', 'v1_dag_to_task_olap')
          AND relkind = 'p'
          AND relnamespace = 'public'::regnamespace
    ) = 4 THEN
        DROP TABLE IF EXISTS v1_olap_partition_migration_state;
    END IF;
END
$mig$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $mig$
BEGIN
    RAISE EXCEPTION 'v1_0_136 (partitioning of v1_statuses_olap, v1_task_events_olap, v1_lookup_table_olap, v1_dag_to_task_olap) is not reversible';
END
$mig$;
-- +goose StatementEnd
