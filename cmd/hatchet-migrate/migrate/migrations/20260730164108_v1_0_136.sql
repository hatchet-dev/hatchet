-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_cagg_task_statuses_minute_do_not_use_directly (
    time_bucket TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count BIGINT NOT NULL DEFAULT 0,
    running_count BIGINT NOT NULL DEFAULT 0,
    completed_count BIGINT NOT NULL DEFAULT 0,
    cancelled_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    evicted_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (time_bucket, tenant_id, workflow_id)
);

CREATE VIEW v1_cagg_task_statuses_minute AS
WITH max_time AS (
    SELECT MAX(time_bucket) AS max_time_bucket
    FROM v1_cagg_task_statuses_minute_do_not_use_directly
), unioned AS (
    SELECT *
    FROM v1_cagg_task_statuses_minute_do_not_use_directly

    UNION

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
    WHERE inserted_at > (SELECT max_time_bucket FROM max_time)
    GROUP BY time_bucket, tenant_id, workflow_id
)

SELECT *
FROM unioned
ORDER BY time_bucket, tenant_id, workflow_id
;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW v1_cagg_task_statuses_minute;
DROP TABLE v1_cagg_task_statuses_minute_do_not_use_directly;
-- +goose StatementEnd
