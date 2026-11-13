-- +goose Up
-- +goose StatementBegin
DROP INDEX v1_event_key_idx;
CREATE INDEX v1_event_key_scope_idx ON v1_event (tenant_id, key, scope);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX v1_event_key_scope_idx;
CREATE INDEX v1_event_key_idx ON v1_event (tenant_id, key);
-- +goose StatementEnd
