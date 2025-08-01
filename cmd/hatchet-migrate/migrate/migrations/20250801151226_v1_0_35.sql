-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_idempotency_key (
    tenant_id UUID NOT NULL,
    key TEXT NOT NULL,
    is_filled BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, key, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_idempotency_key_expires_at_idx ON v1_idempotency_key (tenant_id, expires_at DESC);

SELECT create_v1_range_partition('v1_idempotency_key', NOW()::DATE);
SELECT create_v1_range_partition('v1_idempotency_key', (NOW() + INTERVAL '1 day')::DATE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_idempotency_key;
-- +goose StatementEnd
