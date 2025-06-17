-- +goose Up
-- +goose StatementBegin
CREATE INDEX v1_events_olap_scope_idx ON v1_events_olap (tenant_id, scope) WHERE scope IS NOT NULL;
ALTER TABLE v1_event_to_run_olap ADD COLUMN filter_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_event_to_run_olap DROP COLUMN filter_id;
DROP INDEX v1_events_olap_scope_idx;
-- +goose StatementEnd
