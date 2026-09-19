package grpc

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
)

const (
	// defaultMaxMsgSize applies when the configured maximum is not positive, which connect
	// would otherwise read as no limit at all.
	defaultMaxMsgSize = 4 * 1024 * 1024
	// maxHeaderBytes matches the header list size google.golang.org/grpc servers accept.
	maxHeaderBytes = 16 * 1024 * 1024
	// rpcReadSlack is how long after a call's own timeout its request body stops being read.
	// The handler's context expires first, so the caller is still told the deadline passed.
	rpcReadSlack = 250 * time.Millisecond
	// rpcWriteSlack is how long after a call's own timeout its response stops being written.
	// It is longer than rpcReadSlack so the deadline status itself can still be sent.
	rpcWriteSlack = 2 * time.Second
	// maxRPCTimeout is the longest timeout that gets a transport deadline.
	maxRPCTimeout = 365 * 24 * time.Hour
	// defaultHTTP1BodyReadTimeout is how long an HTTP/1.1 caller has to send its request body.
	defaultHTTP1BodyReadTimeout = 30 * time.Second
	// defaultHTTP1UnaryWriteTimeout is how long an HTTP/1.1 caller has to take a unary response
	// once the server starts writing it.
	defaultHTTP1UnaryWriteTimeout = 30 * time.Second
)

// isGRPC reports whether the request speaks the gRPC protocol proper (not gRPC-Web, which
// carries its status in the body).
func isGRPC(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")

	return contentType == "application/grpc" || strings.HasPrefix(contentType, "application/grpc+")
}

// withStreamAbort gives every handler a way to interrupt a response write that is blocked on a
// peer which has stopped reading. rpcstream.Sender uses it so that a handler can always return.
func withStreamAbort(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)

		ctx := rpcstream.WithAbort(r.Context(), func() {
			// a deadline in the past resets the stream, which fails the blocked write
			_ = rc.SetWriteDeadline(time.Now())
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// transportDeadlines bounds the I/O of a call with deadlines on its connection or stream, which
// is the only thing that interrupts a blocked request body read or response write: a context
// cannot.
type transportDeadlines struct {
	routes                 grpcRoutes
	http1BodyReadTimeout   time.Duration
	http1UnaryWriteTimeout time.Duration
}

// enforce applies two rules.
//
// A call's own timeout (see rpcTimeout) bounds its reads and writes, on every protocol. connect
// turns the timeout into a context deadline, but a caller that stops sending would otherwise
// keep the call open past it. Calls without a timeout keep no deadline: streams are long-lived.
//
// HTTP/1.1 calls get two more bounds, because nothing else reclaims them: the pings that find a
// dead HTTP/2 connection do not exist there. Every procedure reachable over HTTP/1.1 sends
// exactly one request message (connect refuses bidirectional streams on it), so the request
// body must arrive within http1BodyReadTimeout, and a unary response must be taken within
// http1UnaryWriteTimeout of its first byte. Server streams stay unbounded on the write side. An
// idle keep-alive connection is not bounded, which is no different from an idle HTTP/2
// connection: http.Server.IdleTimeout would also close HTTP/2 connections that have no open
// stream, which google.golang.org/grpc never did to its callers.
func (d transportDeadlines) enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		now := time.Now()

		var readDeadline, writeDeadline time.Time

		if timeout, ok := rpcTimeout(r.Header); ok {
			readDeadline = now.Add(timeout + rpcReadSlack)
			writeDeadline = now.Add(timeout + rpcWriteSlack)
		}

		if r.ProtoMajor == 1 {
			// net/http lifts the read deadline itself once the body has been read, when it
			// starts watching the connection for the caller going away, so the deadline only
			// covers the body. A request without a body is already being watched, and an
			// expired deadline there would read as the caller going away and cancel the call.
			if r.ContentLength != 0 {
				if bodyDeadline := now.Add(d.http1BodyReadTimeout); readDeadline.IsZero() || bodyDeadline.Before(readDeadline) {
					readDeadline = bodyDeadline
				}
			} else {
				readDeadline = time.Time{}
			}

			if !d.routes.isServerStreaming(r.URL.Path) {
				w = &deadlineOnWriteResponse{ResponseWriter: w, arm: func() {
					if deadline := time.Now().Add(d.http1UnaryWriteTimeout); writeDeadline.IsZero() || deadline.Before(writeDeadline) {
						_ = rc.SetWriteDeadline(deadline)
					}
				}}
			}
		}

		if !readDeadline.IsZero() {
			_ = rc.SetReadDeadline(readDeadline)
		}

		if !writeDeadline.IsZero() {
			_ = rc.SetWriteDeadline(writeDeadline)
		}

		next.ServeHTTP(w, r)
	})
}

// deadlineOnWriteResponse starts a write deadline when the response starts, so that the time
// the handler takes does not count against the caller.
type deadlineOnWriteResponse struct {
	http.ResponseWriter
	arm  func()
	once sync.Once
}

func (w *deadlineOnWriteResponse) WriteHeader(statusCode int) {
	w.once.Do(w.arm)
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *deadlineOnWriteResponse) Write(p []byte) (int, error) {
	w.once.Do(w.arm)

	return w.ResponseWriter.Write(p)
}

