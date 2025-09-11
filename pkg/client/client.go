package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/client/loader"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"

	cloudrest "github.com/hatchet-dev/hatchet/pkg/client/cloud/rest"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

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

type ClientOpt func(*ClientOpts)

type filesLoaderFunc func() []*types.Workflow

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
	sharedMeta  map[string]string

	cloudRegisterID *string
	runnableActions []string

	filesLoader        filesLoaderFunc
	initWorkflows      bool
	presetWorkerLabels map[string]string
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
		clientConfig, err = loader.GetClientConfigFromConfigFile(cf)

		if err != nil {
			panic(err)
		}
	}

	logger := logger.NewDefaultLogger("client")

	return &ClientOpts{
		tenantId:           clientConfig.TenantId,
		token:              clientConfig.Token,
		l:                  &logger,
		v:                  validator.NewDefaultValidator(),
		tls:                clientConfig.TLSConfig,
		hostPort:           clientConfig.GRPCBroadcastAddress,
		serverURL:          clientConfig.ServerURL,
		filesLoader:        types.DefaultLoader,
		namespace:          clientConfig.Namespace,
		cloudRegisterID:    clientConfig.CloudRegisterID,
		runnableActions:    clientConfig.RunnableActions,
		noGrpcRetry:        clientConfig.NoGrpcRetry,
		sharedMeta:         make(map[string]string),
		presetWorkerLabels: clientConfig.PresetWorkerLabels,
	}
}

// Deprecated: use WithLogger instead
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

func WithLogger(l *zerolog.Logger) ClientOpt {
	return func(opts *ClientOpts) {
		opts.l = l
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

func WithNamespace(namespace string) ClientOpt {
	return func(opts *ClientOpts) {
		opts.namespace = namespace + "_"
	}
}

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

func InitWorkflows() ClientOpt {
	return func(opts *ClientOpts) {
		opts.initWorkflows = true
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

// New creates a new client instance.
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

	if !opts.noGrpcRetry {
		retryOnCodes := []codes.Code{
			codes.ResourceExhausted,
			codes.DeadlineExceeded,
			codes.FailedPrecondition,
			codes.Internal,
			codes.Unavailable,
		}

		retryOpts := []grpc_retry.CallOption{

			grpc_retry.WithBackoff(grpc_retry.BackoffExponentialWithJitter(5*time.Second, 0.10)),
			grpc_retry.WithMax(5),
			grpc_retry.WithPerRetryTimeout(30 * time.Second),
			grpc_retry.WithCodes(retryOnCodes...),
			grpc_retry.WithOnRetryCallback(grpc_retry.OnRetryCallback(func(ctx context.Context, attempt uint, err error) {
				if contains(retryOnCodes, status.Code(err)) {
					opts.l.Debug().Msgf("grpc_retry attempt: %d, backoff for %v", attempt, err)
				}
			})),
		}
		grpcOpts = append(grpcOpts, grpc.WithChainStreamInterceptor(grpc_retry.StreamClientInterceptor(retryOpts...)))
		grpcOpts = append(grpcOpts, grpc.WithChainUnaryInterceptor(grpc_retry.UnaryClientInterceptor(retryOpts...)))
	}

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
		ctxLoader:  newContextLoader(opts.token),
		sharedMeta: opts.sharedMeta,
	}

	subscribe := newSubscribe(conn, shared)
	admin := newAdmin(conn, shared, subscribe)
	dispatcher := newDispatcher(conn, shared, opts.presetWorkerLabels)
	event := newEvent(conn, shared)

	rest, err := rest.NewClientWithResponses(opts.serverURL, rest.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opts.token))
		return nil
	}))

	if err != nil {
		return nil, fmt.Errorf("could not create rest client: %w", err)
	}

	cloudrest, err := cloudrest.NewClientWithResponses(opts.serverURL, cloudrest.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opts.token))
		return nil
	}))

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

func contains(codes []codes.Code, code codes.Code) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}
