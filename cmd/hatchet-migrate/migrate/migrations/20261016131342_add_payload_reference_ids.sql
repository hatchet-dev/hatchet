-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_dag ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_event ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_event ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_tasks_olap ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_dags_olap ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_events_olap ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_events_olap ADD COLUMN IF NOT EXISTS payload_reference_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION extract_uuid7_timestamp(
    id UUID
) RETURNS TIMESTAMPTZ
    LANGUAGE sql
    IMMUTABLE
    PARALLEL SAFE
AS $$
    SELECT 'epoch'::TIMESTAMPTZ + ('x' || REPLACE(LEFT(id::TEXT, 13), '-', ''))::BIT(48)::BIGINT * INTERVAL '1 millisecond';
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS extract_uuid7_timestamp(UUID);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_dag DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_event DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_event DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_tasks_olap DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_dags_olap DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_events_olap DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_events_olap DROP COLUMN IF EXISTS payload_reference_id;
-- +goose StatementEnd
