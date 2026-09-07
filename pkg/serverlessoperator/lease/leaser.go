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

// Config is the leaser's timing. Zero values take the defaults below.
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
	ClaimBatch        int32
}

const (
	defaultTTL               = 15 * time.Second
	defaultHeartbeatInterval = 5 * time.Second
	defaultRebalanceInterval = 5 * time.Second
	defaultSweepInterval     = 10 * time.Minute
	defaultSweepCutoff       = time.Hour
	defaultShedHysteresis    = 0.2
	defaultClaimBatch        = 8

	// maxClaimRounds bounds the claim loop of one tick so a run of zero-weight units, which
	// never spend the budget, cannot keep a tick busy.
	maxClaimRounds = 16
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

	return c
}

// Leaser owns the process row and the set of owned units. Tick and Heartbeat are exported so
// tests can drive one step at a time; Run loops them.
type Leaser struct {
	repo       repository.ServerlessRepository
	reconciler Reconciler
	l          *zerolog.Logger
	owned      map[Unit]int32
	hooks      Hooks
	cfg        Config
	mu         sync.Mutex
	ticked     atomic.Bool
}

func New(repo repository.ServerlessRepository, reconciler Reconciler, cfg Config, l *zerolog.Logger, hooks Hooks) *Leaser {
	return &Leaser{
		repo:       repo,
		reconciler: reconciler,
		cfg:        cfg.withDefaults(),
		l:          l,
		hooks:      hooks,
		owned:      map[Unit]int32{},
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

// Run heartbeats immediately so the process is live before its first tick, then loops the
// heartbeat, rebalance and sweep until ctx is done. Loop errors are logged, not returned:
// a transient database error must not stop the process.
func (s *Leaser) Run(ctx context.Context) error {
	if err := s.Heartbeat(ctx); err != nil {
		return fmt.Errorf("initial heartbeat: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.loop(gctx, s.cfg.HeartbeatInterval, false, func(ctx context.Context) error {
			return s.Heartbeat(ctx)
		})
	})

	g.Go(func() error {
		return s.loop(gctx, s.cfg.RebalanceInterval, true, s.Tick)
	})

	g.Go(func() error {
		return s.loop(gctx, s.cfg.SweepInterval, false, s.sweep)
	})

	return g.Wait()
}

func (s *Leaser) loop(ctx context.Context, interval time.Duration, immediate bool, fn func(context.Context) error) error {
	if immediate {
		if err := fn(ctx); err != nil && ctx.Err() == nil {
			s.l.Error().Err(err).Msg("serverless leaser step failed")
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := fn(ctx); err != nil && ctx.Err() == nil {
				s.l.Error().Err(err).Msg("serverless leaser step failed")
			}
		}
	}
}

// Heartbeat upserts the process row: the only steady-state write of a process.
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

	return s.repo.Processes().Upsert(ctx, opts)
}

func (s *Leaser) sweep(ctx context.Context) error {
	n, err := s.repo.Processes().DeleteExpired(ctx, time.Now().Add(-s.cfg.SweepCutoff))

	if err != nil {
		return fmt.Errorf("sweep expired processes: %w", err)
	}

	if n > 0 {
		s.l.Info().Int64("deleted", n).Msg("swept expired serverless process rows")
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

	live, dead, err := s.repo.Processes().ListLive(ctx)

	if err != nil {
		return fmt.Errorf("list live processes: %w", err)
	}

	unowned, err := s.repo.Leases().CountUnowned(ctx)

	if err != nil {
		return fmt.Errorf("count unowned leases: %w", err)
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

	total := otherWeight + myWeight + unowned.EndpointCount
	fairShare := (total + liveCount - 1) / liveCount
	budget := max(fairShare-myWeight, 0)

	claimed := 0

	// The floor of one unit when holding nothing keeps units from being stranded while the
	// weight estimate is off, pgoutbox style; the budget spreads load, it is not a
	// correctness constraint.
	if (budget > 0 || len(current) == 0) && (unowned.UnitCount > 0 || len(dead) > 0) {
		for round := 0; round < maxClaimRounds; round++ {
			// Each unit weighs at least one endpoint in practice, so a batch never larger
			// than the remaining budget keeps the overshoot to one unit; the floor case
			// claims exactly one.
			limit := int32(1)

			if budget > 0 {
				limit = int32(min(int64(s.cfg.ClaimBatch), budget)) // #nosec G115 -- bounded by ClaimBatch
			}

			rows, err := s.repo.Leases().Claim(ctx, s.cfg.ProcessId, dead, limit)

			if err != nil {
				return fmt.Errorf("claim leases: %w", err)
			}

			for _, row := range rows {
				current[Unit{TenantId: row.TenantID, Shard: row.Shard}] = row.EndpointCount
				myWeight += int64(row.EndpointCount)
				budget -= int64(row.EndpointCount)
				claimed++
			}

			if int32(len(rows)) < limit || budget <= 0 { // #nosec G115 -- bounded by ClaimBatch
				break
			}
		}
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
