-- +goose Up
ALTER TYPE v1_readable_status_olap ADD VALUE IF NOT EXISTS 'EVICTED';

-- +goose Down
-- NOTE: Postgres does not support removing enum values.
-- The 'EVICTED' value in v1_readable_status_olap cannot be reverted.
-- Any EVICTED partitions created by this migration would need to be merged/dropped separately.
