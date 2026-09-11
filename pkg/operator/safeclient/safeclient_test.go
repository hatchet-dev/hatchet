//go:build !e2e && !load && !rampup && !integration

package safeclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSender builds a Sender for tests. It allows AllowEmptyInfraCIDRs and, optionally,
// extra allowed ports so an httptest server (which listens on an ephemeral port) can be
// reached. The port override is TEST-ONLY — production New never permits a non-443 port.
func newTestSender(t *testing.T, allowPorts ...int) *Sender {
	t.Helper()

	l := zerolog.Nop()

	s, err := New(Config{
		AllowEmptyInfraCIDRs: true,
		allowedPortsOverride: allowPorts, // test-only escape hatch
	}, &l)
	require.NoError(t, err)

	return s
}

func TestNew_RequiresInfraCIDRs(t *testing.T) {
	_, err := New(Config{}, nil)
	assert.Error(t, err, "New must fail when InfraBlockedCIDRs is empty without the dev override")

	l := zerolog.Nop()
	_, err = New(Config{AllowEmptyInfraCIDRs: true}, &l)
	assert.NoError(t, err)

	_, err = New(Config{InfraBlockedCIDRs: []string{"203.0.113.0/24"}}, &l)
	assert.NoError(t, err)
}

func TestNew_BadInfraCIDR(t *testing.T) {
	_, err := New(Config{InfraBlockedCIDRs: []string{"nonsense"}}, nil)
	assert.Error(t, err)
}

// TestDeliver_Rejected covers the table of destinations that must be rejected with the
// appropriate typed error WITHOUT completing (or even starting) a network request.
func TestDeliver_Rejected(t *testing.T) {
	s := newTestSender(t)

	tests := []struct {
		name     string
		endpoint string
		wantErr  error
	}{
		{"plain http", "http://example.com/hook", ErrBadScheme},
		{"non-443 port", "https://example.com:8443/hook", ErrBadPort},
		{"loopback v4", "https://127.0.0.1/", ErrBlockedDestination},
		{"loopback v6", "https://[::1]/", ErrBlockedDestination},
		{"rfc1918 10", "https://10.0.0.5/", ErrBlockedDestination},
		{"rfc1918 192", "https://192.168.1.1/", ErrBlockedDestination},
		{"rfc1918 172", "https://172.16.0.1/", ErrBlockedDestination},
		{"metadata", "https://169.254.169.254/latest/meta-data/", ErrBlockedDestination},
		{"cgnat", "https://100.64.0.1/", ErrBlockedDestination},
		{"decimal literal", "https://2130706433/", ErrBlockedDestination},
		{"hex literal", "https://0x7f000001/", ErrBlockedDestination},
		{"octal literal", "https://017700000001/", ErrBlockedDestination},
		{"ipv4-mapped v6", "https://[::ffff:127.0.0.1]/", ErrBlockedDestination},
		{"userinfo", "https://user:pass@example.com/", ErrBlockedDestination},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			res, err := s.Deliver(ctx, http.MethodPost, tc.endpoint, []byte("{}"), nil)
			assert.Nil(t, res)
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr, "endpoint %s", tc.endpoint)
		})
	}
}

func TestValidateEndpoint(t *testing.T) {
	assert.NoError(t, ValidateEndpoint("https://example.com/hook"))
	assert.NoError(t, ValidateEndpoint("https://example.com:443/hook"))
	assert.ErrorIs(t, ValidateEndpoint("http://example.com/hook"), ErrBadScheme)
	assert.ErrorIs(t, ValidateEndpoint("https://example.com:8443/hook"), ErrBadPort)
	assert.ErrorIs(t, ValidateEndpoint("https://user:pass@example.com/"), ErrBlockedDestination)
	assert.ErrorIs(t, ValidateEndpoint("https:///nohost"), ErrBlockedDestination)
}

// newServerSender builds a Sender wired to an httptest TLS server. An httptest server
// listens on loopback (blocked in prod) with a self-signed cert on an ephemeral port (also
// blocked in prod), so this uses the package's TEST-ONLY config escape hatches to permit
// the port, drop the blocklist, and skip TLS verification. Production New permits none of
// this — see TestNew_* and TestDeliver_Rejected, which assert the real policy.
func newServerSender(t *testing.T, srv *httptest.Server, maxBytes int64) (*Sender, string) {
	t.Helper()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	l := zerolog.Nop()

	s, err := New(Config{
		AllowEmptyInfraCIDRs: true,
		MaxResponseBytes:     maxBytes,
		allowedPortsOverride: []int{port},                  // test-only
		testDisableBlocklist: true,                         // test-only
		testAllowedIPs:       []string{"127.0.0.1", "::1"}, // test-only
		testInsecureTLS:      true,                         // test-only
	}, &l)
	require.NoError(t, err)

	return s, srv.URL
}

func TestDeliver_HappyPath(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "bar", r.Header.Get("X-Foo"))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	s, endpoint := newServerSender(t, srv, defaultMaxResponseBytes)

	h := http.Header{}
	h.Set("X-Foo", "bar")

	res, err := s.Deliver(context.Background(), http.MethodPost, endpoint, []byte(`{"hello":"world"}`), h)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "ok", string(res.BodyPrefix))
	assert.Greater(t, res.Duration, time.Duration(0))
}

func TestDeliver_RedirectNotFollowed(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.com/elsewhere", http.StatusFound)
	}))
	defer srv.Close()

	s, endpoint := newServerSender(t, srv, defaultMaxResponseBytes)

	res, err := s.Deliver(context.Background(), http.MethodPost, endpoint, []byte("{}"), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, res.StatusCode)
}

func TestDeliver_ResponseTooLarge(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, strings.Repeat("A", 1024))
	}))
	defer srv.Close()

	s, endpoint := newServerSender(t, srv, 100) // cap below body size

	res, err := s.Deliver(context.Background(), http.MethodPost, endpoint, []byte("{}"), nil)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, ErrResponseTooLarge)
}

func TestDeliver_NoClientImposedTimeout(t *testing.T) {
	// Server delays before responding but makes progress; with a generous ctx the request
	// must complete (proving the client sets no overall timeout of its own).
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "slow-ok")
	}))
	defer srv.Close()

	s, endpoint := newServerSender(t, srv, defaultMaxResponseBytes)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := s.Deliver(ctx, http.MethodPost, endpoint, []byte("{}"), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestDeliver_CallerDeadlineHonored(t *testing.T) {
	// Server hangs well past the caller's deadline.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s, endpoint := newServerSender(t, srv, defaultMaxResponseBytes)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	res, err := s.Deliver(ctx, http.MethodPost, endpoint, []byte("{}"), nil)
	assert.Nil(t, res)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "got %v", err)
}
