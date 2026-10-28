-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_agg_task_statuses_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count BIGINT NOT NULL DEFAULT 0,
    running_count BIGINT NOT NULL DEFAULT 0,
    completed_count BIGINT NOT NULL DEFAULT 0,
    cancelled_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    evicted_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket, workflow_id)
) PARTITION BY RANGE (bucket);

CREATE TABLE v1_agg_task_events_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count BIGINT NOT NULL DEFAULT 0,
    running_count BIGINT NOT NULL DEFAULT 0,
    completed_count BIGINT NOT NULL DEFAULT 0,
    cancelled_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket, workflow_id)
) PARTITION BY RANGE (bucket);

CREATE TABLE v1_agg_events_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    event_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket)
) PARTITION BY RANGE (bucket);

CREATE TABLE v1_agg_task_statuses_minute_delta_olap (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count INTEGER NOT NULL,
    running_count INTEGER NOT NULL,
    completed_count INTEGER NOT NULL,
    cancelled_count INTEGER NOT NULL,
    failed_count INTEGER NOT NULL,
    evicted_count INTEGER NOT NULL,
    PRIMARY KEY (id, tenant_id)
) PARTITION BY HASH (tenant_id);

CREATE TABLE v1_agg_task_events_minute_delta_olap (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count INTEGER NOT NULL,
    running_count INTEGER NOT NULL,
    completed_count INTEGER NOT NULL,
    cancelled_count INTEGER NOT NULL,
    failed_count INTEGER NOT NULL,
    PRIMARY KEY (id, tenant_id)
) PARTITION BY HASH (tenant_id);

CREATE TABLE v1_agg_events_minute_delta_olap (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    event_count INTEGER NOT NULL,
    PRIMARY KEY (id, tenant_id)
) PARTITION BY HASH (tenant_id);

CREATE INDEX v1_agg_task_statuses_minute_delta_olap_idx ON v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket);
CREATE INDEX v1_agg_task_events_minute_delta_olap_idx ON v1_agg_task_events_minute_delta_olap (tenant_id, bucket);
CREATE INDEX v1_agg_events_minute_delta_olap_idx ON v1_agg_events_minute_delta_olap (tenant_id, bucket);

CREATE OR REPLACE FUNCTION claim_agg_deltas(
    delta_table_row anyelement,
    partition_number INT,
    batch_size INT
) RETURNS SETOF anyelement
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
BEGIN
    partition_table := pg_typeof(delta_table_row)::text || '_' || partition_number::text;
    RETURN QUERY EXECUTE format(
        'DELETE FROM %I
         WHERE ctid IN (
             SELECT ctid
             FROM %I
             ORDER BY id
             LIMIT $1
             FOR UPDATE SKIP LOCKED
         )
         RETURNING *',
         partition_table, partition_table)
    USING batch_size;
END;
$$;

SELECT create_v1_hash_partitions('v1_agg_task_statuses_minute_delta_olap'::TEXT, 4);
SELECT create_v1_hash_partitions('v1_agg_task_events_minute_delta_olap'::TEXT, 4);
SELECT create_v1_hash_partitions('v1_agg_events_minute_delta_olap'::TEXT, 4);

CREATE OR REPLACE FUNCTION v1_agg_task_statuses_insert_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count, evicted_count)
    SELECT
        tenant_id,
        date_bin('1 minute', inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        workflow_id,
        COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
        COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
        COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
        COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
        COUNT(*) FILTER (WHERE readable_status = 'FAILED'),
        COUNT(*) FILTER (WHERE readable_status = 'EVICTED')
    FROM new_rows
    GROUP BY 1, 2, 3;

    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION v1_agg_task_statuses_update_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count, evicted_count)
    SELECT *
    FROM (
        SELECT
            tenant_id,
            date_bin('1 minute', inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00') AS bucket,
            workflow_id,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'QUEUED'), 0) AS queued,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'RUNNING'), 0) AS running,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'COMPLETED'), 0) AS completed,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'CANCELLED'), 0) AS cancelled,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'FAILED'), 0) AS failed,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'EVICTED'), 0) AS evicted
        FROM (
            SELECT tenant_id, workflow_id, inserted_at, readable_status, 1 AS sign FROM new_rows
            UNION ALL
            SELECT tenant_id, workflow_id, inserted_at, readable_status, -1 AS sign FROM old_rows
        ) c
        GROUP BY 1, 2, 3
    ) g
    WHERE queued <> 0 OR running <> 0 OR completed <> 0 OR cancelled <> 0 OR failed <> 0 OR evicted <> 0;

    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION v1_agg_task_statuses_delete_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM old_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count, evicted_count)
    SELECT
        tenant_id,
        date_bin('1 minute', inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        workflow_id,
        -COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
        -COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
        -COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
        -COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
        -COUNT(*) FILTER (WHERE readable_status = 'FAILED'),
        -COUNT(*) FILTER (WHERE readable_status = 'EVICTED')
    FROM old_rows
    GROUP BY 1, 2, 3;

    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION v1_agg_task_events_insert_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_events_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count)
    SELECT
        tenant_id,
        date_bin('1 minute', task_inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        workflow_id,
        COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
        COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
        COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
        COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
        COUNT(*) FILTER (WHERE readable_status = 'FAILED')
    FROM new_rows
    WHERE NOT is_dag_orchestrator
    GROUP BY 1, 2, 3;

    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION v1_agg_events_insert_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_events_minute_delta_olap (tenant_id, bucket, event_count)
    SELECT
        tenant_id,
        date_bin('1 minute', seen_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        COUNT(*)
    FROM new_rows
    GROUP BY 1, 2;

    RETURN NULL;
END;
$$;

CREATE TRIGGER v1_agg_task_statuses_insert_trigger
AFTER INSERT ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_statuses_insert_fn();

CREATE TRIGGER v1_agg_task_statuses_update_trigger
AFTER UPDATE ON v1_runs_olap
REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_statuses_update_fn();

CREATE TRIGGER v1_agg_task_statuses_delete_trigger
AFTER DELETE ON v1_runs_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_statuses_delete_fn();

CREATE TRIGGER v1_agg_task_events_insert_trigger
AFTER INSERT ON v1_task_events_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_events_insert_fn();

CREATE TRIGGER v1_agg_events_insert_trigger
AFTER INSERT ON v1_events_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_events_insert_fn();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER v1_agg_events_insert_trigger ON v1_events_olap;
DROP TRIGGER v1_agg_task_events_insert_trigger ON v1_task_events_olap;
DROP TRIGGER v1_agg_task_statuses_delete_trigger ON v1_runs_olap;
DROP TRIGGER v1_agg_task_statuses_update_trigger ON v1_runs_olap;
DROP TRIGGER v1_agg_task_statuses_insert_trigger ON v1_runs_olap;
DROP FUNCTION v1_agg_events_insert_fn();
DROP FUNCTION v1_agg_task_events_insert_fn();
DROP FUNCTION v1_agg_task_statuses_delete_fn();
DROP FUNCTION v1_agg_task_statuses_update_fn();
DROP FUNCTION v1_agg_task_statuses_insert_fn();
DROP FUNCTION claim_agg_deltas(anyelement, INT, INT);
DROP TABLE v1_agg_events_minute_delta_olap;
DROP TABLE v1_agg_task_events_minute_delta_olap;
DROP TABLE v1_agg_task_statuses_minute_delta_olap;
DROP TABLE v1_agg_events_minute_olap;
DROP TABLE v1_agg_task_events_minute_olap;
DROP TABLE v1_agg_task_statuses_minute_olap;
-- +goose StatementEnd
