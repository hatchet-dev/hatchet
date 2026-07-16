-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_payload_location_olap AS ENUM ('INLINE', 'EXTERNAL');

CREATE TABLE v1_payloads_olap (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,

    location v1_payload_location_olap NOT NULL,
    external_location_key TEXT,
    inline_content JSONB,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, external_id, inserted_at),

    CHECK (
        location = 'INLINE'
        OR
        (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
    )
) PARTITION BY RANGE(inserted_at);

-- Create v1_payloads_olap partitions to match existing v1_events_olap partitions.
DO $$
DECLARE
    oldest_date DATE;
    current_date_iter DATE;
    target_date DATE;
BEGIN
    -- Find the oldest v1_events_olap partition's start date from pg_catalog
    SELECT MIN(
        TO_DATE(
            regexp_replace(c.relname, '^v1_events_olap_', ''),
            'YYYYMMDD'
        )
    ) INTO oldest_date
    FROM pg_catalog.pg_class c
    JOIN pg_catalog.pg_inherits i ON c.oid = i.inhrelid
    JOIN pg_catalog.pg_class parent ON i.inhparent = parent.oid
    WHERE parent.relname = 'v1_events_olap'
      AND c.relname ~ '^v1_events_olap_[0-9]{8}$';

    -- Default to today if no v1_events_olap partitions found
    IF oldest_date IS NULL THEN
        oldest_date := NOW()::DATE;
    END IF;

    target_date := (NOW() + INTERVAL '2 day')::DATE;
    current_date_iter := oldest_date;

    WHILE current_date_iter <= target_date LOOP
        PERFORM create_v1_range_partition('v1_payloads_olap'::TEXT, current_date_iter);
        current_date_iter := current_date_iter + INTERVAL '1 day';
    END LOOP;
END;
$$;

ALTER TABLE v1_task_events_olap ADD COLUMN external_id UUID;
ALTER TABLE v1_payload ADD COLUMN external_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_events_olap DROP COLUMN external_id;
ALTER TABLE v1_payload DROP COLUMN external_id;

DROP TABLE v1_payloads_olap;

DROP TYPE v1_payload_location_olap;
-- +goose StatementEnd
