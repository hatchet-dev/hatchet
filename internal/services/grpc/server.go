package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/hatchet-dev/hatchet/internal/services/admin"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts/contractsconnect"
	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	dispatcherconnect "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts/contractsconnect"
	"github.com/hatchet-dev/hatchet/internal/services/grpc/middleware"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	eventsconnect "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts/contractsconnect"
	"github.com/hatchet-dev/hatchet/internal/services/otelcol"
	"github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

// traceServiceExportProcedure is the standard OTLP TraceService method, served for OTel SDK
// compatibility.
const traceServiceExportProcedure = "/opentelemetry.proto.collector.trace.v1.TraceService/Export"

const (
	// serverPingInterval pings the client if the connection has been idle for this long, to
	// ensure the connection is still active.
	serverPingInterval = 30 * time.Second
	// serverPingTimeout closes the connection if a ping is not answered in time.
	serverPingTimeout = 20 * time.Second
	// readHeaderTimeout bounds how long a client may take to send request headers. Bodies are
	// unbounded in time because streams are long-lived.
	readHeaderTimeout = 30 * time.Second
)

type Server struct {
	l           *zerolog.Logger
	a           hatcheterrors.Alerter
	analytics   analytics.Analytics
	port        int
	bindAddress string

	config        *server.ServerConfig
	ingestor      ingestor.Ingestor
	dispatcher    dispatcher.Dispatcher
	dispatcherv1  v1connect.V1DispatcherHandler
	admin         admin.AdminService
	adminv1       adminv1.AdminService
	otelCollector otelcol.OTelCollector
	tls           *tls.Config
	insecure      bool

	shutdownTimeout time.Duration
}

type ServerOpt func(*ServerOpts)

type ServerOpts struct {
	config        *server.ServerConfig
	l             *zerolog.Logger
	a             hatcheterrors.Alerter
	analytics     analytics.Analytics
	port          int
	bindAddress   string
	ingestor      ingestor.Ingestor
	dispatcher    dispatcher.Dispatcher
	dispatcherv1  v1connect.V1DispatcherHandler
	admin         admin.AdminService
	adminv1       adminv1.AdminService
	otelCollector otelcol.OTelCollector
	tls           *tls.Config
	insecure      bool

	shutdownTimeout time.Duration
}

func defaultServerOpts() *ServerOpts {
	logger := logger.NewDefaultLogger("grpc")
	a := hatcheterrors.NoOpAlerter{}
	analytics := analytics.NoOpAnalytics{}
	return &ServerOpts{
		l:           &logger,
		a:           a,
		analytics:   analytics,
		port:        7070,
		bindAddress: "127.0.0.1",
		insecure:    false,

		shutdownTimeout: 10 * time.Second,
	}
}

func WithLogger(l *zerolog.Logger) ServerOpt {
	return func(opts *ServerOpts) {
		opts.l = l
	}
}

func WithAlerter(a hatcheterrors.Alerter) ServerOpt {
	return func(opts *ServerOpts) {
		opts.a = a
	}
}

func WithAnalytics(a analytics.Analytics) ServerOpt {
	return func(opts *ServerOpts) {
		opts.analytics = a
	}
}

func WithBindAddress(bindAddress string) ServerOpt {
	return func(opts *ServerOpts) {
		opts.bindAddress = bindAddress
	}
}

func WithPort(port int) ServerOpt {
	return func(opts *ServerOpts) {
		opts.port = port
	}
}

func WithIngestor(i ingestor.Ingestor) ServerOpt {
	return func(opts *ServerOpts) {
		opts.ingestor = i
	}
}

func WithConfig(config *server.ServerConfig) ServerOpt {
	return func(opts *ServerOpts) {
		opts.config = config
	}
}

func WithTLSConfig(tls *tls.Config) ServerOpt {
	return func(opts *ServerOpts) {
		opts.tls = tls
	}
}

