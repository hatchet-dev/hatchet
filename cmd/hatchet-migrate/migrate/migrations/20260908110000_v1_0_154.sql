-- +goose Up
-- +goose NO TRANSACTION
-- Runs without a wrapping transaction on purpose: Postgres refuses to use a new enum value in
-- the same transaction that added it, and the following migration's serverless tables are
-- created for operators of this kind. The statement is idempotent so a partially applied
-- migration can simply be re-run.

-- SERVERLESS operators anchor the Worker rows of the serverless operator (one per owned
-- (tenant, shard) unit). Leasing for them lives in v1_serverless_lease, not in ClaimOperators.
ALTER TYPE v1_operator_kind ADD VALUE IF NOT EXISTS 'SERVERLESS';

-- +goose Down
-- Postgres cannot drop a value from an enum type, so 'SERVERLESS' stays on v1_operator_kind.
-- Nothing to undo.
