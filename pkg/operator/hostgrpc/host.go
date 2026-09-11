// Package hostgrpc is the out-of-process operator host: it implements pkg/operator.Host over
// pkg/client/operatorclient, speaking OperatorService to the engine with a per-tenant API
// token from a TokenSource. It depends on the client packages and pkg/operator only, so an
// operator binary links it without the engine. The legacy pkg/client is used for two things
// only: turning a token and the HATCHET_CLIENT_* environment into a connection, and the
// durable task listener the Go SDK worker shares.
package hostgrpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/client/operatorclient"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ClientFactory builds the engine client for one token. The default derives the gRPC address
// and TLS settings from the token's claims and HATCHET_CLIENT_* environment, like the SDK.
type ClientFactory func(token string) (engineClient, error)

// Opt configures New.
type Opt func(*opts)

type opts struct {
	tokens    TokenSource
	l         *zerolog.Logger
	newClient ClientFactory
}

func defaultOpts() *opts {
	l := zerolog.Nop()

	return &opts{l: &l}
}

// WithTokenSource sets where the host asks for each tenant's API token. It is required.
func WithTokenSource(tokens TokenSource) Opt {
	return func(o *opts) { o.tokens = tokens }
}

// WithLogger sets the host's logger; the default discards everything.
func WithLogger(l *zerolog.Logger) Opt {
	return func(o *opts) { o.l = l }
}

// WithClientFactory replaces the default client factory; tests inject a fake.
func WithClientFactory(f ClientFactory) Opt {
	return func(o *opts) { o.newClient = f }
}

type cachedClient struct {
	client engineClient
	token  string
}

// closeClient closes the client's gRPC connection. Dropping the reference alone would leave
// the connection's goroutines behind.
func closeClient(c engineClient) {
	_ = c.Close()
}

// Host caches one engine client per tenant. The token is asked from the source on every Open
// so a rotated token replaces the cached client, and the client is closed when it is replaced,
// when ReleaseTenant reports the tenant is no longer served, or when the host closes.
type Host struct {
	tokens    TokenSource
	l         *zerolog.Logger
	newClient ClientFactory
	clients   map[uuid.UUID]*cachedClient
	mu        sync.Mutex
}

// New builds a Host. WithTokenSource is required; the other options have defaults.
func New(fs ...Opt) (*Host, error) {
	o := defaultOpts()

	for _, f := range fs {
		f(o)
	}

	if o.tokens == nil {
		return nil, fmt.Errorf("a token source is required. use WithTokenSource")
	}

	if o.l == nil {
		return nil, fmt.Errorf("a logger is required. use WithLogger or omit it for the default")
	}

	if o.newClient == nil {
		o.newClient = defaultClientFactory(o.l)
	}

	return &Host{
		tokens:    o.tokens,
		l:         o.l,
		newClient: o.newClient,
		clients:   map[uuid.UUID]*cachedClient{},
	}, nil
}

// defaultClientFactory wraps the legacy dial, which panics when the SDK config cannot be loaded
// from the token and environment. The panic is turned into an error so one tenant's bad
// token cannot take the process down.
func defaultClientFactory(l *zerolog.Logger) ClientFactory {
	return func(token string) (c engineClient, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("could not build engine client: %v", r)
			}
		}()

		return dialEngine(token, l)
	}
}

// clientFor returns the cached client for the tenant, rebuilding it when the token changed.
func (h *Host) clientFor(tenantId uuid.UUID, token string) (engineClient, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cached, ok := h.clients[tenantId]

	if ok && cached.token == token {
		return cached.client, nil
	}

	c, err := h.newClient(token)

	if err != nil {
		return nil, err
	}

	if ok {
		closeClient(cached.client)
	}

	h.clients[tenantId] = &cachedClient{client: c, token: token}

	return c, nil
}

func (h *Host) evict(tenantId uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cached, ok := h.clients[tenantId]; ok {
		closeClient(cached.client)
		delete(h.clients, tenantId)
	}
}

// ReleaseTenant closes the tenant's cached client. A multi-tenant operator calls it once it
// serves no more of the tenant, after every session for the tenant is closed.
func (h *Host) ReleaseTenant(tenantId uuid.UUID) {
	h.evict(tenantId)
}

// Close closes every cached client. Sessions are closed by whoever opened them, before the
// host.
func (h *Host) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for tenantId, cached := range h.clients {
		closeClient(cached.client)
		delete(h.clients, tenantId)
	}
}

