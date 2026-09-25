-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS v1_lookup_table_olap_partitioned (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (external_id, inserted_at)
) PARTITION BY RANGE (inserted_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS v1_lookup_table_olap_external_id_inserted_at_idx ON v1_lookup_table_olap (external_id, inserted_at);
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'v1_lookup_table_olap_external_id_inserted_at_uq') THEN
        ALTER TABLE v1_lookup_table_olap ADD CONSTRAINT v1_lookup_table_olap_external_id_inserted_at_uq UNIQUE USING INDEX v1_lookup_table_olap_external_id_inserted_at_idx;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    next_month_start TIMESTAMPTZ := date_trunc('month', NOW()) + INTERVAL '1 month';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'v1_lookup_table_olap_attach_bound') THEN
        EXECUTE format('ALTER TABLE v1_lookup_table_olap ADD CONSTRAINT v1_lookup_table_olap_attach_bound CHECK (inserted_at < %L) NOT VALID', next_month_start);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_lookup_table_olap VALIDATE CONSTRAINT v1_lookup_table_olap_attach_bound;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS v1_dag_to_task_olap_partitioned (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
) PARTITION BY RANGE (dag_inserted_at);
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    tomorrow_start TIMESTAMPTZ := date_trunc('day', NOW() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' + INTERVAL '1 day';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'v1_dag_to_task_olap_attach_bound') THEN
        EXECUTE format('ALTER TABLE v1_dag_to_task_olap ADD CONSTRAINT v1_dag_to_task_olap_attach_bound CHECK (dag_inserted_at < %L) NOT VALID', tomorrow_start);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_dag_to_task_olap VALIDATE CONSTRAINT v1_dag_to_task_olap_attach_bound;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS v1_statuses_olap_partitioned (
    external_id UUID NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    kind v1_run_kind NOT NULL,
    readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    PRIMARY KEY (external_id, inserted_at)
) PARTITION BY RANGE (inserted_at);
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    next_month_start TIMESTAMPTZ := date_trunc('month', NOW()) + INTERVAL '1 month';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'v1_statuses_olap_attach_bound') THEN
        EXECUTE format('ALTER TABLE v1_statuses_olap ADD CONSTRAINT v1_statuses_olap_attach_bound CHECK (inserted_at < %L) NOT VALID', next_month_start);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_statuses_olap VALIDATE CONSTRAINT v1_statuses_olap_attach_bound;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS v1_task_events_olap_partitioned (
    tenant_id UUID NOT NULL,
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    event_type v1_event_type_olap NOT NULL,
    workflow_id UUID NOT NULL,
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    readable_status v1_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    output JSONB,
    worker_id UUID,
    additional__event_data TEXT,
    additional__event_message TEXT,
    durable_invocation_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (task_id, task_inserted_at, id)
) PARTITION BY RANGE (task_inserted_at);

CREATE INDEX IF NOT EXISTS v1_task_events_olap_partitioned_task_id_idx ON v1_task_events_olap_partitioned (task_id);
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    tomorrow_start TIMESTAMPTZ := date_trunc('day', NOW() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' + INTERVAL '1 day';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'v1_task_events_olap_attach_bound') THEN
        EXECUTE format('ALTER TABLE v1_task_events_olap ADD CONSTRAINT v1_task_events_olap_attach_bound CHECK (task_inserted_at < %L) NOT VALID', tomorrow_start);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task_events_olap VALIDATE CONSTRAINT v1_task_events_olap_attach_bound;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_events_olap DROP CONSTRAINT IF EXISTS v1_task_events_olap_attach_bound;
DROP TABLE IF EXISTS v1_task_events_olap_partitioned;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_statuses_olap DROP CONSTRAINT IF EXISTS v1_statuses_olap_attach_bound;
DROP TABLE IF EXISTS v1_statuses_olap_partitioned;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_dag_to_task_olap DROP CONSTRAINT IF EXISTS v1_dag_to_task_olap_attach_bound;
DROP TABLE IF EXISTS v1_dag_to_task_olap_partitioned;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_lookup_table_olap DROP CONSTRAINT IF EXISTS v1_lookup_table_olap_attach_bound;
ALTER TABLE v1_lookup_table_olap DROP CONSTRAINT IF EXISTS v1_lookup_table_olap_external_id_inserted_at_uq;
DROP INDEX IF EXISTS v1_lookup_table_olap_external_id_inserted_at_idx;
DROP TABLE IF EXISTS v1_lookup_table_olap_partitioned;
-- +goose StatementEnd
