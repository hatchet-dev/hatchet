-- +goose Up
-- +goose StatementBegin
-- Identifies the listener session that last activated the worker, so only that session can
-- deactivate it when listener sessions overlap.
ALTER TABLE "Worker" ADD COLUMN "lastListenerSessionId" UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Worker" DROP COLUMN "lastListenerSessionId";
-- +goose StatementEnd
