package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

// namespaceSeparator joins the endpoint namespace to the names it prefixes, the way the SDK's
// HATCHET_CLIENT_NAMESPACE does (client.WithNamespace appends the same underscore).
const namespaceSeparator = "_"

// namespaceLen is the length of the prefix without its separator: a hyphenated uuid.
const namespaceLen = 36

var errEndpointNotFound = errors.New("endpoint not found for namespace")

func namespacePrefix(ns uuid.UUID) string {
	return ns.String() + namespaceSeparator
}

// prefixName applies the namespace to a workflow name or event key. Already-prefixed names
// are left alone so applying twice is harmless, as clientconfig.ApplyNamespace does.
func prefixName(ns uuid.UUID, name string) string {
	prefix := namespacePrefix(ns)

	if strings.HasPrefix(name, prefix) {
		return name
	}

	return prefix + name
}

// prefixAction normalizes an action id the way the engine stores it (types.ParseActionID:
// service first letter lowered, verb lowered) and prefixes the service part. Re-parsing the
// result is a no-op because the prefix starts with a hex digit.
func prefixAction(ns uuid.UUID, action string) (string, error) {
	parsed, err := types.ParseActionID(action)

	if err != nil {
		return "", err
	}

	parsed.Service = prefixName(ns, parsed.Service)

	return parsed.String(), nil
}

// applyNamespace returns a deep copy of wf with the namespace applied to the workflow name,
// the task and on-failure actions and the event trigger keys. Cron triggers are cron
// expressions, not names, so they are left alone.
func applyNamespace(wf *v1.CreateWorkflowVersionRequest, ns uuid.UUID) (*v1.CreateWorkflowVersionRequest, error) {
	if wf == nil {
		return nil, errors.New("workflow is required")
	}

	out, ok := proto.Clone(wf).(*v1.CreateWorkflowVersionRequest)

	if !ok {
		return nil, errors.New("could not clone workflow")
	}

	out.Name = prefixName(ns, out.Name)

	for i, key := range out.EventTriggers {
		out.EventTriggers[i] = prefixName(ns, key)
	}

	tasks := append([]*v1.CreateTaskOpts{}, out.Tasks...)

	if out.OnFailureTask != nil {
		tasks = append(tasks, out.OnFailureTask)
	}

	for i, task := range tasks {
		if task == nil {
			return nil, fmt.Errorf("workflow %s: task at index %d is nil", wf.Name, i)
		}

		if task.Action == "" {
			return nil, fmt.Errorf("workflow %s: task %s is missing required field 'Action'", wf.Name, task.ReadableId)
		}

		action, err := prefixAction(ns, task.Action)

		if err != nil {
			return nil, fmt.Errorf("workflow %s: %w", wf.Name, err)
		}

		task.Action = action
	}

	return out, nil
}

// actionsForWorkflow mirrors grpcoperator.actionsForWorkflow: the action ids a worker must
// register to run every task of the workflow, on-failure task included.
func actionsForWorkflow(wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	if wf == nil {
		return nil, errors.New("workflow is required")
	}

	tasks := append([]*v1.CreateTaskOpts{}, wf.Tasks...)

	if wf.OnFailureTask != nil {
		tasks = append(tasks, wf.OnFailureTask)
	}

	actions := make([]string, 0, len(tasks))

	for i, task := range tasks {
		if task == nil {
			return nil, fmt.Errorf("workflow %s: task at index %d is nil", wf.Name, i)
		}

		if task.Action == "" {
			return nil, fmt.Errorf("workflow %s: task at index %d is missing required field 'Action'", wf.Name, i)
		}

		parsed, err := types.ParseActionID(task.Action)

		if err != nil {
			return nil, fmt.Errorf("workflow %s: %w", wf.Name, err)
		}

		actions = append(actions, parsed.String())
	}

	return actions, nil
}

