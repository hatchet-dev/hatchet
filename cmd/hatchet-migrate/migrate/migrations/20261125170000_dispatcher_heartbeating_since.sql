-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Dispatcher" ADD COLUMN IF NOT EXISTS "heartbeatingSince" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Dispatcher" DROP COLUMN IF EXISTS "heartbeatingSince";
-- +goose StatementEnd
