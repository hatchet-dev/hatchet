-- +goose Up
-- +goose StatementBegin
DROP FUNCTION list_paginated_payloads_for_offload(date, int, uuid, timestamptz, bigint, v1_payload_type);
CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type,
    next_tenant_id uuid,
    next_inserted_at timestamptz,
    next_id bigint,
    next_type v1_payload_type
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
        WHERE
            (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            AND (tenant_id, inserted_at, id, type) <= ($5, $6, $7, $8)
        ORDER BY tenant_id, inserted_at, id, type
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, next_tenant_id, next_inserted_at, next_id, next_type;
END;
$$;

DROP FUNCTION list_paginated_olap_payloads_for_offload(date, int, uuid, uuid, timestamptz);
CREATE OR REPLACE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz,
    next_tenant_id uuid,
    next_external_id uuid,
    next_inserted_at timestamptz
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
        SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
        FROM %I
        WHERE
            (tenant_id, external_id, inserted_at) >= ($1, $2, $3)
            AND (tenant_id, external_id, inserted_at) <= ($4, $5, $6)
        ORDER BY tenant_id, external_id, inserted_at
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, next_tenant_id, next_external_id, next_inserted_at;
END;
$$;

CREATE OR REPLACE FUNCTION create_payload_offload_range_chunks(
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
                CEIL(rn::FLOAT / $6::FLOAT) - 1 AS batch_ix,
                tenant_id::UUID,
                id::BIGINT,
                inserted_at::TIMESTAMPTZ,
                type::v1_payload_type
            FROM paginated
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

CREATE OR REPLACE FUNCTION create_olap_payload_offload_range_chunks(
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    limit_param int,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type
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
        WHERE (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
        ORDER BY tenant_id, inserted_at, id, type
        LIMIT $5
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, limit_param;
END;
$$;

CREATE OR REPLACE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    limit_param int,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz
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
        SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
        FROM %I
        WHERE (tenant_id, external_id, inserted_at) >= ($1, $2, $3)
        ORDER BY tenant_id, external_id, inserted_at
        LIMIT $4
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, limit_param;
END;
$$;

DROP FUNCTION create_payload_offload_range_chunks(date, int, int, uuid, timestamptz, bigint, v1_payload_type);
DROP FUNCTION create_olap_payload_offload_range_chunks(date, int, int, uuid, uuid, timestamptz);
-- +goose StatementEnd
