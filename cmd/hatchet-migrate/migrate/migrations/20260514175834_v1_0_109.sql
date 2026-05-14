-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task_event ALTER COLUMN external_id SET DEFAULT gen_random_uuid();
ALTER TABLE v1_task_event
  ADD CONSTRAINT v1_task_event_external_id_not_null CHECK (external_id IS NOT NULL) NOT VALID;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id SET DEFAULT gen_random_uuid();
ALTER TABLE v1_task_events_olap
  ADD CONSTRAINT v1_task_events_olap_external_id_not_null CHECK (external_id IS NOT NULL) NOT VALID;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
  batch_size INT := 1000;
  last_task_id BIGINT := 0;
  last_task_inserted_at TIMESTAMPTZ := '1970-01-01 00:00:00+00';
BEGIN
  LOOP
    WITH task_batch AS (
        SELECT id, inserted_at
        FROM v1_task
        WHERE (id, inserted_at) > (last_task_id, last_task_inserted_at)
        ORDER BY id, inserted_at
        LIMIT batch_size
    ), updates AS (
        UPDATE v1_task_event
        SET external_id = gen_random_uuid()
        WHERE (task_id, inserted_at) IN (SELECT id, inserted_at FROM task_batch)
    )

    SELECT id, inserted_at INTO last_task_id, last_task_inserted_at
    FROM task_batch
    ORDER BY id DESC, inserted_at DESC
    LIMIT 1
    ;

    EXIT WHEN last_task_id IS NULL;
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
  batch_size INT := 1000;
  last_task_id BIGINT := 0;
  last_task_inserted_at TIMESTAMPTZ := '1970-01-01 00:00:00+00';
BEGIN
  LOOP
    WITH task_batch AS (
        SELECT inserted_at, id
        FROM v1_tasks_olap
        -- pk (ins at, id)
        WHERE (inserted_at, id) > (last_task_inserted_at, last_task_id)
        ORDER BY inserted_at, id
        LIMIT batch_size
    ), updates AS (
        UPDATE v1_task_events_olap
        SET external_id = gen_random_uuid()
        -- pk (task_id, task ins at)
        WHERE (task_id, task_inserted_at) IN (SELECT id, inserted_at FROM task_batch)
    )

    SELECT inserted_at, id INTO last_task_inserted_at, last_task_id
    FROM task_batch
    ORDER BY inserted_at DESC, id DESC
    LIMIT 1
    ;

    EXIT WHEN last_task_inserted_at IS NULL;
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_event VALIDATE CONSTRAINT v1_task_event_external_id_not_null;
ALTER TABLE v1_task_event ALTER COLUMN external_id SET NOT NULL;
ALTER TABLE v1_task_event DROP CONSTRAINT v1_task_event_external_id_not_null;
ALTER TABLE v1_task_events_olap VALIDATE CONSTRAINT v1_task_events_olap_external_id_not_null;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id SET NOT NULL;
ALTER TABLE v1_task_events_olap DROP CONSTRAINT v1_task_events_olap_external_id_not_null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_event ALTER COLUMN external_id DROP NOT NULL;
ALTER TABLE v1_task_event ALTER COLUMN external_id DROP DEFAULT;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id DROP NOT NULL;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id DROP DEFAULT;
-- +goose StatementEnd
