-- +goose Up
-- +goose StatementBegin
-- Only units with endpoints are claimable: an empty unit (shard growth, every endpoint
-- deleted) has nothing to poll, and counting it made a window of empty rows hide the populated
-- units behind it. The claimable index covers exactly the units the claim walks.
DROP INDEX IF EXISTS v1_serverless_lease_claimable_idx;
CREATE INDEX v1_serverless_lease_claimable_idx ON v1_serverless_lease (tenant_id, shard) INCLUDE (endpoint_count) WHERE process_id IS NULL AND endpoint_count > 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS v1_serverless_lease_claimable_idx;
CREATE INDEX v1_serverless_lease_claimable_idx ON v1_serverless_lease (tenant_id, shard) INCLUDE (endpoint_count) WHERE process_id IS NULL;
-- +goose StatementEnd
