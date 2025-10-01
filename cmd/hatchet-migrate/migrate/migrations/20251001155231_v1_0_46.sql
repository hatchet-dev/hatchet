-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_check;
ALTER TABLE v1_payload ADD CONSTRAINT v1_payload_check CHECK (
    location = 'INLINE'
    OR
    (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
) NOT VALID;

ALTER TABLE v1_payload_wal DROP COLUMN operation;

DROP TYPE v1_payload_wal_operation;
CREATE TYPE v1_payload_wal_operation AS ENUM ('REPLICATE_TO_EXTERNAL', 'CUT_OVER_TO_EXTERNAL');

ALTER TABLE v1_payload_wal ADD COLUMN operation v1_payload_wal_operation NOT NULL DEFAULT 'REPLICATE_TO_EXTERNAL';

CREATE OR REPLACE FUNCTION find_matching_tenants_in_payload_wal_partition(
    partition_number INT
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_payload_wal_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT DISTINCT e.tenant_id
            FROM %I e
            WHERE
                (e.offload_at <= NOW() AND e.operation = ''CUT_OVER_TO_EXTERNAL''::v1_payload_wal_operation)
                OR
                e.operation = ''REPLICATE_TO_EXTERNAL''::v1_payload_wal_operation
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload_wal DROP COLUMN operation;

DROP TYPE v1_payload_wal_operation;
CREATE TYPE v1_payload_wal_operation AS ENUM ('CREATE', 'UPDATE', 'DELETE');

ALTER TABLE v1_payload_wal ADD COLUMN operation v1_payload_wal_operation NOT NULL DEFAULT 'CREATE';

ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_check;
ALTER TABLE v1_payload ADD CONSTRAINT v1_payload_check CHECK (
    (location = 'INLINE' AND external_location_key IS NULL)
    OR
    (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
) NOT VALID;

CREATE OR REPLACE FUNCTION find_matching_tenants_in_payload_wal_partition(
    partition_number INT
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_payload_wal_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT DISTINCT e.tenant_id
            FROM %I e
            WHERE e.offload_at < NOW()
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;
-- +goose StatementEnd
