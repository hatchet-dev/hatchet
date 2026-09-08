-- +goose Up
-- +goose StatementBegin
-- Lease claims walk unowned units in (tenant_id, shard) order from a random start instead of
-- sorting the whole candidate population; the covering partial index serves that walk and the
-- capped claimable count without heap reads. It replaces v1_serverless_lease_unowned_idx, whose
-- uniqueness the primary key already provides.
CREATE INDEX IF NOT EXISTS v1_serverless_lease_claimable_idx ON v1_serverless_lease (tenant_id, shard) INCLUDE (endpoint_count) WHERE process_id IS NULL;

DROP INDEX IF EXISTS v1_serverless_lease_unowned_idx;

-- Every process row deletion (sweep or graceful shutdown) now releases the units the row still
-- owns in the same statement. Units whose owner row was deleted before that rule existed are
-- released once here, since claims only consider owners with an expired row.
UPDATE v1_serverless_lease l
SET process_id = NULL, claimed_at = NULL
WHERE
    l.process_id IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM v1_serverless_process p WHERE p.process_id = l.process_id);

-- The routing cache refreshes on every change it needs to see, status transitions included:
-- rows are versioned by the later of updated_at and status_changed_at and paged by (version,
-- id). The version index replaces the updated_at index.
CREATE INDEX IF NOT EXISTS v1_serverless_endpoint_version_idx ON v1_serverless_endpoint (tenant_id, GREATEST(updated_at, COALESCE(status_changed_at, updated_at)), id);

DROP INDEX IF EXISTS v1_serverless_endpoint_updated_idx;

-- The gRPC operator service counts the action links of every worker of an operator once per
-- Listen stream (CountOperatorWorkerActions); the workers are found by (tenant, operator).
CREATE INDEX IF NOT EXISTS "Worker_tenantId_operatorId_idx" ON "Worker" ("tenantId", "operatorId");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "Worker_tenantId_operatorId_idx";

CREATE INDEX IF NOT EXISTS v1_serverless_endpoint_updated_idx ON v1_serverless_endpoint (tenant_id, updated_at);

DROP INDEX IF EXISTS v1_serverless_endpoint_version_idx;

CREATE UNIQUE INDEX IF NOT EXISTS v1_serverless_lease_unowned_idx ON v1_serverless_lease (tenant_id, shard) WHERE process_id IS NULL;

DROP INDEX IF EXISTS v1_serverless_lease_claimable_idx;
-- +goose StatementEnd
