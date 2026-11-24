-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Worker" ADD COLUMN "maxHeartbeatDelayBeforeDeclaredDead" INTERVAL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Worker" DROP COLUMN "maxHeartbeatDelayBeforeDeclaredDead";
-- +goose StatementEnd
