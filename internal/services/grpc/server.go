package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/ratelimit"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/admin"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	dispatcherv1 "github.com/hatchet-dev/hatchet/internal/services/dispatcher/v1"
	"github.com/hatchet-dev/hatchet/internal/services/grpc/middleware"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/encoding"
	gzipcodec "google.golang.org/grpc/encoding/gzip" // Register gzip compression codec
)

type Server struct {
	eventcontracts.UnimplementedEventsServiceServer
	dispatchercontracts.UnimplementedDispatcherServer
	admincontracts.UnimplementedWorkflowServiceServer
	v1contracts.UnimplementedAdminServiceServer
	v1contracts.UnimplementedV1DispatcherServer

	l           *zerolog.Logger
	a           errors.Alerter
	analytics   analytics.Analytics
	port        int
	bindAddress string

	config       *server.ServerConfig
	ingestor     ingestor.Ingestor
	dispatcher   dispatcher.Dispatcher
	dispatcherv1 dispatcherv1.DispatcherService
	admin        admin.AdminService
	adminv1      adminv1.AdminService
	tls          *tls.Config
	insecure     bool
}

type ServerOpt func(*ServerOpts)

type ServerOpts struct {
	config       *server.ServerConfig
	l            *zerolog.Logger
	a            errors.Alerter
	analytics    analytics.Analytics
	port         int
	bindAddress  string
	ingestor     ingestor.Ingestor
	dispatcher   dispatcher.Dispatcher
	dispatcherv1 dispatcherv1.DispatcherService
	admin        admin.AdminService
	adminv1      adminv1.AdminService
	tls          *tls.Config
	insecure     bool
}

func defaultServerOpts() *ServerOpts {
	logger := logger.NewDefaultLogger("grpc")
	a := errors.NoOpAlerter{}
	analytics := analytics.NoOpAnalytics{}
	return &ServerOpts{
		l:           &logger,
		a:           a,
		analytics:   analytics,
		port:        7070,
		bindAddress: "127.0.0.1",
		insecure:    false,
	}
}

func WithLogger(l *zerolog.Logger) ServerOpt {
	return func(opts *ServerOpts) {
		opts.l = l
	}
}

func WithAlerter(a errors.Alerter) ServerOpt {
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

func WithDispatcher(d dispatcher.Dispatcher) ServerOpt {
	return func(opts *ServerOpts) {
		opts.dispatcher = d
	}
}

func WithDispatcherV1(d dispatcherv1.DispatcherService) ServerOpt {
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
		l:            opts.l,
		a:            opts.a,
		analytics:    opts.analytics,
		config:       opts.config,
		port:         opts.port,
		bindAddress:  opts.bindAddress,
		ingestor:     opts.ingestor,
		dispatcher:   opts.dispatcher,
		dispatcherv1: opts.dispatcherv1,
		admin:        opts.admin,
		adminv1:      opts.adminv1,
		tls:          opts.tls,
		insecure:     opts.insecure,
	}, nil
}

// compressionAcceptEncodingInterceptor ensures the server includes grpc-accept-encoding
// in response headers to advertise gzip support, as required by gRPC spec.
// This fixes the grpc-go limitation where registered compressors aren't automatically advertised.
// See: https://github.com/grpc/grpc-go/issues/2786
func (s *Server) compressionAcceptEncodingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	s.l.Debug().Msgf("[compression] Interceptor called for %s", info.FullMethod)

	// Set grpc-accept-encoding in outgoing metadata
	// This header tells the client what compression algorithms the server accepts
	md := metadata.Pairs("grpc-accept-encoding", "gzip,identity")

	// Merge with any existing outgoing metadata
	if existingMD, ok := metadata.FromOutgoingContext(ctx); ok {
		md = metadata.Join(existingMD, md)
		s.l.Debug().Msgf("[compression] Merged grpc-accept-encoding with existing metadata for %s", info.FullMethod)
	} else {
		s.l.Debug().Msgf("[compression] Setting grpc-accept-encoding header for %s", info.FullMethod)
	}

	// Log what we're setting
	if values := md.Get("grpc-accept-encoding"); len(values) > 0 {
		s.l.Debug().Msgf("[compression] Outgoing grpc-accept-encoding: %v", values)
	}

	ctx = metadata.NewOutgoingContext(ctx, md)

	// CRITICAL: SendHeader must be called BEFORE the handler runs
	// This ensures grpc-accept-encoding is in the initial response headers
	if sendErr := grpc.SendHeader(ctx, md); sendErr != nil {
		s.l.Warn().Msgf("[compression] Failed to SendHeader (may be called multiple times): %v", sendErr)
	} else {
		s.l.Debug().Msgf("[compression] Successfully called SendHeader for %s", info.FullMethod)
	}

	// Now call the handler
	resp, err := handler(ctx, req)

	return resp, err
}

func (s *Server) compressionAcceptEncodingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()

	md := metadata.Pairs("grpc-accept-encoding", "gzip,identity")
	if existingMD, ok := metadata.FromOutgoingContext(ctx); ok {
		md = metadata.Join(existingMD, md)
		s.l.Debug().Msgf("[compression] Merged grpc-accept-encoding with existing metadata for stream %s", info.FullMethod)
	} else {
		s.l.Debug().Msgf("[compression] Setting grpc-accept-encoding header for stream %s", info.FullMethod)
	}

	// Log what we're setting
	if values := md.Get("grpc-accept-encoding"); len(values) > 0 {
		s.l.Debug().Msgf("[compression] Outgoing grpc-accept-encoding: %v", values)
	}

	ctx = metadata.NewOutgoingContext(ctx, md)

	// CRITICAL: SendHeader must be called BEFORE the handler runs
	// This ensures grpc-accept-encoding is in the initial response headers
	if sendErr := grpc.SendHeader(ctx, md); sendErr != nil {
		s.l.Warn().Msgf("[compression] Failed to SendHeader for stream (may be called multiple times): %v", sendErr)
	} else {
		s.l.Debug().Msgf("[compression] Successfully called SendHeader for stream %s", info.FullMethod)
	}

	wrapped := &wrappedServerStream{
		ServerStream: ss,
		ctx:          ctx,
	}
	return handler(srv, wrapped)
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

