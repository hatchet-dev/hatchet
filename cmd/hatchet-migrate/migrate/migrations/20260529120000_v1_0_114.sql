-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LeaseKind" ADD VALUE 'PARTITION';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
