-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload ALTER COLUMN external_id SET DEFAULT gen_random_uuid();
ALTER TABLE v1_payload
  ADD CONSTRAINT v1_payload_external_id_not_null CHECK (external_id IS NOT NULL) NOT VALID;

DO $$
DECLARE
  batch_size INT := 10000;
  rows_updated INT;
BEGIN
  LOOP
    UPDATE v1_payload
    SET external_id = gen_random_uuid()
    WHERE ctid IN (
      SELECT ctid FROM v1_payload
      WHERE external_id IS NULL
      LIMIT batch_size
    );
    GET DIAGNOSTICS rows_updated = ROW_COUNT;
    EXIT WHEN rows_updated = 0;
    PERFORM pg_sleep(0.1);
  END LOOP;
END $$;

ALTER TABLE v1_payload VALIDATE CONSTRAINT v1_payload_external_id_not_null;
ALTER TABLE v1_payload ALTER COLUMN external_id SET NOT NULL;
ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_external_id_not_null;

ALTER TABLE v1_payload_cutover_job_offset ADD COLUMN last_external_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;

DROP FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type,
    next_tenant_id uuid,
    next_inserted_at timestamptz,
    next_id bigint,
    next_type v1_payload_type,
    batch_size integer
);
CREATE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_external_id uuid,
    next_external_id uuid,
    batch_size integer
) RETURNS TABLE (
    tenant_id UUID,
    id BIGINT,
    inserted_at TIMESTAMPTZ,
    external_id UUID,
    type v1_payload_type,
    location v1_payload_location,
    external_location_key TEXT,
    inline_content JSONB,
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS MATERIALIZED (
            SELECT tenant_id, id, inserted_at, external_id, type, location,
                external_location_key, inline_content, updated_at
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

        SELECT tenant_id, id, inserted_at, external_id, type, location,
               external_location_key, inline_content, updated_at
        FROM candidates
        WHERE
            external_id >= $1
            AND external_id <= $2
        ORDER BY external_id
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_external_id, next_external_id, batch_size;
END;
$$;

DROP FUNCTION compute_payload_batch_size(
    partition_date DATE,
    last_tenant_id UUID,
    last_inserted_at TIMESTAMPTZ,
    last_id BIGINT,
    last_type v1_payload_type,
    batch_size INTEGER
);
CREATE FUNCTION compute_payload_batch_size(
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

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

DROP FUNCTION create_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type
);
CREATE FUNCTION create_payload_offload_range_chunks(
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

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

DROP FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz,
    next_tenant_id uuid,
    next_external_id uuid,
    next_inserted_at timestamptz,
    batch_size integer
);
CREATE FUNCTION list_paginated_olap_payloads_for_offload(
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

DROP FUNCTION compute_olap_payload_batch_size(
    partition_date DATE,
    last_tenant_id UUID,
    last_external_id UUID,
    last_inserted_at TIMESTAMPTZ,
    batch_size INTEGER
);
CREATE FUNCTION compute_olap_payload_batch_size(
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

DROP FUNCTION create_olap_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz
);
CREATE FUNCTION create_olap_payload_offload_range_chunks(
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
                CEIL(rn::FLOAT / $3::FLOAT) - 1 AS batch_ix,
                external_id::UUID
            FROM paginated
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

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_external_id uuid,
    next_external_id uuid,
    batch_size integer
);
CREATE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type,
    next_tenant_id uuid,
    next_inserted_at timestamptz,
    next_id bigint,
    next_type v1_payload_type,
    batch_size integer
) RETURNS TABLE (
    tenant_id UUID,
    id BIGINT,
    inserted_at TIMESTAMPTZ,
    external_id UUID,
    type v1_payload_type,
    location v1_payload_location,
    external_location_key TEXT,
    inline_content JSONB,
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS MATERIALIZED (
            SELECT tenant_id, id, inserted_at, external_id, type, location,
                external_location_key, inline_content, updated_at
            FROM %I
            WHERE
                (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            ORDER BY tenant_id, inserted_at, id, type

            -- Multiplying by two here to handle an edge case. There is a small chance we miss a row
            -- when a different row is inserted before it, in between us creating the chunks and selecting
            -- them. By multiplying by two to create a "candidate" set, we significantly reduce the chance of us missing
            -- rows in this way, since if a row is inserted before one of our last rows, we will still have
            -- the next row after it in the candidate set.
            LIMIT $9 * 2
        )

        SELECT tenant_id, id, inserted_at, external_id, type, location,
               external_location_key, inline_content, updated_at
        FROM candidates
        WHERE
            (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            AND (tenant_id, inserted_at, id, type) <= ($5, $6, $7, $8)
        ORDER BY tenant_id, inserted_at, id, type
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, next_tenant_id, next_inserted_at, next_id, next_type, batch_size;
END;
$$;

DROP FUNCTION compute_payload_batch_size(
    partition_date DATE,
    last_external_id UUID,
    batch_size INTEGER
);
CREATE FUNCTION compute_payload_batch_size(
    partition_date DATE,
    last_tenant_id UUID,
    last_inserted_at TIMESTAMPTZ,
    last_id BIGINT,
    last_type v1_payload_type,
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS (
            SELECT *
            FROM %I
            WHERE (tenant_id, inserted_at, id, type) >= ($1::UUID, $2::TIMESTAMPTZ, $3::BIGINT, $4::v1_payload_type)
            ORDER BY tenant_id, inserted_at, id, type
            LIMIT $5::INTEGER
        )

        SELECT COALESCE(SUM(pg_column_size(inline_content)), 0) AS total_size_bytes
        FROM candidates
    ', source_partition_name);

    EXECUTE query INTO result_size USING last_tenant_id, last_inserted_at, last_id, last_type, batch_size;

    RETURN result_size;
END;
$$;

DROP FUNCTION create_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_external_id uuid
);
CREATE FUNCTION create_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type
) RETURNS TABLE (
    lower_tenant_id UUID,
    lower_id BIGINT,
    lower_inserted_at TIMESTAMPTZ,
    lower_type v1_payload_type,
    upper_tenant_id UUID,
    upper_id BIGINT,
    upper_inserted_at TIMESTAMPTZ,
    upper_type v1_payload_type
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH paginated AS (
            SELECT tenant_id, id, inserted_at, type, ROW_NUMBER() OVER (ORDER BY tenant_id, inserted_at, id, type) AS rn
            FROM %I
            WHERE (tenant_id, inserted_at, id, type) > ($1, $2, $3, $4)
            ORDER BY tenant_id, inserted_at, id, type
            LIMIT $5::INTEGER
        ), lower_bounds AS (
            SELECT rn::INTEGER / $6::INTEGER AS batch_ix, tenant_id::UUID, id::BIGINT, inserted_at::TIMESTAMPTZ, type::v1_payload_type
            FROM paginated
            WHERE MOD(rn, $6::INTEGER) = 1
        ), upper_bounds AS (
            SELECT
                -- Using `CEIL` and subtracting 1 here to make the `batch_ix` zero indexed like the `lower_bounds` one is.
                -- We need the `CEIL` to handle the case where the number of rows in the window is not evenly divisible by the batch size,
                -- because without CEIL if e.g. there were 5 rows in the window and a batch size of two and we did integer division, we would end
                -- up with batches of index 0, 1, and 1 after dividing and subtracting. With float division and `CEIL`, we get 0, 1, and 2 as expected.
                -- Then we need to subtract one because we compute the batch index by using integer division on the lower bounds, which are all zero indexed.
                CEIL(rn::FLOAT / $6::FLOAT) - 1 AS batch_ix,
                tenant_id::UUID,
                id::BIGINT,
                inserted_at::TIMESTAMPTZ,
                type::v1_payload_type
            FROM paginated
            -- We want to include either the last row of each batch, or the last row of the entire paginated set, which may not line up with a batch end.
            WHERE MOD(rn, $6::INTEGER) = 0 OR rn = (SELECT MAX(rn) FROM paginated)
        )

        SELECT
            lb.tenant_id AS lower_tenant_id,
            lb.id AS lower_id,
            lb.inserted_at AS lower_inserted_at,
            lb.type AS lower_type,
            ub.tenant_id AS upper_tenant_id,
            ub.id AS upper_id,
            ub.inserted_at AS upper_inserted_at,
            ub.type AS upper_type
        FROM lower_bounds lb
        JOIN upper_bounds ub ON lb.batch_ix = ub.batch_ix
        ORDER BY lb.tenant_id, lb.inserted_at, lb.id, lb.type
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, window_size, chunk_size;
END;
$$;

ALTER TABLE v1_payload_cutover_job_offset DROP COLUMN last_external_id;

DROP FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    last_external_id uuid,
    next_external_id uuid,
    batch_size integer
);
CREATE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz,
    next_tenant_id uuid,
    next_external_id uuid,
    next_inserted_at timestamptz,
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
            WHERE
                (tenant_id, external_id, inserted_at) >= ($1, $2, $3)
            ORDER BY tenant_id, external_id, inserted_at

            -- Multiplying by two here to handle an edge case. There is a small chance we miss a row
            -- when a different row is inserted before it, in between us creating the chunks and selecting
            -- them. By multiplying by two to create a "candidate" set, we significantly reduce the chance of us missing
            -- rows in this way, since if a row is inserted before one of our last rows, we will still have
            -- the next row after it in the candidate set.
            LIMIT $7 * 2
        )

        SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
        FROM candidates
        WHERE
            (tenant_id, external_id, inserted_at) >= ($1, $2, $3)
            AND (tenant_id, external_id, inserted_at) <= ($4, $5, $6)
        ORDER BY tenant_id, external_id, inserted_at
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, next_tenant_id, next_external_id, next_inserted_at, batch_size;
END;
$$;

DROP FUNCTION compute_olap_payload_batch_size(
    partition_date DATE,
    last_external_id UUID,
    batch_size INTEGER
);
CREATE FUNCTION compute_olap_payload_batch_size(
    partition_date DATE,
    last_tenant_id UUID,
    last_external_id UUID,
    last_inserted_at TIMESTAMPTZ,
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
            WHERE (tenant_id, external_id, inserted_at) >= ($1::UUID, $2::UUID, $3::TIMESTAMPTZ)
            ORDER BY tenant_id, external_id, inserted_at
            LIMIT $4::INT
        )

        SELECT COALESCE(SUM(pg_column_size(inline_content)), 0) AS total_size_bytes
        FROM candidates
    ', source_partition_name);

    EXECUTE query INTO result_size USING last_tenant_id, last_external_id, last_inserted_at, batch_size;

    RETURN result_size;
END;
$$;

DROP FUNCTION create_olap_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_external_id uuid
);
CREATE FUNCTION create_olap_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz
) RETURNS TABLE (
    lower_tenant_id UUID,
    lower_external_id UUID,
    lower_inserted_at TIMESTAMPTZ,
    upper_tenant_id UUID,
    upper_external_id UUID,
    upper_inserted_at TIMESTAMPTZ
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
            SELECT tenant_id, external_id, inserted_at, ROW_NUMBER() OVER (ORDER BY tenant_id, external_id, inserted_at) AS rn
            FROM %I
            WHERE (tenant_id, external_id, inserted_at) > ($1, $2, $3)
            ORDER BY tenant_id, external_id, inserted_at
            LIMIT $4
        ), lower_bounds AS (
            SELECT rn::INTEGER / $5::INTEGER AS batch_ix, tenant_id::UUID, external_id::UUID, inserted_at::TIMESTAMPTZ
            FROM paginated
            WHERE MOD(rn, $5::INTEGER) = 1
        ), upper_bounds AS (
            SELECT
                CEIL(rn::FLOAT / $5::FLOAT) - 1 AS batch_ix,
                tenant_id::UUID,
                external_id::UUID,
                inserted_at::TIMESTAMPTZ
            FROM paginated
            WHERE MOD(rn, $5::INTEGER) = 0 OR rn = (SELECT MAX(rn) FROM paginated)
        )

        SELECT
            lb.tenant_id AS lower_tenant_id,
            lb.external_id AS lower_external_id,
            lb.inserted_at AS lower_inserted_at,
            ub.tenant_id AS upper_tenant_id,
            ub.external_id AS upper_external_id,
            ub.inserted_at AS upper_inserted_at
        FROM lower_bounds lb
        JOIN upper_bounds ub ON lb.batch_ix = ub.batch_ix
        ORDER BY lb.tenant_id, lb.external_id, lb.inserted_at
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, window_size, chunk_size;
END;
$$;

ALTER TABLE v1_payload ALTER COLUMN external_id DROP DEFAULT;
ALTER TABLE v1_payload ALTER COLUMN external_id DROP NOT NULL;

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
