-- +goose Up
ALTER TABLE "StepBatchConfig" ADD COLUMN IF NOT EXISTS "broadcastOutput" BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE "StepBatchConfig" DROP COLUMN IF EXISTS "broadcastOutput";
