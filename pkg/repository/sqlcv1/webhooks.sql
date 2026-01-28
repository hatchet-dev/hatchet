-- name: CreateWebhook :one
INSERT INTO v1_incoming_webhook (
    tenant_id,
    name,
    source_name,
    event_key_expression,
    scope_expression,
    static_payload,
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
    @tenantId::UUID,
    @name::TEXT,
    @sourceName::v1_incoming_webhook_source_name,
    @eventKeyExpression::TEXT,
    sqlc.narg('scopeExpression')::TEXT,
    sqlc.narg('staticPayload')::JSONB,
    @authMethod::v1_incoming_webhook_auth_type,
    sqlc.narg('authBasicUsername')::TEXT,
    @authBasicPassword::BYTEA,
    sqlc.narg('authApiKeyHeaderName')::TEXT,
    @authApiKeyKey::BYTEA,
    sqlc.narg('authHmacAlgorithm')::v1_incoming_webhook_hmac_algorithm,
    sqlc.narg('authHmacEncoding')::v1_incoming_webhook_hmac_encoding,
    sqlc.narg('authHmacSignatureHeaderName')::TEXT,
    @authHmacWebhookSigningSecret::BYTEA
)
RETURNING *;

-- name: GetWebhook :one
SELECT *
FROM v1_incoming_webhook
WHERE
    name = @name::TEXT
    AND tenant_id = @tenantId::UUID;

-- name: DeleteWebhook :one
DELETE FROM v1_incoming_webhook
WHERE
    tenant_id = @tenantId::UUID
    AND name = @name::TEXT
RETURNING *;

-- name: ListWebhooks :many
SELECT *
FROM v1_incoming_webhook
WHERE
    tenant_id = @tenantId::UUID
    AND (
        @sourceNames::v1_incoming_webhook_source_name[] IS NULL
        OR source_name = ANY(@sourceNames::v1_incoming_webhook_source_name[])
    )
    AND (
        @webhookNames::TEXT[] IS NULL
        OR name = ANY(@webhookNames::TEXT[])
    )
ORDER BY tenant_id, inserted_at DESC
LIMIT COALESCE(sqlc.narg('webhookLimit')::BIGINT, 20000)
OFFSET COALESCE(sqlc.narg('webhookOffset')::BIGINT, 0)
;

-- name: CanCreateWebhook :one
SELECT COUNT(*) < @webhookLimit::INT AS can_create_webhook
FROM v1_incoming_webhook
WHERE
    tenant_id = @tenantId::UUID
;

-- name: UpdateWebhook :one
UPDATE v1_incoming_webhook
SET
    event_key_expression = COALESCE(sqlc.narg('eventKeyExpression')::TEXT, event_key_expression),
    scope_expression = COALESCE(sqlc.narg('scopeExpression')::TEXT, scope_expression),
    static_payload = COALESCE(sqlc.narg('staticPayload')::JSONB, static_payload),
    updated_at = CURRENT_TIMESTAMP
WHERE
    tenant_id = @tenantId::UUID
    AND name = @webhookName::TEXT
RETURNING *;
