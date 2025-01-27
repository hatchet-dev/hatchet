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
        schedule_timeout TEXT NOT NULL DEFAULT '5m',
        step_timeout TEXT NOT NULL DEFAULT '1m',
        priority TINYINT UNSIGNED NOT NULL DEFAULT 1,
        sticky Enum(
            'NONE' = 1,
            'HARD' = 2,
            'SOFT' = 3,
        ) NOT NULL DEFAULT 'NONE',
        desired_worker_id UUID NULL DEFAULT NULL,
        display_name TEXT NOT NULL,
        input TEXT NOT NULL DEFAULT '{}',
        additional_metadata TEXT NOT NULL DEFAULT '{}',
        created_at DateTime('UTC') NOT NULL DEFAULT NOW(),

        PRIMARY KEY (tenant_id, id)
    )
    ENGINE = MergeTree()

    -- https://stackoverflow.com/a/75439879 for more on partitioning
    -- partition by week so we can easily drop old data
    PARTITION BY (toMonday(created_at))
    ORDER BY (tenant_id, id);

    CREATE TABLE task_events (
        id UUID NOT NULL DEFAULT generateUUIDv4(),
        task_id UUID NOT NULL,
        tenant_id UUID NOT NULL,
        event_type Enum(
            'RETRYING' = 1,
            'REASSIGNED' = 2,
            'RETRIED_BY_USER' = 3,
            'CREATED' = 4,
            'QUEUED' = 5,
            'REQUEUED_NO_WORKER' = 6,
            'REQUEUED_RATE_LIMIT' = 7,
            'ASSIGNED' = 8,
            'ACKNOWLEDGED' = 9,
            'SENT_TO_WORKER' = 10,
            'SLOT_RELEASED' = 11,
            'STARTED' = 12,
            'TIMEOUT_REFRESHED' = 13,
            'SCHEDULING_TIMED_OUT' = 14,
            'FINISHED' = 15,
            'FAILED' = 16,
            'CANCELLED' = 17,
            'TIMED_OUT' = 18,
            'RATE_LIMIT_ERROR' = 19
        ) NOT NULL,
        readable_status Enum(
            'QUEUED' = 1,
            'RUNNING' = 2,
            'COMPLETED' = 3,
            'CANCELLED' = 4,
            'FAILED' = 5
        ),
        timestamp DateTime('UTC') NOT NULL,
        retry_count SMALLINT UNSIGNED NOT NULL DEFAULT 0,
        error_message TEXT NOT NULL DEFAULT '',
        output TEXT NOT NULL DEFAULT '{}',
        worker_id UUID NULL DEFAULT NULL,
        additional__event_data TEXT NOT NULL DEFAULT '{}',
        additional__event_message TEXT NOT NULL DEFAULT '',
        created_at DateTime('UTC') NOT NULL DEFAULT NOW(),

        PRIMARY KEY (tenant_id, task_id, timestamp, event_type, retry_count)
    )
    ENGINE = MergeTree()

    -- https://stackoverflow.com/a/75439879 for more on partitioning
    -- partition by week so we can easily drop old data
    PARTITION BY (toMonday(timestamp))
    ORDER BY (tenant_id, task_id, timestamp, event_type, retry_count);
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
