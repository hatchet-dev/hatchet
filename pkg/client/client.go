// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip" // Register gzip compression codec
	"google.golang.org/grpc/keepalive"

	"github.com/hatchet-dev/hatchet/pkg/client/loader"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"

	cloudrest "github.com/hatchet-dev/hatchet/pkg/client/cloud/rest"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// Deprecated: Client is an internal interface used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type Client interface {
	Admin() AdminClient
	Cron() CronClient
	Schedule() ScheduleClient
	Dispatcher() DispatcherClient
	Event() EventClient
	Subscribe() SubscribeClient
	API() *rest.ClientWithResponses
	CloudAPI() *cloudrest.ClientWithResponses
	Logger() *zerolog.Logger
	TenantId() string
	Namespace() string
	CloudRegisterID() *string
	RunnableActions() []string
}

type clientImpl struct {
	conn *grpc.ClientConn

	admin      AdminClient
	cron       CronClient
	schedule   ScheduleClient
	dispatcher DispatcherClient
	event      EventClient
	subscribe  SubscribeClient
	rest       *rest.ClientWithResponses
	cloudrest  *cloudrest.ClientWithResponses

	// the tenant id
	tenantId string

	namespace string

	cloudRegisterID *string
	runnableActions []string

	l *zerolog.Logger

	v validator.Validator
}

// Deprecated: ClientOpt is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type ClientOpt func(*ClientOpts)

type filesLoaderFunc func() []*types.Workflow

// Deprecated: ClientOpts is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type ClientOpts struct {
	tenantId    string
	l           *zerolog.Logger
	v           validator.Validator
	tls         *tls.Config
	hostPort    string
	serverURL   string
	token       string
	namespace   string
	noGrpcRetry bool
	noRetry     bool
	sharedMeta  map[string]string

	cloudRegisterID *string
	runnableActions []string

	filesLoader        filesLoaderFunc
	initWorkflows      bool
	presetWorkerLabels map[string]string

	disableGzipCompression bool
	grpcHeaders            map[string]string

	// Embedded is set by the Go SDK's WithEmbeddedPostgres option and consumed by
	// hatchet.NewClient to bootstrap an in-process engine. It is nil in all other cases.
	Embedded any
}

func defaultClientOpts(token *string, cf *client.ClientConfigFile) *ClientOpts {
	var clientConfig *client.ClientConfig
	var err error

	configLoader := &loader.ConfigLoader{}

	if cf == nil {
		// read from environment variables and hostname by default

		clientConfig, err = configLoader.LoadClientConfig(token)

		if err != nil {
			panic(err)
		}

	} else {
		if token != nil {
			cf.Token = *token
		}
		clientConfig, err = loader.GetClientConfigFromConfigFile(token, cf)

		if err != nil {
			panic(err)
		}
	}

	logger := logger.NewStdErr(&clientConfig.Logger, "client")

	return &ClientOpts{
		tenantId:               clientConfig.TenantId,
		token:                  clientConfig.Token,
		l:                      &logger,
		v:                      validator.NewDefaultValidator(),
		tls:                    clientConfig.TLSConfig,
		hostPort:               clientConfig.GRPCBroadcastAddress,
		serverURL:              clientConfig.ServerURL,
		filesLoader:            types.DefaultLoader,
		namespace:              clientConfig.Namespace,
		cloudRegisterID:        clientConfig.CloudRegisterID,
		runnableActions:        clientConfig.RunnableActions,
		noGrpcRetry:            clientConfig.NoGrpcRetry,
		noRetry:                clientConfig.NoRetry,
		sharedMeta:             make(map[string]string),
		presetWorkerLabels:     clientConfig.PresetWorkerLabels,
		disableGzipCompression: clientConfig.DisableGzipCompression,
	}
}

// Deprecated: WithLogLevel is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func WithLogLevel(lvl string) ClientOpt {
	return func(opts *ClientOpts) {
		logger := logger.NewDefaultLogger("client")
		lvl, err := zerolog.ParseLevel(lvl)

		if err == nil {
			logger = logger.Level(lvl)
		}

		opts.l = &logger
	}
}

// Deprecated: WithLogger is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func WithLogger(l *zerolog.Logger) ClientOpt {
	return func(opts *ClientOpts) {
		opts.l = l
	}
}

// Deprecated: WithTenantId is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func WithTenantId(tenantId string) ClientOpt {
	return func(opts *ClientOpts) {
		opts.tenantId = tenantId
	}
}

// Deprecated: WithHostPort is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func WithHostPort(host string, port int) ClientOpt {
	return func(opts *ClientOpts) {
		opts.hostPort = fmt.Sprintf("%s:%d", host, port)
	}
}

// Deprecated: WithToken is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func WithToken(token string) ClientOpt {
	return func(opts *ClientOpts) {
		opts.token = token
	}
}

// Deprecated: WithNamespace is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func WithNamespace(namespace string) ClientOpt {
	return func(opts *ClientOpts) {
		opts.namespace = namespace + "_"
	}
}

// Deprecated: WithSharedMeta is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func WithSharedMeta(meta map[string]string) ClientOpt {
	return func(opts *ClientOpts) {
		if opts.sharedMeta == nil {
			opts.sharedMeta = make(map[string]string)
		}

		for k, v := range meta {
			opts.sharedMeta[k] = v
		}
	}
}

// Deprecated: InitWorkflows is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func InitWorkflows() ClientOpt {
	return func(opts *ClientOpts) {
		opts.initWorkflows = true
	}
}

func WithGRPCHeaders(headers map[string]string) ClientOpt {
	return func(opts *ClientOpts) {
		if opts.grpcHeaders == nil {
			opts.grpcHeaders = make(map[string]string)
		}
		for k, v := range headers {
			opts.grpcHeaders[k] = v
		}
	}
}

