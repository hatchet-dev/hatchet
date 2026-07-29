-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_idempotency_key DROP CONSTRAINT v1_idempotency_key_pkey;
ALTER TABLE v1_idempotency_key ADD CONSTRAINT v1_idempotency_key_pkey PRIMARY KEY USING INDEX v1_idempotency_key_unique_tenant_key;
CREATE INDEX v1_idempotency_key_expires_at_idx ON v1_idempotency_key (tenant_id, expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_idempotency_key DROP CONSTRAINT v1_idempotency_key_pkey;
ALTER TABLE v1_idempotency_key ADD CONSTRAINT v1_idempotency_key_pkey PRIMARY KEY (tenant_id, expires_at, key);
CREATE UNIQUE INDEX v1_idempotency_key_unique_tenant_key ON v1_idempotency_key (tenant_id, key);
DROP INDEX v1_idempotency_key_expires_at_idx;
-- +goose StatementEnd
