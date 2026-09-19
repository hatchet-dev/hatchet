package grpc

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// These tests hold the transport behaviour a google.golang.org/grpc server gave callers and that
// net/http does not give by default. Each one came out of the first review of the migration.

// stalledDispatcher sends a message larger than the caller's flow-control window from another
// goroutine, the way the dispatcher does, and then wants to return.
type stalledDispatcher struct {
	fakeDispatcher

	sending     chan struct{}
	release     chan struct{}
	handlerDone chan struct{}
	sendResult  chan error
}

func newStalledDispatcher() *stalledDispatcher {
	return &stalledDispatcher{
		sending:     make(chan struct{}),
		release:     make(chan struct{}),
		handlerDone: make(chan struct{}),
		sendResult:  make(chan error, 1),
	}
}

const stalledPayloadSize = 1024 * 1024

func (s *stalledDispatcher) ListenV2(ctx context.Context, _ *dispatchercontracts.WorkerListenRequest, stream *connect.ServerStream[dispatchercontracts.AssignedAction]) error {
	defer close(s.handlerDone)

	sender := rpcstream.NewSender[dispatchercontracts.AssignedAction](ctx, stream)
	defer sender.Close()

	go func() {
		close(s.sending)
		s.sendResult <- sender.Send(&dispatchercontracts.AssignedAction{ActionPayload: strings.Repeat("x", stalledPayloadSize)})
	}()

	// the handler finishes on its own while the send is still blocked
	<-s.release

	return nil
}

func (s *stalledDispatcher) SubscribeToWorkflowRuns(ctx context.Context, stream *connect.BidiStream[dispatchercontracts.SubscribeToWorkflowRunsRequest, dispatchercontracts.WorkflowRunEvent]) error {
	defer close(s.handlerDone)

	sender := rpcstream.NewSender[dispatchercontracts.WorkflowRunEvent](ctx, stream)
	defer sender.Close()

	go func() {
		close(s.sending)
		s.sendResult <- sender.Send(&dispatchercontracts.WorkflowRunEvent{WorkflowRunId: strings.Repeat("x", stalledPayloadSize)})
	}()

	// the handler finishes when the caller half-closes
	for {
		if _, err := stream.Receive(); err != nil {
			return nil
		}
	}
}

// startServerWith starts an insecure server around disp with the given message size limit.
func startServerWith(t *testing.T, disp dispatcher.Dispatcher, maxMsg int) string {
	t.Helper()

	l := zerolog.Nop()

	sc := &server.ServerConfig{Layer: &database.Layer{V1: fakeRepository{}}}
	sc.Logger = &l
	sc.Auth.JWTManager = testJWTManager{}
	sc.Runtime.GRPCMaxMsgSize = maxMsg
	sc.Runtime.GRPCStaticStreamWindowSize = 10 * 1024 * 1024

	port := freePort(t)

	s, err := NewServer(
		WithConfig(sc),
		WithLogger(&l),
		WithDispatcher(disp),
		WithPort(port),
		WithBindAddress("127.0.0.1"),
		WithInsecure(),
		WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}),
		WithShutdownTimeout(time.Second),
	)
	require.NoError(t, err)

	cleanup, err := s.Start()
	require.NoError(t, err)

	t.Cleanup(func() { _ = cleanup() })

	return fmt.Sprintf("127.0.0.1:%d", port)
}

// A caller that keeps its stream open but stops reading leaves a send blocked on flow control,
// with connection pings still answered. The handler must still be able to return.
func TestHandlerReturnsWhileSendIsBlockedOnAStalledPeer(t *testing.T) {
	for name, finish := range map[string]func(t *testing.T, disp *stalledDispatcher, client dispatchercontracts.DispatcherClient, ctx context.Context){
		"handler finishes on its own": func(t *testing.T, disp *stalledDispatcher, client dispatchercontracts.DispatcherClient, ctx context.Context) {
			stream, err := client.ListenV2(ctx, &dispatchercontracts.WorkerListenRequest{WorkerId: "w"})
			require.NoError(t, err)

			_, err = stream.Header()
			require.NoError(t, err)

			requireSendBlocked(t, disp)
			close(disp.release)
		},
		"caller half-closes": func(t *testing.T, disp *stalledDispatcher, client dispatchercontracts.DispatcherClient, ctx context.Context) {
			stream, err := client.SubscribeToWorkflowRuns(ctx)
			require.NoError(t, err)

			_, err = stream.Header()
			require.NoError(t, err)

			requireSendBlocked(t, disp)
			require.NoError(t, stream.CloseSend())
		},
	} {
		t.Run(name, func(t *testing.T) {
			disp := newStalledDispatcher()
			addr := startServerWith(t, disp, 8*1024*1024)

			tr := transports()[0]
			env := &testEnv{addr: addr}
			// a window far smaller than the message, and the test never reads from the stream
			conn := env.dial(t, tr, nil, false, grpcgo.WithStaticStreamWindowSize(65535))

			ctx, cancel := context.WithCancel(authCtx(t, validToken))
			defer cancel()

			finish(t, disp, dispatchercontracts.NewDispatcherClient(conn), ctx)

			select {
			case <-disp.handlerDone:
			case <-time.After(5 * time.Second):
				t.Fatal("handler could not return while a send was blocked on a peer that stopped reading")
			}

			require.Error(t, <-disp.sendResult, "the blocked send must fail rather than outlive its handler")
		})
	}
}