// ParseNamespace extracts the endpoint namespace from a registered action id of the form
// <uuid>_<service>:<verb>.
func ParseNamespace(actionId string) (uuid.UUID, bool) {
	if len(actionId) <= namespaceLen || actionId[namespaceLen:namespaceLen+1] != namespaceSeparator {
		return uuid.Nil, false
	}

	ns, err := uuid.Parse(actionId[:namespaceLen])

	if err != nil {
		return uuid.Nil, false
	}

	return ns, true
}

// sortedUnion merges action lists, dropping duplicates and empties, and sorts the result so
// two unions compare with slices.Equal.
func sortedUnion(lists ...[]string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)

	for _, list := range lists {
		for _, action := range list {
			if action == "" {
				continue
			}

			if _, ok := seen[action]; ok {
				continue
			}

			seen[action] = struct{}{}
			out = append(out, action)
		}
	}

	sort.Strings(out)

	return out
}

// endpointConfig is the mutable, row-derived part of a cached endpoint. It is replaced as a
// whole on refresh so readers take one consistent snapshot.
type endpointConfig struct {
	secretErr             error
	name                  string
	healthcheckUrl        string
	triggerUrl            string
	secret                string
	secretEnc             string
	statusError           string
	registeredActions     []string
	updatedAt             time.Time
	slots                 int32
	durableSlots          int32
	requestTimeoutSeconds int32
	pollIntervalSeconds   int32
	inlineWaitBudgetMs    int32
	enabled               bool
	healthy               bool
	healthKnown           bool
}

// cachedEndpoint is one endpoint of a served tenant. Identity fields never change; cfg is
// swapped under the cache lock; workflows are what this process learned from the endpoint's
// healthcheck (namespaced), also guarded by the cache lock. The limiters survive refreshes
// so in-flight deliveries keep their slot: limiter bounds non-durable deliveries,
// durableLimiter bounds open durable websockets.
type cachedEndpoint struct {
	cfg            *endpointConfig
	limiter        *slotLimiter
	durableLimiter *slotLimiter
	workflows      []*v1.CreateWorkflowVersionRequest
	id             uuid.UUID
	tenantId       uuid.UUID
	namespace      uuid.UUID
	shard          int32
}

// routingCache is a served tenant's endpoints keyed by namespace and by id, with decrypted
// secrets and the union of registered_actions over enabled endpoints. Load reads every row;
// Refresh reads rows updated since the last read. Health flips do not bump updated_at, so
// the cache's health view is whatever the owner last wrote plus this process's own writes.
type routingCache struct {
	byNamespace map[uuid.UUID]*cachedEndpoint
	byId        map[uuid.UUID]*cachedEndpoint
	repo        repository.ServerlessEndpointRepository
	enc         encryption.EncryptionService
	l           *zerolog.Logger
	union       []string
	since       time.Time
	lastLoad    time.Time
	tenantId    uuid.UUID
	mu          sync.RWMutex
}

func newRoutingCache(tenantId uuid.UUID, repo repository.ServerlessEndpointRepository, enc encryption.EncryptionService, l *zerolog.Logger) *routingCache {
	return &routingCache{
		tenantId:    tenantId,
		repo:        repo,
		enc:         enc,
		l:           l,
		byNamespace: map[uuid.UUID]*cachedEndpoint{},
		byId:        map[uuid.UUID]*cachedEndpoint{},
	}
}

