-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task_event ALTER COLUMN external_id SET DEFAULT gen_random_uuid();
ALTER TABLE v1_task_event
  ADD CONSTRAINT v1_task_event_external_id_not_null CHECK (external_id IS NOT NULL) NOT VALID;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
  batch_size INT := 1000;
  last_task_id BIGINT := 0;
  last_inserted_at TIMESTAMPTZ := '1970-01-01 00:00:00+00';
BEGIN
  LOOP
    WITH task_batch AS (
        SELECT id, inserted_at
        FROM v1_task
        WHERE (id, inserted_at) > (last_task_id, last_inserted_at)
        ORDER BY id, inserted_at
        LIMIT batch_size
    ), updates AS (
        UPDATE v1_task_event
        SET external_id = gen_random_uuid()
        WHERE (task_id, inserted_at) IN (SELECT id, inserted_at FROM task_batch)
    )

    SELECT MAX(id), MAX(inserted_at) INTO last_task_id, last_inserted_at
    FROM task_batch
    ;

    EXIT WHEN last_task_id IS NULL;
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_event VALIDATE CONSTRAINT v1_task_event_external_id_not_null;
ALTER TABLE v1_task_event ALTER COLUMN external_id SET NOT NULL;
ALTER TABLE v1_task_event DROP CONSTRAINT v1_task_event_external_id_not_null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_event ALTER COLUMN external_id DROP NOT NULL;
ALTER TABLE v1_task_event ALTER COLUMN external_id DROP DEFAULT;
-- +goose StatementEnd
