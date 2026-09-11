// Package lease is the serverless operator's process membership and unit ownership: one
// heartbeat row per process, fair-share claiming of (tenant, shard) units with FOR UPDATE SKIP
// LOCKED, shedding above fair share, and a periodic sweep of expired process rows. The pattern
// is pgoutbox's consumer session leasing with indexed candidate lookups instead of a table
// scan.
package lease

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// Unit is one lease unit.
type Unit = repository.ServerlessUnit

// Reconciler is what the leaser drives: the runner opens and closes registrations and
// pollers as ownership changes and reports in-flight deliveries so shedding can prefer idle
// units.
type Reconciler interface {
	UnitsGained(ctx context.Context, units []Unit)
	UnitsLost(ctx context.Context, units []Unit)
	InFlight(unit Unit) int
}

// Hooks are optional observers for metrics. Nil funcs are skipped.
type Hooks struct {
	Claimed    func(n int)
	Shed       func(n int)
	Rebalanced func(d time.Duration)
	Owned      func(units, endpoints int)
}

// Config is the leaser's timing and claim sizing. Zero values take the defaults below.
type Config struct {
	Hostname          string
	Version           string
	ProcessId         uuid.UUID
	TTL               time.Duration
	HeartbeatInterval time.Duration
	RebalanceInterval time.Duration
	SweepInterval     time.Duration
	SweepCutoff       time.Duration
	ShedHysteresis    float64

	// ClaimBatch is the most units one claim statement takes; MaxClaimPerTick caps how many a
	// tick claims in total and sizes the sample of the claimable population each tick counts.
	// The tick's budget is its fair share of the claimable units, so a takeover of many units
	// spreads evenly over the live processes and a single survivor takes at most
	// MaxClaimPerTick per tick.
	ClaimBatch      int32
	MaxClaimPerTick int32
}

const (
	defaultTTL               = 15 * time.Second
	defaultHeartbeatInterval = 5 * time.Second
	defaultRebalanceInterval = 5 * time.Second
	defaultSweepInterval     = 10 * time.Minute
	defaultSweepCutoff       = time.Hour
	defaultShedHysteresis    = 0.2
	defaultClaimBatch        = 64
	defaultMaxClaimPerTick   = 1024

	// initialHeartbeatBackoffMax caps the retry interval of the first heartbeat.
	initialHeartbeatBackoffMax = 30 * time.Second
)

func (c Config) withDefaults() Config {
	if c.TTL <= 0 {
		c.TTL = defaultTTL
	}

	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = defaultHeartbeatInterval
	}

	if c.RebalanceInterval <= 0 {
		c.RebalanceInterval = defaultRebalanceInterval
	}

	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}

	if c.SweepCutoff <= 0 {
		c.SweepCutoff = defaultSweepCutoff
	}

	if c.ShedHysteresis <= 0 {
		c.ShedHysteresis = defaultShedHysteresis
	}

	if c.ClaimBatch <= 0 {
		c.ClaimBatch = defaultClaimBatch
	}

	if c.MaxClaimPerTick <= 0 {
		c.MaxClaimPerTick = defaultMaxClaimPerTick
	}

	if c.ClaimBatch > c.MaxClaimPerTick {
		c.ClaimBatch = c.MaxClaimPerTick
	}

	return c
}

// Leaser owns the process row and the set of owned units. Tick and Heartbeat are exported so
// tests can drive one step at a time; Run loops them.
type Leaser struct {
	repo       repository.ServerlessRepository
	reconciler Reconciler
	l          *zerolog.Logger
	owned      map[Unit]int32
	// kick asks the rebalance loop for an immediate tick, after a heartbeat lapse.
	kick  chan struct{}
	hooks Hooks
	cfg   Config
	// lastHeartbeat is when the last heartbeat succeeded; a gap longer than the TTL means the
	// process row expired in between and other processes may have taken its units.
	lastHeartbeat time.Time
	// initialBackoff is the first retry interval of the initial heartbeat; tests shorten it.
	initialBackoff time.Duration
	mu             sync.Mutex
	ticked         atomic.Bool
}

func New(repo repository.ServerlessRepository, reconciler Reconciler, cfg Config, l *zerolog.Logger, hooks Hooks) *Leaser {
	return &Leaser{
		repo:           repo,
		reconciler:     reconciler,
		cfg:            cfg.withDefaults(),
		l:              l,
		hooks:          hooks,
		owned:          map[Unit]int32{},
		kick:           make(chan struct{}, 1),
		initialBackoff: time.Second,
	}
}

// Ready reports whether the first rebalance tick has completed.
func (s *Leaser) Ready() bool {
	return s.ticked.Load()
}

