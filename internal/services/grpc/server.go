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
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/hatchet-dev/hatchet/internal/services/admin"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts/contractsconnect"
	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	dispatcherconnect "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts/contractsconnect"
	"github.com/hatchet-dev/hatchet/internal/services/grpc/middleware"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	eventsconnect "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts/contractsconnect"
	"github.com/hatchet-dev/hatchet/internal/services/otelcol"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

// traceServiceExportProcedure is the standard OTLP TraceService method, served for OTel SDK
// compatibility.
const (
	traceServiceName            = "opentelemetry.proto.collector.trace.v1.TraceService"
	traceServiceExportProcedure = "/" + traceServiceName + "/Export"
)

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
	operatorSvc   v1connect.OperatorServiceHandler
	otelCollector otelcol.OTelCollector
	tls           *tls.Config
	insecure      bool

	shutdownTimeout time.Duration

	http1BodyReadTimeout   time.Duration
	http1UnaryWriteTimeout time.Duration
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
	operatorSvc   v1connect.OperatorServiceHandler
	otelCollector otelcol.OTelCollector
	tls           *tls.Config
	insecure      bool

	shutdownTimeout time.Duration

	http1BodyReadTimeout   time.Duration
	http1UnaryWriteTimeout time.Duration
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

		http1BodyReadTimeout:   defaultHTTP1BodyReadTimeout,
		http1UnaryWriteTimeout: defaultHTTP1UnaryWriteTimeout,
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

// WithOperatorService registers the v1.OperatorService for out-of-process operators. When it is
// not set the service is not registered and callers receive Unimplemented.
func WithOperatorService(o v1connect.OperatorServiceHandler) ServerOpt {
	return func(opts *ServerOpts) {
		opts.operatorSvc = o
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

	// the engine always serves the whole API; only the OTel collector is optional
	switch {
	case opts.ingestor == nil:
		return nil, fmt.Errorf("ingestor is required. use WithIngestor")
	case opts.dispatcher == nil:
		return nil, fmt.Errorf("dispatcher is required. use WithDispatcher")
	case opts.dispatcherv1 == nil:
		return nil, fmt.Errorf("v1 dispatcher is required. use WithDispatcherV1")
	case opts.admin == nil:
		return nil, fmt.Errorf("admin service is required. use WithAdmin")
	case opts.adminv1 == nil:
		return nil, fmt.Errorf("v1 admin service is required. use WithAdminV1")
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
		operatorSvc:   opts.operatorSvc,
		otelCollector: opts.otelCollector,
		tls:           opts.tls,
		insecure:      opts.insecure,

		shutdownTimeout: opts.shutdownTimeout,

		http1BodyReadTimeout:   opts.http1BodyReadTimeout,
		http1UnaryWriteTimeout: opts.http1UnaryWriteTimeout,
	}, nil
}

func (s *Server) Start() (func() error, error) {
	return s.startGRPC()
}

// handlerOptions returns the options shared by every handler. Interceptors run in the order
// listed, outermost first.
func (s *Server) handlerOptions() ([]connect.HandlerOption, error) {
	limit := s.config.Runtime.GRPCRateLimit
	if limit == 0 {
		limit = 1000
	}
	burst := limit
	limiter := middleware.NewHatchetRateLimiter(rate.Limit(limit), int(burst), s.l)

	interceptors := []connect.Interceptor{
		middleware.NewTelemetryInterceptor(),
		middleware.LoggingInterceptor(s.l),
		middleware.NewErrorInterceptor(s.a, s.l).Interceptor(),
		middleware.RecoveryInterceptor(s.a, s.l),
	}

	// unary-only interceptors registered by extensions run closest to the handler
	for _, interceptor := range s.config.GRPCInterceptors {
		interceptors = append(interceptors, interceptor)
	}

	maxMsgSize := s.config.Runtime.GRPCMaxMsgSize
	if maxMsgSize <= 0 {
		maxMsgSize = defaultMaxMsgSize
	}

	return []connect.HandlerOption{
		// authentication and rate limits run on the request headers, before any body is read
		connect.WithRequestGate(middleware.NewRequestGate(middleware.NewAuthN(s.config), limiter, s.l)),
		connect.WithInterceptors(interceptors...),
		connect.WithReadMaxBytes(maxMsgSize),
		connect.WithSendMaxBytes(maxMsgSize),
		withBoundedGzip(maxMsgSize),
	}, nil
}

func (s *Server) handler() (http.Handler, error) {
	opts, err := s.handlerOptions()

	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	routes := grpcRoutes{}

	routes.addService(eventcontracts.File_events_proto.Services().ByName("EventsService"))
	mux.Handle(eventsconnect.NewEventsServiceHandler(s.ingestor, opts...))

	routes.addService(dispatchercontracts.File_dispatcher_proto.Services().ByName("Dispatcher"))
	mux.Handle(dispatcherconnect.NewDispatcherHandler(s.dispatcher, opts...))

	routes.addService(v1contracts.File_v1_dispatcher_proto.Services().ByName("V1Dispatcher"))
	mux.Handle(v1connect.NewV1DispatcherHandler(s.dispatcherv1, opts...))

	routes.addService(admincontracts.File_workflows_proto.Services().ByName("WorkflowService"))
	mux.Handle(contractsconnect.NewWorkflowServiceHandler(s.admin, opts...))

	routes.addService(v1contracts.File_v1_workflows_proto.Services().ByName("AdminService"))
	mux.Handle(v1connect.NewAdminServiceHandler(s.adminv1, opts...))

	if s.operatorSvc != nil {
		routes.addService(v1contracts.File_v1_operator_proto.Services().ByName("OperatorService"))
		mux.Handle(v1connect.NewOperatorServiceHandler(s.operatorSvc, opts...))
	}

	if s.otelCollector != nil {
		// Register as the standard OTLP TraceService for OTEL SDK compatibility
		routes.add(traceServiceName, "Export", false)
		mux.Handle(traceServiceExportProcedure, connect.NewUnaryHandlerSimple(
			traceServiceExportProcedure,
			func(ctx context.Context, req *collectortracev1.ExportTraceServiceRequest) (*collectortracev1.ExportTraceServiceResponse, error) {
				return s.otelCollector.Export(ctx, req)
			},
			opts...,
		))
	}

	deadlines := transportDeadlines{
		routes:                 routes,
		http1BodyReadTimeout:   s.http1BodyReadTimeout,
		http1UnaryWriteTimeout: s.http1UnaryWriteTimeout,
	}

	// outermost first
	return withCORS(s.config.Runtime.AllowedOrigins, routes.unimplemented(deadlines.enforce(withStreamAbort(matchRequestCompression(mux))))), nil
}

// matchRequestCompression compresses a gRPC response only when its request was compressed,
// which is the rule gRPC servers follow and SDKs are sized for. gRPC clients advertise
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
	// server is insecure. HTTP/1.1 is what fetch-based Connect callers get from serverless
	// runtimes and from load balancers that do not speak HTTP/2 to their backends; it carries
	// unary calls and server streams, and transportDeadlines bounds it.
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
		MaxHeaderBytes:    maxHeaderBytes,
		// streams are long-lived, so there is no read, write or idle timeout; dead HTTP/2
		// connections are found by the pings below, and per-call deadlines come from
		// transportDeadlines
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