func (s *Server) Start() (func() error, error) {
	return s.startGRPC()
}

func (s *Server) startGRPC() (func() error, error) {
	s.l.Debug().Msgf("starting grpc server on %s:%d", s.bindAddress, s.port)
	s.l.Info().Msg("gzip compression enabled for gRPC server")

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.bindAddress, s.port))

	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	serverOpts := []grpc.ServerOption{}

	if s.insecure {
		serverOpts = append(serverOpts, grpc.Creds(insecure.NewCredentials()))
	} else {
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(s.tls)))
	}

	authMiddleware := middleware.NewAuthN(s.config)

	grpcPanicRecoveryHandler := func(p any) (err error) {
		panicErr, ok := p.(error)

		var panicStr string

		if !ok {
			panicStr, ok = p.(string)

			if !ok {
				panicStr = "Could not determine panic error"
			}
		} else {
			panicStr = panicErr.Error()
		}

		err = fmt.Errorf("recovered from panic: %s. Stack: %s", panicStr, string(debug.Stack()))

		s.l.Err(err).Msg("")
		s.a.SendAlert(context.Background(), err, nil)
		return status.Errorf(codes.Internal, "An internal error occurred")
	}
	limit := s.config.Runtime.GRPCRateLimit
	if limit == 0 {
		limit = 1000
	}
	burst := limit
	limiter := middleware.NewHatchetRateLimiter(rate.Limit(limit), int(burst), s.l)

	errorInterceptor := middleware.NewErrorInterceptor(s.a, s.l)

	opts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(
		logging.StreamServerInterceptor(middleware.InterceptorLogger(s.l), opts...),
		s.compressionAcceptEncodingStreamInterceptor, // Advertise gzip support in grpc-accept-encoding header
		auth.StreamServerInterceptor(authMiddleware.Middleware),
		middleware.ServerNameStreamingInterceptor,
		ratelimit.StreamServerInterceptor(limiter),
		errorInterceptor.ErrorStreamServerInterceptor(),
		recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
	))

	// Prepare base unary interceptors
	baseUnaryInterceptors := []grpc.UnaryServerInterceptor{
		logging.UnaryServerInterceptor(middleware.InterceptorLogger(s.l), opts...),
		s.compressionAcceptEncodingInterceptor, // Advertise gzip support in grpc-accept-encoding header
		auth.UnaryServerInterceptor(authMiddleware.Middleware),
		middleware.AttachServerNameInterceptor,
		ratelimit.UnaryServerInterceptor(limiter),
		errorInterceptor.ErrorUnaryServerInterceptor(),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
	}

	if len(s.config.GRPCInterceptors) > 0 {
		baseUnaryInterceptors = append(baseUnaryInterceptors, s.config.GRPCInterceptors...)
	}

	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(baseUnaryInterceptors...))

	var enforcement = keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(enforcement))

	var kasp = keepalive.ServerParameters{
		// ping the client every 30 seconds if idle to ensure the connection is still active
		Time: 30 * time.Second,
	}

	serverOpts = append(serverOpts, grpc.KeepaliveParams(kasp))

	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(
		s.config.Runtime.GRPCMaxMsgSize,
	), grpc.MaxSendMsgSize(
		s.config.Runtime.GRPCMaxMsgSize,
	))

	serverOpts = append(serverOpts, grpc.StaticStreamWindowSize(
		s.config.Runtime.GRPCStaticStreamWindowSize,
	))

	serverOpts = append(serverOpts, grpc.StatsHandler(
		otelgrpc.NewServerHandler(),
	))

	// Force gzip compressor registration by referencing the package
	// This ensures the package's init() function runs and registers the compressor
	_ = gzipcodec.Name

	// Ensure gzip compressor is registered before server creation
	// This guarantees the server will advertise gzip in grpc-accept-encoding header
	if compressor := encoding.GetCompressor("gzip"); compressor == nil {
		s.l.Warn().Msg("gzip compressor not registered - compression may not be advertised")
	} else {
		s.l.Debug().Msg("gzip compressor confirmed registered")
	}

	grpcServer := grpc.NewServer(serverOpts...)

	if s.ingestor != nil {
		eventcontracts.RegisterEventsServiceServer(grpcServer, s.ingestor)
	}

	if s.dispatcher != nil {
		dispatchercontracts.RegisterDispatcherServer(grpcServer, s.dispatcher)
	}

	if s.dispatcherv1 != nil {
		v1contracts.RegisterV1DispatcherServer(grpcServer, s.dispatcherv1)
	}

	if s.admin != nil {
		admincontracts.RegisterWorkflowServiceServer(grpcServer, s.admin)
	}

	if s.adminv1 != nil {
		v1contracts.RegisterAdminServiceServer(grpcServer, s.adminv1)
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			panic(fmt.Errorf("failed to serve: %w", err))
		}
	}()

	cleanup := func() error {
		grpcServer.GracefulStop()
		return nil
	}

	return cleanup, nil
}
