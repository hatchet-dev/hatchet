//go:build !e2e && !load && !rampup && !integration

package safeclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doyensec/safeurl"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// effectiveTransport is the transport the Sender actually sends through: safeurl clones
// the one it is given.
func effectiveTransport(t *testing.T, s *Sender) *http.Transport {
	t.Helper()

	switch c := s.client.(type) {
	case *safeurl.WrappedClient:
		return c.Client.Transport.(*http.Transport)
	case *http.Client:
		return c.Transport.(*http.Transport)
	default:
		t.Fatalf("unexpected client %T", s.client)
		return nil
	}
}

// TestIdleConnectionsAreBounded is the correctness F19 / performance F7 regression: the
// transport must have a global idle connection limit and an idle timeout, in both policy
// modes, and closing the Sender must release what is pooled.
func TestIdleConnectionsAreBounded(t *testing.T) {
	var idle, closed atomic.Int32

	ports := make([]int, 0, 10)
	servers := make([]*httptest.Server, 0, 10)

	for i := 0; i < 10; i++ {
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
			switch state {
			case http.StateIdle:
				idle.Add(1)
			case http.StateClosed:
				closed.Add(1)
			}
		}
		srv.StartTLS()
		t.Cleanup(srv.Close)

		_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
		require.NoError(t, err)
		p, err := strconv.Atoi(port)
		require.NoError(t, err)

		ports = append(ports, p)
		servers = append(servers, srv)
	}

	l := zerolog.Nop()

	s, err := New(Config{
		AllowEmptyInfraCIDRs: true,
		allowedPortsOverride: ports,
		testDisableBlocklist: true,
		testAllowedIPs:       []string{"127.0.0.1", "::1"},
		testInsecureTLS:      true,
	}, &l)
	require.NoError(t, err)

	for _, srv := range servers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := s.Deliver(ctx, http.MethodPost, srv.URL, nil, nil)
		cancel()
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool { return idle.Load() == 10 }, 5*time.Second, time.Millisecond)

	transport := effectiveTransport(t, s)
	assert.Positive(t, transport.MaxIdleConns, "no global idle connection limit")
	assert.Positive(t, transport.MaxIdleConnsPerHost, "no per-host idle connection limit")
	assert.Positive(t, transport.IdleConnTimeout, "no idle connection timeout")

	s.CloseIdleConnections()
	require.Eventually(t, func() bool { return closed.Load() == 10 }, 5*time.Second, time.Millisecond, "pooled connections survive CloseIdleConnections")

	insecure, err := New(Config{InsecureDestinations: true}, &l)
	require.NoError(t, err)

	transport = effectiveTransport(t, insecure)
	assert.Positive(t, transport.MaxIdleConns)
	assert.Positive(t, transport.MaxIdleConnsPerHost)
	assert.Positive(t, transport.IdleConnTimeout)
}

// TestIdleConnectionLimitsAreConfigurable checks the knobs reach the transport.
func TestIdleConnectionLimitsAreConfigurable(t *testing.T) {
	l := zerolog.Nop()

	s, err := New(Config{
		AllowEmptyInfraCIDRs: true,
		MaxIdleConns:         7,
		MaxIdleConnsPerHost:  3,
		IdleConnTimeout:      11 * time.Second,
	}, &l)
	require.NoError(t, err)

	transport := effectiveTransport(t, s)
	assert.Equal(t, 7, transport.MaxIdleConns)
	assert.Equal(t, 3, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 11*time.Second, transport.IdleConnTimeout)
}

type countingBody struct {
	io.Reader
	n int
}

func (r *countingBody) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.n += n

	return n, err
}

type stubDoer struct {
	body io.ReadCloser
}

func (d stubDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: d.body}, nil
}

