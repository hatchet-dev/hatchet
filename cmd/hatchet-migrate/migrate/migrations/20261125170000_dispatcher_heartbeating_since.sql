-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Dispatcher" ADD COLUMN IF NOT EXISTS "heartbeatingSince" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Dispatcher" DROP COLUMN IF EXISTS "heartbeatingSince";
-- +goose StatementEnd
