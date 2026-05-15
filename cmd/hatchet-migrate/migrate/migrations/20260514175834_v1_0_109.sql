-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task_event
  ADD CONSTRAINT v1_task_event_external_id_not_null CHECK (external_id IS NOT NULL) NOT VALID;
ALTER TABLE v1_task_events_olap
  ADD CONSTRAINT v1_task_events_olap_external_id_not_null CHECK (external_id IS NOT NULL) NOT VALID;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
  batch_size INT := 1000;
  last_id BIGINT := 0;
BEGIN
  CREATE TEMP TABLE tmp_109_task_event_nulls AS
    SELECT id, task_id, task_inserted_at
    FROM v1_task_event
    WHERE external_id IS NULL;
  CREATE INDEX ON tmp_109_task_event_nulls (id);

  LOOP
    WITH batch AS (
        SELECT id, task_id, task_inserted_at
        FROM tmp_109_task_event_nulls
        WHERE id > last_id
        ORDER BY id
        LIMIT batch_size
    ), updates AS (
        UPDATE v1_task_event e
        SET external_id = gen_random_uuid()
        FROM batch b
        WHERE e.id = b.id
          AND e.task_id = b.task_id
          AND e.task_inserted_at = b.task_inserted_at
    )
    SELECT MAX(id) INTO last_id FROM batch;
    EXIT WHEN last_id IS NULL;
  END LOOP;

END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
  batch_size INT := 1000;
  last_id BIGINT := 0;
BEGIN
  CREATE TEMP TABLE tmp_109_task_events_olap_nulls AS
    SELECT id FROM v1_task_events_olap WHERE external_id IS NULL;
  CREATE INDEX ON tmp_109_task_events_olap_nulls (id);

  LOOP
    WITH batch AS (
        SELECT id
        FROM tmp_109_task_events_olap_nulls
        WHERE id > last_id
        ORDER BY id
        LIMIT batch_size
    ), updates AS (
        UPDATE v1_task_events_olap
        SET external_id = gen_random_uuid()
        WHERE id IN (SELECT id FROM batch)
    )
    SELECT MAX(id) INTO last_id FROM batch;
    EXIT WHEN last_id IS NULL;
  END LOOP;

END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_event VALIDATE CONSTRAINT v1_task_event_external_id_not_null;
ALTER TABLE v1_task_event ALTER COLUMN external_id SET NOT NULL;
ALTER TABLE v1_task_event ALTER COLUMN external_id SET DEFAULT gen_random_uuid();
ALTER TABLE v1_task_event DROP CONSTRAINT v1_task_event_external_id_not_null;
ALTER TABLE v1_task_events_olap VALIDATE CONSTRAINT v1_task_events_olap_external_id_not_null;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id SET NOT NULL;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id SET DEFAULT gen_random_uuid();
ALTER TABLE v1_task_events_olap DROP CONSTRAINT v1_task_events_olap_external_id_not_null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_event ALTER COLUMN external_id DROP NOT NULL;
ALTER TABLE v1_task_event ALTER COLUMN external_id DROP DEFAULT;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id DROP NOT NULL;
ALTER TABLE v1_task_events_olap ALTER COLUMN external_id DROP DEFAULT;
-- +goose StatementEnd
