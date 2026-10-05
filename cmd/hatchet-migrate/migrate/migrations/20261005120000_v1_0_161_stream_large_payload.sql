-- +goose Up
-- +goose StatementBegin
-- a message whose payload was uploaded ahead of its publish references it here,
-- with an empty payload of its own
ALTER TABLE v1_stream_message ADD COLUMN payload_id UUID;
ALTER TABLE v1_stream_message ADD COLUMN payload_inserted_at TIMESTAMPTZ;

-- Payloads too large for a gRPC publish. Insert-only; partitions are dropped
-- on the same schedule as v1_stream_message's, so an upload no publish
-- references needs no cleanup of its own.
CREATE TABLE v1_stream_payload (
    id UUID NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    payload BYTEA NOT NULL,

    CONSTRAINT v1_stream_payload_pkey PRIMARY KEY (tenant_id, id, inserted_at)
) PARTITION BY RANGE(inserted_at);

SELECT create_v1_hourly_range_partition('v1_stream_payload', NOW() + make_interval(hours => h))
FROM generate_series(0, 24) AS h;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_stream_payload;
ALTER TABLE v1_stream_message DROP COLUMN payload_inserted_at;
ALTER TABLE v1_stream_message DROP COLUMN payload_id;
-- +goose StatementEnd
