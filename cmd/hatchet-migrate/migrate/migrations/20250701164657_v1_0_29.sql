-- +goose Up
-- +goose StatementBegin

-- +goose NO TRANSACTION

DO $$
DECLARE
    snapshot_xmin BIGINT;
    partition_record RECORD;
    create_sql TEXT;
    rename_sql TEXT;
    old_name TEXT;
    new_name TEXT;
BEGIN
    SELECT txid_current() INTO snapshot_xmin;

    CREATE TABLE v1_lookup_table_new (
        tenant_id UUID NOT NULL,
        external_id UUID NOT NULL,
        task_id BIGINT,
        dag_id BIGINT,
        inserted_at TIMESTAMPTZ NOT NULL,

        PRIMARY KEY (external_id, inserted_at)
    ) PARTITION BY RANGE(inserted_at);

    FOR partition_record IN
        WITH wk AS (
            SELECT DISTINCT
                DATE_TRUNC('week', inserted_at)::DATE AS week_start
            FROM v1_lookup_table
        )

        SELECT
            'v1_lookup_table_' || TO_CHAR(week_start, 'YYYYMMDD') AS partition_name,
            'FOR VALUES FROM (''' || week_start || ''') TO (''' || (week_start + INTERVAL '1 week')::DATE || ''')' AS partition_bound
        FROM wk
        ORDER BY week_start
    LOOP
        create_sql := format(
            'CREATE TABLE %I PARTITION OF v1_lookup_table_new %s',
            partition_record.partition_name || '_new',
            partition_record.partition_bound
        );
        EXECUTE create_sql;
    END LOOP;

    INSERT INTO v1_lookup_table_new (tenant_id, external_id, task_id, dag_id, inserted_at)
    SELECT tenant_id, external_id, task_id, dag_id, inserted_at
    FROM v1_lookup_table
    WHERE xmin::text::bigint < snapshot_xmin;

    LOCK TABLE v1_lookup_table IN ACCESS EXCLUSIVE MODE;

    INSERT INTO v1_lookup_table_new (tenant_id, external_id, task_id, dag_id, inserted_at)
    SELECT tenant_id, external_id, task_id, dag_id, inserted_at
    FROM v1_lookup_table
    WHERE xmin::text::bigint >= snapshot_xmin;

    DROP TABLE v1_lookup_table;
    ALTER TABLE v1_lookup_table_new RENAME TO v1_lookup_table;

    FOR partition_record IN
        SELECT c.relname as partition_name
        FROM pg_class c
        JOIN pg_inherits i ON c.oid = i.inhrelid
        JOIN pg_class parent ON i.inhparent = parent.oid
        WHERE parent.relname = 'v1_lookup_table'
          AND c.relname LIKE '%_new'
        ORDER BY c.relname
    LOOP
        old_name := partition_record.partition_name;
        new_name := regexp_replace(old_name, '_new$', '');
        rename_sql := format('ALTER TABLE %I RENAME TO %I', old_name, new_name);
        EXECUTE rename_sql;
    END LOOP;

    FOR partition_record IN
        SELECT
            c.relname as partition_name,
            i.relname as index_name
        FROM pg_class c
        JOIN pg_inherits inh ON c.oid = inh.inhrelid
        JOIN pg_class parent ON inh.inhparent = parent.oid
        JOIN pg_index idx ON c.oid = idx.indrelid
        JOIN pg_class i ON idx.indexrelid = i.oid
        WHERE parent.relname = 'v1_lookup_table'
        AND i.relname LIKE '%_new_%'
        ORDER BY c.relname, i.relname
    LOOP
        old_name := partition_record.index_name;
        new_name := regexp_replace(old_name, '_new_', '_');
        rename_sql := format('ALTER INDEX %I RENAME TO %I', old_name, new_name);
        EXECUTE rename_sql;
    END LOOP;
END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
