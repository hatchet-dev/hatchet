-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    partition_row record;
    renamed_name text;
BEGIN
    LOCK TABLE v1_payload IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE v1_payload_new IN ACCESS EXCLUSIVE MODE;

    DROP TRIGGER IF EXISTS v1_payload_mirror ON v1_payload;
    DROP FUNCTION IF EXISTS v1_payload_mirror_fn();

    DROP TABLE v1_payload CASCADE;

    ALTER TABLE v1_payload_new RENAME TO v1_payload;
    ALTER INDEX v1_payload_new_pkey RENAME TO v1_payload_pkey;

    FOR partition_row IN
        SELECT inhrelid::regclass::text AS partition_name
        FROM pg_inherits
        WHERE inhparent = 'v1_payload'::regclass
    LOOP
        renamed_name := 'v1_payload_' || substring(partition_row.partition_name from '(\d{8})$');
        EXECUTE format('ALTER TABLE %I RENAME TO %I', partition_row.partition_name, renamed_name);

        IF to_regclass(partition_row.partition_name || '_pkey') IS NOT NULL THEN
            EXECUTE format('ALTER INDEX %I RENAME TO %I', partition_row.partition_name || '_pkey', renamed_name || '_pkey');
        END IF;
    END LOOP;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
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
            inserted_at_date IS NOT NULL
            AND inserted_at_date >= %L::DATE
            AND inserted_at_date < %L::DATE
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
                INSERT INTO %I (tenant_id, id, inserted_at, inserted_at_date, external_id, type, location, external_location_key, inline_content, updated_at)
                VALUES (NEW.tenant_id, NEW.id, NEW.inserted_at, NEW.inserted_at_date, NEW.external_id, NEW.type, NEW.location, NEW.external_location_key, NEW.inline_content, NEW.updated_at)
                ON CONFLICT (external_id, inserted_at_date) DO UPDATE
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
                    external_id = NEW.external_id
                    AND inserted_at_date = NEW.inserted_at_date;
                RETURN NEW;
            ELSIF TG_OP = ''DELETE'' THEN
                DELETE FROM %I
                WHERE
                    external_id = OLD.external_id
                    AND inserted_at_date = OLD.inserted_at_date;
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
-- +goose StatementEnd
-- +goose StatementBegin
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
-- +goose StatementBegin
DROP FUNCTION IF EXISTS diff_payload_source_and_target_partitions(date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $migration$
DECLARE
    partition_row record;
    renamed_name text;
    partition_date date;
BEGIN
    LOCK TABLE v1_payload IN ACCESS EXCLUSIVE MODE;

    ALTER TABLE v1_payload RENAME TO v1_payload_new;
    ALTER INDEX v1_payload_pkey RENAME TO v1_payload_new_pkey;

    FOR partition_row IN
        SELECT inhrelid::regclass::text AS partition_name
        FROM pg_inherits
        WHERE inhparent = 'v1_payload_new'::regclass
    LOOP
        renamed_name := 'v1_payload_new_' || substring(partition_row.partition_name from '(\d{8})$');
        EXECUTE format('ALTER TABLE %I RENAME TO %I', partition_row.partition_name, renamed_name);

        IF to_regclass(partition_row.partition_name || '_pkey') IS NOT NULL THEN
            EXECUTE format('ALTER INDEX %I RENAME TO %I', partition_row.partition_name || '_pkey', renamed_name || '_pkey');
        END IF;
    END LOOP;

    CREATE TABLE v1_payload (
        tenant_id UUID NOT NULL,
        id BIGINT NOT NULL,
        inserted_at TIMESTAMPTZ NOT NULL,
        inserted_at_date DATE NOT NULL DEFAULT CURRENT_TIMESTAMP::DATE,
        external_id UUID NOT NULL DEFAULT gen_random_uuid(),
        type v1_payload_type NOT NULL,
        location v1_payload_location NOT NULL,
        external_location_key TEXT,
        inline_content JSONB,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

        PRIMARY KEY (tenant_id, inserted_at, id, type),
        CONSTRAINT v1_payload_check CHECK (
            location = 'INLINE'
            OR
            (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
        )
    ) PARTITION BY RANGE(inserted_at);

    FOR partition_row IN
        SELECT inhrelid::regclass::text AS partition_name
        FROM pg_inherits
        WHERE inhparent = 'v1_payload_new'::regclass
    LOOP
        partition_date := to_date(substring(partition_row.partition_name from '(\d{8})$'), 'YYYYMMDD');
        PERFORM create_v1_range_partition('v1_payload', partition_date);
    END LOOP;

    INSERT INTO v1_payload (tenant_id, id, inserted_at, inserted_at_date, external_id, type, location, external_location_key, inline_content, updated_at)
    SELECT tenant_id, id, inserted_at, inserted_at_date, external_id, type, location, external_location_key, inline_content, updated_at
    FROM v1_payload_new;

    CREATE INDEX v1_payload_external_id_idx ON v1_payload (external_id ASC);
END;
$migration$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_payload_mirror_fn()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'INSERT' THEN
		INSERT INTO v1_payload_new (
			tenant_id,
			id,
			inserted_at,
			inserted_at_date,
			external_id,
			type,
			location,
			external_location_key,
			inline_content,
			updated_at
		) VALUES (
			NEW.tenant_id,
			NEW.id,
			NEW.inserted_at,
			NEW.inserted_at_date,
			NEW.external_id,
			NEW.type,
			NEW.location,
			NEW.external_location_key,
			NEW.inline_content,
			NEW.updated_at
		)
		ON CONFLICT (external_id, inserted_at_date)
		DO UPDATE SET
			tenant_id             = EXCLUDED.tenant_id,
			id                    = EXCLUDED.id,
			inserted_at           = EXCLUDED.inserted_at,
			type                  = EXCLUDED.type,
			location              = EXCLUDED.location,
			external_location_key = EXCLUDED.external_location_key,
			inline_content        = EXCLUDED.inline_content,
			updated_at            = EXCLUDED.updated_at;
		RETURN NEW;
	ELSIF TG_OP = 'UPDATE' THEN
		UPDATE v1_payload_new SET
			tenant_id             = NEW.tenant_id,
			id                    = NEW.id,
			inserted_at           = NEW.inserted_at,
			type                  = NEW.type,
			location              = NEW.location,
			external_location_key = NEW.external_location_key,
			inline_content        = NEW.inline_content,
			updated_at            = NEW.updated_at
		WHERE external_id = NEW.external_id AND inserted_at_date = NEW.inserted_at_date;
		RETURN NEW;
	ELSIF TG_OP = 'DELETE' THEN
		DELETE FROM v1_payload_new
		WHERE external_id = OLD.external_id AND inserted_at_date = OLD.inserted_at_date;
		RETURN OLD;
	END IF;
	RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE TRIGGER v1_payload_mirror
AFTER INSERT OR UPDATE OR DELETE ON v1_payload
FOR EACH ROW EXECUTE FUNCTION v1_payload_mirror_fn();
-- +goose StatementEnd
-- +goose StatementBegin
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
-- +goose StatementEnd
-- +goose StatementBegin
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
-- +goose StatementEnd
-- +goose StatementBegin
DROP FUNCTION IF EXISTS diff_payload_source_and_target_partitions(date);
CREATE OR REPLACE FUNCTION diff_payload_source_and_target_partitions(
    partition_date date
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
    temp_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payload_offload_tmp_%s', partition_date_str) INTO temp_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        SELECT tenant_id, id, inserted_at, external_id, type, location, external_location_key, inline_content, updated_at
        FROM %I source
        WHERE NOT EXISTS (
            SELECT 1
            FROM %I AS target
            WHERE
                source.tenant_id = target.tenant_id
                AND source.inserted_at = target.inserted_at
                AND source.id = target.id
                AND source.type = target.type
        )
    ', source_partition_name, temp_partition_name);

    RETURN QUERY EXECUTE query;
END;
$$;
-- +goose StatementEnd
