-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
  chunk_start TIMESTAMPTZ;
  chunk_end TIMESTAMPTZ;
  max_time TIMESTAMPTZ;
BEGIN
  SELECT MIN(inserted_at), MAX(inserted_at)
  INTO chunk_start, max_time
  FROM v1_statuses_olap;

  IF chunk_start IS NULL THEN
    RETURN;
  END IF;

  chunk_start := date_trunc('hour', chunk_start);

  WHILE chunk_start <= max_time LOOP
    chunk_end := chunk_start + INTERVAL '1 hour';

    INSERT INTO v1_cagg_task_statuses_minute_do_not_use_directly (
        time_bucket,
        tenant_id,
        workflow_id,
        queued_count,
        running_count,
        completed_count,
        cancelled_count,
        failed_count,
        evicted_count
    )
    SELECT
        date_bin('1 minute', inserted_at, TIMESTAMP '2001-01-01') AS time_bucket,
        tenant_id,
        workflow_id,
        COUNT(*) FILTER (WHERE readable_status = 'QUEUED') AS queued_count,
        COUNT(*) FILTER (WHERE readable_status = 'RUNNING') AS running_count,
        COUNT(*) FILTER (WHERE readable_status = 'COMPLETED') AS completed_count,
        COUNT(*) FILTER (WHERE readable_status = 'CANCELLED') AS cancelled_count,
        COUNT(*) FILTER (WHERE readable_status = 'FAILED') AS failed_count,
        COUNT(*) FILTER (WHERE readable_status = 'EVICTED') AS evicted_count
    FROM v1_statuses_olap
    WHERE inserted_at >= chunk_start
      AND inserted_at < chunk_end
    GROUP BY time_bucket, tenant_id, workflow_id
    ON CONFLICT (time_bucket, tenant_id, workflow_id) DO UPDATE SET
        queued_count = EXCLUDED.queued_count,
        running_count = EXCLUDED.running_count,
        completed_count = EXCLUDED.completed_count,
        cancelled_count = EXCLUDED.cancelled_count,
        failed_count = EXCLUDED.failed_count,
        evicted_count = EXCLUDED.evicted_count;

    chunk_start := chunk_end;
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
TRUNCATE TABLE v1_cagg_task_statuses_minute_do_not_use_directly;
-- +goose StatementEnd