// Owned snapshots the owned units.
func (s *Leaser) Owned() []Unit {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Unit, 0, len(s.owned))

	for u := range s.owned {
		out = append(out, u)
	}

	sortUnits(out)

	return out
}

func (s *Leaser) counts() (units, endpoints int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, n := range s.owned {
		endpoints += n
	}

	return int32(len(s.owned)), endpoints // #nosec G115 -- unit counts are small
}

// Run heartbeats until the process row is live, retrying with backoff so a database that is
// unreachable at startup delays the process instead of stopping it, then loops the heartbeat,
// rebalance and sweep until ctx is done. Loop errors are logged, not returned: a transient
// database error must not stop the process.
func (s *Leaser) Run(ctx context.Context) error {
	if err := s.initialHeartbeat(ctx); err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.loop(gctx, s.cfg.HeartbeatInterval, false, nil, s.Heartbeat)
	})

	g.Go(func() error {
		return s.loop(gctx, s.cfg.RebalanceInterval, true, s.kick, s.Tick)
	})

	g.Go(func() error {
		return s.loop(gctx, s.cfg.SweepInterval, false, nil, s.sweep)
	})

	return g.Wait()
}

func (s *Leaser) initialHeartbeat(ctx context.Context) error {
	backoff := s.initialBackoff

	for {
		err := s.Heartbeat(ctx)

		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return fmt.Errorf("initial heartbeat: %w", err)
		}

		s.l.Error().Err(err).Dur("retry_in", backoff).Msg("initial serverless heartbeat failed; retrying")

		select {
		case <-ctx.Done():
			return fmt.Errorf("initial heartbeat: %w", ctx.Err())
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, initialHeartbeatBackoffMax)
	}
}

func (s *Leaser) loop(ctx context.Context, interval time.Duration, immediate bool, kick <-chan struct{}, fn func(context.Context) error) error {
	run := func() {
		if err := fn(ctx); err != nil && ctx.Err() == nil {
			s.l.Error().Err(err).Msg("serverless leaser step failed")
		}
	}

	if immediate {
		run()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			run()
		case <-kick:
			run()
		}
	}
}

// Heartbeat upserts the process row: the only steady-state write of a process. A heartbeat
// that lands more than the TTL after the previous one found the row expired in between; the
// lease table is re-read at once, since other processes may have taken units meanwhile.
func (s *Leaser) Heartbeat(ctx context.Context) error {
	units, endpoints := s.counts()

	opts := repository.UpsertServerlessProcessOpts{
		ProcessId:     s.cfg.ProcessId,
		TTL:           s.cfg.TTL,
		UnitCount:     units,
		EndpointCount: endpoints,
	}

	if s.cfg.Hostname != "" {
		opts.Hostname = &s.cfg.Hostname
	}

	if s.cfg.Version != "" {
		opts.Version = &s.cfg.Version
	}

	if err := s.repo.Processes().Upsert(ctx, opts); err != nil {
		return err
	}

	now := time.Now()

	s.mu.Lock()
	lapsed := !s.lastHeartbeat.IsZero() && now.Sub(s.lastHeartbeat) > s.cfg.TTL
	s.lastHeartbeat = now
	s.mu.Unlock()

	if lapsed {
		s.l.Warn().Msg("serverless heartbeat lapsed beyond the lease TTL; ownership is re-read from the lease table")

		select {
		case s.kick <- struct{}{}:
		default:
		}
	}

	return nil
}

func (s *Leaser) sweep(ctx context.Context) error {
	deleted, released, err := s.repo.Processes().DeleteExpired(ctx, time.Now().Add(-s.cfg.SweepCutoff))

	if err != nil {
		return fmt.Errorf("sweep expired processes: %w", err)
	}

	if deleted > 0 {
		s.l.Info().Int64("deleted", deleted).Int64("released_units", released).Msg("swept expired serverless process rows")
	}

	return nil
}

