-- +goose Up
-- +goose StatementBegin
DROP INDEX v1_payload_external_id_idx;
DROP INDEX v1_payloads_olap_external_id_idx;

CREATE OR REPLACE FUNCTION swap_v1_payload_partition_with_temp(
    partition_date date
) RETURNS text
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    temp_table_name varchar;
    old_pk_name varchar;
    new_pk_name varchar;
    old_external_id_unique_constraint_name varchar;
    new_external_id_unique_constraint_name varchar;
    partition_start date;
    partition_end date;
    trigger_function_name varchar;
    trigger_name varchar;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payload_offload_tmp_%s', partition_date_str) INTO temp_table_name;
    SELECT format('v1_payload_offload_tmp_%s_pkey', partition_date_str) INTO old_pk_name;
    SELECT format('v1_payload_%s_pkey', partition_date_str) INTO new_pk_name;
    SELECT format('v1_payload_offload_tmp_%s_external_id_key', partition_date_str) INTO old_external_id_unique_constraint_name;
    SELECT format('v1_payload_%s_external_id_uq', partition_date_str) INTO new_external_id_unique_constraint_name;
    SELECT format('sync_to_%s', temp_table_name) INTO trigger_function_name;
    SELECT format('trigger_sync_to_%s', temp_table_name) INTO trigger_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = temp_table_name) THEN
        RAISE EXCEPTION 'Temp table % does not exist', temp_table_name;
    END IF;

    partition_start := partition_date;
    partition_end := partition_date + INTERVAL '1 day';

    EXECUTE format(
        'ALTER TABLE %I SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor = ''0.05'',
            autovacuum_vacuum_threshold = ''25'',
            autovacuum_analyze_threshold = ''25'',
            autovacuum_vacuum_cost_delay = ''10'',
            autovacuum_vacuum_cost_limit = ''1000''
        )',
        temp_table_name
    );
    RAISE NOTICE 'Set autovacuum settings on partition %', temp_table_name;

    LOCK TABLE v1_payload IN ACCESS EXCLUSIVE MODE;

    RAISE NOTICE 'Dropping trigger from partition %', source_partition_name;
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', trigger_name, source_partition_name);

    RAISE NOTICE 'Dropping trigger function %', trigger_function_name;
    EXECUTE format('DROP FUNCTION IF EXISTS %I()', trigger_function_name);

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE NOTICE 'Dropping old partition %', source_partition_name;
        EXECUTE format('ALTER TABLE v1_payload DETACH PARTITION %I', source_partition_name);
        EXECUTE format('DROP TABLE %I CASCADE', source_partition_name);
    END IF;

    RAISE NOTICE 'Renaming primary key % to %', old_pk_name, new_pk_name;
    EXECUTE format('ALTER INDEX %I RENAME TO %I', old_pk_name, new_pk_name);

    RAISE NOTICE 'Renaming external_id unique constraint % to %', old_external_id_unique_constraint_name, new_external_id_unique_constraint_name;
    EXECUTE format('ALTER TABLE %I RENAME CONSTRAINT %I TO %I', temp_table_name, old_external_id_unique_constraint_name, new_external_id_unique_constraint_name);

    RAISE NOTICE 'Renaming temp table % to %', temp_table_name, source_partition_name;
    EXECUTE format('ALTER TABLE %I RENAME TO %I', temp_table_name, source_partition_name);

    RAISE NOTICE 'Attaching new partition % to v1_payload', source_partition_name;
    EXECUTE format(
        'ALTER TABLE v1_payload ATTACH PARTITION %I FOR VALUES FROM (%L) TO (%L)',
        source_partition_name,
        partition_start,
        partition_end
    );

    RAISE NOTICE 'Dropping hack check constraint';
    EXECUTE format(
        'ALTER TABLE %I DROP CONSTRAINT %I',
        source_partition_name,
        temp_table_name || '_iat_chk_bounds'
    );

    RAISE NOTICE 'Successfully swapped partition %', source_partition_name;
    RETURN source_partition_name;
END;
$$;

DO $$
DECLARE
    partition_with_temp_named_constraint record;