// Unwrap lets http.ResponseController reach the connection.
func (w *deadlineOnWriteResponse) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// rpcTimeout reads the timeout a caller attached to its call: grpc-timeout for gRPC and
// gRPC-Web, Connect-Timeout-Ms for Connect. Malformed values are left for connect to reject.
func rpcTimeout(header http.Header) (time.Duration, bool) {
	if v := header.Get("Grpc-Timeout"); v != "" {
		if len(v) < 2 || len(v) > 9 {
			return 0, false
		}

		var unit time.Duration

		switch v[len(v)-1] {
		case 'H':
			unit = time.Hour
		case 'M':
			unit = time.Minute
		case 'S':
			unit = time.Second
		case 'm':
			unit = time.Millisecond
		case 'u':
			unit = time.Microsecond
		case 'n':
			unit = time.Nanosecond
		default:
			return 0, false
		}

		n, err := strconv.ParseInt(v[:len(v)-1], 10, 64)

		// eight digits of hours do not fit in a Duration; a timeout that long needs no deadline
		if err != nil || n < 0 || n > int64(maxRPCTimeout/unit) {
			return 0, false
		}

		return time.Duration(n) * unit, true
	}

	if v := header.Get("Connect-Timeout-Ms"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)

		if err != nil || n < 0 || n > int64(maxRPCTimeout/time.Millisecond) {
			return 0, false
		}

		return time.Duration(n) * time.Millisecond, true
	}

	return 0, false
}

// grpcRoutes is the set of mounted procedures, by service and method name. The value is whether
// the method streams its response.
type grpcRoutes map[string]map[string]bool

func (routes grpcRoutes) addService(service protoreflect.ServiceDescriptor) {
	methods := service.Methods()

	for i := range methods.Len() {
		method := methods.Get(i)

		routes.add(string(service.FullName()), string(method.Name()), method.IsStreamingServer())
	}
}

func (routes grpcRoutes) add(service, method string, serverStreaming bool) {
	if routes[service] == nil {
		routes[service] = map[string]bool{}
	}

	routes[service][method] = serverStreaming
}

// splitProcedure splits /Service/Method. ok is false for a path without a method.
func splitProcedure(path string) (service, method string, ok bool) {
	procedure := strings.TrimPrefix(path, "/")
	pos := strings.LastIndex(procedure, "/")

	if pos == -1 {
		return "", "", false
	}

	return procedure[:pos], procedure[pos+1:], true
}

func (routes grpcRoutes) isServerStreaming(path string) bool {
	service, method, _ := splitProcedure(path)

	return routes[service][method]
}

// unimplemented answers gRPC calls to procedures that are not mounted the way
// google.golang.org/grpc servers do: a trailers-only Unimplemented status naming the service or
// method, before authentication. Without it the caller gets a bare HTTP 404, which gRPC clients
// read as Unimplemented but without the message. Other protocols keep the 404.
func (routes grpcRoutes) unimplemented(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isGRPC(r) {
			next.ServeHTTP(w, r)
			return
		}

		var message string

		if service, method, ok := splitProcedure(r.URL.Path); !ok {
			message = fmt.Sprintf("malformed method name: %q", r.URL.Path)
		} else if methods, knownService := routes[service]; !knownService {
			message = fmt.Sprintf("unknown service %v", service)
		} else if _, knownMethod := methods[method]; !knownMethod {
			message = fmt.Sprintf("unknown method %v for service %v", method, service)
		}

		if message == "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", strconv.Itoa(int(connect.CodeUnimplemented)))
		w.Header().Set("Grpc-Message", grpcPercentEncode(message))
		w.WriteHeader(http.StatusOK)
	})
}

// grpcPercentEncode encodes a grpc-message value: bytes outside printable ASCII, and the percent
// sign, are percent-encoded.
func grpcPercentEncode(msg string) string {
	const upperHex = "0123456789ABCDEF"

	var out strings.Builder

	for i := range len(msg) {
		if c := msg[i]; c >= ' ' && c <= '~' && c != '%' {
			out.WriteByte(c)
		} else {
			out.WriteByte('%')
			out.WriteByte(upperHex[c>>4])
			out.WriteByte(upperHex[c&15])
		}
	}

	return out.String()
}

var errDecompressedTooLarge = errors.New("decompressed message is larger than the configured maximum")

// boundedGzipReader is a gzip decompressor that inflates at most limit+1 bytes per message:
// enough for connect to see that a message is over the limit and reject it, as it does with the
// standard reader, but no more. connect drains an over-limit message to measure it, and with the
// standard reader that drain inflates whatever a small compressed body expands to.
type boundedGzipReader struct {
	gzip.Reader
	limit     int64
	remaining int64
}

func newBoundedGzipReader(limit int) *boundedGzipReader {
	return &boundedGzipReader{limit: int64(limit)}
}

func (b *boundedGzipReader) Reset(r io.Reader) error {
	b.remaining = b.limit + 1

	return b.Reader.Reset(r)
}

func (b *boundedGzipReader) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, errDecompressedTooLarge
	}

	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}

	n, err := b.Reader.Read(p)
	b.remaining -= int64(n)

	return n, err
}

// withBoundedGzip replaces connect's default gzip support with a decompressor bounded by the
// read limit. The compressor is the one connect uses by default.
func withBoundedGzip(readMaxBytes int) connect.HandlerOption {
	return connect.WithCompression(
		"gzip",
		func() connect.Decompressor { return newBoundedGzipReader(readMaxBytes) },
		func() connect.Compressor { return gzip.NewWriter(io.Discard) },
	)
}
