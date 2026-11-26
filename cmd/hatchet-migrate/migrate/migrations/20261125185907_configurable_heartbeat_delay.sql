-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Worker"
    ADD COLUMN "maxHeartbeatDelayBeforeDeclaredDead" INTERVAL NOT NULL DEFAULT INTERVAL '30 seconds',
    ADD CONSTRAINT "Worker_maxHeartbeatDelayBeforeDeclaredDead_bounds"
        CHECK ("maxHeartbeatDelayBeforeDeclaredDead" BETWEEN INTERVAL '4 seconds' AND INTERVAL '1 hour') NOT VALID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Worker" DROP COLUMN "maxHeartbeatDelayBeforeDeclaredDead";
-- +goose StatementEnd