// TestDeliver_StopsReadingAtTheResponseCap is the correctness F20 regression: an 8-byte cap
// must not consume a 64 KiB body before reporting ErrResponseTooLarge.
func TestDeliver_StopsReadingAtTheResponseCap(t *testing.T) {
	body := &countingBody{Reader: bytes.NewReader(make([]byte, 64*1024))}

	s, err := New(Config{InsecureDestinations: true, MaxResponseBytes: 8}, nil)
	require.NoError(t, err)

	s.client = stubDoer{body: io.NopCloser(body)}

	_, err = s.Deliver(context.Background(), http.MethodGet, "http://endpoint.test", nil, nil)
	require.ErrorIs(t, err, ErrResponseTooLarge)
	assert.LessOrEqual(t, body.n, 9, "8-byte cap consumed %d bytes", body.n)
}

// TestDeliver_OversizeReturnsBeforeTheDeadline is the security F11 regression: a streaming
// endpoint that exceeds the cap and then stalls must not hold the delivery until the
// caller's deadline.
func TestDeliver_OversizeReturnsBeforeTheDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 1025))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	s, err := New(Config{InsecureDestinations: true, MaxResponseBytes: 1024}, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err = s.Deliver(ctx, http.MethodPost, srv.URL, nil, nil)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrResponseTooLarge)
	assert.Less(t, elapsed, time.Second, "returned only after %s", elapsed)
}

// TestTransportErrorsAreRedacted is the security F12 regression: the errors Deliver and
// DialContext return reach tenants as task errors and endpoint status, so they carry the
// endpoint host and a stable category but no resolved addresses, ports or resolver detail.
func TestTransportErrorsAreRedacted(t *testing.T) {
	l := zerolog.Nop()

	t.Run("connection refused", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		_, port, err := net.SplitHostPort(ln.Addr().String())
		require.NoError(t, err)
		require.NoError(t, ln.Close())

		s, err := New(Config{InsecureDestinations: true}, &l)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// The configured host is what the tenant may see; the address it resolved to, the
		// port, the path and the socket error are not.
		_, err = s.Deliver(ctx, http.MethodPost, "http://localhost:"+port+"/hook?token=abc", nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "localhost")
		assert.NotContains(t, err.Error(), "127.0.0.1", "resolved address leaked: %s", err)
		assert.NotContains(t, err.Error(), port, "port leaked: %s", err)
		assert.NotContains(t, err.Error(), "token=abc", "query string leaked: %s", err)
		assert.NotContains(t, err.Error(), "connection refused", "socket detail leaked: %s", err)
		assert.False(t, errors.Is(err, context.DeadlineExceeded))

		var endpointErr *EndpointError
		require.ErrorAs(t, err, &endpointErr)
		assert.Equal(t, StageConnect, endpointErr.Stage)
	})

	t.Run("dns failure", func(t *testing.T) {
		s, err := New(Config{InsecureDestinations: true}, &l)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = s.Deliver(ctx, http.MethodPost, "http://endpoint.invalid/hook", nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint.invalid")
		assert.NotContains(t, strings.ToLower(err.Error()), "lookup ", "resolver detail leaked: %s", err)
		assert.NotContains(t, err.Error(), "no such host", "resolver detail leaked: %s", err)
	})

	t.Run("blocked resolution", func(t *testing.T) {
		s := newTestSender(t)

		_, err := s.DialContext(context.Background(), "tcp", "localhost:443")
		require.ErrorIs(t, err, ErrBlockedDestination)
		assert.NotContains(t, err.Error(), "127.0.0.1", "blocked address leaked: %s", err)
		assert.NotContains(t, err.Error(), "::1", "blocked address leaked: %s", err)
	})

	t.Run("dial failure", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		_, port, err := net.SplitHostPort(ln.Addr().String())
		require.NoError(t, err)
		require.NoError(t, ln.Close())

		p, err := strconv.Atoi(port)
		require.NoError(t, err)

		s, err := New(Config{
			AllowEmptyInfraCIDRs: true,
			allowedPortsOverride: []int{p},
			testDisableBlocklist: true,
		}, &l)
		require.NoError(t, err)

		_, err = s.DialContext(context.Background(), "tcp", "127.0.0.1:"+port)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), port, "port leaked: %s", err)
		assert.NotContains(t, err.Error(), "connection refused", "socket detail leaked: %s", err)
	})
}
