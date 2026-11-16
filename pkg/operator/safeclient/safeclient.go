// Package safeclient provides an SSRF-hardened HTTP client for delivering outbound
// requests to user-supplied endpoints. It is built on github.com/doyensec/safeurl, which
// validates the resolved IP at connection time (so it is safe against DNS rebinding), and
// layers on an explicit, auditable denylist plus synchronous pre-checks.
//
// Policy enforced by this package:
//   - only scheme https, only port 443;
//   - all internal/reserved IP space is blocked, plus any caller-supplied
//     Config.InfraBlockedCIDRs;
//   - redirects are never followed (3xx is surfaced to the caller as the result status);
//   - the response body is capped at Config.MaxResponseBytes, and reading stops there;
//   - HTTP/1.1 only, so every retained connection is in the transport's idle pool, which
//     is bounded globally, per host and in time (Config.MaxIdleConns, MaxIdleConnsPerHost,
//     IdleConnTimeout); CloseIdleConnections releases them;
//   - transport errors are returned as EndpointError, worded for the tenant, with the
//     detail in the operator log;
//   - the overall request deadline is owned by the CALLER via context.Context: this
//     package imposes no overall request timeout.
package safeclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/doyensec/safeurl"
	"github.com/rs/zerolog"
)

const (
	defaultConnectTimeout = 5 * time.Second
	// defaultMaxResponseBytes matches the gRPC server's default max message size
	// (GRPCMaxMsgSize, 4194304 bytes) in pkg/config/server so response bodies and gRPC
	// payloads share the same ceiling. Callers can override via Config.MaxResponseBytes.
	defaultMaxResponseBytes = 4 * 1024 * 1024 // 4 MiB
	allowedScheme           = "https"
	allowedPort             = 443

	// Idle pool defaults. One operator process talks to many distinct origins, most of them
	// rarely, so the pool is bounded globally and expires quickly; per host a few
	// connections cover a burst of deliveries to one endpoint.
	defaultMaxIdleConns        = 256
	defaultMaxIdleConnsPerHost = 4
	defaultIdleConnTimeout     = 90 * time.Second
)

// Config controls the SSRF policy and resource limits of a Sender.
type Config struct {
	InfraBlockedCIDRs    []string
	allowedPortsOverride []int
	testAllowedIPs       []string
	ConnectTimeout       time.Duration
	MaxResponseBytes     int64

	MaxRedirects int

	// MaxIdleConns, MaxIdleConnsPerHost and IdleConnTimeout bound the transport's idle
	// connection pool, which is every connection the Sender retains: it speaks HTTP/1.1
	// only. Zero values take the defaults (256, 4 and 90s).
	MaxIdleConns         int
	MaxIdleConnsPerHost  int
	IdleConnTimeout      time.Duration
	AllowEmptyInfraCIDRs bool
	EnableIPv6           bool

	// InsecureDestinations disables the SSRF policy for local development and e2e runs:
	// plain http, any port, and loopback and private ranges are all allowed, and the
	// request goes through a plain http.Client instead of safeurl. TLS verification stays
	// on. It is only honored when set explicitly by the operator's own configuration;
	// nothing in this package turns it on.
	InsecureDestinations bool

	testDisableBlocklist bool
	testInsecureTLS      bool
}

// DeliveryResult is the outcome of a successful (network-completed) delivery attempt. A
// 3xx status is reported here, not followed.
type DeliveryResult struct {
	BodyPrefix []byte
	StatusCode int
	Duration   time.Duration
}

// httpDoer is the subset of http.Client the Sender uses: the safeurl WrappedClient in the
// default policy, a plain http.Client under InsecureDestinations.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Sender delivers outbound HTTP requests under the SSRF policy. Construct one with New and
// reuse it; it is safe for concurrent use.
type Sender struct {
	client httpDoer
	// transport is the transport the client actually uses: under the SSRF policy the clone
	// safeurl installed its dialer on, so it holds the pooled connections.
	transport      *http.Transport
	blocklist      *blocklist
	l              *zerolog.Logger
	allowedPorts   []int
	maxBytes       int64
	connectTimeout time.Duration
	insecure       bool
	enableIPv6     bool
}