// Tick is one rebalance: refresh ownership from the lease table, compute the fair share by
// endpoint weight, claim up to the budget from unowned and dead-process units, shed above
// fair share plus hysteresis, then hand the ownership diff to the reconciler.
func (s *Leaser) Tick(ctx context.Context) error {
	start := time.Now()

	defer func() {
		if s.hooks.Rebalanced != nil {
			s.hooks.Rebalanced(time.Since(start))
		}
	}()

	current, err := s.listOwned(ctx)

	if err != nil {
		return err
	}

	live, _, err := s.repo.Processes().ListLive(ctx)

	if err != nil {
		return fmt.Errorf("list live processes: %w", err)
	}

	myWeight := weightOf(current)

	// Self is counted from local state, not from its process row: the row's counts lag by
	// up to one heartbeat and may not exist yet on the first tick.
	liveCount := int64(1)
	otherWeight := int64(0)

	for _, p := range live {
		if p.ProcessID == s.cfg.ProcessId {
			continue
		}

		liveCount++
		otherWeight += int64(p.EndpointCount)
	}

	// Units still held by dead processes are claimable, so they count toward what the live
	// processes share; without them a process that already owns its fair share has no budget
	// to take a crashed process's units over. The count is a sample of at most
	// MaxClaimPerTick units, whatever the fleet's size, so every replica's count costs the
	// same bounded index walk each tick; only units with endpoints are counted, since only
	// those are claimed. A full sample means the backlog is at least a tick's worth for this
	// process, which is all the budget below needs to know.
	sampleLimit := int64(s.cfg.MaxClaimPerTick)
	claimable, err := s.repo.Leases().CountClaimable(ctx, sampleLimit)

	if err != nil {
		return fmt.Errorf("count claimable leases: %w", err)
	}

	saturated := claimable.UnitCount >= sampleLimit

	total := otherWeight + myWeight + claimable.EndpointCount
	fairShare := (total + liveCount - 1) / liveCount
	weightBudget := max(fairShare-myWeight, 0)

	// The unit budget is this process's share of the claimable units, so a takeover spreads
	// over the live processes within a tick instead of one process taking a fixed batch.
	unitBudget := min((claimable.UnitCount+liveCount-1)/liveCount, int64(s.cfg.MaxClaimPerTick))

	// A saturated sample is a backlog of unknown size beyond it (a cold start, a large
	// process gone): every live process takes a full tick's worth now, the concurrent walks
	// start at random keys and skip each other's locks, and the next tick sheds whatever
	// overshot the fair share once the counts are exact again. Waiting for an exact fleet-wide
	// count would cost every replica a scan of the whole backlog each tick.
	if saturated {
		unitBudget = int64(s.cfg.MaxClaimPerTick)
		weightBudget = max(weightBudget, claimable.EndpointCount)
	}

	// The floor of one unit when holding nothing keeps units from being stranded while the
	// weight estimate is off, pgoutbox style; the budgets spread load, they are not
	// correctness constraints. Progress with units held is guaranteed by the count itself:
	// it covers only units with endpoints, so a claimable population raises the fair share
	// above what the live processes hold and at least one of them gets a positive budget.
	if len(current) == 0 && claimable.UnitCount > 0 {
		unitBudget = max(unitBudget, 1)
		weightBudget = max(weightBudget, 1)
	}

	claimed := 0

	if weightBudget > 0 && unitBudget > 0 {
		n, err := s.claim(ctx, current, unitBudget, weightBudget)

		if err != nil {
			return err
		}

		claimed = n
		myWeight = weightOf(current)
	}

	if claimed > 0 && s.hooks.Claimed != nil {
		s.hooks.Claimed(claimed)
	}

	shed := 0

	// A tick that claimed never sheds: a claim batch can overshoot the budget by one batch
	// and shedding it straight back would be churn. The next tick sheds if still above the
	// hysteresis threshold.
	if claimed == 0 && liveCount > 1 && float64(myWeight) > float64(fairShare)*(1+s.cfg.ShedHysteresis) {
		toShed := s.pickShed(current, myWeight-fairShare)

		if len(toShed) > 0 {
			rows, err := s.repo.Leases().Shed(ctx, s.cfg.ProcessId, toShed)

			if err != nil {
				return fmt.Errorf("shed leases: %w", err)
			}

			for _, row := range rows {
				delete(current, Unit{TenantId: row.TenantID, Shard: row.Shard})
				shed++
			}
		}
	}

	if shed > 0 && s.hooks.Shed != nil {
		s.hooks.Shed(shed)
	}

	gained, lost := s.commit(current)

	if s.hooks.Owned != nil {
		s.hooks.Owned(len(current), int(weightOf(current)))
	}

	if len(lost) > 0 {
		s.reconciler.UnitsLost(ctx, lost)
	}

	if len(gained) > 0 {
		s.reconciler.UnitsGained(ctx, gained)
	}

	if claimed > 0 || shed > 0 {
		s.l.Info().
			Int("claimed", claimed).
			Int("shed", shed).
			Int("owned", len(current)).
			Int64("weight", weightOf(current)).
			Int64("fair_share", fairShare).
			Msg("serverless lease rebalance")
	}

	s.ticked.Store(true)

	return nil
}

