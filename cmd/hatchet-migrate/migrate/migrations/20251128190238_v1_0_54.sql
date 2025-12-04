-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_payload_cutover_job_offset (
    key DATE PRIMARY KEY,
    last_offset BIGINT NOT NULL,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE OR REPLACE FUNCTION copy_v1_payload_partition_structure(
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payload_offload_tmp_%s', partition_date_str) INTO target_table_name;
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
                INSERT INTO %I (tenant_id, id, inserted_at, external_id, type, location, external_location_key, inline_content, updated_at)
                VALUES (NEW.tenant_id, NEW.id, NEW.inserted_at, NEW.external_id, NEW.type, NEW.location, NEW.external_location_key, NEW.inline_content, NEW.updated_at)
                ON CONFLICT (tenant_id, id, inserted_at, type) DO UPDATE
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
                    AND id = NEW.id
                    AND inserted_at = NEW.inserted_at
                    AND type = NEW.type;
                RETURN NEW;
            ELSIF TG_OP = ''DELETE'' THEN
                DELETE FROM %I
                WHERE
                    tenant_id = OLD.tenant_id
                    AND id = OLD.id
                    AND inserted_at = OLD.inserted_at
                    AND type = OLD.type;
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

CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    limit_param int,
    offset_param bigint
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
        SELECT tenant_id, id, inserted_at, external_id, type, location,
               external_location_key, inline_content, updated_at
        FROM %I
        ORDER BY tenant_id, inserted_at, id, type
        LIMIT $1
        OFFSET $2
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING limit_param, offset_param;
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payload_cutover_job_offset;
DROP FUNCTION copy_v1_payload_partition_structure(date);
DROP FUNCTION list_paginated_payloads_for_offload(date, int, bigint);
DROP FUNCTION swap_v1_payload_partition_with_temp(date);
-- +goose StatementEnd
