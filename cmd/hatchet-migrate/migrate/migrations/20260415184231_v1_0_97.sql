-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_incoming_webhook ADD COLUMN return_event_as_response_payload BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE v1_task
    ADD COLUMN triggering_event_external_id UUID,
    ADD COLUMN triggering_event_key TEXT
;

ALTER TABLE v1_tasks_olap
    ADD COLUMN triggering_event_external_id UUID,
    ADD COLUMN triggering_event_key TEXT
;

ALTER TABLE v1_match
    ADD COLUMN trigger_event_external_id UUID,
    ADD COLUMN trigger_event_key TEXT
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_incoming_webhook DROP COLUMN return_event_as_response_payload;
ALTER TABLE v1_task
    DROP COLUMN triggering_event_external_id,
    DROP COLUMN triggering_event_key
;
ALTER TABLE v1_tasks_olap
    DROP COLUMN triggering_event_external_id,
    DROP COLUMN triggering_event_key
;

ALTER TABLE v1_match
    DROP COLUMN trigger_event_external_id,
    DROP COLUMN trigger_event_key
;
-- +goose StatementEnd
