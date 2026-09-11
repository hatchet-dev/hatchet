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

// cachedClient is one tenant's engine client and the sessions that use it. refs counts the
// sessions holding it; evicted records that the host no longer hands it out (its token was
// rotated, its tenant released or the host closed). An evicted client is closed once its last
// session releases it, so a teardown of one session never closes the connection another
// session of the same tenant is still using. All fields are guarded by the host's mu.
type cachedClient struct {
	client  engineClient
	token   string
	refs    int
	evicted bool
	closed  bool
}

// closeLocked closes the client's gRPC connection once. Dropping the reference alone would
// leave the connection's goroutines behind. The caller holds the host's mu.
func (c *cachedClient) closeLocked() {
	if c.closed {
		return
	}

	c.closed = true
	_ = c.client.Close()
}

// Host caches one engine client per tenant. The token is asked from the source on every
// connect so a rotated token replaces the cached client. A client is evicted when it is
// replaced, when ReleaseTenant reports the tenant is no longer served, or when the host
// closes; it is closed when it is evicted and no session holds it any more.
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

// clientFor returns the cached client for the tenant, rebuilding it when the token changed,
// with one reference on it held for the caller. The caller releases the reference exactly
// once, when the session it opened over the client is gone. A replaced client is evicted:
// closed now when nothing holds it, otherwise by its last release.
func (h *Host) clientFor(tenantId uuid.UUID, token string) (engineClient, func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cached, ok := h.clients[tenantId]

	if !ok || cached.token != token {
		c, err := h.newClient(token)

		if err != nil {
			return nil, nil, err
		}

		if ok {
			h.evictLocked(tenantId, cached)
		}

		cached = &cachedClient{client: c, token: token}
		h.clients[tenantId] = cached
	}

	cached.refs++

	return cached.client, h.releaser(cached), nil
}

// releaser returns the release for one reference on cached; it runs once.
func (h *Host) releaser(cached *cachedClient) func() {
	var once sync.Once

	return func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()

			cached.refs--

			if cached.evicted && cached.refs <= 0 {
				cached.closeLocked()
			}
		})
	}
}

// evictLocked drops cached from the map and closes it unless a session still holds it, in
// which case the last release closes it. The caller holds mu.
func (h *Host) evictLocked(tenantId uuid.UUID, cached *cachedClient) {
	cached.evicted = true

	if h.clients[tenantId] == cached {
		delete(h.clients, tenantId)
	}

	if cached.refs <= 0 {
		cached.closeLocked()
	}
}

func (h *Host) evict(tenantId uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cached, ok := h.clients[tenantId]; ok {
		h.evictLocked(tenantId, cached)
	}
}

// ReleaseTenant evicts the tenant's cached client: the next Open builds a new one. A
// multi-tenant operator calls it once it serves no more of the tenant. A session of the tenant
// that is still open (one being drained, or one opened again while an older one is torn down)
// keeps the client until it closes; the eviction only stops the client being handed out.
func (h *Host) ReleaseTenant(tenantId uuid.UUID) {
	h.evict(tenantId)
}

// Close closes every cached client, whether or not a session still holds it. Sessions are
// closed by whoever opened them, before the host.
func (h *Host) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for tenantId, cached := range h.clients {
		cached.evicted = true
		cached.closeLocked()
		delete(h.clients, tenantId)
	}
}