// Open implements operator.Host. The identity must name the operator by name: OperatorService
// upserts one GRPC row per (tenant, name), so an existing row cannot be opened by id, and a kind
// other than GRPC is refused. An Unauthenticated connect drops the cached client and asks the
// source once more, so a token rotated between two Opens is used without waiting for the
// source's own reload. The initial action set is streamed to the engine right after the connect
// and flushed before the session is returned; the client keeps it as the desired set and
// replays it when a reconnect does not resume the worker. Assigned actions reach opts.Handler
// from the session's deliver loop, which runs until Close.
func (h *Host) Open(ctx context.Context, id operator.Identity, opts operator.OpenOpts) (operator.Session, error) {
	if opts.Handler == nil {
		return nil, errors.New("hostgrpc: an action handler is required")
	}

	if id.OperatorId != nil {
		return nil, fmt.Errorf("hostgrpc: an existing operator row cannot be opened by id over OperatorService: %w", operator.ErrNotSupported)
	}

	if id.Kind != "" && id.Kind != sqlcv1.V1OperatorKindGRPC {
		return nil, fmt.Errorf("hostgrpc: operator kind %s cannot be registered over OperatorService: %w", id.Kind, operator.ErrNotSupported)
	}

	if id.Name == "" {
		return nil, errors.New("hostgrpc: an operator name is required")
	}

	if opts.ResumeWorkerId != nil {
		return nil, fmt.Errorf("hostgrpc: resuming a named worker: %w", operator.ErrNotSupported)
	}

	if opts.WorkerName != "" {
		return nil, fmt.Errorf("hostgrpc: naming the worker: %w", operator.ErrNotSupported)
	}

	cs, err := h.connect(ctx, id, opts)

	if err != nil && status.Code(err) == codes.Unauthenticated {
		h.evict(id.TenantId)
		cs, err = h.connect(ctx, id, opts)
	}

	if err != nil {
		return nil, err
	}

	// The engine reports the tenant it authenticated the token as; the session must belong
	// to the tenant the identity names, whatever the source handed out.
	reg, err := parseRegistration(cs.Registration())

	if err != nil {
		_ = cs.Close(operatorclient.WithoutDrain())
		return nil, err
	}

	if reg.TenantId != id.TenantId {
		_ = cs.Close(operatorclient.WithoutDrain())

		return nil, fmt.Errorf("hostgrpc: session for tenant %s was authenticated as tenant %s; the token source is misconfigured", id.TenantId, reg.TenantId)
	}

	if len(opts.Actions) > 0 {
		cs.AddActions(opts.Actions...)

		if err := cs.Flush(ctx); err != nil {
			_ = cs.Close(operatorclient.WithoutDrain())
			return nil, fmt.Errorf("hostgrpc: could not register initial actions for tenant %s: %w", id.TenantId, err)
		}
	}

	s := newSession(cs, reg, opts.Handler, h.l)

	if err := s.startDelivery(); err != nil {
		_ = cs.Close(operatorclient.WithoutDrain())
		return nil, err
	}

	return s, nil
}

func (h *Host) connect(ctx context.Context, id operator.Identity, opts operator.OpenOpts) (operatorclient.Session, error) {
	token, err := h.tokens.Token(ctx, id.TenantId)

	if err != nil {
		if errors.Is(err, ErrNoToken) {
			return nil, fmt.Errorf("tenant %s: %w", id.TenantId, ErrNoToken)
		}

		return nil, fmt.Errorf("could not resolve token for tenant %s: %w", id.TenantId, err)
	}

	// A token whose tenant claim names another tenant is refused before a client is built;
	// the engine's own answer is checked again after connecting.
	if claims, err := loaderutils.GetConfFromJWT(token); err == nil && claims.TenantId != "" && claims.TenantId != id.TenantId.String() {
		return nil, fmt.Errorf("token for tenant %s carries the tenant claim %s; the token source is misconfigured", id.TenantId, claims.TenantId)
	}

	c, err := h.clientFor(id.TenantId, token)

	if err != nil {
		return nil, fmt.Errorf("tenant %s: %w", id.TenantId, err)
	}

	labels := make(map[string]interface{}, len(opts.Labels))

	for k, v := range opts.Labels {
		labels[k] = v
	}

	return c.Operator().Connect(ctx, &operatorclient.ConnectRequest{
		Name:       id.Name,
		SlotConfig: opts.SlotConfig,
		Labels:     labels,
	})
}

// parseRegistration turns the client's string ids into the contract's.
func parseRegistration(reg operatorclient.Registration) (operator.Registration, error) {
	tenantId, err := uuid.Parse(reg.TenantId)

	if err != nil {
		return operator.Registration{}, fmt.Errorf("hostgrpc: the engine returned tenant id %q: %w", reg.TenantId, err)
	}

	operatorId, err := uuid.Parse(reg.OperatorId)

	if err != nil {
		return operator.Registration{}, fmt.Errorf("hostgrpc: the engine returned operator id %q: %w", reg.OperatorId, err)
	}

	workerId, err := uuid.Parse(reg.WorkerId)

	if err != nil {
		return operator.Registration{}, fmt.Errorf("hostgrpc: the engine returned worker id %q: %w", reg.WorkerId, err)
	}

	return operator.Registration{TenantId: tenantId, OperatorId: operatorId, WorkerId: workerId, Resumed: reg.Resumed}, nil
}
