package client

import (
	"crypto/tls"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/hatchet-dev/hatchet/pkg/client/loader"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type Client interface {
	Admin() AdminClient
	Dispatcher() DispatcherClient
	Event() EventClient
}

type clientImpl struct {
	conn *grpc.ClientConn

	admin      AdminClient
	dispatcher DispatcherClient
	event      EventClient

	// the tenant id
	tenantId string

	l *zerolog.Logger

	v validator.Validator
}

type ClientOpt func(*ClientOpts)

type filesLoaderFunc func() []*types.Workflow

type ClientOpts struct {
	tenantId string
	l        *zerolog.Logger
	v        validator.Validator
	tls      *tls.Config
	hostPort string

	filesLoader   filesLoaderFunc
	initWorkflows bool
}

func defaultClientOpts() *ClientOpts {
	// read from environment variables and hostname by default
	configLoader := &loader.ConfigLoader{}

	clientConfig, err := configLoader.LoadClientConfig()

	if err != nil {
		panic(err)
	}

	logger := logger.NewDefaultLogger("client")

	return &ClientOpts{
		tenantId:    clientConfig.TenantId,
		l:           &logger,
		v:           validator.NewDefaultValidator(),
		tls:         clientConfig.TLSConfig,
		hostPort:    "localhost:7070",
		filesLoader: types.DefaultLoader,
	}
}

func WithTenantId(tenantId string) ClientOpt {
	return func(opts *ClientOpts) {
		opts.tenantId = tenantId
	}
}

func WithHostPort(host string, port int) ClientOpt {
	return func(opts *ClientOpts) {
		opts.hostPort = fmt.Sprintf("%s:%d", host, port)
	}
}

func InitWorkflows() ClientOpt {
	return func(opts *ClientOpts) {
		opts.initWorkflows = true
	}
}

// WithWorkflows sets the workflow files to use for the worker. If this is not passed in, the workflows files will be loaded
// from the .hatchet folder in the current directory.
func WithWorkflows(files []*types.Workflow) ClientOpt {
	return func(opts *ClientOpts) {
		opts.filesLoader = func() []*types.Workflow {
			return files
		}
	}
}

type sharedClientOpts struct {
	tenantId string
	l        *zerolog.Logger
	v        validator.Validator
}

// New creates a new client instance.
func New(fs ...ClientOpt) (Client, error) {
	opts := defaultClientOpts()

	for _, f := range fs {
		f(opts)
	}

	// if no TLS, exit
	if opts.tls == nil {
		return nil, fmt.Errorf("tls config is required")
	}

	opts.l.Debug().Msgf("connecting to %s with TLS server name %s", opts.hostPort, opts.tls.ServerName)

	conn, err := grpc.Dial(
		opts.hostPort,
		grpc.WithTransportCredentials(credentials.NewTLS(opts.tls)),
	)

	if err != nil {
		return nil, err
	}

	shared := &sharedClientOpts{
		tenantId: opts.tenantId,
		l:        opts.l,
		v:        opts.v,
	}

	admin := newAdmin(conn, shared)
	dispatcher := newDispatcher(conn, shared)
	event := newEvent(conn, shared)

	// if init workflows is set, then we need to initialize the workflows
	if opts.initWorkflows {
		if err := initWorkflows(opts.filesLoader, admin); err != nil {
			return nil, fmt.Errorf("could not init workflows: %w", err)
		}
	}

	return &clientImpl{
		conn:       conn,
		tenantId:   opts.tenantId,
		l:          opts.l,
		admin:      admin,
		dispatcher: dispatcher,
		event:      event,
		v:          opts.v,
	}, nil
}

func (c *clientImpl) Admin() AdminClient {
	return c.admin
}

func (c *clientImpl) Dispatcher() DispatcherClient {
	return c.dispatcher
}

func (c *clientImpl) Event() EventClient {
	return c.event
}

func initWorkflows(fl filesLoaderFunc, adminClient AdminClient) error {
	files := fl()

	for _, file := range files {
		if err := adminClient.PutWorkflow(file); err != nil {
			return fmt.Errorf("could not create workflow: %w", err)
		}
	}

	return nil
}