// Open implements operator.Host. The identity must name the operator by name: OperatorService
// upserts one GRPC row per (tenant, name), so an existing row cannot be opened by id, and a kind
// other than GRPC is refused. An Unauthenticated connect drops the cached client and asks the
// source once more, so a token rotated between two Opens is used without waiting for the
// source's own reload. The initial action set is streamed to the engine right after the connect
// and flushed before the session is returned; the session keeps it as the desired set and
// restores it when it has to open a new client session. Assigned actions reach opts.Handler
// from the session's delivery, which runs until Close.
//
// The session supervises its client session: when the client's stream fails for good (its
// token was revoked, say) the session connects again through the host, which asks the source
// for the tenant's current token, and resumes delivery. It gives up, and reports so through
// Done and Err, only on a failure no retry can fix.
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

	// The engine registers every wire session as SELF: the row is kept alive by this host's
	// stream, and an engine-leased row can never be driven over the wire.
	if id.Leasing != "" && id.Leasing != sqlcv1.V1OperatorLeasingSELF {
		return nil, fmt.Errorf("hostgrpc: operator leasing %s cannot be registered over OperatorService: %w", id.Leasing, operator.ErrNotSupported)
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

	cs, reg, release, err := h.openClientSession(ctx, id, opts)

	if err != nil {
		return nil, err
	}

	if len(opts.Actions) > 0 {
		cs.AddActions(opts.Actions...)

		if err := cs.Flush(ctx); err != nil {
			_ = cs.Close(operatorclient.WithoutDrain())
			release()

			return nil, fmt.Errorf("hostgrpc: could not register initial actions for tenant %s: %w", id.TenantId, err)
		}
	}

	s := newSession(cs, reg, opts.Handler, h.l)
	s.release = release
	s.startQueueSize = startQueueSizeFor(opts.SlotConfig)
	s.addDesired(opts.Actions)
	s.reconnect = func(ctx context.Context) (operatorclient.Session, operator.Registration, func(), error) {
		return h.openClientSession(ctx, id, opts)
	}

	if err := s.startDelivery(); err != nil {
		_ = cs.Close(operatorclient.WithoutDrain())
		release()

		return nil, err
	}

	return s, nil
}

// errTenantMismatch marks a session the engine authenticated as another tenant than the one
// the identity names: the token source is misconfigured, and no retry fixes that.
var errTenantMismatch = errors.New("hostgrpc: the token source is misconfigured")

// openClientSession connects a client session for the identity and verifies it: an
// Unauthenticated connect drops the cached client and asks the source once more, and the
// tenant the engine authenticated the token as must be the one the identity names, whatever
// the source handed out. The returned release gives the client reference back and is called
// once, when the client session is gone.
func (h *Host) openClientSession(ctx context.Context, id operator.Identity, opts operator.OpenOpts) (operatorclient.Session, operator.Registration, func(), error) {
	cs, release, err := h.connect(ctx, id, opts)

	if err != nil && status.Code(err) == codes.Unauthenticated {
		h.evict(id.TenantId)
		cs, release, err = h.connect(ctx, id, opts)
	}

	if err != nil {
		return nil, operator.Registration{}, nil, err
	}

	reg, err := parseRegistration(cs.Registration())

	if err != nil {
		_ = cs.Close(operatorclient.WithoutDrain())
		release()

		return nil, operator.Registration{}, nil, err
	}

	if reg.TenantId != id.TenantId {
		_ = cs.Close(operatorclient.WithoutDrain())
		release()

		return nil, operator.Registration{}, nil, fmt.Errorf("session for tenant %s was authenticated as tenant %s: %w", id.TenantId, reg.TenantId, errTenantMismatch)
	}

	return cs, reg, release, nil
}

// connect asks the source for the tenant's token, takes a reference on the tenant's client and
// connects. The reference is released here when the connect fails.
func (h *Host) connect(ctx context.Context, id operator.Identity, opts operator.OpenOpts) (operatorclient.Session, func(), error) {
	token, err := h.tokens.Token(ctx, id.TenantId)

	if err != nil {
		if errors.Is(err, ErrNoToken) {
			return nil, nil, fmt.Errorf("tenant %s: %w", id.TenantId, ErrNoToken)
		}

		return nil, nil, fmt.Errorf("could not resolve token for tenant %s: %w", id.TenantId, err)
	}

	// A token whose tenant claim names another tenant is refused before a client is built;
	// the engine's own answer is checked again after connecting.
	if claims, err := loaderutils.GetConfFromJWT(token); err == nil && claims.TenantId != "" && claims.TenantId != id.TenantId.String() {
		return nil, nil, fmt.Errorf("token for tenant %s carries the tenant claim %s: %w", id.TenantId, claims.TenantId, errTenantMismatch)
	}

	c, release, err := h.clientFor(id.TenantId, token)

	if err != nil {
		return nil, nil, fmt.Errorf("tenant %s: %w", id.TenantId, err)
	}

	labels := make(map[string]interface{}, len(opts.Labels))

	for k, v := range opts.Labels {
		labels[k] = v
	}

	cs, err := c.Operator().Connect(ctx, &operatorclient.ConnectRequest{
		Name:       id.Name,
		SlotConfig: opts.SlotConfig,
		Labels:     labels,
	})

	if err != nil {
		release()
		return nil, nil, err
	}

	return cs, release, nil
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
