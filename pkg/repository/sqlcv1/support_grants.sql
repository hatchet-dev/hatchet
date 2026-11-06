-- name: RevokeSupportGrant :exec
INSERT INTO v1_support_grant_revocation (
    grant_id,
    tenant_id,
    expires_at
) VALUES (
    @grantId::uuid,
    @tenantId::uuid,
    @expiresAt::timestamptz
)
ON CONFLICT (grant_id) DO NOTHING;

-- name: IsSupportGrantRevoked :one
SELECT EXISTS (
    SELECT 1
    FROM v1_support_grant_revocation
    WHERE grant_id = @grantId::uuid
) AS revoked;

-- name: DeleteExpiredSupportGrantRevocations :exec
DELETE FROM v1_support_grant_revocation
WHERE expires_at < CURRENT_TIMESTAMP;
