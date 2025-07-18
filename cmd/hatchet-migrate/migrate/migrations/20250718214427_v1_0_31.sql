-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_events_olap ADD COLUMN triggering_webhook_name TEXT;

CREATE TABLE v1_incoming_webhook_validation_failures (
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,

    tenant_id UUID NOT NULL,

    -- webhook names are tenant-unique
    incoming_webhook_name TEXT NOT NULL,

    error TEXT NOT NULL,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_incoming_webhook_validation_failures_tenant_id_incoming_webhook_name_idx ON v1_incoming_webhook_validation_failures (tenant_id, incoming_webhook_name);

SELECT create_v1_range_partition('v1_incoming_webhook_validation_failures', NOW()::DATE);
SELECT create_v1_range_partition('v1_incoming_webhook_validation_failures', (NOW() - INTERVAL '1 day')::DATE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_events_olap DROP COLUMN triggering_webhook_name;
DROP TABLE v1_incoming_webhook_validation_failures;
-- +goose StatementEnd
