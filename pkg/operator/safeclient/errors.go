package safeclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/doyensec/safeurl"
)

// Typed errors returned by the client. Callers (e.g. a retry queue or API validation
// layer) use errors.Is to distinguish policy-blocked failures — which must NOT be retried
// and should be surfaced to the user — from transient network failures, which are
// retryable. Any error returned from Deliver that does not match one of these (nor a
// context error) should be treated as a transient/retryable network failure.
var (
	// ErrBlockedDestination indicates the destination resolved to (or is) a blocked IP,
	// or otherwise violated the destination policy (credentials in URL, invalid host).
	ErrBlockedDestination = errors.New("safeclient: destination blocked by SSRF policy")

	// ErrBadScheme indicates a scheme other than https.
	ErrBadScheme = errors.New("safeclient: scheme not allowed (https only)")

	// ErrBadPort indicates a port other than 443.
	ErrBadPort = errors.New("safeclient: port not allowed (443 only)")

	// ErrResponseTooLarge indicates the response body exceeded Config.MaxResponseBytes.
	ErrResponseTooLarge = errors.New("safeclient: response body exceeded maximum size")
)

// blockReason is the value recorded in the safeclient_requests_blocked_total{reason=...}
// metric for a policy block.
type blockReason string

const (
	reasonScheme      blockReason = "scheme"
	reasonPort        blockReason = "port"
	reasonDestination blockReason = "destination"
	reasonCredentials blockReason = "credentials"
)

// mapSafeurlError translates an error returned from safeurl's WrappedClient.Do (or our own
// pre-checks) into one of our typed errors. The first return value is the metric reason
// for policy blocks, or the empty string for non-policy (transient/context) errors.
//
// safeurl returns scheme/credentials/host validation errors directly, while port and
// resolved-IP errors surface from the dialer wrapped inside a *url.Error; errors.As
// traverses that wrapping.
func mapSafeurlError(err error) (blockReason, error) {
	if err == nil {
		return "", nil
	}

	if _, ok := errors.AsType[*safeurl.AllowedSchemeError](err); ok {
		return reasonScheme, fmt.Errorf("%w: %v", ErrBadScheme, err)
	}

	if _, ok := errors.AsType[*safeurl.AllowedPortError](err); ok {
		return reasonPort, fmt.Errorf("%w: %v", ErrBadPort, err)
	}

	if _, ok := errors.AsType[*safeurl.SendingCredentialsBlockedError](err); ok {
		return reasonCredentials, fmt.Errorf("%w: %v", ErrBlockedDestination, err)
	}

	_, isIPErr := errors.AsType[*safeurl.AllowedIPError](err)
	_, isIPv6Err := errors.AsType[*safeurl.IPv6BlockedError](err)
	_, isInvalidHostErr := errors.AsType[*safeurl.InvalidHostError](err)
	_, isHostErr := errors.AsType[*safeurl.AllowedHostError](err)

	if isIPErr || isIPv6Err || isInvalidHostErr || isHostErr {
		return reasonDestination, fmt.Errorf("%w: %v", ErrBlockedDestination, err)
	}

	// Context errors stay matchable via errors.Is so callers can treat caller-owned
	// deadline/cancellation distinctly from policy blocks. These are not policy blocks, so
	// no metric reason is returned.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "", err
	}

	// Everything else is a transient/network failure: retryable.
	return "", err
}
