-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION compute_payload_batch_size(
    partition_date DATE
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
            LIMIT $5::INT
        )

        SELECT SUM(pg_column_size(inline_content)) AS total_size_bytes
        FROM candidates
    ', source_partition_name);

    EXECUTE query INTO result_size;

    RETURN result_size;
END;
$$;

CREATE OR REPLACE FUNCTION compute_olap_payload_batch_size(
    partition_date DATE
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

        SELECT SUM(pg_column_size(inline_content)) AS total_size_bytes
        FROM candidates
    ', source_partition_name);

    EXECUTE query INTO result_size;

    RETURN result_size;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION compute_payload_batch_size(DATE);
DROP FUNCTION compute_olap_payload_batch_size(DATE);
-- +goose StatementEnd
