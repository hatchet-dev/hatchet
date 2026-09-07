package durable

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

// NetDialer opens the TCP connection under the operator's SSRF policy. *safeclient.Sender
// satisfies it; tests use a plain net.Dialer against loopback.
type NetDialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// ErrNoSecret is returned when the endpoint has no signing secret to sign the upgrade with.
var ErrNoSecret = errors.New("endpoint has no signing secret")

// UpgradeError is returned when the endpoint answered the upgrade with something other than
// 101. Status is the HTTP status it sent.
type UpgradeError struct {
	Status int
}

func (e *UpgradeError) Error() string {
	msg := fmt.Sprintf("endpoint refused websocket upgrade with status %d", e.Status)

	if text := http.StatusText(e.Status); text != "" {
		msg += " " + text
	}

	return msg
}

// Retryable reports whether the refusal is transient by the same rule as non-durable
// responses: 5xx, 408, 425 and 429 are retried, other statuses are not.
func (e *UpgradeError) Retryable() bool {
	return e.Status >= http.StatusInternalServerError ||
		e.Status == http.StatusRequestTimeout ||
		e.Status == http.StatusTooEarly ||
		e.Status == http.StatusTooManyRequests
}

// upgradeURL rewrites the trigger URL to its websocket scheme: https becomes wss; http
// becomes ws only under InsecureDestinations, where the safeclient policy is off as well.
func upgradeURL(trigger string, insecure bool) (string, error) {
	u, err := url.Parse(trigger)

	if err != nil {
		return "", fmt.Errorf("%w: %v", safeclient.ErrBlockedDestination, err)
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		if !insecure {
			return "", fmt.Errorf("%w: got %q", safeclient.ErrBadScheme, u.Scheme)
		}

		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("%w: got %q", safeclient.ErrBadScheme, u.Scheme)
	}

	if u.User != nil {
		return "", fmt.Errorf("%w: userinfo not allowed in URL", safeclient.ErrBlockedDestination)
	}

	if u.Hostname() == "" {
		return "", fmt.Errorf("%w: empty host", safeclient.ErrBlockedDestination)
	}

	return u.String(), nil
}

// newNonce returns 16 random bytes, base64url without padding.
func newNonce() (string, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate nonce: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// signedUpgradeHeaders builds the bodyless upgrade's authorization headers. The signature
// covers timestamp, nonce, task id and invocation so a captured upgrade cannot be replayed
// for another task or, once the endpoint tracks nonces, at all.
func signedUpgradeHeaders(secret, endpointId, taskId string, invocation int32, now time.Time, nonce string) (http.Header, error) {
	if secret == "" {
		return nil, fmt.Errorf("%w: endpoint %s", ErrNoSecret, endpointId)
	}

	timestamp := strconv.FormatInt(now.Unix(), 10)
	inv := strconv.FormatInt(int64(invocation), 10)

	sig, err := signature.Sign(contract.UpgradeSigningPayload(timestamp, nonce, taskId, inv), secret)

	if err != nil {
		return nil, fmt.Errorf("could not sign upgrade: %w", err)
	}

	h := http.Header{}
	h.Set(contract.EndpointIdHeader, endpointId)
	h.Set(contract.TimestampHeader, timestamp)
	h.Set(contract.NonceHeader, nonce)
	h.Set(contract.TaskIdHeader, taskId)
	h.Set(contract.InvocationHeader, inv)
	h.Set(contract.SignatureHeader, sig)

	return h, nil
}

// dial performs the signed upgrade through the policy dialer. TLS is negotiated by the
// websocket library on top of the dialed connection with the URL's host name, so certificate
// verification is unaffected by the dialer connecting to a validated IP.
func dial(ctx context.Context, p *Params) (*websocket.Conn, error) {
	target, err := upgradeURL(p.TriggerURL, p.Insecure)

	if err != nil {
		return nil, err
	}

	nonce, err := p.newNonce()

	if err != nil {
		return nil, err
	}

	headers, err := signedUpgradeHeaders(p.Secret, p.EndpointId, p.TaskId, p.Invocation, p.now(), nonce)

	if err != nil {
		return nil, err
	}

	dialer := &websocket.Dialer{
		NetDialContext:   p.Dialer.DialContext,
		HandshakeTimeout: p.HandshakeTimeout,
		TLSClientConfig:  p.tlsConfig,
		// Never pick up HTTP_PROXY / HTTPS_PROXY from the environment.
		Proxy: nil,
	}

	if dialer.TLSClientConfig == nil {
		dialer.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	conn, resp, err := dialer.DialContext(ctx, target, headers) //nolint:bodyclose // gorilla closes the response body on a failed handshake

	if err != nil {
		if errors.Is(err, websocket.ErrBadHandshake) && resp != nil {
			return nil, &UpgradeError{Status: resp.StatusCode}
		}

		return nil, err
	}

	return conn, nil
}