// New validates cfg, builds the safeurl-backed client, and returns a Sender. It fails if
// InfraBlockedCIDRs is empty (unless AllowEmptyInfraCIDRs) or if any blocked CIDR — default
// or infra — fails to parse. l may be nil (logging is then disabled).
func New(cfg Config, l *zerolog.Logger) (*Sender, error) {
	cfg = cfg.withDefaults()

	if cfg.InsecureDestinations {
		return newInsecureSender(cfg, l), nil
	}

	if len(cfg.InfraBlockedCIDRs) == 0 && !cfg.AllowEmptyInfraCIDRs {
		return nil, fmt.Errorf("safeclient: InfraBlockedCIDRs is required (set AllowEmptyInfraCIDRs for local dev)")
	}

	bl, err := newBlocklist(cfg.InfraBlockedCIDRs)

	if err != nil {
		return nil, fmt.Errorf("safeclient: could not parse blocked CIDRs: %w", err)
	}

	ports := []int{allowedPort}
	if len(cfg.allowedPortsOverride) > 0 {
		ports = cfg.allowedPortsOverride
	}

	allBlocked := make([]string, 0, len(DefaultBlockedCIDRs)+len(cfg.InfraBlockedCIDRs))
	allBlocked = append(allBlocked, DefaultBlockedCIDRs...)
	allBlocked = append(allBlocked, cfg.InfraBlockedCIDRs...)

	if cfg.testDisableBlocklist {
		allBlocked = nil
		bl = &blocklist{}
	}

	transport := cfg.newTransport()

	if cfg.testInsecureTLS {
		transport.TLSClientConfig.InsecureSkipVerify = true // #nosec G402 -- test-only
	}

	builder := safeurl.GetConfigBuilder().
		SetAllowedSchemes(allowedScheme).
		SetAllowedPorts(ports...).
		SetBlockedIPsCIDR(allBlocked...).
		SetCheckRedirect(noRedirects).
		EnableIPv6(cfg.EnableIPv6).
		AllowSendingCredentials(false).
		SetTransport(transport)

	if len(cfg.testAllowedIPs) > 0 {
		builder = builder.SetAllowedIPs(cfg.testAllowedIPs...)
	}
	// NOTE: deliberately never call SetTimeout — the caller owns the overall request
	// budget via context.Context (see Deliver).

	client := safeurl.Client(builder.Build())

	// safeurl clones the transport it was given and installs its own dialer on the clone;
	// the clone is the one that pools connections, so it is the one CloseIdleConnections
	// has to reach.
	effective, ok := client.Client.Transport.(*http.Transport)

	if !ok {
		return nil, fmt.Errorf("safeclient: safeurl installed a %T transport, expected *http.Transport", client.Client.Transport)
	}

	return &Sender{
		client:         client,
		transport:      effective,
		blocklist:      bl,
		allowedPorts:   ports,
		maxBytes:       cfg.MaxResponseBytes,
		connectTimeout: cfg.ConnectTimeout,
		enableIPv6:     cfg.EnableIPv6,
		l:              l,
	}, nil
}

// newInsecureSender builds the development-only Sender: no scheme, port or destination
// checks, no proxy, no redirects. The response cap and the caller-owned deadline still
// apply.
func newInsecureSender(cfg Config, l *zerolog.Logger) *Sender {
	transport := cfg.newTransport()

	if l != nil {
		l.Warn().Msg("safeclient: SSRF policy disabled by InsecureDestinations; never run this way in production")
	}

	return &Sender{
		client:         &http.Client{Transport: transport, CheckRedirect: noRedirects},
		transport:      transport,
		blocklist:      &blocklist{},
		maxBytes:       cfg.MaxResponseBytes,
		connectTimeout: cfg.ConnectTimeout,
		l:              l,
		insecure:       true,
		enableIPv6:     true,
	}
}

// withDefaults fills the zero limits.
func (cfg Config) withDefaults() Config {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}

	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = defaultMaxResponseBytes
	}

	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = defaultMaxIdleConns
	}

	if cfg.MaxIdleConnsPerHost <= 0 {
		cfg.MaxIdleConnsPerHost = defaultMaxIdleConnsPerHost
	}

	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = defaultIdleConnTimeout
	}

	return cfg
}

