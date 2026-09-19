package grpc

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// enforceRPCTimeout ties a call's own timeout to its transport. connect turns the timeout into
// a context deadline, but a context cannot interrupt a request body read or a response write, so
// a caller that stops sending would keep the call open past its deadline. Calls without a
// timeout keep no deadline: streams are long-lived.
func enforceRPCTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if timeout, ok := rpcTimeout(r.Header); ok {
			rc := http.NewResponseController(w)
			now := time.Now()

			_ = rc.SetReadDeadline(now.Add(timeout + rpcReadSlack))
			_ = rc.SetWriteDeadline(now.Add(timeout + rpcWriteSlack))
		}

		next.ServeHTTP(w, r)
	})
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

// grpcRoutes is the set of mounted procedures, by service and method name.
type grpcRoutes map[string]map[string]struct{}

func (routes grpcRoutes) addService(service protoreflect.ServiceDescriptor) {
	methods := service.Methods()

	for i := range methods.Len() {
		routes.add(string(service.FullName()), string(methods.Get(i).Name()))
	}
}

func (routes grpcRoutes) add(service, method string) {
	if routes[service] == nil {
		routes[service] = map[string]struct{}{}
	}

	routes[service][method] = struct{}{}
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

		procedure := strings.TrimPrefix(r.URL.Path, "/")
		pos := strings.LastIndex(procedure, "/")

		var message string

		if pos == -1 {
			message = fmt.Sprintf("malformed method name: %q", r.URL.Path)
		} else {
			service, method := procedure[:pos], procedure[pos+1:]

			if methods, knownService := routes[service]; !knownService {
				message = fmt.Sprintf("unknown service %v", service)
			} else if _, knownMethod := methods[method]; !knownMethod {
				message = fmt.Sprintf("unknown method %v for service %v", method, service)
			}
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