// Load replaces the cache with the tenant's current rows, dropping endpoints that vanished.
func (c *routingCache) Load(ctx context.Context) error {
	rows, err := c.repo.ListForTenant(ctx, c.tenantId)

	if err != nil {
		return fmt.Errorf("could not load endpoints for tenant %s: %w", c.tenantId, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	seen := make(map[uuid.UUID]struct{}, len(rows))

	for _, row := range rows {
		seen[row.ID] = struct{}{}
		c.upsertLocked(row)
	}

	for id, ep := range c.byId {
		if _, ok := seen[id]; ok {
			continue
		}

		delete(c.byId, id)
		delete(c.byNamespace, ep.namespace)
	}

	c.lastLoad = time.Now()
	c.recomputeUnionLocked()

	return nil
}

// Refresh applies rows updated since the last Load or Refresh. Deleted endpoints are not
// visible here; Load drops them.
func (c *routingCache) Refresh(ctx context.Context) error {
	c.mu.RLock()
	since := c.since
	c.mu.RUnlock()

	// Overlap by a second so a row committed with the same updated_at as the previous
	// watermark is not skipped; re-applying a row is idempotent.
	rows, err := c.repo.ListUpdatedSince(ctx, c.tenantId, since.Add(-time.Second))

	if err != nil {
		return fmt.Errorf("could not refresh endpoints for tenant %s: %w", c.tenantId, err)
	}

	if len(rows) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, row := range rows {
		c.upsertLocked(row)
	}

	c.recomputeUnionLocked()

	return nil
}

// LastLoad is when the cache was last fully loaded; the runner schedules full reloads on it.
func (c *routingCache) LastLoad() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastLoad
}

func (c *routingCache) upsertLocked(row *sqlcv1.V1ServerlessEndpoint) {
	ep, ok := c.byId[row.ID]

	if !ok {
		ep = &cachedEndpoint{
			id:             row.ID,
			tenantId:       row.TenantID,
			namespace:      row.Namespace,
			shard:          row.Shard,
			limiter:        newSlotLimiter(int(row.Slots)),
			durableLimiter: newSlotLimiter(int(row.DurableSlots)),
		}

		c.byId[row.ID] = ep
		c.byNamespace[row.Namespace] = ep
	}

	prev := ep.cfg

	cfg := &endpointConfig{
		name:                  row.Name,
		healthcheckUrl:        row.HealthcheckUrl,
		triggerUrl:            row.TriggerUrl,
		secretEnc:             row.SigningSecretEnc,
		slots:                 row.Slots,
		durableSlots:          row.DurableSlots,
		requestTimeoutSeconds: row.RequestTimeoutSeconds,
		pollIntervalSeconds:   row.PollIntervalSeconds,
		inlineWaitBudgetMs:    row.InlineWaitBudgetMs,
		enabled:               row.Enabled,
		healthKnown:           row.Healthy.Valid,
		healthy:               row.Healthy.Valid && row.Healthy.Bool,
		registeredActions:     append([]string{}, row.RegisteredActions...),
	}

	if row.StatusError.Valid {
		cfg.statusError = row.StatusError.String
	}

	if row.UpdatedAt.Valid {
		cfg.updatedAt = row.UpdatedAt.Time
	}

	// Decrypt only when the ciphertext changed: decryption is the expensive part of a
	// refresh and secrets rotate rarely.
	if prev != nil && prev.secretEnc == cfg.secretEnc {
		cfg.secret = prev.secret
		cfg.secretErr = prev.secretErr
	} else {
		cfg.secret, cfg.secretErr = c.decryptSecret(cfg.secretEnc)

		if cfg.secretErr != nil {
			c.l.Error().Err(cfg.secretErr).Str("endpoint_id", row.ID.String()).Msg("could not decrypt endpoint signing secret")
		}
	}

	// A status this process wrote itself is newer than a row read before the write landed
	// only when updated_at did not move; keep the local health view in that case.
	if prev != nil && prev.healthKnown && cfg.updatedAt.Equal(prev.updatedAt) {
		cfg.healthKnown = prev.healthKnown
		cfg.healthy = prev.healthy
		cfg.statusError = prev.statusError
	}

	if prev == nil || prev.slots != cfg.slots {
		ep.limiter.resize(int(cfg.slots))
	}

	if prev == nil || prev.durableSlots != cfg.durableSlots {
		ep.durableLimiter.resize(int(cfg.durableSlots))
	}

	ep.cfg = cfg

	if cfg.updatedAt.After(c.since) {
		c.since = cfg.updatedAt
	}
}

func (c *routingCache) decryptSecret(enc string) (string, error) {
	if enc == "" {
		return "", errors.New("endpoint has no signing secret")
	}

	if c.enc == nil {
		return "", errors.New("no encryption service configured")
	}

	secret, err := c.enc.DecryptString(enc, contract.SigningSecretEncryptionDataID)

	if err != nil {
		return "", fmt.Errorf("could not decrypt signing secret: %w", err)
	}

	return secret, nil
}

func (c *routingCache) recomputeUnionLocked() {
	lists := make([][]string, 0, len(c.byId))

	for _, ep := range c.byId {
		if ep.cfg.enabled {
			lists = append(lists, ep.cfg.registeredActions)
		}
	}

	c.union = sortedUnion(lists...)
}

// ActionUnion is the sorted union of registered_actions over the tenant's enabled endpoints:
// the action set every registration for the tenant advertises.
func (c *routingCache) ActionUnion() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return append([]string{}, c.union...)
}

