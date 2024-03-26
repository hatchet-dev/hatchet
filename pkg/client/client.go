package client

import (
	"crypto/tls"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/hatchet-dev/hatchet/internal/config/client"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/hatchet-dev/hatchet/pkg/client/loader"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type Client interface {
	Admin() AdminClient
	Dispatcher() DispatcherClient
	Event() EventClient
	Run() RunClient
}

type clientImpl struct {
	conn *grpc.ClientConn

	admin      AdminClient
	dispatcher DispatcherClient
	event      EventClient
	run        RunClient

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
	token    string

	filesLoader   filesLoaderFunc
	initWorkflows bool
}

func defaultClientOpts(cf *client.ClientConfigFile) *ClientOpts {
	var clientConfig *client.ClientConfig
	var err error

	configLoader := &loader.ConfigLoader{}

	if cf == nil {
		// read from environment variables and hostname by default

		clientConfig, err = configLoader.LoadClientConfig()

		if err != nil {
			panic(err)
		}

	} else {
		clientConfig, err = loader.GetClientConfigFromConfigFile(cf)

		if err != nil {
			panic(err)
		}
	}

	logger := logger.NewDefaultLogger("client")

	return &ClientOpts{
		tenantId:    clientConfig.TenantId,
		token:       clientConfig.Token,
		l:           &logger,
		v:           validator.NewDefaultValidator(),
		tls:         clientConfig.TLSConfig,
		hostPort:    clientConfig.GRPCBroadcastAddress,
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

func WithToken(token string) ClientOpt {
	return func(opts *ClientOpts) {
		opts.token = token
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
	tenantId  string
	l         *zerolog.Logger
	v         validator.Validator
	ctxLoader *contextLoader
}

// New creates a new client instance.
func New(fs ...ClientOpt) (Client, error) {
	opts := defaultClientOpts(nil)

	for _, f := range fs {
		f(opts)
	}

	return newFromOpts(opts)
}

func NewFromConfigFile(cf *client.ClientConfigFile, fs ...ClientOpt) (Client, error) {
	opts := defaultClientOpts(cf)

	for _, f := range fs {
		f(opts)
	}

	return newFromOpts(opts)
}

func newFromOpts(opts *ClientOpts) (Client, error) {
	if opts.token == "" {
		return nil, fmt.Errorf("token is required")
	}

	var transportCreds credentials.TransportCredentials

	if opts.tls == nil {
		opts.l.Debug().Msgf("connecting to %s without TLS", opts.hostPort)

		transportCreds = insecure.NewCredentials()
	} else {
		opts.l.Debug().Msgf("connecting to %s with TLS server name %s", opts.hostPort, opts.tls.ServerName)

		transportCreds = credentials.NewTLS(opts.tls)
	}

	conn, err := grpc.Dial(
		opts.hostPort,
		grpc.WithTransportCredentials(transportCreds),
	)

	if err != nil {
		return nil, err
	}

	shared := &sharedClientOpts{
		tenantId:  opts.tenantId,
		l:         opts.l,
		v:         opts.v,
		ctxLoader: newContextLoader(opts.token),
	}

	admin := newAdmin(conn, shared)
	dispatcher := newDispatcher(conn, shared)
	event := newEvent(conn, shared)
	run := newRun(conn, shared)

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
		run:        run,
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

func (c *clientImpl) Run() RunClient {
	return c.run
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
