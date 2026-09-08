// Package grpclink is the out-of-process Link: registrations are OperatorSessions opened over
// the engine's OperatorService with a per-tenant API token from a TenantTokenExchange.
package grpclink

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // OperatorService's client lives in the legacy client package
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// ClientFactory builds the engine client for one token. The default derives the gRPC address
// and TLS settings from the token's claims and HATCHET_CLIENT_* environment, like the SDK.
type ClientFactory func(token string) (client.Client, error) //nolint:staticcheck // see import

// Options configures a Link.
type Options struct {
	Logger *zerolog.Logger

	// NewClient replaces the default client factory; tests inject a fake.
	NewClient ClientFactory

	// OperatorName is the OperatorService operator name every registration connects as. The
	// engine upserts one operator row per tenant by this name.
	OperatorName string
}

type cachedClient struct {
	client client.Client //nolint:staticcheck // see import
	token  string
}

// closeClient closes the client's connection when the client exposes one. The SDK client
// owns a gRPC connection; dropping the reference alone would leave its goroutines behind.
func closeClient(c client.Client) { //nolint:staticcheck // see import
	if closer, ok := c.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

// Link caches one engine client per tenant. The token is asked from the exchange on every
// Open so a rotated token replaces the cached client, and the client is closed when it is
// replaced or when the core reports the tenant is no longer served, after every registration
// for the tenant is closed.
type Link struct {
	exchange  TenantTokenExchange
	l         *zerolog.Logger
	newClient ClientFactory
	clients   map[uuid.UUID]*cachedClient
	name      string
	mu        sync.Mutex
}

// New builds a Link over exchange.
func New(exchange TenantTokenExchange, opts Options) *Link {
	l := opts.Logger

	if l == nil {
		nop := zerolog.Nop()
		l = &nop
	}

	factory := opts.NewClient

	if factory == nil {
		factory = defaultClientFactory(l)
	}

	name := opts.OperatorName

	if name == "" {
		name = "serverless"
	}

	return &Link{
		exchange:  exchange,
		l:         l,
		newClient: factory,
		clients:   map[uuid.UUID]*cachedClient{},
		name:      name,
	}
}

// defaultClientFactory wraps client.New, which panics when the SDK config cannot be loaded
// from the token and environment. The panic is turned into an error so one tenant's bad
// token cannot take the process down.
func defaultClientFactory(l *zerolog.Logger) ClientFactory {
	return func(token string) (c client.Client, err error) { //nolint:staticcheck // see import
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("could not build engine client: %v", r)
			}
		}()

		return client.New(client.WithToken(token), client.WithLogger(l)) //nolint:staticcheck // see import
	}
}

// clientFor returns the cached client for the tenant, rebuilding it when the token changed.
func (g *Link) clientFor(tenantId uuid.UUID, token string) (client.Client, error) { //nolint:staticcheck // see import
	g.mu.Lock()
	defer g.mu.Unlock()

	cached, ok := g.clients[tenantId]

	if ok && cached.token == token {
		return cached.client, nil
	}

	c, err := g.newClient(token)

	if err != nil {
		return nil, err
	}

	if ok {
		closeClient(cached.client)
	}

	g.clients[tenantId] = &cachedClient{client: c, token: token}

	return c, nil
}

func (g *Link) evict(tenantId uuid.UUID) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if cached, ok := g.clients[tenantId]; ok {
		closeClient(cached.client)
		delete(g.clients, tenantId)
	}
}

// ReleaseTenant implements link.TenantReleaser.
func (g *Link) ReleaseTenant(tenantId uuid.UUID) {
	g.evict(tenantId)
}

// Open implements link.Link. An Unauthenticated connect drops the cached client and asks the
// exchange once more, so a token rotated between two Opens is used without waiting for the
// exchange's own reload. The initial action set is streamed to the engine right after the
// connect and flushed before the registration is returned; the session keeps it as the
// desired set and replays it when a reconnect does not resume the worker.
func (g *Link) Open(ctx context.Context, tenantId uuid.UUID, opts link.OpenOpts) (link.Registration, error) {
	session, err := g.connect(ctx, tenantId, opts)

	if err != nil && status.Code(err) == codes.Unauthenticated {
		g.evict(tenantId)
		session, err = g.connect(ctx, tenantId, opts)
	}

	if err != nil {
		return nil, err
	}

	// The engine reports the tenant it authenticated the token as; a unit's registration
	// must belong to the unit's tenant, whatever the exchange handed out.
	if got := session.Registration().TenantId; got != tenantId.String() {
		_ = session.Close()

		return nil, fmt.Errorf("registration for tenant %s was authenticated as tenant %s; the token exchange is misconfigured", tenantId, got)
	}

	if len(opts.Actions) > 0 {
		session.AddActions(opts.Actions...)

		if err := session.Flush(ctx); err != nil {
			_ = session.Close()
			return nil, fmt.Errorf("could not register initial actions for tenant %s: %w", tenantId, err)
		}
	}

	return &registration{session: session}, nil
}