// Workflows returns the namespaced workflows this process knows for the tenant's enabled
// endpoints, for the Open call of a new registration.
func (c *routingCache) Workflows() []*v1.CreateWorkflowVersionRequest {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*v1.CreateWorkflowVersionRequest, 0)

	for _, ep := range c.byId {
		if ep.cfg.enabled {
			out = append(out, ep.workflows...)
		}
	}

	return out
}

// Endpoints snapshots the tenant's endpoints (enabled or not).
func (c *routingCache) Endpoints() []*cachedEndpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*cachedEndpoint, 0, len(c.byId))

	for _, ep := range c.byId {
		out = append(out, ep)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].id.String() < out[j].id.String()
	})

	return out
}

func (c *routingCache) Endpoint(id uuid.UUID) (*cachedEndpoint, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ep, ok := c.byId[id]

	return ep, ok
}

// Config snapshots an endpoint's mutable configuration.
func (c *routingCache) Config(ep *cachedEndpoint) *endpointConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return ep.cfg
}

func (c *routingCache) lookup(ns uuid.UUID) (*cachedEndpoint, *endpointConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ep, ok := c.byNamespace[ns]

	if !ok || !ep.cfg.enabled {
		return nil, nil, false
	}

	return ep, ep.cfg, true
}

// Route resolves the endpoint an action id belongs to. A miss triggers one full reload,
// which also drops deleted endpoints, before failing with errEndpointNotFound.
func (c *routingCache) Route(ctx context.Context, actionId string) (*cachedEndpoint, *endpointConfig, error) {
	ns, ok := ParseNamespace(actionId)

	if !ok {
		return nil, nil, fmt.Errorf("%w: action %q carries no namespace", errEndpointNotFound, actionId)
	}

	if ep, cfg, ok := c.lookup(ns); ok {
		return ep, cfg, nil
	}

	if err := c.Load(ctx); err != nil {
		return nil, nil, err
	}

	if ep, cfg, ok := c.lookup(ns); ok {
		return ep, cfg, nil
	}

	return nil, nil, fmt.Errorf("%w: %s", errEndpointNotFound, ns)
}

// SetHealthcheck records what an owned endpoint's healthcheck produced: its namespaced
// workflows and action set. The action set replaces registered_actions in the cache so the
// union changes here as soon as the owner writes it, without waiting for a refresh. It
// returns whether the union changed.
func (c *routingCache) SetHealthcheck(id uuid.UUID, workflows []*v1.CreateWorkflowVersionRequest, actions []string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	ep, ok := c.byId[id]

	if !ok {
		return false
	}

	ep.workflows = workflows

	cfg := *ep.cfg
	cfg.registeredActions = append([]string{}, actions...)
	ep.cfg = &cfg

	before := c.union
	c.recomputeUnionLocked()

	return !stringsEqual(before, c.union)
}

// SetStatus records a status transition this process wrote, so a later refresh that returns
// the row unchanged does not resurrect the old value.
func (c *routingCache) SetStatus(id uuid.UUID, healthy bool, statusError string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ep, ok := c.byId[id]

	if !ok {
		return
	}

	cfg := *ep.cfg
	cfg.healthKnown = true
	cfg.healthy = healthy
	cfg.statusError = statusError
	ep.cfg = &cfg
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
