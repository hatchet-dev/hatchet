-- +goose Up
-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION diff_payload_source_and_target_partitions(date);
DROP FUNCTION diff_olap_payload_source_and_target_partitions(date);
-- +goose StatementEnd