// claim takes units into current until either budget is spent or nothing claimable is left.
// The walk over unowned units starts at a random key, so concurrent claimers spread over the
// population instead of contending on its head, and wraps around to the beginning once.
func (s *Leaser) claim(ctx context.Context, current map[Unit]int32, unitBudget, weightBudget int64) (int, error) {
	claimed := 0
	after := Unit{TenantId: randomTenantKey()}
	wrapped := false

	for unitBudget > 0 && weightBudget > 0 {
		limit := int32(min(int64(s.cfg.ClaimBatch), unitBudget, weightBudget)) // #nosec G115 -- bounded by ClaimBatch

		rows, err := s.repo.Leases().Claim(ctx, s.cfg.ProcessId, after, limit)

		if err != nil {
			return claimed, fmt.Errorf("claim leases: %w", err)
		}

		for _, row := range rows {
			unit := Unit{TenantId: row.TenantID, Shard: row.Shard}
			current[unit] = row.EndpointCount
			unitBudget--
			// Every unit spends at least one weight unit so a run of empty units cannot keep
			// a tick claiming past its unit budget.
			weightBudget -= max(int64(row.EndpointCount), 1)
			claimed++

			if lessUnit(after, unit) {
				after = unit
			}
		}

		if int32(len(rows)) == limit { // #nosec G115 -- bounded by ClaimBatch
			continue
		}

		if wrapped || after.TenantId == uuid.Nil {
			break
		}

		after = Unit{}
		wrapped = true
	}

	return claimed, nil
}

// randomTenantKey is a random point in the tenant id space: tenant ids are random uuids, so
// a fresh one is distributed like them and starting a walk there spreads concurrent claimers
// over the unowned population.
func randomTenantKey() uuid.UUID {
	return uuid.New()
}

func (s *Leaser) listOwned(ctx context.Context) (map[Unit]int32, error) {
	rows, err := s.repo.Leases().ListOwned(ctx, s.cfg.ProcessId)

	if err != nil {
		return nil, fmt.Errorf("list owned leases: %w", err)
	}

	current := make(map[Unit]int32, len(rows))

	for _, row := range rows {
		current[Unit{TenantId: row.TenantID, Shard: row.Shard}] = row.EndpointCount
	}

	return current, nil
}

// pickShed chooses units whose weights fit within excess, smallest first, skipping units with
// in-flight deliveries and zero-weight units (shedding them changes nothing).
func (s *Leaser) pickShed(current map[Unit]int32, excess int64) []Unit {
	type candidate struct {
		unit   Unit
		weight int32
	}

	candidates := make([]candidate, 0, len(current))

	for unit, weight := range current {
		if weight <= 0 || s.reconciler.InFlight(unit) > 0 {
			continue
		}

		candidates = append(candidates, candidate{unit: unit, weight: weight})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].weight != candidates[j].weight {
			return candidates[i].weight < candidates[j].weight
		}

		return lessUnit(candidates[i].unit, candidates[j].unit)
	})

	out := make([]Unit, 0)

	for _, c := range candidates {
		if int64(c.weight) > excess {
			break
		}

		out = append(out, c.unit)
		excess -= int64(c.weight)
	}

	return out
}

// commit replaces the owned set and returns the diff.
func (s *Leaser) commit(current map[Unit]int32) (gained, lost []Unit) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for unit := range current {
		if _, ok := s.owned[unit]; !ok {
			gained = append(gained, unit)
		}
	}

	for unit := range s.owned {
		if _, ok := current[unit]; !ok {
			lost = append(lost, unit)
		}
	}

	s.owned = current

	sortUnits(gained)
	sortUnits(lost)

	return gained, lost
}

// Release gives every owned unit back in one statement and forgets them locally. The caller
// drains and closes registrations afterwards, then calls DeleteProcess.
func (s *Leaser) Release(ctx context.Context) ([]Unit, error) {
	if _, err := s.repo.Leases().ReleaseAll(ctx, s.cfg.ProcessId); err != nil {
		return nil, fmt.Errorf("release leases: %w", err)
	}

	s.mu.Lock()
	released := make([]Unit, 0, len(s.owned))

	for unit := range s.owned {
		released = append(released, unit)
	}

	s.owned = map[Unit]int32{}
	s.mu.Unlock()

	sortUnits(released)

	return released, nil
}

// DeleteProcess removes the process row, the last step of a graceful shutdown.
func (s *Leaser) DeleteProcess(ctx context.Context) error {
	return s.repo.Processes().Delete(ctx, s.cfg.ProcessId)
}

func weightOf(units map[Unit]int32) int64 {
	var total int64

	for _, n := range units {
		total += int64(n)
	}

	return total
}

func lessUnit(a, b Unit) bool {
	if a.TenantId != b.TenantId {
		return a.TenantId.String() < b.TenantId.String()
	}

	return a.Shard < b.Shard
}

func sortUnits(units []Unit) {
	sort.Slice(units, func(i, j int) bool {
		return lessUnit(units[i], units[j])
	})
}
