-- +goose no transaction
-- +goose Up
-- https://stackoverflow.com/a/70958260
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_payload_wal_0_poll_idx ON v1_payload_wal_0 (tenant_id, offload_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_payload_wal_1_poll_idx ON v1_payload_wal_1 (tenant_id, offload_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_payload_wal_2_poll_idx ON v1_payload_wal_2 (tenant_id, offload_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_payload_wal_3_poll_idx ON v1_payload_wal_3 (tenant_id, offload_at);
CREATE INDEX IF NOT EXISTS v1_payload_wal_poll_idx ON v1_payload_wal (tenant_id, offload_at);

-- +goose Down
DROP INDEX IF EXISTS v1_payload_wal_poll_idx;