// newTransport builds the transport both policy modes share: no proxy, HTTP/1.1 only, a
// hung TLS handshake backstopped by ConnectTimeout and a bounded idle pool.
//
// The Sender speaks HTTP/1.1 only. HTTP/2 connections live in net/http's h2 pool, which
// MaxIdleConns does not bound: a fleet polling many distinct origins would retain one
// connection per origin for as long as the origin is revisited. HTTP/1.1 keep-alive under
// the bounded idle pool is the explicit policy, so every retained connection is counted
// and evicted by the transport. Three settings make that hold together with the dialer
// safeurl installs: ForceAttemptHTTP2 stays off, TLSNextProto is an empty map so net/http
// never registers the h2 upgrade, and ALPN offers http/1.1 alone so an origin cannot
// select h2 and have its frames read by the HTTP/1 parser. The SSRF dial-time check runs
// at the TCP layer regardless of the HTTP version.
func (cfg Config) newTransport() *http.Transport {
	return &http.Transport{
		// Backstop for a hung TLS handshake; see Config.ConnectTimeout caveat.
		TLSHandshakeTimeout: cfg.ConnectTimeout,
		// Never pick up HTTP_PROXY / HTTPS_PROXY from the environment.
		Proxy:             nil,
		ForceAttemptHTTP2: false,
		TLSNextProto:      map[string]func(string, *tls.Conn) http.RoundTripper{},
		TLSClientConfig:   &tls.Config{NextProtos: []string{"http/1.1"}, MinVersion: tls.VersionTLS12},

		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
	}
}

// CloseIdleConnections drops every pooled connection. Call it when the Sender is retired
// or when the origins it served are gone.
func (s *Sender) CloseIdleConnections() {
	if s.transport != nil {
		s.transport.CloseIdleConnections()
	}
}

