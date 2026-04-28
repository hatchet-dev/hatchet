-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Worker" ADD COLUMN "actionsHash" BYTEA;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Worker" DROP COLUMN "actionsHash";
-- +goose StatementEnd