func (g *Link) connect(ctx context.Context, tenantId uuid.UUID, opts link.OpenOpts) (client.OperatorSession, error) {
	token, err := g.exchange.Token(ctx, tenantId)

	if err != nil {
		if errors.Is(err, link.ErrNoToken) {
			return nil, fmt.Errorf("tenant %s: %w", tenantId, link.ErrNoToken)
		}

		return nil, fmt.Errorf("could not resolve token for tenant %s: %w", tenantId, err)
	}

	// A token whose tenant claim names another tenant is refused before a client is built;
	// the engine's own answer is checked again after connecting.
	if claims, err := loaderutils.GetConfFromJWT(token); err == nil && claims.TenantId != "" && claims.TenantId != tenantId.String() {
		return nil, fmt.Errorf("token for tenant %s carries the tenant claim %s; the token exchange is misconfigured", tenantId, claims.TenantId)
	}

	c, err := g.clientFor(tenantId, token)

	if err != nil {
		return nil, fmt.Errorf("tenant %s: %w", tenantId, err)
	}

	labels := make(map[string]interface{}, len(opts.Labels))

	for k, v := range opts.Labels {
		labels[k] = v
	}

	return c.Operator().Connect(ctx, &client.ConnectOperatorRequest{
		Name:       g.name,
		SlotConfig: opts.SlotConfig,
		Labels:     labels,
	})
}

// registration adapts an OperatorSession to link.Registration. hub is created by the first
// OpenDurable and holds the session's one DurableTaskListener.
type registration struct {
	session client.OperatorSession
	hub     *durableHub
	mu      sync.Mutex
	closed  bool
}

func (r *registration) WorkerId() string {
	return r.session.Registration().WorkerId
}

func (r *registration) Actions(ctx context.Context) (<-chan *contracts.AssignedAction, <-chan error, error) {
	return r.session.Actions(ctx)
}

// PutWorkflow implements link.Registration over the admin service; the session derives the
// action ids without touching the streamed action set.
func (r *registration) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	_, actions, err := r.session.PutWorkflow(ctx, wf)

	if err != nil {
		return nil, err
	}

	return actions, nil
}

// AddActions implements link.Registration. The session coalesces and chunks the delta onto
// the Listen stream; nothing is sent until its flusher runs, so the call never blocks.
func (r *registration) AddActions(_ context.Context, ids []string) error {
	r.session.AddActions(ids...)
	return nil
}

// RemoveActions implements link.Registration; see AddActions.
func (r *registration) RemoveActions(_ context.Context, ids []string) error {
	r.session.RemoveActions(ids...)
	return nil
}

// Flush implements link.Registration: it waits until the session has sent every queued delta
// and reports the last send failure.
func (r *registration) Flush(ctx context.Context) error {
	return r.session.Flush(ctx)
}

func (r *registration) SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error {
	_, err := r.session.SendStepActionEvent(ctx, ev)
	return err
}

// OpenDurable implements link.Registration over the session's DurableTaskListener, one per
// registration, multiplexed by task external id and invocation.
func (r *registration) OpenDurable(_ context.Context, taskExternalId string, invocation int32) (link.DurableChannel, error) {
	r.mu.Lock()

	if r.closed {
		r.mu.Unlock()
		return nil, errors.New("registration closed")
	}

	if r.hub == nil {
		r.hub = newDurableHub(r.session)
	}

	hub := r.hub
	r.mu.Unlock()

	return hub.open(taskExternalId, invocation)
}

func (r *registration) Close() error {
	r.mu.Lock()
	r.closed = true
	hub := r.hub
	r.mu.Unlock()

	if hub != nil {
		hub.closeAll()
	}

	return r.session.Close()
}
