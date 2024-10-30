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
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/admin"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/grpc/middleware"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

type Server struct {
	eventcontracts.UnimplementedEventsServiceServer
	dispatchercontracts.UnimplementedDispatcherServer
	admincontracts.UnimplementedWorkflowServiceServer

	l           *zerolog.Logger
	a           errors.Alerter
	analytics   analytics.Analytics
	port        int
	bindAddress string

	config     *server.ServerConfig
	ingestor   ingestor.Ingestor
	dispatcher dispatcher.Dispatcher
	admin      admin.AdminService
	tls        *tls.Config
	insecure   bool
}

type ServerOpt func(*ServerOpts)

type ServerOpts struct {
	config      *server.ServerConfig
	l           *zerolog.Logger
	a           errors.Alerter
	analytics   analytics.Analytics
	port        int
	bindAddress string
	ingestor    ingestor.Ingestor
	dispatcher  dispatcher.Dispatcher
	admin       admin.AdminService
	tls         *tls.Config
	insecure    bool
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

func WithAdmin(a admin.AdminService) ServerOpt {
	return func(opts *ServerOpts) {
		opts.admin = a
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
		l:           opts.l,
		a:           opts.a,
		analytics:   opts.analytics,
		config:      opts.config,
		port:        opts.port,
		bindAddress: opts.bindAddress,
		ingestor:    opts.ingestor,
		dispatcher:  opts.dispatcher,
		admin:       opts.admin,
		tls:         opts.tls,
		insecure:    opts.insecure,
	}, nil
}

func (s *Server) Start() (func() error, error) {
	return s.startGRPC()
}

func (s *Server) startGRPC() (func() error, error) {
	s.l.Debug().Msgf("starting grpc server on %s:%d", s.bindAddress, s.port)

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
		auth.StreamServerInterceptor(authMiddleware.Middleware),
		middleware.ServerNameStreamingInterceptor,
		ratelimit.StreamServerInterceptor(limiter),
		errorInterceptor.ErrorStreamServerInterceptor(),
		recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
	))

	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(
		logging.UnaryServerInterceptor(middleware.InterceptorLogger(s.l), opts...),
		auth.UnaryServerInterceptor(authMiddleware.Middleware),
		middleware.AttachServerNameInterceptor,
		ratelimit.UnaryServerInterceptor(limiter),
		errorInterceptor.ErrorUnaryServerInterceptor(),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
	))

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

	grpcServer := grpc.NewServer(serverOpts...)

	if s.ingestor != nil {
		eventcontracts.RegisterEventsServiceServer(grpcServer, s.ingestor)
	}

	if s.dispatcher != nil {
		dispatchercontracts.RegisterDispatcherServer(grpcServer, s.dispatcher)
	}

	if s.admin != nil {
		admincontracts.RegisterWorkflowServiceServer(grpcServer, s.admin)
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
