-- +goose Up
-- +goose StatementBegin
CREATE INDEX v1_events_olap_scope_idx ON v1_events_olap (tenant_id, scope, seen_at DESC) WHERE scope IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX v1_events_olap_scope_idx;
-- +goose StatementEnd