type sharedClientOpts struct {
	tenantId   string
	namespace  string
	l          *zerolog.Logger
	v          validator.Validator
	ctxLoader  *contextLoader
	sharedMeta map[string]string
}

// Deprecated: New is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func New(fs ...ClientOpt) (Client, error) {
	var token *string
	initOpts := &ClientOpts{}
	for _, f := range fs {
		f(initOpts)
	}
	if initOpts.token != "" {
		token = &initOpts.token
	}

	opts := defaultClientOpts(token, nil)

	for _, f := range fs {
		f(opts)
	}

	return newFromOpts(opts)
}

// Deprecated: NewFromConfigFile is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewFromConfigFile(cf *client.ClientConfigFile, fs ...ClientOpt) (Client, error) {
	opts := defaultClientOpts(nil, cf)

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

	keepAliveParams := keepalive.ClientParameters{
		Time:                10 * time.Second, // grpc.keepalive_time_ms: 10 * 1000
		Timeout:             60 * time.Second, // grpc.keepalive_timeout_ms: 60 * 1000
		PermitWithoutStream: true,             // grpc.keepalive_permit_without_calls: 1
	}

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithKeepaliveParams(keepAliveParams),
	}

	if !opts.disableGzipCompression {
		grpcOpts = append(grpcOpts, grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		opts.l.Info().Msg("gzip compression enabled for gRPC client")
	} else {
		opts.l.Info().Msg("gzip compression disabled for gRPC client")
	}

	grpcRetryEnabled := !opts.noRetry && !opts.noGrpcRetry
	restRetryEnabled := !opts.noRetry
	grpcOpts = append(grpcOpts, retry.GRPCDialOptions(opts.l, grpcRetryEnabled)...)

	conn, err := grpc.NewClient(
		opts.hostPort,
		grpcOpts...,
	)

	if err != nil {
		return nil, err
	}

	shared := &sharedClientOpts{
		tenantId:   opts.tenantId,
		namespace:  opts.namespace,
		l:          opts.l,
		v:          opts.v,
		ctxLoader:  newContextLoader(opts.token, opts.grpcHeaders),
		sharedMeta: opts.sharedMeta,
	}

	subscribe := newSubscribe(conn, shared)
	admin := newAdmin(conn, shared, subscribe)
	dispatcher := newDispatcher(conn, shared, opts.presetWorkerLabels)
	event := newEvent(conn, shared)

	authEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opts.token))
		return nil
	}

	restOpts := []rest.ClientOption{rest.WithRequestEditorFn(authEditor)}
	cloudRestOpts := []cloudrest.ClientOption{cloudrest.WithRequestEditorFn(authEditor)}
	if restRetryEnabled {
		restDoer, retryErr := retry.NewRestDoer(&http.Client{})
		if retryErr != nil {
			return nil, fmt.Errorf("could not create rest retry doer: %w", retryErr)
		}
		restOpts = append(restOpts, rest.WithHTTPClient(restDoer))
		cloudRestOpts = append(cloudRestOpts, cloudrest.WithHTTPClient(restDoer))
	}

	rest, err := rest.NewClientWithResponses(opts.serverURL, restOpts...)

	if err != nil {
		return nil, fmt.Errorf("could not create rest client: %w", err)
	}

	cloudrest, err := cloudrest.NewClientWithResponses(opts.serverURL, cloudRestOpts...)

	if err != nil {
		return nil, fmt.Errorf("could not create cloud REST client: %w", err)
	}

	cronClient, err := NewCronClient(rest, opts.l, opts.v, opts.tenantId, opts.namespace)

	if err != nil {
		return nil, fmt.Errorf("could not create cron client: %w", err)
	}

	scheduleClient, err := NewScheduleClient(rest, opts.l, opts.v, opts.tenantId, opts.namespace)

	if err != nil {
		return nil, fmt.Errorf("could not create schedule client: %w", err)
	}

	// if init workflows is set, then we need to initialize the workflows
	if opts.initWorkflows {
		if err := initWorkflows(opts.filesLoader, admin); err != nil {
			return nil, fmt.Errorf("could not init workflows: %w", err)
		}
	}

	return &clientImpl{
		conn:            conn,
		tenantId:        opts.tenantId,
		l:               opts.l,
		admin:           admin,
		cron:            cronClient,
		schedule:        scheduleClient,
		dispatcher:      dispatcher,
		subscribe:       subscribe,
		event:           event,
		v:               opts.v,
		rest:            rest,
		cloudrest:       cloudrest,
		namespace:       opts.namespace,
		cloudRegisterID: opts.cloudRegisterID,
		runnableActions: opts.runnableActions,
	}, nil
}

func (c *clientImpl) Admin() AdminClient {
	return c.admin
}

func (c *clientImpl) Cron() CronClient {
	return c.cron
}

func (c *clientImpl) Schedule() ScheduleClient {
	return c.schedule
}

func (c *clientImpl) Dispatcher() DispatcherClient {
	return c.dispatcher
}

func (c *clientImpl) Event() EventClient {
	return c.event
}

func (c *clientImpl) Subscribe() SubscribeClient {
	return c.subscribe
}

func (c *clientImpl) Logger() *zerolog.Logger {
	return c.l
}

func (c *clientImpl) API() *rest.ClientWithResponses {
	return c.rest
}

func (c *clientImpl) CloudAPI() *cloudrest.ClientWithResponses {
	return c.cloudrest
}

func (c *clientImpl) TenantId() string {
	return c.tenantId
}

func (c *clientImpl) Namespace() string {
	return c.namespace
}

func (c *clientImpl) CloudRegisterID() *string {
	return c.cloudRegisterID
}

func (c *clientImpl) RunnableActions() []string {
	return c.runnableActions
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
