-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_TOPIC';
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_MESSAGE';

-- metadata for streams
CREATE TABLE v1_stream_topic (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT v1_stream_topic_tenant_ns_topic_key UNIQUE (tenant_id, namespace, topic)
);

-- xact_id stores the transaction id that this row was inserted with
-- when querying, we filter based on the minimum in-flight transaction number,
-- thus all transactions started after the oldest in flight transaction are ignored
-- strictly ordering messages by transaction *start* rather than commit.
-- producer_id/producer_seq have no default: every publish is required to
-- supply them (see api-contracts/v1/streams.proto).
CREATE TABLE v1_stream_message (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    payload BYTEA NOT NULL,
    xact_id XID8 NOT NULL DEFAULT pg_current_xact_id(),
    producer_id TEXT NOT NULL,
    producer_seq BIGINT NOT NULL,

    CONSTRAINT v1_stream_message_pkey PRIMARY KEY (tenant_id, namespace, topic, inserted_at, id)
) PARTITION BY RANGE(inserted_at);

SELECT create_v1_range_partition('v1_stream_message', NOW()::DATE);
SELECT create_v1_range_partition('v1_stream_message', (NOW() + INTERVAL '1 day')::DATE);

-- v1_stream_producer_cursor tracks, per (tenant, namespace, topic,
-- producer_id), the last producer_seq durably applied to v1_stream_message.
CREATE TABLE v1_stream_producer_cursor (
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    producer_id TEXT NOT NULL,
    last_seq BIGINT NOT NULL,

    CONSTRAINT v1_stream_producer_cursor_pkey PRIMARY KEY (tenant_id, namespace, topic, producer_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_stream_producer_cursor;
DROP TABLE v1_stream_message;
DROP TABLE v1_stream_topic;
-- +goose StatementEnd