// noRedirects tells the underlying http.Client to return the 3xx response as-is rather
// than following it, so the caller sees the redirect status as a delivery outcome.
func noRedirects(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// Deliver issues a request with the given method, body, and headers to endpoint,
// enforcing the full SSRF policy. The HTTP method is chosen by the caller (e.g.
// http.MethodPost, http.MethodGet). It never follows redirects and caps the response body
// at the configured maximum.
//
// The overall timeout/deadline is owned by the CALLER via ctx: this method sets no
// http.Client.Timeout and wraps no internal context. If ctx has no deadline, the request
// may run unbounded — callers should set a deadline.
//
// Retries: callers must re-invoke Deliver for each attempt so that full validation —
// including fresh DNS resolution and the dial-time IP check — runs every time. Never cache
// a resolved IP across attempts.
//
// It returns a *DeliveryResult on a completed request (including 3xx), or a typed error:
// ErrBadScheme, ErrBadPort, ErrBlockedDestination (do not retry — surface to the user),
// ErrResponseTooLarge, or an *EndpointError wrapping the context/network error
// (retryable). Use errors.Is to distinguish them; an EndpointError's message is safe to
// show the tenant and the wrapped detail is logged here.
func (s *Sender) Deliver(ctx context.Context, method, endpoint string, body []byte, headers http.Header) (*DeliveryResult, error) {
	start := time.Now()

	// Synchronous, network-free pre-check. This runs on every attempt (so retries
	// re-validate) and rejects bad scheme/port/userinfo and obfuscated IP-literal hosts
	// before any I/O. The dial-time check in safeurl remains the authoritative enforcement
	// point for DNS-resolved hosts.
	if reason, err := s.validate(endpoint); err != nil {
		s.recordBlocked(hostOf(endpoint), reason, err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))

	if err != nil {
		return nil, fmt.Errorf("safeclient: could not build request: %w", err)
	}

	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	if len(body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	host := req.URL.Hostname()

	resp, err := s.client.Do(req)

	if err != nil {
		reason, mapped := mapSafeurlError(host, err)

		if reason != "" {
			s.recordBlocked(host, reason, err)
			return nil, mapped
		}

		return nil, s.publicError(host, mapped)
	}

	defer resp.Body.Close()

	// Read one byte past the cap to tell "exactly the cap" from "over it", and nothing
	// more: an oversize response is rejected as soon as the cap is crossed and the body is
	// closed unread, which drops the connection instead of draining a stream the endpoint
	// controls.
	limited := io.LimitReader(resp.Body, s.maxBytes+1)

	prefix, err := io.ReadAll(limited)

	if err != nil {
		return nil, s.publicError(host, err)
	}

	if int64(len(prefix)) > s.maxBytes {
		return nil, ErrResponseTooLarge
	}

	return &DeliveryResult{
		StatusCode: resp.StatusCode,
		BodyPrefix: prefix,
		Duration:   time.Since(start),
	}, nil
}

// validate performs the network-free scheme/port/userinfo/host-literal checks. On failure
// it returns the metric reason and a typed error; on success it returns ("", nil).
func (s *Sender) validate(endpoint string) (blockReason, error) {
	if s.insecure {
		return validateInsecureURL(endpoint)
	}

	if reason, err := validateURL(endpoint, s.allowedPorts); err != nil {
		return reason, err
	}

	// Reject IP-literal hosts (including obfuscated decimal/hex/octal forms) that fall in a
	// blocked range, before any network I/O.
	u, err := url.Parse(endpoint)
	if err != nil {
		return reasonDestination, fmt.Errorf("%w: endpoint URL could not be parsed", ErrBlockedDestination)
	}

	if ip, ok := parseHostLiteralIP(u.Hostname()); ok && s.blocklist.isBlockedIP(ip) {
		return reasonDestination, fmt.Errorf("%w: literal host %s", ErrBlockedDestination, u.Hostname())
	}

	return "", nil
}

// publicError logs the transport failure and returns its tenant-facing form.
func (s *Sender) publicError(host string, err error) error {
	public := PublicError(host, err)

	var endpointErr *EndpointError

	if s.l != nil && errors.As(public, &endpointErr) {
		// Never log request bodies or URLs. Host, stage and the transport error only.
		s.l.Warn().
			Str("host", host).
			Str("stage", string(endpointErr.Stage)).
			Err(logCause(err)).
			Msg("outbound request failed")
	}

	return public
}

// recordBlocked logs a policy block with its cause, which is the operator's to see: the
// tenant gets the public error the caller returns.
func (s *Sender) recordBlocked(host string, reason blockReason, err error) {
	if s.l == nil {
		return
	}

	// Never log request bodies or URLs. Host, matched rule and the cause only.
	s.l.Warn().
		Str("host", host).
		Str("reason", string(reason)).
		Err(logCause(err)).
		Msg("blocked outbound request")
}

// hostOf extracts the hostname from a raw URL for logging, tolerating parse errors.
func hostOf(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return u.Hostname()
	}

	return ""
}

// ValidateEndpoint runs the synchronous, network-free scheme/port/userinfo checks against
// a raw URL. It is exposed for reuse at the endpoint-registration API boundary so invalid
// URLs can be rejected with a clear message at submission time.
//
// This is a UX convenience only: the dial-time IP check in Deliver remains the real
// enforcement point. Do NOT pre-resolve DNS and store the resolved IP — resolution must
// happen fresh on every delivery attempt.
func ValidateEndpoint(rawURL string) error {
	_, err := validateURL(rawURL, []int{allowedPort})
	return err
}

// validateInsecureURL is the InsecureDestinations pre-check: http or https with a host.
func validateInsecureURL(rawURL string) (blockReason, error) {
	u, err := url.Parse(rawURL)

	if err != nil {
		return reasonDestination, fmt.Errorf("%w: endpoint URL could not be parsed", ErrBlockedDestination)
	}

	if u.Scheme != allowedScheme && u.Scheme != "http" {
		return reasonScheme, fmt.Errorf("%w: got %q", ErrBadScheme, u.Scheme)
	}

	if u.Hostname() == "" {
		return reasonDestination, fmt.Errorf("%w: empty host", ErrBlockedDestination)
	}

	return "", nil
}

// validateURL is the shared scheme/port/userinfo validator. allowedPorts is the set of
// permitted explicit ports (production: just 443). On failure it returns the metric reason
// and a typed error; on success it returns ("", nil).
func validateURL(rawURL string, allowedPorts []int) (blockReason, error) {
	u, err := url.Parse(rawURL)

	if err != nil {
		return reasonDestination, fmt.Errorf("%w: endpoint URL could not be parsed", ErrBlockedDestination)
	}

	if u.Scheme != allowedScheme {
		return reasonScheme, fmt.Errorf("%w: got %q", ErrBadScheme, u.Scheme)
	}

	if u.User != nil {
		return reasonCredentials, fmt.Errorf("%w: userinfo not allowed in URL", ErrBlockedDestination)
	}

	if u.Hostname() == "" {
		return reasonDestination, fmt.Errorf("%w: empty host", ErrBlockedDestination)
	}

	// An explicit port must be in the allowed set. No port (empty) defaults to 443 and is
	// fine.
	if p := u.Port(); p != "" {
		pn, err := strconv.Atoi(p)

		if err != nil || !slices.Contains(allowedPorts, pn) {
			return reasonPort, fmt.Errorf("%w: got %q", ErrBadPort, p)
		}
	}

	return "", nil
}
