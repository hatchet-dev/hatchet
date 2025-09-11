package v1

import (
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	grpcMetadata "google.golang.org/grpc/metadata"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/logger"

	"context"
)

type GRPCClient struct {
	l     *zerolog.Logger
	admin admincontracts.AdminServiceClient
}

type clientOpts struct {
	l        *zerolog.Logger
	hostPort string
	token    string
	tls      *tls.Config
}

type GRPCClientOpt func(*clientOpts)

func WithHostPort(hostPort string) func(*clientOpts) {
	// validate the hostPort
	return func(opts *clientOpts) {
		opts.hostPort = hostPort
	}
}

func WithToken(token string) func(*clientOpts) {
	return func(opts *clientOpts) {
		opts.token = token
	}
}

func WithTLS(tls *tls.Config) func(*clientOpts) {
	return func(opts *clientOpts) {
		opts.tls = tls
	}
}

func WithLogger(l *zerolog.Logger) func(*clientOpts) {
	return func(opts *clientOpts) {
		opts.l = l
	}
}

func defaultOpts() *clientOpts {
	l := logger.NewDefaultLogger("client")

	return &clientOpts{
		l: &l,
	}
}

func validateOpts(opts *clientOpts) error {
	if opts.hostPort == "" {
		return errors.New("hostPort is required")
	}

	if opts.token == "" {
		return errors.New("token is required")
	}

	_, _, err := net.SplitHostPort(opts.hostPort)

	if err != nil {
		return err
	}

	return nil
}

func NewGRPCClient(fs ...GRPCClientOpt) (*GRPCClient, error) {
	opts := defaultOpts()

	for _, f := range fs {
		f(opts)
	}

	err := validateOpts(opts)

	if err != nil {
		return nil, err
	}

	var transportCreds credentials.TransportCredentials

	if opts.tls == nil {
		opts.l.Warn().Msgf("connecting to %s without TLS", opts.hostPort)

		transportCreds = insecure.NewCredentials()
	} else {
		opts.l.Debug().Msgf("connecting to %s with TLS server name %s", opts.hostPort, opts.tls.ServerName)

		transportCreds = credentials.NewTLS(opts.tls)
	}

	keepAliveParams := keepalive.ClientParameters{
		Time:                10 * time.Second, // grpc.keepalive_time_ms: 10 * 1000
		Timeout:             60 * time.Second, // grpc.keepalive_timeout_ms: 60 * 1000
		PermitWithoutStream: true,             // grpc.keepalive_permit_without_calls: 1
	}

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithKeepaliveParams(keepAliveParams),
	}

	conn, err := grpc.NewClient(
		opts.hostPort,
		grpcOpts...,
	)

	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		l:     opts.l,
		admin: admincontracts.NewAdminServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Admin() admincontracts.AdminServiceClient {
	return c.admin
}

func AuthContext(ctx context.Context, token string) context.Context {
	md := grpcMetadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})

	return grpcMetadata.NewOutgoingContext(ctx, md)
}
