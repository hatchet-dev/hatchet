package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

// namespaceLen is the length of the prefix without its separator: a hyphenated uuid.
const namespaceLen = 36

var errEndpointNotFound = errors.New("endpoint not found for namespace")

// namespacePrefix is what prefixName prepends: contract.NamespacePrefix over the endpoint's
// namespace, joined by the separator the SDK's HATCHET_CLIENT_NAMESPACE uses.
func namespacePrefix(ns uuid.UUID) string {
	return contract.NamespacePrefix(ns.String())
}

// prefixName applies the namespace to a workflow name, action service or event key through
// contract.ApplyNamespace, the rule the durable relay applies to the names an endpoint
// references in nested requests, so what the operator registers and what the relay confines
// an endpoint to are prefixed by one implementation. Already-prefixed names are left alone,
// so applying twice is harmless.
func prefixName(ns uuid.UUID, name string) string {
	return contract.ApplyNamespace(ns.String(), name)
}

// prefixAction normalizes an action id the way the engine stores it (types.ParseActionID:
// service first letter lowered, verb lowered) and prefixes the service part. Re-parsing the
// result is a no-op because the prefix starts with a hex digit. The engine's rule that both
// parts are non-empty is checked before the prefix, which would otherwise hide an empty
// service.
func prefixAction(ns uuid.UUID, action string) (string, error) {
	parsed, err := types.ParseActionID(action)

	if err != nil {
		return "", err
	}

	if parsed.Service == "" || parsed.Verb == "" {
		return "", fmt.Errorf("invalid action %q: service and verb are required", action)
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
	if len(actionId) <= namespaceLen || actionId[namespaceLen:namespaceLen+1] != contract.NamespaceSeparator {
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
	statusChangedAt       time.Time
	requestTimeoutSeconds int32
	pollIntervalSeconds   int32
	inlineWaitBudgetMs    int32
	enabled               bool
	healthy               bool
	healthKnown           bool
}

// version is the row version the cache orders refreshes by: the later of updated_at and
// status_changed_at, the same expression ListUpdatedSince keys on.
func (cfg *endpointConfig) version() time.Time {
	if cfg.statusChangedAt.After(cfg.updatedAt) {
		return cfg.statusChangedAt
	}

	return cfg.updatedAt
}

// contribution is what the endpoint adds to the tenant's action union: its registered
// actions while enabled, nothing otherwise.
func (cfg *endpointConfig) contribution() []string {
	if cfg == nil || !cfg.enabled {
		return nil
	}

	return cfg.registeredActions
}

// cachedEndpoint is one endpoint of a served tenant. Identity fields never change; cfg is
// swapped under the cache lock.
type cachedEndpoint struct {
	cfg       *endpointConfig
	id        uuid.UUID
	tenantId  uuid.UUID
	namespace uuid.UUID
	shard     int32
}

// unionDelta is one published change of the union: the actions that entered it and the ones
// that left it between revision rev-1 and rev.
type unionDelta struct {
	added   []string
	removed []string
	rev     uint64
}

// unionLogSize bounds the delta log; a registration further behind than that diffs against
// the full union instead.
const unionLogSize = 64

// routingCache is a served tenant's endpoints keyed by namespace and by id, with decrypted
// secrets and the union of registered_actions over enabled endpoints.
//
// The union is a reference count per action: how many enabled endpoints advertise it. An
// action enters the union on the 0 to 1 transition and leaves it on 1 to 0, so a change to
// one endpoint costs that endpoint's actions, whatever the tenant's size. Every batch of row
// changes (a load, a refresh, a page of a gained unit) publishes at most one revision, with
// its delta appended to a bounded log; registrations catch up from the log by revision and
// only fall back to a full diff when they are further behind than the log reaches. The sorted
// form is built on demand, once per revision.
//
// Load reads every row; Refresh reads rows whose version (the later of updated_at and
// status_changed_at) is past the watermark, so configuration, registered_actions and status
// changes all surface. Rows already applied at the same version are skipped.
type routingCache struct {
	byNamespace map[uuid.UUID]*cachedEndpoint
	byId        map[uuid.UUID]*cachedEndpoint
	repo        repository.ServerlessEndpointRepository
	enc         encryption.EncryptionService
	l           *zerolog.Logger

	counts    map[string]int
	sorted    []string
	log       []unionDelta
	since     time.Time
	lastLoad  time.Time
	tenantId  uuid.UUID
	sinceId   uuid.UUID
	rev       uint64
	sortedRev uint64
	mu        sync.RWMutex
}

func newRoutingCache(tenantId uuid.UUID, repo repository.ServerlessEndpointRepository, enc encryption.EncryptionService, l *zerolog.Logger) *routingCache {
	return &routingCache{
		tenantId:    tenantId,
		repo:        repo,
		enc:         enc,
		l:           l,
		byNamespace: map[uuid.UUID]*cachedEndpoint{},
		byId:        map[uuid.UUID]*cachedEndpoint{},
		counts:      map[string]int{},
	}
}

// batch accumulates the union delta of one group of row changes; publishLocked turns it
// into a revision.
type batch struct {
	added   map[string]struct{}
	removed map[string]struct{}
}

func newBatch() *batch {
	return &batch{added: map[string]struct{}{}, removed: map[string]struct{}{}}
}

// enter counts one more enabled endpoint advertising action.
func (b *batch) enter(c *routingCache, action string) {
	c.counts[action]++

	if c.counts[action] != 1 {
		return
	}

	if _, ok := b.removed[action]; ok {
		delete(b.removed, action)
		return
	}

	b.added[action] = struct{}{}
}

// leave counts one fewer enabled endpoint advertising action.
func (b *batch) leave(c *routingCache, action string) {
	c.counts[action]--

	if c.counts[action] > 0 {
		return
	}

	delete(c.counts, action)

	if _, ok := b.added[action]; ok {
		delete(b.added, action)
		return
	}

	b.removed[action] = struct{}{}
}

// move replaces an endpoint's contribution from prev to next.
func (b *batch) move(c *routingCache, prev, next []string) {
	if len(prev) == 0 && len(next) == 0 {
		return
	}

	if stringsEqual(prev, next) {
		return
	}

	nextSet := make(map[string]struct{}, len(next))

	for _, action := range next {
		if action == "" {
			continue
		}

		if _, dup := nextSet[action]; dup {
			continue
		}

		nextSet[action] = struct{}{}
	}

	prevSet := make(map[string]struct{}, len(prev))

	for _, action := range prev {
		if action == "" {
			continue
		}

		if _, dup := prevSet[action]; dup {
			continue
		}

		prevSet[action] = struct{}{}

		if _, keep := nextSet[action]; !keep {
			b.leave(c, action)
		}
	}

	for action := range nextSet {
		if _, had := prevSet[action]; !had {
			b.enter(c, action)
		}
	}
}

// publishLocked bumps the revision when the batch changed the union and records the delta.
func (c *routingCache) publishLocked(b *batch) bool {
	if len(b.added) == 0 && len(b.removed) == 0 {
		return false
	}

	c.rev++

	delta := unionDelta{rev: c.rev, added: make([]string, 0, len(b.added)), removed: make([]string, 0, len(b.removed))}

	for action := range b.added {
		delta.added = append(delta.added, action)
	}

	for action := range b.removed {
		delta.removed = append(delta.removed, action)
	}

	c.log = append(c.log, delta)

	if len(c.log) > unionLogSize {
		c.log = c.log[len(c.log)-unionLogSize:]
	}

	return true
}

// Load replaces the cache with the tenant's current rows, dropping endpoints that vanished.
func (c *routingCache) Load(ctx context.Context) error {
	rows, err := c.repo.ListForTenant(ctx, c.tenantId)

	if err != nil {
		return fmt.Errorf("could not load endpoints for tenant %s: %w", c.tenantId, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	b := newBatch()
	seen := make(map[uuid.UUID]struct{}, len(rows))

	for _, row := range rows {
		seen[row.ID] = struct{}{}
		c.upsertLocked(b, row)
	}

	for id, ep := range c.byId {
		if _, ok := seen[id]; ok {
			continue
		}

		b.move(c, ep.cfg.contribution(), nil)
		delete(c.byId, id)
		delete(c.byNamespace, ep.namespace)
	}

	c.lastLoad = time.Now()
	c.publishLocked(b)

	return nil
}

// Refresh applies rows versioned past the watermark. Deleted endpoints are not visible here;
// Load drops them. A row whose commit lands after a refresh read past its version is caught
// by the next full load.
func (c *routingCache) Refresh(ctx context.Context) error {
	c.mu.RLock()
	since, sinceId := c.since, c.sinceId
	c.mu.RUnlock()

	rows, err := c.repo.ListUpdatedSince(ctx, c.tenantId, since, sinceId)

	if err != nil {
		return fmt.Errorf("could not refresh endpoints for tenant %s: %w", c.tenantId, err)
	}

	if len(rows) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.applyRowsLocked(rows)

	return nil
}

// applyRowsLocked upserts a batch of rows and publishes once.
func (c *routingCache) applyRowsLocked(rows []*sqlcv1.V1ServerlessEndpoint) {
	b := newBatch()

	for _, row := range rows {
		c.upsertLocked(b, row)
	}

	c.publishLocked(b)
}

// LastLoad is when the cache was last fully loaded; the runner schedules full reloads on it.
func (c *routingCache) LastLoad() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastLoad
}

func (c *routingCache) upsertLocked(b *batch, row *sqlcv1.V1ServerlessEndpoint) {
	ep, ok := c.byId[row.ID]

	if !ok {
		ep = &cachedEndpoint{
			id:        row.ID,
			tenantId:  row.TenantID,
			namespace: row.Namespace,
			shard:     row.Shard,
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
		requestTimeoutSeconds: row.RequestTimeoutSeconds,
		pollIntervalSeconds:   row.PollIntervalSeconds,
		inlineWaitBudgetMs:    row.InlineWaitBudgetMs,
		enabled:               row.Enabled,
		healthKnown:           row.Healthy.Valid,
		healthy:               row.Healthy.Valid && row.Healthy.Bool,
		registeredActions:     row.RegisteredActions,
	}

	if row.StatusError.Valid {
		cfg.statusError = row.StatusError.String
	}

	if row.UpdatedAt.Valid {
		cfg.updatedAt = row.UpdatedAt.Time
	}

	if row.StatusChangedAt.Valid {
		cfg.statusChangedAt = row.StatusChangedAt.Time
	}

	c.advanceWatermarkLocked(cfg.version(), row.ID)

	// A row already applied at this version changes nothing; the refresh window and the
	// full reload both return rows the cache has seen.
	if prev != nil && prev.updatedAt.Equal(cfg.updatedAt) && prev.statusChangedAt.Equal(cfg.statusChangedAt) && stringsEqual(prev.registeredActions, cfg.registeredActions) {
		return
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

	// A status this process wrote after the row was read is newer than the row's; the
	// database timestamps of both writes decide, so clock skew plays no part.
	if prev != nil && prev.healthKnown && prev.statusChangedAt.After(cfg.statusChangedAt) {
		cfg.healthKnown = prev.healthKnown
		cfg.healthy = prev.healthy
		cfg.statusError = prev.statusError
		cfg.statusChangedAt = prev.statusChangedAt
	}

	b.move(c, prev.contribution(), cfg.contribution())

	ep.cfg = cfg
}

// advanceWatermarkLocked moves the refresh keyset to (version, id) when it is later.
func (c *routingCache) advanceWatermarkLocked(version time.Time, id uuid.UUID) {
	if version.After(c.since) || (version.Equal(c.since) && id.String() > c.sinceId.String()) {
		c.since = version
		c.sinceId = id
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

// Revision is the union's current revision; it changes exactly when the union does.
func (c *routingCache) Revision() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.rev
}

// ActionUnion is the sorted union of registered_actions over the tenant's enabled endpoints,
// the action set a registration opens with, and its revision. The slice is shared and must
// not be modified. It is built once per revision, and only on demand: a registration's sync
// never needs it, so a changed union costs the sort at most once per open. The keys are
// copied under the read lock and sorted outside any lock, since sorting a million ids would
// otherwise hold every route on the tenant; the result is published if the revision it was
// built for is still current, else built again.
func (c *routingCache) ActionUnion() ([]string, uint64) {
	for {
		c.mu.RLock()

		if c.sortedRev == c.rev && c.sorted != nil {
			sorted, rev := c.sorted, c.rev
			c.mu.RUnlock()

			return sorted, rev
		}

		rev := c.rev
		sorted := make([]string, 0, len(c.counts))

		for action := range c.counts {
			sorted = append(sorted, action)
		}

		c.mu.RUnlock()

		sort.Strings(sorted)

		c.mu.Lock()

		if c.rev == rev {
			c.sorted = sorted
			c.sortedRev = rev
			c.mu.Unlock()

			return sorted, rev
		}

		c.mu.Unlock()
	}
}

// DeltasSince coalesces the union changes after revision rev into the delta that brings a
// set at rev to the current revision, which is returned with it. ok is false when the log no
// longer reaches back to rev, in which case the caller diffs its set with DiffAgainst. A
// caller at the current revision gets an empty delta.
func (c *routingCache) DeltasSince(rev uint64) (added, removed []string, current uint64, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if rev == c.rev {
		return nil, nil, c.rev, true
	}

	if rev > c.rev || len(c.log) == 0 || c.log[0].rev > rev+1 {
		return nil, nil, c.rev, false
	}

	addedSet := map[string]struct{}{}
	removedSet := map[string]struct{}{}

	for _, delta := range c.log {
		if delta.rev <= rev {
			continue
		}

		for _, action := range delta.added {
			if _, ok := removedSet[action]; ok {
				delete(removedSet, action)
				continue
			}

			addedSet[action] = struct{}{}
		}

		for _, action := range delta.removed {
			if _, ok := addedSet[action]; ok {
				delete(addedSet, action)
				continue
			}

			removedSet[action] = struct{}{}
		}
	}

	return sortedKeys(addedSet), sortedKeys(removedSet), c.rev, true
}

// DiffAgainst is the delta from have to the current union (the ids to add and the ids to
// remove) with the union's revision, for a caller whose revision the log no longer reaches.
// It walks both sets once under the read lock and sorts nothing but the delta.
func (c *routingCache) DiffAgainst(have map[string]struct{}) (added, removed []string, rev uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for action := range c.counts {
		if _, ok := have[action]; !ok {
			added = append(added, action)
		}
	}

	for action := range have {
		if _, ok := c.counts[action]; !ok {
			removed = append(removed, action)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)

	return added, removed, c.rev
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))

	for action := range set {
		out = append(out, action)
	}

	sort.Strings(out)

	return out
}

// endpointsOnShards snapshots the endpoints on the given shards, in no particular order.
func (c *routingCache) endpointsOnShards(shards map[int32]struct{}) []*cachedEndpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*cachedEndpoint, 0)

	for _, ep := range c.byId {
		if _, ok := shards[ep.shard]; ok {
			out = append(out, ep)
		}
	}

	return out
}

// Endpoints snapshots the tenant's endpoints (enabled or not), sorted by id for callers that
// need a stable order, such as tests.
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

// SetHealthcheck records the action set an owned endpoint's healthcheck produced. It
// replaces registered_actions in the cache so the union changes here as soon as the owner
// learns it, without waiting for a refresh. It returns whether the union changed.
func (c *routingCache) SetHealthcheck(id uuid.UUID, actions []string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	ep, ok := c.byId[id]

	if !ok {
		return false
	}

	cfg := *ep.cfg
	cfg.registeredActions = append([]string{}, actions...)

	b := newBatch()
	b.move(c, ep.cfg.contribution(), cfg.contribution())
	ep.cfg = &cfg

	return c.publishLocked(b)
}

// SetStatus records a status transition this process wrote, stamped with the database's
// status_changed_at of the write, so a refresh returning an older row cannot resurrect the
// previous value.
func (c *routingCache) SetStatus(id uuid.UUID, healthy bool, statusError string, changedAt time.Time) {
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
	cfg.statusChangedAt = changedAt
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
