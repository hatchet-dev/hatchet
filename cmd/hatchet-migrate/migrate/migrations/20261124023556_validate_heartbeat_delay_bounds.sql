-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Worker" VALIDATE CONSTRAINT "Worker_maxHeartbeatDelayBeforeDeclaredDead_bounds";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- no op
-- +goose StatementEnd