func requireSendBlocked(t *testing.T, disp *stalledDispatcher) {
	t.Helper()

	<-disp.sending

	select {
	case err := <-disp.sendResult:
		t.Fatalf("send did not block: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
}

func h2cClient(t *testing.T) *http.Client {
	t.Helper()

	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)

	tr := &http.Transport{Protocols: protocols}
	t.Cleanup(tr.CloseIdleConnections)

	return &http.Client{Transport: tr}
}

func grpcFrame(t *testing.T, msg proto.Message) []byte {
	t.Helper()

	data, err := proto.Marshal(msg)
	require.NoError(t, err)

	frame := make([]byte, 5, len(data)+5)
	binary.BigEndian.PutUint32(frame[1:], uint32(len(data))) // nolint:gosec

	return append(frame, data...)
}

func rawGRPCRequest(t *testing.T, ctx context.Context, addr, path string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+addr+path, body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("Te", "trailers")
	req.Header.Set("Authorization", "Bearer "+validToken)

	return req
}

func grpcStatusOf(res *http.Response) (code, message string) {
	if code := res.Header.Get("Grpc-Status"); code != "" {
		return code, res.Header.Get("Grpc-Message")
	}

	return res.Trailer.Get("Grpc-Status"), res.Trailer.Get("Grpc-Message")
}

// A caller that declares a timeout and then stops sending its request must be answered by the
// server once the timeout passes, not held until the caller gives up.
func TestRPCTimeoutInterruptsAStalledRequestBody(t *testing.T) {
	env := startTestServer(t, transports()[0], nil, 0)

	body, writer := io.Pipe()
	defer writer.Close()

	// far longer than the call's own timeout: the server has to be the one to finish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := rawGRPCRequest(t, ctx, env.addr, "/Dispatcher/Register", body)
	req.Header.Set("Grpc-Timeout", "200m")

	go func() {
		// an incomplete frame: a prefix announcing five bytes, and one byte of them
		_, _ = writer.Write([]byte{0, 0, 0, 0, 5, 10})
	}()

	begin := time.Now()

	res, err := h2cClient(t).Do(req)
	require.NoError(t, err)

	defer res.Body.Close()

	_, _ = io.ReadAll(res.Body)

	code, _ := grpcStatusOf(res)

	assert.Equal(t, "4", code, "deadline exceeded")
	assert.Less(t, time.Since(begin), 3*time.Second)
}

func TestRPCTimeoutHeaderParsing(t *testing.T) {
	for value, want := range map[string]time.Duration{
		"20m": 20 * time.Millisecond,
		"3S":  3 * time.Second,
		"2M":  2 * time.Minute,
		"1H":  time.Hour,
		"5u":  5 * time.Microsecond,
		"7n":  7 * time.Nanosecond,
	} {
		got, ok := rpcTimeout(http.Header{"Grpc-Timeout": {value}})
		assert.True(t, ok, value)
		assert.Equal(t, want, got, value)
	}

	for _, value := range []string{"bad", "m", "10", "-1S", "123456789S", "1.5S", "99999999H"} {
		_, ok := rpcTimeout(http.Header{"Grpc-Timeout": {value}})
		assert.False(t, ok, value)
	}

	got, ok := rpcTimeout(http.Header{"Connect-Timeout-Ms": {"1500"}})
	assert.True(t, ok)
	assert.Equal(t, 1500*time.Millisecond, got)

	_, ok = rpcTimeout(http.Header{})
	assert.False(t, ok)
}

// google.golang.org/grpc servers answer a call to a procedure they do not have with a
// trailers-only Unimplemented status that names it, before authentication.
func TestUnknownProceduresGetAGRPCStatus(t *testing.T) {
	env := startTestServer(t, transports()[0], nil, 0)
	client := h2cClient(t)

	for path, message := range map[string]string{
		"/Dispatcher/NotAMethod": "unknown method NotAMethod for service Dispatcher",
		"/Absent/Method":         "unknown service Absent",
		// mounted on other servers, not on this one
		"/EventsService/Push": "unknown service EventsService",
		"/nomethod":           `malformed method name: "/nomethod"`,
	} {
		req := rawGRPCRequest(t, context.Background(), env.addr, path, bytes.NewReader(grpcFrame(t, &dispatchercontracts.WorkerRegisterRequest{})))
		req.Header.Del("Authorization")

		res, err := client.Do(req)
		require.NoError(t, err, path)

		body, _ := io.ReadAll(res.Body)
		res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode, path)
		assert.Equal(t, "application/grpc", res.Header.Get("Content-Type"), path)
		assert.Equal(t, "12", res.Header.Get("Grpc-Status"), "%s: status belongs in the headers of a trailers-only response", path)
		assert.Equal(t, message, res.Header.Get("Grpc-Message"), path)
		assert.Empty(t, body, path)
	}

	t.Run("grpc-go client sees the message", func(t *testing.T) {
		conn := env.dial(t, transports()[0], nil, false)

		err := conn.Invoke(authCtx(t, validToken), "/Dispatcher/NotAMethod", &dispatchercontracts.GetVersionRequest{}, &dispatchercontracts.GetVersionResponse{})
		assert.Equal(t, codes.Unimplemented, status.Code(err))
		assert.Equal(t, "unknown method NotAMethod for service Dispatcher", status.Convert(err).Message())
	})

	t.Run("mounted procedures are not affected", func(t *testing.T) {
		res, err := client.Do(rawGRPCRequest(t, context.Background(), env.addr, "/Dispatcher/Register", bytes.NewReader(grpcFrame(t, &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"}))))
		require.NoError(t, err)

		_, _ = io.ReadAll(res.Body)
		res.Body.Close()

		code, _ := grpcStatusOf(res)
		assert.Equal(t, "0", code)
	})

	t.Run("other protocols keep the plain 404", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "http://"+env.addr+"/Absent/Method", strings.NewReader("{}"))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		require.NoError(t, err)

		res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		assert.Empty(t, res.Header.Get("Grpc-Status"))
	})
}

func TestGRPCMessageEncoding(t *testing.T) {
	assert.Equal(t, "unknown service Absent", grpcPercentEncode("unknown service Absent"))
	assert.Equal(t, "100%25 %E2%9C%93%0A", grpcPercentEncode("100% ✓\n"))
}

// google.golang.org/grpc servers accept 16 MiB of metadata; net/http stops at 1 MiB unless told
// otherwise.
func TestLargeMetadataIsAccepted(t *testing.T) {
	tr := transports()[0]
	env := startTestServer(t, tr, nil, 0)
	client := dispatchercontracts.NewDispatcherClient(env.dial(t, tr, nil, false))

	// the limit is advertised in the server's settings, which the client has after one call
	_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "warm"})
	require.NoError(t, err)

	res, err := client.Register(
		authCtx(t, validToken, "large-metadata", strings.Repeat("m", 2*1024*1024)),
		&dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"},
	)
	require.NoError(t, err)
	assert.Equal(t, "w", res.WorkerName)
}

