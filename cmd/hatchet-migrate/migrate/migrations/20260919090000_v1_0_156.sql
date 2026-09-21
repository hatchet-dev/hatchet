-- +goose NO TRANSACTION
-- Runs without a wrapping transaction so the ALTER TABLE on "Worker" commits, and releases its
-- lock, before the backfills run. A failure part way through leaves the earlier statements
-- applied, so the "Worker" statement can be re-run as is.

-- +goose Up

ALTER TABLE "Worker"
    ADD COLUMN IF NOT EXISTS "operatorActionCount" INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "isExemptFromLimits" BOOLEAN NOT NULL DEFAULT false;

ALTER TYPE v1_operator_kind ADD VALUE IF NOT EXISTS 'GRPC';

CREATE TYPE v1_operator_leasing_manager AS ENUM ('SELF', 'DISPATCHER');

ALTER TABLE v1_operator ADD COLUMN leasing_manager v1_operator_leasing_manager NOT NULL DEFAULT 'SELF';

-- Every DAG operator row was claimed by a dispatcher before the column existed.
UPDATE v1_operator SET leasing_manager = 'DISPATCHER' WHERE kind = 'DAG';

-- ensureDAGOperator checks and then inserts, so two concurrent registrations could have created
-- a tenant's DAG operator twice. The oldest row keeps its name so the unique index can build.
UPDATE v1_operator dup
SET name = dup.name || '-' || dup.id::text
FROM v1_operator first
WHERE
    first.tenant_id = dup.tenant_id
    AND first.name = dup.name
    AND first.kind = dup.kind
    AND (first.created_at, first.id) < (dup.created_at, dup.id);

CREATE UNIQUE INDEX v1_operator_tenant_name_kind_key ON v1_operator (tenant_id, name, kind);

-- Only operator workers' counts are read.
UPDATE "Worker" w
SET "operatorActionCount" = (SELECT count(*) FROM "_ActionToWorker" aw WHERE aw."B" = w."id")
WHERE w."operatorId" IS NOT NULL;

-- The DAG operator's workers were the only ones the limit queries left out before.
UPDATE "Worker" w
SET "isExemptFromLimits" = true
FROM v1_operator op
WHERE op.id = w."operatorId" AND op.kind = 'DAG';

-- +goose Down

ALTER TABLE "Worker"
    DROP COLUMN IF EXISTS "isExemptFromLimits",
    DROP COLUMN IF EXISTS "operatorActionCount";

DROP INDEX IF EXISTS v1_operator_tenant_name_kind_key;

ALTER TABLE v1_operator DROP COLUMN leasing_manager;

DROP TYPE IF EXISTS v1_operator_leasing_manager;

-- Postgres cannot drop a value from an enum type, so 'GRPC' stays on v1_operator_kind.
