-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_incoming_webhook_auth_type AS ENUM ('BASIC', 'API_KEY', 'HMAC');
CREATE TYPE v1_incoming_webhook_hmac_algorithm AS ENUM ('SHA1', 'SHA256', 'SHA512', 'MD5');
CREATE TYPE v1_incoming_webhook_hmac_encoding AS ENUM ('HEX', 'BASE64', 'BASE64URL');

-- Can add more sources in the future
CREATE TYPE v1_incoming_webhook_source_name AS ENUM ('GENERIC', 'GITHUB', 'STRIPE');

CREATE TABLE v1_incoming_webhook (
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,

    source_name v1_incoming_webhook_source_name NOT NULL,

    -- CEL expression that creates an event key
    -- from the payload of the webhook
    event_key_expression TEXT NOT NULL,

    auth_method v1_incoming_webhook_auth_type NOT NULL,

    auth__basic__username TEXT,
    auth__basic__password BYTEA,

    auth__api_key__header_name TEXT,
    auth__api_key__key BYTEA,

    auth__hmac__algorithm v1_incoming_webhook_hmac_algorithm,
    auth__hmac__encoding v1_incoming_webhook_hmac_encoding,
    auth__hmac__signature_header_name TEXT,
    auth__hmac__webhook_signing_secret BYTEA,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, name),
    CHECK (
        (
            auth_method = 'BASIC'
            AND (
                auth__basic__username IS NOT NULL
                AND auth__basic__password IS NOT NULL
            )
        )
        OR
        (
            auth_method = 'API_KEY'
            AND (
                auth__api_key__header_name IS NOT NULL
                AND auth__api_key__key IS NOT NULL
            )
        )
        OR
        (
            auth_method = 'HMAC'
            AND (
                auth__hmac__algorithm IS NOT NULL
                AND auth__hmac__encoding IS NOT NULL
                AND auth__hmac__signature_header_name IS NOT NULL
                AND auth__hmac__webhook_signing_secret IS NOT NULL
            )
        )
    ),
    CHECK (LENGTH(event_key_expression) > 0),
    CHECK (LENGTH(name) > 0)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_incoming_webhook;
DROP TYPE v1_incoming_webhook_auth_type;
DROP TYPE v1_incoming_webhook_hmac_algorithm;
DROP TYPE v1_incoming_webhook_hmac_encoding;
DROP TYPE v1_incoming_webhook_source_name;
-- +goose StatementEnd
