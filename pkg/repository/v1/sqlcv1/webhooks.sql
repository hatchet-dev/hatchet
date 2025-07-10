-- name: CreateWebhook :one
INSERT INTO v1_incoming_webhook (
    id,
    tenant_id,
    source_name,
    name,
    event_key_expression,
    auth_method,
    auth__basic__username,
    auth__basic__password,
    auth__api_key__header_name,
    auth__api_key__key,
    auth__hmac__algorithm,
    auth__hmac__encoding,
    auth__hmac__signature_header_name,
    auth__hmac__webhook_signing_secret

) VALUES (
    @id::UUID,
    @tenantId::UUID,
    @sourceName::v1_incoming_webhook_source_name,
    @name::TEXT,
    @eventKeyExpression::TEXT,
    @authMethod::v1_incoming_webhook_auth_type,
    @authBasicUsername::TEXT,
    @authBasicPassword::BYTEA,
    @authApiKeyHeaderName::TEXT,
    @authApiKeyKey::BYTEA,
    sqlc.narg('authHmacAlgorithm')::v1_incoming_webhook_hmac_algorithm,
    sqlc.narg('authHmacEncoding')::v1_incoming_webhook_hmac_encoding,
    @authHmacSignatureHeaderName::TEXT,
    @authHmacWebhookSigningSecret::BYTEA
)
RETURNING *;

-- name: GetWebhook :one
SELECT *
FROM v1_incoming_webhook
WHERE
    id = @id::UUID
    AND tenant_id = @tenantId::UUID;

-- name: DeleteWebhook :one
DELETE FROM v1_incoming_webhook
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
RETURNING *;

-- name: ListWebhooks :many
SELECT *
FROM v1_incoming_webhook
WHERE
    tenant_id = @tenantId::UUID
    AND (
        @sourceNames::TEXT[] IS NULL
        OR source_name = ANY(@sourceNames::TEXT[])
    )
    AND (
        @webhookNames::TEXT[] IS NULL
        OR name = ANY(@webhookNames::TEXT[])
    )
ORDER BY inserted_at DESC
LIMIT COALESCE(sqlc.narg('webhookLimit')::BIGINT, 20000)
OFFSET COALESCE(sqlc.narg('webhookOffset')::BIGINT, 0)
;