-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_worker_slot_config_worker_id_idx ON v1_worker_slot_config (worker_id);

-- +goose Down
DROP INDEX IF EXISTS v1_worker_slot_config_worker_id_idx;
