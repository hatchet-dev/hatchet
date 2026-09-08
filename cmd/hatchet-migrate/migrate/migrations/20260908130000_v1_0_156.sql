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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS v1_serverless_lease_unowned_idx ON v1_serverless_lease (tenant_id, shard) WHERE process_id IS NULL;

DROP INDEX IF EXISTS v1_serverless_lease_claimable_idx;
-- +goose StatementEnd