func WithInsecure() ServerOpt {
	return func(opts *ServerOpts) {
		opts.insecure = true
	}
}

// WithShutdownTimeout bounds how long the server waits for in-flight RPCs and
// streams to drain on graceful shutdown before forcing a hard stop. Non-positive
// values are ignored.
func WithShutdownTimeout(timeout time.Duration) ServerOpt {
	return func(opts *ServerOpts) {
		if timeout > 0 {
			opts.shutdownTimeout = timeout
		}
	}
}

func WithDispatcher(d dispatcher.Dispatcher) ServerOpt {
	return func(opts *ServerOpts) {
		opts.dispatcher = d
	}
}

func WithDispatcherV1(d v1connect.V1DispatcherHandler) ServerOpt {
	return func(opts *ServerOpts) {
		opts.dispatcherv1 = d
	}
}

func WithAdmin(a admin.AdminService) ServerOpt {
	return func(opts *ServerOpts) {
		opts.admin = a
	}
}

func WithAdminV1(a adminv1.AdminService) ServerOpt {
	return func(opts *ServerOpts) {
		opts.adminv1 = a
	}
}

func WithOTelCollector(oc otelcol.OTelCollector) ServerOpt {
	return func(opts *ServerOpts) {
		opts.otelCollector = oc
	}
}

func NewServer(fs ...ServerOpt) (*Server, error) {
	opts := defaultServerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.config == nil {
		return nil, fmt.Errorf("config is required. use WithConfig")
	}

	if opts.tls == nil {
		return nil, fmt.Errorf("tls config is required. use WithTLSConfig")
	}

	newLogger := opts.l.With().Str("service", "grpc").Logger()
	opts.l = &newLogger

	return &Server{
		l:             opts.l,
		a:             opts.a,
		analytics:     opts.analytics,
		config:        opts.config,
		port:          opts.port,
		bindAddress:   opts.bindAddress,
		ingestor:      opts.ingestor,
		dispatcher:    opts.dispatcher,
		dispatcherv1:  opts.dispatcherv1,
		admin:         opts.admin,
		adminv1:       opts.adminv1,
		otelCollector: opts.otelCollector,
		tls:           opts.tls,
		insecure:      opts.insecure,

		shutdownTimeout: opts.shutdownTimeout,
	}, nil
}

func (s *Server) Start() (func() error, error) {
	return s.startGRPC()
}

// handlerOptions returns the options shared by every handler. Interceptors run in the order
// listed, outermost first.
func (s *Server) handlerOptions() ([]connect.HandlerOption, error) {
	otelInterceptor, err := otelconnect.NewInterceptor(
		// continue the caller's trace instead of starting a linked root span
		otelconnect.WithTrustRemote(),
		otelconnect.WithoutServerPeerAttributes(),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create otel interceptor: %w", err)
	}

	limit := s.config.Runtime.GRPCRateLimit
	if limit == 0 {
		limit = 1000
	}
	burst := limit
	limiter := middleware.NewHatchetRateLimiter(rate.Limit(limit), int(burst), s.l)

	interceptors := []connect.Interceptor{
		otelInterceptor,
		middleware.LoggingInterceptor(s.l),
		middleware.NewAuthN(s.config).Interceptor(),
		limiter.Interceptor(),
		middleware.NewErrorInterceptor(s.a, s.l).Interceptor(),
		middleware.RecoveryInterceptor(s.a, s.l),
	}

	// unary-only interceptors registered by extensions run closest to the handler
	for _, interceptor := range s.config.GRPCInterceptors {
		interceptors = append(interceptors, interceptor)
	}

	return []connect.HandlerOption{
		connect.WithInterceptors(interceptors...),
		connect.WithReadMaxBytes(s.config.Runtime.GRPCMaxMsgSize),
		connect.WithSendMaxBytes(s.config.Runtime.GRPCMaxMsgSize),
	}, nil
}

