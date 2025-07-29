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

CREATE TYPE v1_payload_wal_operation AS ENUM ('INSERT', 'UPDATE', 'DELETE');

CREATE TABLE v1_payload_wal (
    tenant_id UUID NOT NULL,
    offload_at TIMESTAMPTZ NOT NULL,
    payload_id BIGINT NOT NULL,
    payload_inserted_at TIMESTAMPTZ NOT NULL,
    payload_type v1_payload_type NOT NULL,
    operation v1_payload_wal_operation NOT NULL,

    PRIMARY KEY (offload_at, payload_id, payload_inserted_at, payload_type, tenant_id),
    CONSTRAINT "v1_payload_wal_payload" FOREIGN KEY (payload_id, payload_inserted_at, payload_type, tenant_id) REFERENCES v1_payload (id, inserted_at, type, tenant_id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payload_wal;
DROP TYPE v1_payload_wal_operation;

DROP TABLE v1_payload;
DROP TYPE v1_payload_type;
-- +goose StatementEnd
