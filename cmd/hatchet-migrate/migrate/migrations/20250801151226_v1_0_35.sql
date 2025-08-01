-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_idempotency_key (
    tenant_id UUID NOT NULL,

    key TEXT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,
    claimed_by_external_id UUID,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, expires_at, key)
);

CREATE UNIQUE INDEX v1_idempotency_key_expires_at_idx ON v1_idempotency_key (tenant_id, key);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_idempotency_key;
-- +goose StatementEnd
