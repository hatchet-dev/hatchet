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

SELECT create_v1_range_partition('v1_payloads_olap'::TEXT, NOW()::DATE);
SELECT create_v1_range_partition('v1_payloads_olap'::TEXT, (NOW() + INTERVAL '1 day')::DATE);
SELECT create_v1_range_partition('v1_payloads_olap'::TEXT, (NOW() + INTERVAL '2 day')::DATE);

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
