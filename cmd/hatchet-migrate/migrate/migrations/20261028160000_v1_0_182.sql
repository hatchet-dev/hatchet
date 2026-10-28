-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Dispatcher" ADD COLUMN IF NOT EXISTS "brokerId" TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Dispatcher" DROP COLUMN IF EXISTS "brokerId";
-- +goose StatementEnd
