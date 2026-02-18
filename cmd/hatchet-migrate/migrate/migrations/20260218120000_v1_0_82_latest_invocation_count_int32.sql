-- +goose Up
-- Alters latest_invocation_count from BIGINT to INTEGER for invocation counts (int32) across the stack.
-- In PostgreSQL 11+, this propagates to all partitions of v1_durable_event_log_file.
ALTER TABLE v1_durable_event_log_file
    ALTER COLUMN latest_invocation_count TYPE INTEGER;

-- +goose Down
ALTER TABLE v1_durable_event_log_file
    ALTER COLUMN latest_invocation_count TYPE BIGINT;
