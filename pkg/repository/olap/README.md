# Setup

1. `curl https://clickhouse.com/ | sh`
2. `sudo ./clickhouse install`
3. `clickhouse client --host "$CLICKHOUSE_SECURE_NATIVE_HOSTNAME" --secure --password "$CLICKHOUSE_PASSWORD"`

4. ```sql
    CREATE TABLE tasks (
        id UUID NOT NULL,
        tenant_id UUID NOT NULL,
        queue TEXT NOT NULL,
        action_id TEXT NOT NULL,
        schedule_timeout TEXT NOT NULL,
        step_timeout TEXT,
        priority INTEGER NOT NULL DEFAULT 1,
        sticky Enum(
            'HARD' = 1,
            'SOFT' = 2,
        ),
        desired_worker_id UUID,
        display_name TEXT NOT NULL,
        input TEXT NOT NULL,
        worker_id UUID NOT NULL,
        created_at DateTime('UTC') NOT NULL DEFAULT NOW(),

        PRIMARY KEY (id)
    )
    ENGINE = MergeTree()

    -- https://stackoverflow.com/a/75439879 for more on partitioning
    -- partition by tenant id since we'll rarely (or never) query across tenants
    -- partition by week so we can easily drop old data
    PARTITION BY (tenant_id, toMonday(created_at))
    ORDER BY (id)

    CREATE TABLE task_events (
        id UUID NOT NULL DEFAULT generateUUIDv4(),
        task_id UUID NOT NULL,
        tenant_id UUID NOT NULL,
        status Enum(
            'REQUEUED_NO_WORKER' = 1,
            'REQUEUED_RATE_LIMIT' = 2,
            'SCHEDULING_TIMED_OUT' = 3,
            'ASSIGNED' = 4,
            'STARTED' = 5,
            'FINISHED' = 6,
            'FAILED' = 7,
            'RETRYING' = 8,
            'CANCELLED' = 9,
            'TIMED_OUT' = 10,
            'REASSIGNED' = 11,
            'SLOT_RELEASED' = 12,
            'TIMEOUT_REFRESHED' = 13,
            'RETRIED_BY_USER' = 14,
            'SENT_TO_WORKER' = 15,
            'WORKFLOW_RUN_GROUP_KEY_SUCCEEDED' = 16,
            'WORKFLOW_RUN_GROUP_KEY_FAILED' = 17,
            'RATE_LIMIT_ERROR' = 18,
            'ACKNOWLEDGED' = 19,
            'CREATED' = 20
        ) NOT NULL,
        timestamp DateTime('UTC') NOT NULL,
        retry_count INTEGER NOT NULL DEFAULT 0,
        error_message TEXT NULL DEFAULT NULL,
        additional__event_data TEXT,
        additional__event_message TEXT,
        additional__event_severity Enum(
            'INFO' = 1,
            'WARNING' = 2,
            'CRITICAL' = 3
        ),
        additional__event_reason Enum(
            'ACKNOWLEDGED' = 1,
            'ASSIGNED' = 2,
            'CANCELLED' = 3,
            'FAILED' = 4,
            'FINISHED' = 5,
            'REASSIGNED' = 6,
            'REQUEUED_NO_WORKER' = 7,
            'REQUEUED_RATE_LIMIT' = 8,
            'RETRIED_BY_USER' = 9,
            'RETRYING' = 10,
            'SCHEDULING_TIMED_OUT' = 11,
            'SLOT_RELEASED' = 12,
            'STARTED' = 13,
            'TIMED_OUT' = 14,
            'TIMEOUT_REFRESHED' = 15,
            'WORKFLOW_RUN_GROUP_KEY_FAILED' = 16,
            'WORKFLOW_RUN_GROUP_KEY_SUCCEEDED' = 17
        ),
        created_at DateTime('UTC') NOT NULL DEFAULT NOW(),

        CONSTRAINT check__failed_state_has_error CHECK CASE WHEN status = 'FAILED' THEN error_message IS NOT NULL ELSE error_message IS NULL END,
        CONSTRAINT check__input_is_valid_json CHECK isValidJSON(input),
        CONSTRAINT check__additional__event_data_is_valid_json CHECK isValidJSON(additional__event_data),

        PRIMARY KEY (task_id, timestamp, status)
    )
    ENGINE = MergeTree()

    -- https://stackoverflow.com/a/75439879 for more on partitioning
    -- partition by tenant id since we'll rarely (or never) query across tenants
    -- partition by week so we can easily drop old data
    PARTITION BY (tenant_id, toMonday(timestamp))
    ORDER BY (task_id, timestamp, status)
   ```

5. ```sql
   -- Create events
   INSERT INTO events (task_id, worker_id, tenant_id, status, timestamp, retry_count, error_message)
   VALUES
    (1, 1, '44bffbf3-5530-4378-94f3-0c85dc719159', 'CREATED', '2021-01-01 00:00:00', 0, NULL),
    (2, 1, '44bffbf3-5530-4378-94f3-0c85dc719159', 'ASSIGNED', '2024-01-01 12:34:56', 1, NULL),
    (2, 1, '44bffbf3-5530-4378-94f3-0c85dc719159', 'FAILED', '2024-01-01 12:34:58', 1, 'A foobar went wrong')
   ```

6. ```sql
   -- workflows runs view on /workflow-runs index
   WITH rows_assigned AS (
       SELECT id, ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY timestamp DESC) AS row_num
       FROM events
       -- Filtering logic here
   )

   SELECT *
   FROM events e
   JOIN rows_assigned ra ON e.id = ra.id
   WHERE ra.row_num = 1
   ```

7. ```sql
   -- Single workflow run on /workflow-runs/:id

   SELECT *
   FROM events
   WHERE task_id = 2 -- :id
   ```