// The listener speaks HTTP/2 only, as it did when google.golang.org/grpc served it. HTTP/1.1
// requests would have no ping to find a caller that stops sending its body.
func TestHTTP1IsRefused(t *testing.T) {
	t.Run("insecure", func(t *testing.T) {
		env := startTestServer(t, transports()[0], nil, 0)

		conn, err := net.Dial("tcp", env.addr)
		require.NoError(t, err)

		defer conn.Close()

		_, err = fmt.Fprintf(conn, "POST /Dispatcher/Register HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nAuthorization: Bearer %s\r\nContent-Length: 2\r\n\r\n{}", validToken)
		require.NoError(t, err)

		require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

		res, err := http.ReadResponse(bufio.NewReader(conn), nil)
		// the connection is closed without an answer; a refusal with a status would do as well
		if err == nil {
			defer res.Body.Close()

			assert.GreaterOrEqual(t, res.StatusCode, 400)
		}

		requireHandlerNotReached(t, env)
	})

	t.Run("tls", func(t *testing.T) {
		pki := newTestPKI(t)
		tr := transports()[1]
		env := startTestServer(t, tr, pki, 0)

		cfg := tr.clientTLS(pki, false)
		cfg.NextProtos = []string{"http/1.1"}

		client := &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}, Timeout: 5 * time.Second}

		req, err := http.NewRequest(http.MethodPost, "https://"+env.addr+"/Dispatcher/Register", strings.NewReader("{}"))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+validToken)

		res, err := client.Do(req)
		if err == nil {
			defer res.Body.Close()

			assert.GreaterOrEqual(t, res.StatusCode, 400)
		}

		requireHandlerNotReached(t, env)
	})
}

