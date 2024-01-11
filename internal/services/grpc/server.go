package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/services/admin"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Server struct {
	eventcontracts.UnimplementedEventsServiceServer
	dispatchercontracts.UnimplementedDispatcherServer
	admincontracts.UnimplementedWorkflowServiceServer

	l           *zerolog.Logger
	port        int
	bindAddress string

	ingestor   ingestor.Ingestor
	dispatcher dispatcher.Dispatcher
	admin      admin.AdminService
	tls        *tls.Config
}

type ServerOpt func(*ServerOpts)

type ServerOpts struct {
	l           *zerolog.Logger
	port        int
	bindAddress string
	ingestor    ingestor.Ingestor
	dispatcher  dispatcher.Dispatcher
	admin       admin.AdminService
	tls         *tls.Config
}

func defaultServerOpts() *ServerOpts {
	logger := logger.NewDefaultLogger("grpc")

	return &ServerOpts{
		l:           &logger,
		port:        7070,
		bindAddress: "127.0.0.1",
	}
}

func WithLogger(l *zerolog.Logger) ServerOpt {
	return func(opts *ServerOpts) {
		opts.l = l
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

func WithTLSConfig(tls *tls.Config) ServerOpt {
	return func(opts *ServerOpts) {
		opts.tls = tls
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

	if opts.tls == nil {
		return nil, fmt.Errorf("tls config is required. use WithTLSConfig")
	}

	newLogger := opts.l.With().Str("service", "grpc").Logger()
	opts.l = &newLogger

	return &Server{
		l:           opts.l,
		port:        opts.port,
		bindAddress: opts.bindAddress,
		ingestor:    opts.ingestor,
		dispatcher:  opts.dispatcher,
		admin:       opts.admin,
		tls:         opts.tls,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	return s.startGRPC(ctx)
}

func (s *Server) startGRPC(ctx context.Context) error {
	s.l.Debug().Msgf("starting grpc server on %s:%d", s.bindAddress, s.port)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.bindAddress, s.port))

	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(s.tls)))

	if s.ingestor != nil {
		eventcontracts.RegisterEventsServiceServer(grpcServer, s.ingestor)
	}

	if s.dispatcher != nil {
		dispatchercontracts.RegisterDispatcherServer(grpcServer, s.dispatcher)
	}

	if s.admin != nil {
		admincontracts.RegisterWorkflowServiceServer(grpcServer, s.admin)
	}

	// Start listening
	return grpcServer.Serve(lis)
}
