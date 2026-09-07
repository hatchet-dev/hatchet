-- +goose Up
-- +goose NO TRANSACTION
-- Runs without a wrapping transaction on purpose: Postgres refuses to use a new enum value in
-- the same transaction that added it, and the partial index below filters on that value. Both
-- statements are idempotent so a partially applied migration can simply be re-run.

ALTER TYPE v1_operator_kind ADD VALUE IF NOT EXISTS 'GRPC';

-- GRPC operators are upserted by name on connect, so the name must be unique within a tenant for
-- that kind. Other kinds are created through the REST API and may share names.
CREATE UNIQUE INDEX IF NOT EXISTS v1_operator_grpc_tenant_name_key ON v1_operator (tenant_id, name) WHERE kind = 'GRPC';

-- +goose Down
-- Postgres cannot drop a value from an enum type, so 'GRPC' stays on v1_operator_kind; only the
-- index is removed.
DROP INDEX IF EXISTS v1_operator_grpc_tenant_name_key;
