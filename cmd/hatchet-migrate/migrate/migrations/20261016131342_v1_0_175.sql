-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_dag ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_event ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_task_event ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_tasks_olap ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_dags_olap ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_events_olap ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_task_events_olap ADD COLUMN payload_reference_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN payload_reference_id;
ALTER TABLE v1_dag DROP COLUMN payload_reference_id;
ALTER TABLE v1_event DROP COLUMN payload_reference_id;
ALTER TABLE v1_task_event DROP COLUMN payload_reference_id;
ALTER TABLE v1_tasks_olap DROP COLUMN payload_reference_id;
ALTER TABLE v1_dags_olap DROP COLUMN payload_reference_id;
ALTER TABLE v1_events_olap DROP COLUMN payload_reference_id;
ALTER TABLE v1_task_events_olap DROP COLUMN payload_reference_id;
-- +goose StatementEnd
