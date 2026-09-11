package durable

import (
	"bytes"
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
	"sync"
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

// ErrUpgradeHeadersTooLarge is returned when the endpoint's upgrade response headers exceed
// Params.MaxUpgradeHeaderBytes. The handshake is parsed before the websocket frame limit
// applies, so this is the only bound on what an endpoint can make the operator allocate for
// the response.
var ErrUpgradeHeadersTooLarge = errors.New("endpoint upgrade response headers exceeded the limit")

// headerTerminator ends the HTTP response head; nothing after it counts toward the budget.
var headerTerminator = []byte("\r\n\r\n")

// headerLimitConn budgets the bytes read from the connection until the end of the HTTP
// response head. It wraps the decrypted stream (the TLS connection for wss, the TCP one for
// ws), so the budget is on what the handshake parser allocates, not on TLS records. Once the
// terminator has been seen every read passes through untouched.
type headerLimitConn struct {
	net.Conn
	mu        sync.Mutex
	remaining int64
	tail      []byte
	done      bool
}

func newHeaderLimitConn(conn net.Conn, limit int64) *headerLimitConn {
	return &headerLimitConn{Conn: conn, remaining: limit}
}

func (c *headerLimitConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.done || n == 0 {
		return n, err
	}

	// The terminator may straddle reads: search the last three bytes of the previous read
	// together with this one.
	window := append(append([]byte{}, c.tail...), p[:n]...)

	if i := bytes.Index(window, headerTerminator); i >= 0 {
		counted := int64(i + len(headerTerminator) - len(c.tail))
		c.done = true
		c.tail = nil

		if counted > c.remaining {
			return 0, ErrUpgradeHeadersTooLarge
		}

		return n, err
	}

	c.remaining -= int64(n)

	if c.remaining < 0 {
		return 0, ErrUpgradeHeadersTooLarge
	}

	keep := len(headerTerminator) - 1

	if len(window) < keep {
		keep = len(window)
	}

	c.tail = append([]byte{}, window[len(window)-keep:]...)

	return n, err
}

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
// covers the endpoint id, timestamp, nonce, task id and invocation so a captured upgrade
// cannot be replayed for another task or endpoint or, once the endpoint tracks nonces, at
// all.
func signedUpgradeHeaders(secret, endpointId, taskId string, invocation int32, now time.Time, nonce string) (http.Header, error) {
	if secret == "" {
		return nil, fmt.Errorf("%w: endpoint %s", ErrNoSecret, endpointId)
	}

	timestamp := strconv.FormatInt(now.Unix(), 10)
	inv := strconv.FormatInt(int64(invocation), 10)

	sig, err := signature.Sign(contract.UpgradeSigningPayload(endpointId, timestamp, nonce, taskId, inv), secret)

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

// dial performs the signed upgrade through the policy dialer. TLS is negotiated here on top
// of the dialed connection with the URL's host name, so certificate verification is
// unaffected by the dialer connecting to a validated IP, and the handshake response is read
// through a headerLimitConn on the decrypted stream so its headers are bounded by
// Params.MaxUpgradeHeaderBytes.
func dial(ctx context.Context, p *Params) (*websocket.Conn, error) {
	target, err := upgradeURL(p.TriggerURL, p.Insecure)

	if err != nil {
		return nil, err
	}

	targetURL, err := url.Parse(target)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", safeclient.ErrBlockedDestination, err)
	}

	nonce, err := p.newNonce()

	if err != nil {
		return nil, err
	}

	headers, err := signedUpgradeHeaders(p.Secret, p.EndpointId, p.TaskId, p.Invocation, p.now(), nonce)

	if err != nil {
		return nil, err
	}

	tlsConfig := p.tlsConfig

	if tlsConfig == nil {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	dialer := &websocket.Dialer{
		// ws (InsecureDestinations only): the policy dial, header-budgeted.
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := p.Dialer.DialContext(ctx, network, addr)

			if err != nil {
				return nil, err
			}

			return newHeaderLimitConn(conn, p.MaxUpgradeHeaderBytes), nil
		},
		// wss: the policy dial, TLS with the URL's host name, then the header budget on the
		// decrypted stream. With this set the library does no TLS of its own.
		NetDialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialTLS(ctx, p, tlsConfig, targetURL.Hostname(), network, addr)
		},
		HandshakeTimeout: p.HandshakeTimeout,
		// Never pick up HTTP_PROXY / HTTPS_PROXY from the environment.
		Proxy: nil,
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

// dialTLS opens the policy connection and negotiates TLS on it with serverName, then wraps
// it so the upgrade response head is budgeted.
func dialTLS(ctx context.Context, p *Params, base *tls.Config, serverName, network, addr string) (net.Conn, error) {
	raw, err := p.Dialer.DialContext(ctx, network, addr)

	if err != nil {
		return nil, err
	}

	cfg := base.Clone()

	if cfg.ServerName == "" {
		cfg.ServerName = serverName
	}

	tlsConn := tls.Client(raw, cfg)

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = raw.Close()

		// The handshake error reaches the tenant as the task error: name the host and
		// the certificate problem, nothing about the connection.
		p.Logger.Warn().Err(err).Str("host", serverName).Str("endpoint_id", p.EndpointId).Msg("durable upgrade TLS handshake failed")

		return nil, safeclient.PublicError(serverName, err)
	}

	return newHeaderLimitConn(tlsConn, p.MaxUpgradeHeaderBytes), nil
}
