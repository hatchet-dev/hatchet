-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LeaseKind" ADD VALUE 'TABLE_PARTITION_MAINTENANCE';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
