-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Worker" ADD COLUMN "actionHash" BYTEA;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Worker" DROP COLUMN "actionHash";
-- +goose StatementEnd
