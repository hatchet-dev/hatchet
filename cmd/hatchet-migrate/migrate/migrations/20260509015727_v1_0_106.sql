-- +goose Up
-- +goose StatementBegin
LOCK TABLE v1_queue_item IN ACCESS EXCLUSIVE MODE;

WITH duplicate_tasks AS (
    SELECT
        task_id,
        task_inserted_at,
        retry_count,
        COUNT(*) AS count
    FROM v1_queue_item
    GROUP BY task_id, task_inserted_at, retry_count
    HAVING COUNT(*) > 1
), with_row_numbers AS (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY task_id, task_inserted_at, retry_count ORDER BY id) AS rn
    FROM v1_queue_item
    WHERE (task_id, task_inserted_at, retry_count) IN (
        SELECT task_id, task_inserted_at, retry_count
        FROM duplicate_tasks
    )
)

DELETE FROM v1_queue_item
WHERE id IN (
    SELECT id
    FROM with_row_numbers
    WHERE rn > 1
);

DROP INDEX v1_queue_item_task_idx;

CREATE UNIQUE INDEX v1_queue_item_task_idx ON v1_queue_item (
    task_id ASC,
    task_inserted_at ASC,
    retry_count ASC
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX v1_queue_item_task_idx;

CREATE INDEX v1_queue_item_task_idx ON v1_queue_item (
    task_id ASC,
    task_inserted_at ASC,
    retry_count ASC
);
-- +goose StatementEnd
