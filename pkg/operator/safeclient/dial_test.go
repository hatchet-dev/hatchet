//go:build !e2e && !load && !rampup && !integration

package safeclient

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDialContext_BlockedDestinations(t *testing.T) {
	s := newTestSender(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tests := []struct {
		name    string
		addr    string
		wantErr error
	}{
		{"loopback v4", "127.0.0.1:443", ErrBlockedDestination},
		{"loopback v6", "[::1]:443", ErrBlockedDestination},
		{"rfc1918", "10.0.0.5:443", ErrBlockedDestination},
		{"link-local metadata", "169.254.169.254:443", ErrBlockedDestination},
		{"obfuscated decimal loopback", "2130706433:443", ErrBlockedDestination},
		{"non-443 port", "example.com:8443", ErrBadPort},
		{"missing port", "example.com", ErrBlockedDestination},
		{"localhost name resolves to loopback", "localhost:443", ErrBlockedDestination},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := s.DialContext(ctx, "tcp", tt.addr)

			if conn != nil {
				_ = conn.Close()
			}

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}

	conn, err := s.DialContext(ctx, "udp", "example.com:443")
	assert.ErrorIs(t, err, ErrBlockedDestination, "only tcp networks")
	assert.Nil(t, conn)
}

// TestDialContext_AllowedIPDialsValidatedAddress reaches a loopback listener with the
// blocklist disabled, which is the only way to exercise the connect path without a public
// address. The dialer connects to the resolved IP, not the name.
func TestDialContext_AllowedIPDialsValidatedAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	l := zerolog.Nop()

	s, err := New(Config{
		AllowEmptyInfraCIDRs: true,
		allowedPortsOverride: []int{port},
		testDisableBlocklist: true,
	}, &l)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := s.DialContext(ctx, "tcp", "localhost:"+portStr)
	require.NoError(t, err)

	defer conn.Close()

	assert.Equal(t, srv.Listener.Addr().String(), conn.RemoteAddr().String())
}

func TestDialContext_InsecureAllowsLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l := zerolog.Nop()

	s, err := New(Config{InsecureDestinations: true}, &l)
	require.NoError(t, err)
	assert.True(t, s.InsecureDestinations())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := s.DialContext(ctx, "tcp", srv.Listener.Addr().String())
	require.NoError(t, err)
	_ = conn.Close()

	strict := newTestSender(t)
	assert.False(t, strict.InsecureDestinations())

	_, err = strict.DialContext(ctx, "tcp", srv.Listener.Addr().String())
	assert.Error(t, err, "the strict sender still refuses loopback on an ephemeral port")
}