func (s *Server) handler() (http.Handler, error) {
	opts, err := s.handlerOptions()

	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	if s.ingestor != nil {
		mux.Handle(eventsconnect.NewEventsServiceHandler(s.ingestor, opts...))
	}

	if s.dispatcher != nil {
		mux.Handle(dispatcherconnect.NewDispatcherHandler(s.dispatcher, opts...))
	}

	if s.dispatcherv1 != nil {
		mux.Handle(v1connect.NewV1DispatcherHandler(s.dispatcherv1, opts...))
	}

	if s.admin != nil {
		mux.Handle(contractsconnect.NewWorkflowServiceHandler(s.admin, opts...))
	}

	if s.adminv1 != nil {
		mux.Handle(v1connect.NewAdminServiceHandler(s.adminv1, opts...))
	}

	if s.otelCollector != nil {
		// Register as the standard OTLP TraceService for OTEL SDK compatibility
		mux.Handle(traceServiceExportProcedure, connect.NewUnaryHandlerSimple(
			traceServiceExportProcedure,
			func(ctx context.Context, req *collectortracev1.ExportTraceServiceRequest) (*collectortracev1.ExportTraceServiceResponse, error) {
				return s.otelCollector.Export(ctx, req)
			},
			opts...,
		))
	}

	return matchRequestCompression(mux), nil
}

// matchRequestCompression keeps the response compression rule gRPC clients have always had
// from this server: a response is compressed only when its request was. gRPC clients advertise
// gzip on every call whether or not they were configured to compress, and connect compresses
// whenever the client advertises support, which would put every assigned action through gzip.
func matchRequestCompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			if encoding := r.Header.Get("Grpc-Encoding"); encoding == "" || encoding == "identity" {
				r.Header.Del("Grpc-Accept-Encoding")
			} else {
				r.Header.Set("Grpc-Accept-Encoding", encoding)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) startGRPC() (func() error, error) {
	s.l.Debug().Msgf("starting grpc server on %s:%d", s.bindAddress, s.port)
	s.l.Info().Msg("gzip compression enabled for gRPC server")

	handler, err := s.handler()

	if err != nil {
		return nil, err
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.bindAddress, s.port))

	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	// gRPC clients need HTTP/2: negotiated over TLS, or with prior knowledge (h2c) when the
	// server is insecure. HTTP/1.1 stays on for Connect and gRPC-Web unary calls.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)

	if s.insecure {
		protocols.SetUnencryptedHTTP2(true)
	} else {
		protocols.SetHTTP2(true)
	}

	httpServer := &http.Server{
		Handler:           handler,
		Protocols:         protocols,
		ReadHeaderTimeout: readHeaderTimeout,
		// streams are long-lived, so there is no read, write or idle timeout; dead connections
		// are found by the pings below
		HTTP2: &http.HTTP2Config{
			// gRPC clients multiplex every call and stream of a worker over one connection
			MaxConcurrentStreams: math.MaxInt32,
			// the connection window has to be at least as large as the stream window for a
			// single upload to use all of it
			MaxReceiveBufferPerStream:     int(s.config.Runtime.GRPCStaticStreamWindowSize),
			MaxReceiveBufferPerConnection: int(s.config.Runtime.GRPCStaticStreamWindowSize),
			SendPingTimeout:               serverPingInterval,
			PingTimeout:                   serverPingTimeout,
		},
	}

	if !s.insecure {
		httpServer.TLSConfig = s.tls.Clone()
	}

	go func() {
		var err error

		if s.insecure {
			err = httpServer.Serve(lis)
		} else {
			err = httpServer.ServeTLS(lis, "", "")
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(fmt.Errorf("failed to serve: %w", err))
		}
	}()

	cleanup := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			s.l.Error().Msgf("grpc server did not drain within %s, forcing a hard stop", s.shutdownTimeout)

			return httpServer.Close()
		}

		s.l.Debug().Msg("grpc server stopped gracefully")

		return nil
	}

	return cleanup, nil
}
