package middleware

const (
	// IsExchangeTokenContextKey is the context key used to indicate that the request
	// is authenticated with an exchange token.
	IsExchangeTokenContextKey = "is_exchange_token"

	// IsSupportTokenContextKey is only ever set alongside IsExchangeTokenContextKey
	IsSupportTokenContextKey = "is_support_token"

	SupportClaimsContextKey = "support-claims"

	// StatusClientClosedRequest is the non-standard status code (nginx convention) for a
	// client that disconnected before the server could respond. Go's net/http has no
	// constant for it.
	StatusClientClosedRequest = 499
)
