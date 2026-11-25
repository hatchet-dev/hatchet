-- +goose Up
-- +goose StatementBegin
-- A task event is counted in the minute it happened. Event timestamps can come from worker
-- clocks, so they are capped at the inserting transaction's time: a delta never lands in a
-- future minute, which may not have an aggregate partition yet.
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
        date_bin('1 minute', LEAST(event_timestamp, now()), TIMESTAMPTZ '1970-01-01 00:00:00+00'),
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
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
-- +goose StatementEnd