BEGIN
    FOR partition_with_temp_named_constraint IN
        SELECT
            c.conrelid::regclass::text AS partition_name,
            c.conname AS constraint_name
        FROM pg_constraint c
        JOIN pg_inherits i ON i.inhrelid = c.conrelid
        WHERE
            i.inhparent = 'v1_payload'::regclass
            AND c.conname ~ '^v1_payload_offload_tmp_[0-9]{8}_external_id_key$'
    LOOP
        EXECUTE format(
            'ALTER TABLE %I RENAME CONSTRAINT %I TO %I',
            partition_with_temp_named_constraint.partition_name,
            partition_with_temp_named_constraint.constraint_name,
            partition_with_temp_named_constraint.partition_name || '_external_id_uq'
        );
    END LOOP;
END $$;

DROP FUNCTION copy_v1_payloads_olap_partition_structure(date);
DROP FUNCTION list_paginated_olap_payloads_for_offload(date, uuid, uuid, integer);
DROP FUNCTION create_olap_payload_offload_range_chunks(date, integer, integer, uuid);
DROP FUNCTION diff_olap_payload_source_and_target_partitions(date);
DROP FUNCTION compute_olap_payload_batch_size(date, uuid, integer);
DROP FUNCTION swap_v1_payloads_olap_partition_with_temp(date);
DROP TABLE v1_payloads_olap_cutover_job_offset;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX v1_payload_external_id_idx ON v1_payload (external_id ASC);
CREATE INDEX v1_payloads_olap_external_id_idx ON v1_payloads_olap (external_id ASC);

CREATE OR REPLACE FUNCTION swap_v1_payload_partition_with_temp(
    partition_date date
) RETURNS text
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    temp_table_name varchar;
    old_pk_name varchar;
    new_pk_name varchar;
    old_ext_id_idx_name varchar;
    new_ext_id_idx_name varchar;
    partition_start date;
    partition_end date;
    trigger_function_name varchar;
    trigger_name varchar;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payload_offload_tmp_%s', partition_date_str) INTO temp_table_name;
    SELECT format('v1_payload_offload_tmp_%s_pkey', partition_date_str) INTO old_pk_name;
    SELECT format('v1_payload_%s_pkey', partition_date_str) INTO new_pk_name;
    SELECT format('v1_payload_offload_tmp_%s_external_id_idx', partition_date_str) INTO old_ext_id_idx_name;
    SELECT format('v1_payload_%s_external_id_idx', partition_date_str) INTO new_ext_id_idx_name;
    SELECT format('sync_to_%s', temp_table_name) INTO trigger_function_name;
    SELECT format('trigger_sync_to_%s', temp_table_name) INTO trigger_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = temp_table_name) THEN
        RAISE EXCEPTION 'Temp table % does not exist', temp_table_name;
    END IF;

    partition_start := partition_date;
    partition_end := partition_date + INTERVAL '1 day';

    EXECUTE format(
        'ALTER TABLE %I SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor = ''0.05'',
            autovacuum_vacuum_threshold = ''25'',
            autovacuum_analyze_threshold = ''25'',
            autovacuum_vacuum_cost_delay = ''10'',
            autovacuum_vacuum_cost_limit = ''1000''
        )',
        temp_table_name
    );
    RAISE NOTICE 'Set autovacuum settings on partition %', temp_table_name;

    LOCK TABLE v1_payload IN ACCESS EXCLUSIVE MODE;

    RAISE NOTICE 'Dropping trigger from partition %', source_partition_name;
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', trigger_name, source_partition_name);

    RAISE NOTICE 'Dropping trigger function %', trigger_function_name;
    EXECUTE format('DROP FUNCTION IF EXISTS %I()', trigger_function_name);

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE NOTICE 'Dropping old partition %', source_partition_name;
        EXECUTE format('ALTER TABLE v1_payload DETACH PARTITION %I', source_partition_name);
        EXECUTE format('DROP TABLE %I CASCADE', source_partition_name);
    END IF;

    RAISE NOTICE 'Renaming primary key % to %', old_pk_name, new_pk_name;
    EXECUTE format('ALTER INDEX %I RENAME TO %I', old_pk_name, new_pk_name);

    RAISE NOTICE 'Renaming external_id index % to %', old_ext_id_idx_name, new_ext_id_idx_name;
    EXECUTE format('ALTER INDEX %I RENAME TO %I', old_ext_id_idx_name, new_ext_id_idx_name);

    RAISE NOTICE 'Renaming temp table % to %', temp_table_name, source_partition_name;
    EXECUTE format('ALTER TABLE %I RENAME TO %I', temp_table_name, source_partition_name);

    RAISE NOTICE 'Attaching new partition % to v1_payload', source_partition_name;
    EXECUTE format(
        'ALTER TABLE v1_payload ATTACH PARTITION %I FOR VALUES FROM (%L) TO (%L)',
        source_partition_name,
        partition_start,
        partition_end
    );

    RAISE NOTICE 'Dropping hack check constraint';
    EXECUTE format(
        'ALTER TABLE %I DROP CONSTRAINT %I',
        source_partition_name,
        temp_table_name || '_iat_chk_bounds'
    );

    RAISE NOTICE 'Successfully swapped partition %', source_partition_name;
    RETURN source_partition_name;
