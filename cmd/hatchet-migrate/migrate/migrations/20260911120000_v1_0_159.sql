-- +goose Up
-- +goose StatementBegin

-- Runs inside goose's transaction. The enum type is created here rather than extended, and
-- Postgres only refuses to use a value that ALTER TYPE ... ADD VALUE added in the same
-- transaction; a value of a type created in the transaction is usable at once, so the column
-- and the backfill below can follow in one transaction and no NO TRANSACTION is needed.
--
-- The version jumps from v1_0_154 to v1_0_159 because belanger/serverless-operator, which
-- stacks on this branch, owns v1_0_155 to v1_0_158.

-- What an operator is (kind) and who keeps it alive (leasing) are separate axes. MANAGED rows
-- are assigned to a dispatcher by ClaimOperators and built from a factory inside the engine;
-- SELF rows keep themselves alive, through a Listen stream out of process or their own leaser in
-- process, and are never claimed.
CREATE TYPE v1_operator_leasing AS ENUM ('MANAGED', 'SELF');

ALTER TABLE v1_operator ADD COLUMN leasing v1_operator_leasing NOT NULL DEFAULT 'SELF';

-- Every DAG operator row was claimed by the engine before the column existed.
UPDATE v1_operator SET leasing = 'MANAGED' WHERE kind = 'DAG';

-- SERVERLESS is retired: the serverless operator is a GRPC contract operator that leases itself
-- in both modes. The enum value stays (Postgres cannot drop one, and databases migrated by the
-- serverless branch carry it); comparing on the text form works whether or not this database
-- ever added it.
UPDATE v1_operator SET kind = 'GRPC' WHERE kind::text = 'SERVERLESS';

-- Rows are unique per (tenant, name, kind) from here on, for every kind. The two partial
-- indexes only covered the kinds that were upserted by name. A duplicate is renamed rather than
-- deleted: the oldest row keeps its name, a later one gets its id as a suffix. (A duplicate can
-- exist from a race creating a tenant's DAG operator, or from a SERVERLESS row and a GRPC row
-- of the same name, which the update above just folded into one kind.)
UPDATE v1_operator dup
SET name = dup.name || '-' || dup.id::text
FROM v1_operator first
WHERE
    first.tenant_id = dup.tenant_id
    AND first.name = dup.name
    AND first.kind = dup.kind
    AND (first.created_at, first.id) < (dup.created_at, dup.id);

DROP INDEX IF EXISTS v1_operator_grpc_tenant_name_key;
DROP INDEX IF EXISTS v1_operator_serverless_tenant_name_key;

CREATE UNIQUE INDEX v1_operator_tenant_name_kind_key ON v1_operator (tenant_id, name, kind);

-- Whether a worker counts against the tenant's WORKER and WORKER_SLOT limits is decided when
-- the worker is created, by whoever hosts it: the in-process host exempts every worker it
-- creates, the wire meters every worker it registers. It is a fact about the worker, not
-- about the operator row, so the limit queries read it here rather than through a join.
ALTER TABLE "Worker" ADD COLUMN "exemptFromLimits" BOOLEAN NOT NULL DEFAULT false;

-- The DAG operator's workers were the only ones the limit queries left out before the column
-- existed.
UPDATE "Worker" w
SET "exemptFromLimits" = true
FROM v1_operator op
WHERE op.id = w."operatorId" AND op.kind = 'DAG';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE "Worker" DROP COLUMN IF EXISTS "exemptFromLimits";

DROP INDEX IF EXISTS v1_operator_tenant_name_kind_key;

-- The GRPC partial index comes back; the SERVERLESS one belongs to the serverless branch's own
-- migrations. Rows folded from SERVERLESS into GRPC, and renamed duplicates, stay as they are.
CREATE UNIQUE INDEX IF NOT EXISTS v1_operator_grpc_tenant_name_key ON v1_operator (tenant_id, name) WHERE kind = 'GRPC';

ALTER TABLE v1_operator DROP COLUMN IF EXISTS leasing;

DROP TYPE IF EXISTS v1_operator_leasing;

-- +goose StatementEnd
