package authmode

// Fixed claims of the embedded auth-disabled token. The loader pins the JWT manager to these so the
// one committed token validates on every authdisabled instance.
const (
	EmbeddedTokenIssuer      = "http://localhost:8888" //nolint:gosec // URL, not a credential
	EmbeddedTokenAudience    = "http://localhost:8888" //nolint:gosec // URL, not a credential
	EmbeddedTokenServerURL   = "http://localhost:8888" //nolint:gosec // URL, not a credential
	EmbeddedTokenGRPCAddress = "localhost:7077"
	EmbeddedTokenTenantID    = "707d0855-80ab-4e1f-a156-f1c4546cbf52" //nolint:gosec // tenant UUID, not a credential
	EmbeddedTokenID          = "0a0a0a0a-0000-4000-8000-000000000001" //nolint:gosec // token UUID, not a credential
)