END;
$$;

CREATE TABLE v1_payloads_olap_cutover_job_offset (
    key DATE PRIMARY KEY,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    lease_process_id UUID NOT NULL DEFAULT gen_random_uuid(),
    lease_expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    last_external_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID
);

CREATE OR REPLACE FUNCTION copy_v1_payloads_olap_partition_structure(
    partition_date date
) RETURNS text
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    target_table_name varchar;
    trigger_function_name varchar;
    trigger_name varchar;
    partition_start date;
    partition_end date;
BEGIN
    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s', partition_date_str) INTO target_table_name;
    SELECT format('sync_to_%s', target_table_name) INTO trigger_function_name;
    SELECT format('trigger_sync_to_%s', target_table_name) INTO trigger_name;
    partition_start := partition_date;
    partition_end := partition_date + INTERVAL '1 day';

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Source partition % does not exist', source_partition_name;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = target_table_name) THEN
        RAISE NOTICE 'Target table % already exists, skipping creation', target_table_name;
        RETURN target_table_name;
    END IF;

    EXECUTE format(
        'CREATE TABLE %I (LIKE %I INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING INDEXES)',
        target_table_name,
        source_partition_name
    );

    EXECUTE format('
        ALTER TABLE %I
        ADD CONSTRAINT %I
        CHECK (
            inserted_at IS NOT NULL
            AND inserted_at >= %L::TIMESTAMPTZ
            AND inserted_at < %L::TIMESTAMPTZ
        )
        ',
        target_table_name,
        target_table_name || '_iat_chk_bounds',
        partition_start,
        partition_end
    );

    EXECUTE format('
        CREATE OR REPLACE FUNCTION %I() RETURNS trigger
            LANGUAGE plpgsql AS $func$
        BEGIN
            IF TG_OP = ''INSERT'' THEN
                INSERT INTO %I (tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at)
                VALUES (NEW.tenant_id, NEW.external_id, NEW.location, NEW.external_location_key, NEW.inline_content, NEW.inserted_at, NEW.updated_at)
                ON CONFLICT (tenant_id, external_id, inserted_at) DO UPDATE
                SET
                    location = EXCLUDED.location,
                    external_location_key = EXCLUDED.external_location_key,
                    inline_content = EXCLUDED.inline_content,
                    updated_at = EXCLUDED.updated_at;
                RETURN NEW;
            ELSIF TG_OP = ''UPDATE'' THEN
                UPDATE %I
                SET
                    location = NEW.location,
                    external_location_key = NEW.external_location_key,
                    inline_content = NEW.inline_content,
                    updated_at = NEW.updated_at
                WHERE
                    tenant_id = NEW.tenant_id
                    AND external_id = NEW.external_id
                    AND inserted_at = NEW.inserted_at
                    ;
                RETURN NEW;
            ELSIF TG_OP = ''DELETE'' THEN
                DELETE FROM %I
                WHERE
                    tenant_id = OLD.tenant_id
                    AND external_id = OLD.external_id
                    AND inserted_at = OLD.inserted_at
                    ;
                RETURN OLD;
            END IF;
            RETURN NULL;
        END;
        $func$;
    ', trigger_function_name, target_table_name, target_table_name, target_table_name);

    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', trigger_name, source_partition_name);

    EXECUTE format('
        CREATE TRIGGER %I
        AFTER INSERT OR UPDATE OR DELETE ON %I
        FOR EACH ROW
        EXECUTE FUNCTION %I();
    ', trigger_name, source_partition_name, trigger_function_name);

    RAISE NOTICE 'Created table % as a copy of partition % with sync trigger', target_table_name, source_partition_name;

    RETURN target_table_name;
END;
$$;

CREATE OR REPLACE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    last_external_id uuid,
    next_external_id uuid,
    batch_size integer
) RETURNS TABLE (
    tenant_id UUID,
    external_id UUID,
    location v1_payload_location_olap,
    external_location_key TEXT,
    inline_content JSONB,
    inserted_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS MATERIALIZED (
            SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
            FROM %I
            WHERE external_id >= $1::UUID
            ORDER BY external_id

            -- Multiplying by two here to handle an edge case. There is a small chance we miss a row
            -- when a different row is inserted before it, in between us creating the chunks and selecting
            -- them. By multiplying by two to create a "candidate" set, we significantly reduce the chance of us missing
            -- rows in this way, since if a row is inserted before one of our last rows, we will still have
            -- the next row after it in the candidate set.
            LIMIT $3 * 2
        )

        SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
        FROM candidates
        WHERE
            external_id >= $1
            AND external_id <= $2
        ORDER BY external_id
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_external_id, next_external_id, batch_size;
END;
$$;

CREATE OR REPLACE FUNCTION create_olap_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_external_id uuid
) RETURNS TABLE (
    lower_external_id UUID,
    upper_external_id UUID
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH paginated AS (
            SELECT external_id, ROW_NUMBER() OVER (ORDER BY external_id) AS rn
            FROM %I
            WHERE external_id > $1::UUID
            ORDER BY external_id
            LIMIT $2::INTEGER
        ), lower_bounds AS (
            SELECT rn::INTEGER / $3::INTEGER AS batch_ix, external_id::UUID
            FROM paginated
            WHERE MOD(rn, $3::INTEGER) = 1
        ), upper_bounds AS (
            SELECT
                -- Using `CEIL` and subtracting 1 here to make the `batch_ix` zero indexed like the `lower_bounds` one is.
                -- We need the `CEIL` to handle the case where the number of rows in the window is not evenly divisible by the batch size,
                -- because without CEIL if e.g. there were 5 rows in the window and a batch size of two and we did integer division, we would end
                -- up with batches of index 0, 1, and 1 after dividing and subtracting. With float division and `CEIL`, we get 0, 1, and 2 as expected.
                -- Then we need to subtract one because we compute the batch index by using integer division on the lower bounds, which are all zero indexed.
                CEIL(rn::FLOAT / $3::FLOAT) - 1 AS batch_ix,
                external_id::UUID
            FROM paginated
            -- We want to include either the last row of each batch, or the last row of the entire paginated set, which may not line up with a batch end.
            WHERE MOD(rn, $3::INTEGER) = 0 OR rn = (SELECT MAX(rn) FROM paginated)
        )

        SELECT
            lb.external_id AS lower_external_id,
            ub.external_id AS upper_external_id
        FROM lower_bounds lb
        JOIN upper_bounds ub ON lb.batch_ix = ub.batch_ix
        ORDER BY lb.external_id
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_external_id, window_size, chunk_size;
END;
$$;

CREATE OR REPLACE FUNCTION diff_olap_payload_source_and_target_partitions(
    partition_date date
) RETURNS TABLE (
    tenant_id UUID,
    external_id UUID,
    inserted_at TIMESTAMPTZ,
    location v1_payload_location_olap,
    external_location_key TEXT,
    inline_content JSONB,
    updated_at TIMESTAMPTZ
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    temp_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s', partition_date_str) INTO temp_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        SELECT tenant_id, external_id, inserted_at, location, external_location_key, inline_content, updated_at
        FROM %I source
        WHERE NOT EXISTS (
            SELECT 1
            FROM %I AS target
            WHERE
                source.tenant_id = target.tenant_id
                AND source.external_id = target.external_id
                AND source.inserted_at = target.inserted_at
        )
    ', source_partition_name, temp_partition_name);

    RETURN QUERY EXECUTE query;
END;
$$;

CREATE OR REPLACE FUNCTION compute_olap_payload_batch_size(
    partition_date DATE,
    last_external_id UUID,
    batch_size INTEGER
) RETURNS BIGINT
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str TEXT;
    source_partition_name TEXT;
    query TEXT;
    result_size BIGINT;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS (
            SELECT *
            FROM %I
            WHERE external_id >= $1::UUID
            ORDER BY external_id
            LIMIT $2::INTEGER
        )

        SELECT COALESCE(SUM(pg_column_size(inline_content)), 0) AS total_size_bytes
        FROM candidates
    ', source_partition_name);

    EXECUTE query INTO result_size USING last_external_id, batch_size;

    RETURN result_size;
END;
$$;

CREATE OR REPLACE FUNCTION swap_v1_payloads_olap_partition_with_temp(
    partition_date date
) RETURNS text
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    temp_table_name varchar;
    old_pk_name varchar;
    new_pk_name varchar;
    old_ext_id_idx_name varchar;
    new_ext_id_idx_name varchar;
    partition_start date;
    partition_end date;
    trigger_function_name varchar;
    trigger_name varchar;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s', partition_date_str) INTO temp_table_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s_pkey', partition_date_str) INTO old_pk_name;
    SELECT format('v1_payloads_olap_%s_pkey', partition_date_str) INTO new_pk_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s_external_id_idx', partition_date_str) INTO old_ext_id_idx_name;
    SELECT format('v1_payloads_olap_%s_external_id_idx', partition_date_str) INTO new_ext_id_idx_name;
    SELECT format('sync_to_%s', temp_table_name) INTO trigger_function_name;
    SELECT format('trigger_sync_to_%s', temp_table_name) INTO trigger_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = temp_table_name) THEN
        RAISE EXCEPTION 'Temp table % does not exist', temp_table_name;
    END IF;

    partition_start := partition_date;
    partition_end := partition_date + INTERVAL '1 day';

    EXECUTE format(
        'ALTER TABLE %I SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor = ''0.05'',
            autovacuum_vacuum_threshold = ''25'',
            autovacuum_analyze_threshold = ''25'',
            autovacuum_vacuum_cost_delay = ''10'',
            autovacuum_vacuum_cost_limit = ''1000''
        )',
        temp_table_name
    );
    RAISE NOTICE 'Set autovacuum settings on partition %', temp_table_name;

    LOCK TABLE v1_payloads_olap IN ACCESS EXCLUSIVE MODE;

    RAISE NOTICE 'Dropping trigger from partition %', source_partition_name;
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', trigger_name, source_partition_name);

    RAISE NOTICE 'Dropping trigger function %', trigger_function_name;
    EXECUTE format('DROP FUNCTION IF EXISTS %I()', trigger_function_name);

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE NOTICE 'Dropping old partition %', source_partition_name;
        EXECUTE format('ALTER TABLE v1_payloads_olap DETACH PARTITION %I', source_partition_name);
        EXECUTE format('DROP TABLE %I CASCADE', source_partition_name);
    END IF;

    RAISE NOTICE 'Renaming primary key % to %', old_pk_name, new_pk_name;
    EXECUTE format('ALTER INDEX %I RENAME TO %I', old_pk_name, new_pk_name);

    RAISE NOTICE 'Renaming external_id index % to %', old_ext_id_idx_name, new_ext_id_idx_name;
    EXECUTE format('ALTER INDEX %I RENAME TO %I', old_ext_id_idx_name, new_ext_id_idx_name);

    RAISE NOTICE 'Renaming temp table % to %', temp_table_name, source_partition_name;
    EXECUTE format('ALTER TABLE %I RENAME TO %I', temp_table_name, source_partition_name);

    RAISE NOTICE 'Attaching new partition % to v1_payloads_olap', source_partition_name;
    EXECUTE format(
        'ALTER TABLE v1_payloads_olap ATTACH PARTITION %I FOR VALUES FROM (%L) TO (%L)',
        source_partition_name,
        partition_start,
        partition_end
    );

    RAISE NOTICE 'Dropping hack check constraint';
    EXECUTE format(
        'ALTER TABLE %I DROP CONSTRAINT %I',
        source_partition_name,
        temp_table_name || '_iat_chk_bounds'
    );

    RAISE NOTICE 'Successfully swapped partition %', source_partition_name;
    RETURN source_partition_name;
END;
$$;
-- +goose StatementEnd
