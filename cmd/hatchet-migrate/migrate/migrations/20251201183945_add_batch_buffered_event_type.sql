-- +goose Up
-- +goose StatementBegin
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'BATCH_BUFFERED';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'no-op: cannot remove enum values';
-- +goose StatementEnd
