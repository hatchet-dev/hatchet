-- +goose Up
-- +goose StatementBegin

-- "operatorActionCount" is the number of "_ActionToWorker" rows the worker holds, kept for
-- the per-operator action budget: every path that links or unlinks actions maintains it, so
-- the budget is a sum over the operator's workers rather than a count of their links. Only
-- operator workers are read, and only they are backfilled; an SDK worker's count is zero
-- until it next registers and nothing reads it.
ALTER TABLE "Worker" ADD COLUMN IF NOT EXISTS "operatorActionCount" INTEGER NOT NULL DEFAULT 0;

UPDATE "Worker" w
SET "operatorActionCount" = (SELECT count(*) FROM "_ActionToWorker" aw WHERE aw."B" = w."id")
WHERE w."operatorId" IS NOT NULL;

-- "actionHash" is untouched: the hash encoding is the one workers already carry, so a worker
-- registered by an older engine keeps a hash the new engine computes the same way.

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- The column goes; "actionHash" was never rewritten, so there is nothing to restore.
ALTER TABLE "Worker" DROP COLUMN IF EXISTS "operatorActionCount";

-- +goose StatementEnd
