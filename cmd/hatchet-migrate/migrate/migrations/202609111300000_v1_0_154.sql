-- +goose NO TRANSACTION
-- Runs without a wrapping transaction on purpose: goose then sends each statement on its own in
-- autocommit mode. Postgres refuses to use a value that ALTER TYPE ... ADD VALUE added in the
-- same transaction, and the backfills below use the values this file adds, so the statements
-- have to commit one at a time. The cost is that a failure part way through leaves the earlier
-- statements applied; undo them by hand with the Down section before re-running.
--
-- The goose id is fifteen digits because main's 202609101223115_v1_0_153.sql is, and a version
-- has to sort after every applied one or goose refuses it as a missing migration.

-- +goose Up

-- What an operator is: GRPC is a contract operator written against pkg/operator, hostable in
-- process or out of process. The value is added before the column below so an existing database
-- and a fresh one end with the same enum.
ALTER TYPE v1_operator_kind ADD VALUE IF NOT EXISTS 'GRPC';

-- Who keeps an operator alive is a separate axis from what it is. For a DISPATCHER row the
-- dispatcher claims the row through ClaimOperators and builds the operator from a factory
-- inside the engine; a SELF row keeps itself alive, through a Listen stream out of process or
-- its own leaser in process, and is never claimed.
CREATE TYPE v1_operator_leasing_manager AS ENUM ('SELF', 'DISPATCHER');

ALTER TABLE v1_operator ADD COLUMN leasing_manager v1_operator_leasing_manager NOT NULL DEFAULT 'SELF';

-- Every DAG operator row was claimed by a dispatcher before the column existed.
UPDATE v1_operator SET leasing_manager = 'DISPATCHER' WHERE kind = 'DAG';

-- Rows are unique per (tenant, name, kind) from here on: GRPC rows are upserted by name on
-- registration, and the other kinds follow the same rule. A duplicate (possible from a race
-- creating a tenant's DAG operator) is renamed rather than deleted: the oldest row keeps its
-- name, a later one gets its id as a suffix.
UPDATE v1_operator dup
SET name = dup.name || '-' || dup.id::text
FROM v1_operator first
WHERE
    first.tenant_id = dup.tenant_id
    AND first.name = dup.name
    AND first.kind = dup.kind
    AND (first.created_at, first.id) < (dup.created_at, dup.id);

CREATE UNIQUE INDEX v1_operator_tenant_name_kind_key ON v1_operator (tenant_id, name, kind);

-- "operatorActionCount" is the number of "_ActionToWorker" rows the worker holds, kept for the
-- per-operator action budget: every path that links or unlinks actions maintains it, so the
-- budget is a sum over the operator's workers rather than a count of their links. Only operator
-- workers are read, and only they are backfilled; an SDK worker's count is zero until it next
-- registers and nothing reads it. "actionHash" is untouched: the hash encoding is the one
-- workers already carry.
ALTER TABLE "Worker" ADD COLUMN "operatorActionCount" INTEGER NOT NULL DEFAULT 0;

UPDATE "Worker" w
SET "operatorActionCount" = (SELECT count(*) FROM "_ActionToWorker" aw WHERE aw."B" = w."id")
WHERE w."operatorId" IS NOT NULL;

-- Whether a worker counts against the tenant's WORKER and WORKER_SLOT limits is decided when
-- the worker is created, by whoever hosts it: the in-process host exempts every worker it
-- creates, the wire meters every worker it registers. It is a fact about the worker, not about
-- the operator row, so the limit queries read it here rather than through a join.
ALTER TABLE "Worker" ADD COLUMN "exemptFromLimits" BOOLEAN NOT NULL DEFAULT false;

-- The DAG operator's workers were the only ones the limit queries left out before the column
-- existed.
UPDATE "Worker" w
SET "exemptFromLimits" = true
FROM v1_operator op
WHERE op.id = w."operatorId" AND op.kind = 'DAG';

-- +goose Down

ALTER TABLE "Worker" DROP COLUMN "exemptFromLimits";

ALTER TABLE "Worker" DROP COLUMN "operatorActionCount";

DROP INDEX IF EXISTS v1_operator_tenant_name_kind_key;

ALTER TABLE v1_operator DROP COLUMN leasing_manager;

DROP TYPE IF EXISTS v1_operator_leasing_manager;

-- Postgres cannot drop a value from an enum type, so 'GRPC' stays on v1_operator_kind. Renamed
-- duplicates keep their new names.
