//go:build !e2e && !load && !rampup && !integration

package safeclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// connCounter counts the server side of a test server's connections through ConnState.
type connCounter struct {
	opened atomic.Int64
	closed atomic.Int64
}

func (c *connCounter) connState(_ net.Conn, s http.ConnState) {
	switch s {
	case http.StateNew:
		c.opened.Add(1)
	case http.StateClosed:
		c.closed.Add(1)
	}
}

func (c *connCounter) open() int64 {
	return c.opened.Load() - c.closed.Load()
}

// waitOpen waits for the server-side open connection count to settle at or below want.
func (c *connCounter) waitOpen(t *testing.T, want int64) int64 {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	for c.open() > want && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	return c.open()
}

// newProtoServer starts a TLS test server that records the protocol major version of the
// last request it served, with HTTP/2 offered when h2 is set.
func newProtoServer(t *testing.T, h2 bool, counter *connCounter) (*httptest.Server, *atomic.Int64, int) {
	t.Helper()

	var proto atomic.Int64

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proto.Store(int64(r.ProtoMajor))
		_, _ = fmt.Fprint(w, `{}`)
	}))

	if counter != nil {
		srv.Config.ConnState = counter.connState
	}

	srv.EnableHTTP2 = h2
	srv.StartTLS()
	t.Cleanup(srv.Close)

	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)

	pn, err := strconv.Atoi(port)
	require.NoError(t, err)

	return srv, &proto, pn
}

// The Sender speaks HTTP/1.1 even to an origin that offers HTTP/2, so every connection it
// keeps is in the transport's bounded idle pool.
func TestDeliver_NegotiatesHTTP1AgainstHTTP2Origin(t *testing.T) {
	srv, proto, port := newProtoServer(t, true, nil)

	l := zerolog.Nop()

	s, err := New(Config{
		AllowEmptyInfraCIDRs: true,
		allowedPortsOverride: []int{port},                  // test-only
		testDisableBlocklist: true,                         // test-only
		testAllowedIPs:       []string{"127.0.0.1", "::1"}, // test-only
		testInsecureTLS:      true,                         // test-only
	}, &l)
	require.NoError(t, err)

	defer s.CloseIdleConnections()

	res, err := s.Deliver(context.Background(), http.MethodPost, srv.URL, []byte("{}"), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "{}", string(res.BodyPrefix))
	assert.EqualValues(t, 1, proto.Load(), "the request must be served over HTTP/1.1")
}

// Polling many distinct origins retains at most MaxIdleConns connections, whether or not the
// origins offer HTTP/2, and CloseIdleConnections drops them all. Every origin is dialed to
// the one test server: the test replaces the dialer safeurl installed on the effective
// transport, which bypasses safeurl's dial-time address check for this test alone (the
// blocklist is disabled here anyway); the connection accounting under test is the
// transport's, above the dialer.
func TestDeliver_IdlePoolBoundsRetainedConnectionsAcrossOrigins(t *testing.T) {
	const (
		origins     = 500
		maxIdle     = 8
		maxIdleHost = 4
	)

	for _, h2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("origin_http2=%t", h2), func(t *testing.T) {
			counter := &connCounter{}
			srv, proto, port := newProtoServer(t, h2, counter)

			l := zerolog.Nop()

			s, err := New(Config{
				AllowEmptyInfraCIDRs: true,
				MaxIdleConns:         maxIdle,
				MaxIdleConnsPerHost:  maxIdleHost,
				allowedPortsOverride: []int{port},                  // test-only
				testDisableBlocklist: true,                         // test-only
				testAllowedIPs:       []string{"127.0.0.1", "::1"}, // test-only
				testInsecureTLS:      true,                         // test-only
			}, &l)
			require.NoError(t, err)

			defer s.CloseIdleConnections()

			tr := s.transport
			require.NotNil(t, tr)
			assert.Equal(t, maxIdle, tr.MaxIdleConns)
			assert.Equal(t, maxIdleHost, tr.MaxIdleConnsPerHost)

			target := srv.Listener.Addr().String()
			tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, target)
			}

			for i := 0; i < origins; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, err := s.Deliver(ctx, http.MethodPost, fmt.Sprintf("https://origin-%d.test:%d/", i, port), []byte("{}"), nil)
				cancel()
				require.NoError(t, err)
			}

			assert.EqualValues(t, 1, proto.Load(), "requests must be served over HTTP/1.1")

			open := counter.waitOpen(t, maxIdle)
			t.Logf("origin_http2=%t origins=%d opened=%d retained=%d max_idle=%d", h2, origins, counter.opened.Load(), open, maxIdle)
			assert.LessOrEqual(t, open, int64(maxIdle), "retained connections must not exceed MaxIdleConns")

			s.CloseIdleConnections()

			open = counter.waitOpen(t, 0)
			assert.Zero(t, open, "CloseIdleConnections must drop every pooled connection")
		})
	}
}
