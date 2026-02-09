-- +goose Up
-- +goose StatementBegin

-- Add new monitoring/observability event types.
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'CANCELLING';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'CANCELLED_CONFIRMED';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'CANCELLATION_FAILED';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'DURABLE_EVICTED';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'DURABLE_RESUMING';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Recreate enum without the new values.
CREATE TYPE v1_event_type_olap_old AS ENUM (
    'RETRYING',
    'REASSIGNED',
    'RETRIED_BY_USER',
    'CREATED',
    'QUEUED',
    'REQUEUED_NO_WORKER',
    'REQUEUED_RATE_LIMIT',
    'ASSIGNED',
    'ACKNOWLEDGED',
    'SENT_TO_WORKER',
    'SLOT_RELEASED',
    'STARTED',
    'TIMEOUT_REFRESHED',
    'SCHEDULING_TIMED_OUT',
    'FINISHED',
    'FAILED',
    'CANCELLED',
    'TIMED_OUT',
    'RATE_LIMIT_ERROR',
    'SKIPPED',
    'COULD_NOT_SEND_TO_WORKER'
);

ALTER TABLE v1_task_events_olap_tmp
    ALTER COLUMN event_type TYPE v1_event_type_olap_old
    USING event_type::text::v1_event_type_olap_old;

ALTER TABLE v1_task_events_olap
    ALTER COLUMN event_type TYPE v1_event_type_olap_old
    USING event_type::text::v1_event_type_olap_old;

DROP TYPE v1_event_type_olap;
ALTER TYPE v1_event_type_olap_old RENAME TO v1_event_type_olap;

-- +goose StatementEnd
