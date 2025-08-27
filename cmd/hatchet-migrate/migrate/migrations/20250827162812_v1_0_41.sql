-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_payload_type AS ENUM ('TASK_INPUT', 'DAG_INPUT', 'TASK_OUTPUT');

CREATE TABLE v1_payload (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL,
    type v1_payload_type NOT NULL,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, inserted_at, id, type)
) PARTITION BY RANGE(inserted_at);

SELECT create_v1_range_partition('v1_payload'::TEXT, NOW()::DATE);
SELECT create_v1_range_partition('v1_payload'::TEXT, (NOW() + INTERVAL '1 day')::DATE);

CREATE TYPE v1_payload_wal_operation AS ENUM ('CREATE', 'UPDATE', 'DELETE');

CREATE TABLE v1_payload_wal (
    tenant_id UUID NOT NULL,
    offload_at TIMESTAMPTZ NOT NULL,
    payload_id BIGINT NOT NULL,
    payload_inserted_at TIMESTAMPTZ NOT NULL,
    payload_type v1_payload_type NOT NULL,
    operation v1_payload_wal_operation NOT NULL,

    offload_process_lease_id UUID,
    offload_process_lease_expires_at TIMESTAMPTZ,

    PRIMARY KEY (offload_at, tenant_id, payload_id, payload_inserted_at, payload_type),
    CONSTRAINT "v1_payload_wal_payload" FOREIGN KEY (payload_id, payload_inserted_at, payload_type, tenant_id) REFERENCES v1_payload (id, inserted_at, type, tenant_id) ON DELETE CASCADE
) PARTITION BY HASH (tenant_id);

SELECT create_v1_hash_partitions('v1_payload_wal'::TEXT, 4);

CREATE INDEX idx_payload_wal_lease_expiry ON v1_payload_wal (tenant_id, offload_process_lease_id) WHERE offload_process_lease_id IS NOT NULL;
CREATE INDEX idx_payload_wal_lease_acquisition ON v1_payload_wal (tenant_id, offload_process_lease_id) WHERE offload_process_lease_id IS NULL;

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
                e.offload_at < NOW()
                AND (e.offload_process_lease_id IS NULL OR e.offload_process_lease_expires_at < NOW())
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payload_wal;
DROP TYPE v1_payload_wal_operation;

DROP TABLE v1_payload;
DROP TYPE v1_payload_type;
-- +goose StatementEnd
