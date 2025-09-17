-- +goose Up
-- +goose StatementBegin
CREATE INDEX v1_payload_wal_payload_lookup_idx ON v1_payload_wal (payload_id, payload_inserted_at, payload_type, tenant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX v1_payload_wal_payload_lookup_idx;
-- +goose StatementEnd
