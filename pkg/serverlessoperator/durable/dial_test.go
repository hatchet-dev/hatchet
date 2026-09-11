//go:build !e2e && !load && !rampup && !integration

package durable

import (
	"context"
	"crypto/tls"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// paddedUpgrader answers the upgrade with headerBytes of padding in a response header.
func paddedUpgrader(headerBytes int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := http.Header{"X-Padding": []string{strings.Repeat("x", headerBytes)}}
		up := websocket.Upgrader{}

		if c, err := up.Upgrade(w, r, h); err == nil {
			_ = c.Close()
		}
	})
}

func dialParams(url string) Params {
	p := Params{
		Dialer:     loopbackDialer{},
		TriggerURL: url,
		Secret:     testSecret,
		EndpointId: "ep-1",
		TaskId:     testTaskId,
		Invocation: 3,
		Insecure:   true,
	}
	p.withDefaults()

	return p
}

// TestDialRejectsOversizedUpgradeHeaders is the security F01 regression: the handshake
// response headers are parsed before the frame limit applies, so an endpoint could make
// the operator allocate an arbitrarily large header block. The limit has to bound the
// decrypted HTTP bytes, so the TLS case is checked as well as the plain one.
func TestDialRejectsOversizedUpgradeHeaders(t *testing.T) {
	const headerBytes = 16 * 1024 * 1024

	t.Run("plain", func(t *testing.T) {
		srv := httptest.NewServer(paddedUpgrader(headerBytes))
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		p := dialParams(srv.URL)

		conn, err := dial(ctx, &p)

		if conn != nil {
			_ = conn.Close()
		}

		require.ErrorIs(t, err, ErrUpgradeHeadersTooLarge)

		out := classifyDialError(ctx, &p, err)
		assert.Equal(t, KindFailed, out.Kind)
		assert.False(t, out.Retry)
		assert.Contains(t, out.Error, "upgrade response headers exceeded")
	})

	t.Run("tls", func(t *testing.T) {
		srv := httptest.NewTLSServer(paddedUpgrader(headerBytes))
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		p := dialParams(srv.URL)
		p.Insecure = false
		p.tlsConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 -- loopback test server

		conn, err := dial(ctx, &p)

		if conn != nil {
			_ = conn.Close()
		}

		require.ErrorIs(t, err, ErrUpgradeHeadersTooLarge)
	})

	t.Run("under the limit", func(t *testing.T) {
		srv := httptest.NewServer(paddedUpgrader(1024))
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		p := dialParams(srv.URL)

		conn, err := dial(ctx, &p)
		require.NoError(t, err)
		_ = conn.Close()
	})

	t.Run("configured limit", func(t *testing.T) {
		srv := httptest.NewServer(paddedUpgrader(4096))
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		p := dialParams(srv.URL)
		p.MaxUpgradeHeaderBytes = 2048

		conn, err := dial(ctx, &p)

		if conn != nil {
			_ = conn.Close()
		}

		require.ErrorIs(t, err, ErrUpgradeHeadersTooLarge)
	})
}

// TestHeaderLimitConnStopsCountingAfterTheHeaders checks the wrapper only budgets the
// handshake: frames after the blank line are not counted, whatever their size, and the
// terminator is found across read boundaries.
func TestHeaderLimitConnStopsCountingAfterTheHeaders(t *testing.T) {
	head := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n\r\n"
	body := strings.Repeat("y", 4096)

	c := newHeaderLimitConn(&scriptedConn{data: []byte(head + body)}, 64)

	// Reads of 3 bytes split the "\r\n\r\n" terminator across calls.
	buf := make([]byte, 3)
	total := 0

	for total < len(head)+len(body) {
		n, err := c.Read(buf)
		require.NoError(t, err)
		total += n
	}

	assert.Equal(t, len(head)+len(body), total)

	over := newHeaderLimitConn(&scriptedConn{data: []byte(strings.Repeat("h", 65) + "\r\n\r\n")}, 64)
	_, err := over.Read(make([]byte, 128))
	assert.ErrorIs(t, err, ErrUpgradeHeadersTooLarge)
}

// scriptedConn is a net.Conn whose reads return data in whatever chunk size the caller
// asks for.
type scriptedConn struct {
	netConnStub
	data []byte
}

func (c *scriptedConn) Read(p []byte) (int, error) {
	if len(c.data) == 0 {
		return 0, io.EOF
	}

	n := copy(p, c.data)
	c.data = c.data[n:]

	return n, nil
}

type netConnStub struct{}

func (netConnStub) Read([]byte) (int, error)         { return 0, io.EOF }
func (netConnStub) Write(p []byte) (int, error)      { return len(p), nil }
func (netConnStub) Close() error                     { return nil }
func (netConnStub) LocalAddr() net.Addr              { return nil }
func (netConnStub) RemoteAddr() net.Addr             { return nil }
func (netConnStub) SetDeadline(time.Time) error      { return nil }
func (netConnStub) SetReadDeadline(time.Time) error  { return nil }
func (netConnStub) SetWriteDeadline(time.Time) error { return nil }

// The upgrade signature is bound to the endpoint it was made for: the same headers verify
// under that endpoint's id and under no other, so a signing secret shared by two endpoints
// does not let one present the other's upgrade.
func TestSignedUpgradeHeadersAreBoundToTheEndpoint(t *testing.T) {
	const secret = "upgrade-secret"

	now := time.Unix(1_700_000_000, 0)
	h, err := signedUpgradeHeaders(secret, "endpoint-a", "task-1", 2, now, "nonce-1")
	require.NoError(t, err)

	payloadFor := func(endpointId string) string {
		return contract.UpgradeSigningPayload(
			endpointId,
			h.Get(contract.TimestampHeader),
			h.Get(contract.NonceHeader),
			h.Get(contract.TaskIdHeader),
			h.Get(contract.InvocationHeader),
		)
	}

	assert.Equal(t, "endpoint-a", h.Get(contract.EndpointIdHeader))
	assert.True(t, signature.Verify(payloadFor("endpoint-a"), secret, h.Get(contract.SignatureHeader)))
	assert.False(t, signature.Verify(payloadFor("endpoint-b"), secret, h.Get(contract.SignatureHeader)), "the signature does not verify under another endpoint id")
}
