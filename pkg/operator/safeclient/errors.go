package safeclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"

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

// mapSafeurlError translates an error returned from safeurl's WrappedClient.Do into one of
// our typed errors, worded for the tenant. The first return value is the metric reason for
// policy blocks, or the empty string for non-policy (transient/context) errors, which are
// returned as they are for the caller to wrap.
//
// A policy failure's public form names the configured host and the policy category and
// nothing else: safeurl's own messages carry the resolved address and the full request URL,
// query string included, which must not reach task errors or endpoint status. The cause is
// logged by the caller through logCause.
//
// safeurl returns scheme/credentials/host validation errors directly, while port and
// resolved-IP errors surface from the dialer wrapped inside a *url.Error; errors.As
// traverses that wrapping.
func mapSafeurlError(host string, err error) (blockReason, error) {
	if err == nil {
		return "", nil
	}

	if _, ok := errors.AsType[*safeurl.AllowedSchemeError](err); ok {
		return reasonScheme, fmt.Errorf("%w: endpoint host %s", ErrBadScheme, host)
	}

	if _, ok := errors.AsType[*safeurl.AllowedPortError](err); ok {
		return reasonPort, fmt.Errorf("%w: endpoint host %s", ErrBadPort, host)
	}

	if _, ok := errors.AsType[*safeurl.SendingCredentialsBlockedError](err); ok {
		return reasonCredentials, fmt.Errorf("%w: userinfo not allowed in URL", ErrBlockedDestination)
	}

	_, isIPErr := errors.AsType[*safeurl.AllowedIPError](err)
	_, isIPv6Err := errors.AsType[*safeurl.IPv6BlockedError](err)
	_, isInvalidHostErr := errors.AsType[*safeurl.InvalidHostError](err)
	_, isHostErr := errors.AsType[*safeurl.AllowedHostError](err)

	if isIPErr || isIPv6Err || isInvalidHostErr || isHostErr {
		return reasonDestination, fmt.Errorf("%w: endpoint host %s resolves to an address the policy does not allow", ErrBlockedDestination, host)
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

// logCause is the form of a transport or policy error the operator log gets: a *url.Error
// carries the whole request URL, userinfo and query string included, so its inner error is
// logged instead, under the host the caller logs beside it.
func logCause(err error) error {
	var urlErr *url.Error

	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}

	return err
}

// Stage is the step of an outbound request at which a transport failure happened.
type Stage string

const (
	StageResolve Stage = "resolve"
	StageConnect Stage = "connect"
	StageTLS     Stage = "tls"
	StageRead    Stage = "read"
	StageRequest Stage = "request"
)

// EndpointError is a transport failure on the way to an endpoint, worded for the tenant:
// it names the endpoint host and the stage that failed and nothing else, because the
// message ends up in task errors and endpoint status. Resolved addresses, ports, resolver
// and socket detail stay in the operator log, where the Sender writes them. The cause is
// still reachable through errors.Is and errors.As for classification (context errors,
// net.Error timeouts).
type EndpointError struct {
	cause error
	Host  string
	Stage Stage
}

func (e *EndpointError) Error() string {
	switch e.Stage {
	case StageResolve:
		return fmt.Sprintf("safeclient: could not resolve endpoint host %s", e.Host)
	case StageConnect:
		return fmt.Sprintf("safeclient: could not connect to endpoint host %s", e.Host)
	case StageTLS:
		msg := fmt.Sprintf("safeclient: TLS handshake with endpoint host %s failed", e.Host)

		// Certificate verification failures describe the endpoint's own certificate, which
		// is what the tenant needs to fix.
		if detail := certificateDetail(e.cause); detail != "" {
			msg += ": " + detail
		}

		return msg
	case StageRead:
		return fmt.Sprintf("safeclient: connection to endpoint host %s failed while reading the response", e.Host)
	default:
		return fmt.Sprintf("safeclient: request to endpoint host %s failed", e.Host)
	}
}

func (e *EndpointError) Unwrap() error {
	return e.cause
}

// PublicError wraps a transport error for the tenant (see EndpointError). Policy errors and
// errors that are already public are returned as they are; nil stays nil.
func PublicError(host string, err error) error {
	if err == nil {
		return nil
	}

	var public *EndpointError

	if errors.As(err, &public) {
		return err
	}

	if errors.Is(err, ErrBlockedDestination) || errors.Is(err, ErrBadScheme) || errors.Is(err, ErrBadPort) || errors.Is(err, ErrResponseTooLarge) {
		return err
	}

	return &EndpointError{Host: host, Stage: stageOf(err), cause: err}
}

// stageOf classifies a transport error by its innermost cause.
func stageOf(err error) Stage {
	var dnsErr *net.DNSError

	if errors.As(err, &dnsErr) {
		return StageResolve
	}

	if certificateDetail(err) != "" {
		return StageTLS
	}

	var (
		recordErr tls.RecordHeaderError
		alertErr  tls.AlertError
	)

	if errors.As(err, &recordErr) || errors.As(err, &alertErr) {
		return StageTLS
	}

	var opErr *net.OpError

	if errors.As(err, &opErr) {
		switch opErr.Op {
		case "dial":
			return StageConnect
		case "read", "write":
			return StageRead
		}
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return StageRead
	}

	return StageRequest
}

// certificateDetail returns the message of a certificate verification failure in err, or
// the empty string when err is not one.
func certificateDetail(err error) string {
	var (
		verifyErr *tls.CertificateVerificationError
		authErr   x509.UnknownAuthorityError
		hostErr   x509.HostnameError
		invalid   x509.CertificateInvalidError
	)

	switch {
	case errors.As(err, &verifyErr):
		return verifyErr.Error()
	case errors.As(err, &authErr):
		return authErr.Error()
	case errors.As(err, &hostErr):
		return hostErr.Error()
	case errors.As(err, &invalid):
		return invalid.Error()
	}

	return ""
}
