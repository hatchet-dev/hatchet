-- +goose Up
-- +goose StatementBegin

-- "actionCount" is the number of "_ActionToWorker" rows the worker holds. Every path that
-- links or unlinks actions maintains it, so the per-operator action budget is a sum over the
-- operator's workers rather than a count of their links. It is backfilled for operator
-- workers, the only ones the budget applies to; SDK workers start at the default and are
-- recounted when they next register.
ALTER TABLE "Worker" ADD COLUMN IF NOT EXISTS "actionCount" INTEGER NOT NULL DEFAULT 0;

UPDATE "Worker" w
SET "actionCount" = (SELECT count(*) FROM "_ActionToWorker" aw WHERE aw."B" = w."id")
WHERE w."operatorId" IS NOT NULL;

-- "actionHash" is untouched: the hash encoding is the one workers already carry, so a worker
-- registered by an older engine keeps a hash the new engine computes the same way.

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- The column goes; "actionHash" was never rewritten, so there is nothing to restore.
ALTER TABLE "Worker" DROP COLUMN IF EXISTS "actionCount";

-- +goose StatementEnd
