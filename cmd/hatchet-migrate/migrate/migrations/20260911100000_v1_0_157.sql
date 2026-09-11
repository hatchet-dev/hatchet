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

-- The canonical action hash changed its encoding: each id is now prefixed by its byte length
-- instead of followed by a separator, so ids containing the separator no longer collide.
-- Active workers are recomputed in place, since the scheduler groups them by hash and a
-- stale value would group them with nothing. Every other worker with a hash has it cleared:
-- the scheduler reads a worker without a hash through the join, and an operator worker
-- refreshes its hash when a session resumes it, so recomputing every inactive worker's
-- links here would only be work for rows that mostly age out.
UPDATE "Worker" w
SET "actionHash" = (
    SELECT sha256(coalesce(
        string_agg(
            int4send(octet_length(convert_to(a."actionId", 'UTF8'))) || convert_to(a."actionId", 'UTF8'),
            ''::bytea
            ORDER BY a."actionId" COLLATE "C"
        ),
        ''::bytea
    ))
    FROM "_ActionToWorker" aw
    JOIN "Action" a ON a."id" = aw."A"
    WHERE aw."B" = w."id"
)
WHERE w."isActive" AND w."actionHash" IS NOT NULL;

UPDATE "Worker" w
SET "actionHash" = NULL
WHERE NOT w."isActive" AND w."actionHash" IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- The column goes; the hashes are not restored to the old encoding, because the old code
-- recomputes a worker's hash from its links on the next registration or delta and reads a
-- NULL hash through the join in the meantime.
ALTER TABLE "Worker" DROP COLUMN IF EXISTS "actionCount";

-- +goose StatementEnd