func requireHandlerNotReached(t *testing.T, env *testEnv) {
	t.Helper()

	env.disp.mu.Lock()
	defer env.disp.mu.Unlock()

	require.Nil(t, env.disp.lastCtx, "an HTTP/1.1 request reached a handler")
}

type countingDecompressor struct {
	connect.Decompressor
	n *atomic.Int64
}

func (c *countingDecompressor) Read(p []byte) (int, error) {
	n, err := c.Decompressor.Read(p)
	c.n.Add(int64(n))

	return n, err
}

// connect drains an over-limit message to report its size. With the standard gzip reader that
// drain inflates everything a small compressed body expands to; the bounded reader stops one
// byte past the limit, as google.golang.org/grpc does.
func TestGzipExpansionStopsAtTheReadLimit(t *testing.T) {
	var inflated atomic.Int64

	opts := []connect.HandlerOption{
		connect.WithReadMaxBytes(maxMsgSize),
		connect.WithCompression(
			"gzip",
			func() connect.Decompressor {
				return &countingDecompressor{Decompressor: newBoundedGzipReader(maxMsgSize), n: &inflated}
			},
			func() connect.Compressor { return gzip.NewWriter(io.Discard) },
		),
	}

	handler := connect.NewUnaryHandlerSimple("/Dispatcher/Register", (&fakeDispatcher{}).Register, opts...)

	post := func(workerName string) *httptest.ResponseRecorder {
		var compressed bytes.Buffer

		z := gzip.NewWriter(&compressed)
		_, err := z.Write([]byte(`{"workerName":"` + workerName + `"}`))
		require.NoError(t, err)
		require.NoError(t, z.Close())

		req := httptest.NewRequest(http.MethodPost, "http://engine/Dispatcher/Register", &compressed)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		return rec
	}

	rec := post(strings.Repeat("x", 8*maxMsgSize))

	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "resource exhausted")
	assert.Equal(t, int64(maxMsgSize+1), inflated.Load())
}

func TestBoundedGzipReaderAcceptsMessagesUpToTheLimit(t *testing.T) {
	for size, wantErr := range map[int]bool{0: false, 1: false, 63: false, 64: false, 65: true, 4096: true} {
		var compressed bytes.Buffer

		z := gzip.NewWriter(&compressed)
		_, err := z.Write(bytes.Repeat([]byte("x"), size))
		require.NoError(t, err)
		require.NoError(t, z.Close())

		reader := newBoundedGzipReader(64)
		require.NoError(t, reader.Reset(&compressed))

		// what connect does: read one byte past the limit, then drain if that byte was there
		got, err := io.ReadAll(io.LimitReader(reader, 65))
		require.NoError(t, err, size)

		if !wantErr {
			assert.Len(t, got, size)
			require.NoError(t, reader.Close())

			continue
		}

		assert.Len(t, got, 65, size)

		_, err = io.Copy(io.Discard, reader)
		assert.ErrorIs(t, err, errDecompressedTooLarge, size)
	}
}

func TestBoundedGzipReaderIsReusable(t *testing.T) {
	reader := newBoundedGzipReader(8)

	for range 3 {
		var compressed bytes.Buffer

		z := gzip.NewWriter(&compressed)
		_, _ = z.Write([]byte("12345678"))
		require.NoError(t, z.Close())

		require.NoError(t, reader.Reset(&compressed))

		got, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "12345678", string(got))
		require.NoError(t, reader.Close())

		// what connect's pool does before putting a decompressor back
		_ = reader.Reset(http.NoBody)
	}
}

// A maximum message size of zero or less must not remove the limit.
func TestNonPositiveMaxMessageSizeKeepsTheDefaultLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			addr := startServerWith(t, &fakeDispatcher{}, limit)
			env := &testEnv{addr: addr}

			client := dispatchercontracts.NewDispatcherClient(env.dial(t, transports()[0], nil, false,
				grpcgo.WithDefaultCallOptions(grpcgo.MaxCallSendMsgSize(16*1024*1024), grpcgo.MaxCallRecvMsgSize(16*1024*1024))))

			_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: strings.Repeat("w", 1024)})
			require.NoError(t, err)

			_, err = client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: strings.Repeat("w", defaultMaxMsgSize+1)})
			assert.Equal(t, codes.ResourceExhausted, status.Code(err))
		})
	}
}
