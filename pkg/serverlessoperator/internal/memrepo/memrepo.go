// Package memrepo is an in-memory repository.ServerlessRepository for unit tests of the
// serverless operator core and leaser. It implements the methods the core uses and records
// the writes tests assert on. Methods the core never calls are left to the embedded nil
// interface and panic when reached.
package memrepo

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Unit aliases the lease unit.
type Unit = repository.ServerlessUnit

// StatusWrite is one UpdateStatus call.
type StatusWrite struct {
	Error      *string
	EndpointId uuid.UUID
	Healthy    bool
}

// ActionWrite is one UpdateRegisteredActions call.
type ActionWrite struct {
	Actions    []string
	EndpointId uuid.UUID
}

type process struct {
	row     sqlcv1.V1ServerlessProcess
	expired bool
}

// Repo is the in-memory repository. Every method is safe for concurrent use.
type Repo struct {
	endpoints  map[uuid.UUID]*sqlcv1.V1ServerlessEndpoint
	processes  map[uuid.UUID]*process
	leases     map[Unit]*sqlcv1.V1ServerlessLease
	Now        func() time.Time
	FailWrites error

	// Recorded calls. Read them through the snapshot accessors below, which take the lock.
	statusWrites  []StatusWrite
	actionWrites  []ActionWrite
	heartbeats    []repository.UpsertServerlessProcessOpts
	claimCalls    []int32
	shedCalls     [][]Unit
	deletedProcs  []uuid.UUID
	releaseAlls   int
	listForTenant int
	listSince     int

	mu sync.Mutex
}

func (r *Repo) StatusWrites() []StatusWrite {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]StatusWrite{}, r.statusWrites...)
}

func (r *Repo) ActionWrites() []ActionWrite {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]ActionWrite{}, r.actionWrites...)
}

func (r *Repo) Heartbeats() []repository.UpsertServerlessProcessOpts {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]repository.UpsertServerlessProcessOpts{}, r.heartbeats...)
}

func (r *Repo) ClaimCalls() []int32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]int32{}, r.claimCalls...)
}

func (r *Repo) ShedCalls() [][]Unit {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([][]Unit{}, r.shedCalls...)
}

func (r *Repo) DeletedProcs() []uuid.UUID {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]uuid.UUID{}, r.deletedProcs...)
}

func (r *Repo) ReleaseAlls() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.releaseAlls
}

// ListForTenantCalls counts full loads; ListSinceCalls counts incremental refreshes.
func (r *Repo) ListForTenantCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.listForTenant
}

func (r *Repo) ListSinceCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.listSince
}

func New() *Repo {
	return &Repo{
		endpoints: map[uuid.UUID]*sqlcv1.V1ServerlessEndpoint{},
		processes: map[uuid.UUID]*process{},
		leases:    map[Unit]*sqlcv1.V1ServerlessLease{},
		Now:       time.Now,
	}
}

// AddEndpoint stores a copy of ep, stamps updated_at, and creates or bumps its lease unit.
func (r *Repo) AddEndpoint(ep *sqlcv1.V1ServerlessEndpoint) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *ep
	cp.UpdatedAt = pgtype.Timestamptz{Time: r.Now(), Valid: true}
	r.endpoints[cp.ID] = &cp

	unit := Unit{TenantId: cp.TenantID, Shard: cp.Shard}

	lease, ok := r.leases[unit]

	if !ok {
		lease = &sqlcv1.V1ServerlessLease{TenantID: unit.TenantId, Shard: unit.Shard}
		r.leases[unit] = lease
	}

	lease.EndpointCount++
}

// UpdateEndpoint mutates a stored endpoint and bumps updated_at.
func (r *Repo) UpdateEndpoint(id uuid.UUID, fn func(ep *sqlcv1.V1ServerlessEndpoint)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ep, ok := r.endpoints[id]

	if !ok {
		return
	}

	fn(ep)
	ep.UpdatedAt = pgtype.Timestamptz{Time: r.Now(), Valid: true}
}

// RemoveEndpoint hard-deletes an endpoint and decrements its unit's count.
func (r *Repo) RemoveEndpoint(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ep, ok := r.endpoints[id]

	if !ok {
		return
	}

	delete(r.endpoints, id)

	if lease, ok := r.leases[Unit{TenantId: ep.TenantID, Shard: ep.Shard}]; ok {
		lease.EndpointCount--
	}
}

// Endpoint returns a copy of a stored endpoint.
func (r *Repo) Endpoint(id uuid.UUID) *sqlcv1.V1ServerlessEndpoint {
	r.mu.Lock()
	defer r.mu.Unlock()

	ep, ok := r.endpoints[id]

	if !ok {
		return nil
	}

	cp := *ep

	return &cp
}

// SetLease creates or overwrites a lease row.
func (r *Repo) SetLease(unit Unit, owner *uuid.UUID, endpointCount int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.leases[unit] = &sqlcv1.V1ServerlessLease{TenantID: unit.TenantId, Shard: unit.Shard, ProcessID: owner, EndpointCount: endpointCount}
}

// Lease returns a copy of a lease row.
func (r *Repo) Lease(unit Unit) *sqlcv1.V1ServerlessLease {
	r.mu.Lock()
	defer r.mu.Unlock()

	l, ok := r.leases[unit]

	if !ok {
		return nil
	}

	cp := *l

	return &cp
}

// SetProcess creates or overwrites a process row.
func (r *Repo) SetProcess(id uuid.UUID, unitCount, endpointCount int32, expired bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.processes[id] = &process{
		row:     sqlcv1.V1ServerlessProcess{ProcessID: id, UnitCount: unitCount, EndpointCount: endpointCount},
		expired: expired,
	}
}

func (r *Repo) Endpoints() repository.ServerlessEndpointRepository { return &endpoints{r: r} }
func (r *Repo) Tenants() repository.ServerlessTenantRepository     { return &tenants{} }
func (r *Repo) Processes() repository.ServerlessProcessRepository  { return &processes{r: r} }
func (r *Repo) Leases() repository.ServerlessLeaseRepository       { return &leases{r: r} }

type endpoints struct {
	repository.ServerlessEndpointRepository
	r *Repo
}

func copyEndpoint(ep *sqlcv1.V1ServerlessEndpoint) *sqlcv1.V1ServerlessEndpoint {
	cp := *ep
	cp.RegisteredActions = append([]string{}, ep.RegisteredActions...)

	return &cp
}

func sortEndpoints(out []*sqlcv1.V1ServerlessEndpoint) {
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID.String() < out[j].ID.String()
	})
}

func (e *endpoints) ListForUnits(_ context.Context, units []Unit, afterId uuid.UUID, limit int64) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	e.r.mu.Lock()
	defer e.r.mu.Unlock()

	want := map[Unit]struct{}{}

	for _, u := range units {
		want[u] = struct{}{}
	}

	out := make([]*sqlcv1.V1ServerlessEndpoint, 0)

	for _, ep := range e.r.endpoints {
		if _, ok := want[Unit{TenantId: ep.TenantID, Shard: ep.Shard}]; !ok {
			continue
		}

		if ep.ID.String() <= afterId.String() {
			continue
		}

		out = append(out, copyEndpoint(ep))
	}

	sortEndpoints(out)

	if int64(len(out)) > limit {
		out = out[:limit]
	}

	return out, nil
}

func (e *endpoints) ListForTenant(_ context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	e.r.mu.Lock()
	defer e.r.mu.Unlock()

	e.r.listForTenant++

	out := make([]*sqlcv1.V1ServerlessEndpoint, 0)

	for _, ep := range e.r.endpoints {
		if ep.TenantID == tenantId {
			out = append(out, copyEndpoint(ep))
		}
	}

	sortEndpoints(out)

	return out, nil
}

func (e *endpoints) ListUpdatedSince(_ context.Context, tenantId uuid.UUID, since time.Time) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	e.r.mu.Lock()
	defer e.r.mu.Unlock()

	e.r.listSince++

	out := make([]*sqlcv1.V1ServerlessEndpoint, 0)

	for _, ep := range e.r.endpoints {
		if ep.TenantID == tenantId && ep.UpdatedAt.Time.After(since) {
			out = append(out, copyEndpoint(ep))
		}
	}

	sortEndpoints(out)

	return out, nil
}

func (e *endpoints) UpdateStatus(_ context.Context, endpointId uuid.UUID, healthy bool, statusError *string) error {
	e.r.mu.Lock()
	defer e.r.mu.Unlock()

	if e.r.FailWrites != nil {
		return e.r.FailWrites
	}

	e.r.statusWrites = append(e.r.statusWrites, StatusWrite{EndpointId: endpointId, Healthy: healthy, Error: statusError})

	if ep, ok := e.r.endpoints[endpointId]; ok {
		ep.Healthy = pgtype.Bool{Bool: healthy, Valid: true}
		ep.StatusError = pgtype.Text{}

		if statusError != nil {
			ep.StatusError = pgtype.Text{String: *statusError, Valid: true}
		}
	}

	return nil
}

func (e *endpoints) UpdateRegisteredActions(_ context.Context, endpointId uuid.UUID, actions []string) error {
	e.r.mu.Lock()
	defer e.r.mu.Unlock()

	if e.r.FailWrites != nil {
		return e.r.FailWrites
	}

	e.r.actionWrites = append(e.r.actionWrites, ActionWrite{EndpointId: endpointId, Actions: append([]string{}, actions...)})

	if ep, ok := e.r.endpoints[endpointId]; ok {
		ep.RegisteredActions = append([]string{}, actions...)
		ep.UpdatedAt = pgtype.Timestamptz{Time: e.r.Now(), Valid: true}
	}

	return nil
}

type tenants struct {
	repository.ServerlessTenantRepository
}

type processes struct {
	repository.ServerlessProcessRepository
	r *Repo
}

func (p *processes) Upsert(_ context.Context, opts repository.UpsertServerlessProcessOpts) error {
	p.r.mu.Lock()
	defer p.r.mu.Unlock()

	if p.r.FailWrites != nil {
		return p.r.FailWrites
	}

	p.r.heartbeats = append(p.r.heartbeats, opts)

	row := sqlcv1.V1ServerlessProcess{
		ProcessID:     opts.ProcessId,
		UnitCount:     opts.UnitCount,
		EndpointCount: opts.EndpointCount,
		ExpiresAt:     pgtype.Timestamptz{Time: p.r.Now().Add(opts.TTL), Valid: true},
	}

	p.r.processes[opts.ProcessId] = &process{row: row}

	return nil
}

func (p *processes) ListLive(_ context.Context) ([]*sqlcv1.V1ServerlessProcess, []uuid.UUID, error) {
	p.r.mu.Lock()
	defer p.r.mu.Unlock()

	live := make([]*sqlcv1.V1ServerlessProcess, 0)
	dead := make([]uuid.UUID, 0)

	for id, proc := range p.r.processes {
		if proc.expired {
			dead = append(dead, id)
			continue
		}

		row := proc.row
		live = append(live, &row)
	}

	sort.Slice(live, func(i, j int) bool { return live[i].ProcessID.String() < live[j].ProcessID.String() })
	sort.Slice(dead, func(i, j int) bool { return dead[i].String() < dead[j].String() })

	return live, dead, nil
}

func (p *processes) DeleteExpired(_ context.Context, _ time.Time) (int64, error) {
	p.r.mu.Lock()
	defer p.r.mu.Unlock()

	var n int64

	for id, proc := range p.r.processes {
		if proc.expired {
			delete(p.r.processes, id)
			n++
		}
	}

	return n, nil
}

func (p *processes) Delete(_ context.Context, processId uuid.UUID) error {
	p.r.mu.Lock()
	defer p.r.mu.Unlock()

	p.r.deletedProcs = append(p.r.deletedProcs, processId)
	delete(p.r.processes, processId)

	return nil
}

type leases struct {
	repository.ServerlessLeaseRepository
	r *Repo
}

func sortedUnits(m map[Unit]*sqlcv1.V1ServerlessLease) []Unit {
	out := make([]Unit, 0, len(m))

	for u := range m {
		out = append(out, u)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].TenantId != out[j].TenantId {
			return out[i].TenantId.String() < out[j].TenantId.String()
		}

		return out[i].Shard < out[j].Shard
	})

	return out
}

// Claim takes claimable units in deterministic order (the database randomizes).
func (l *leases) Claim(_ context.Context, processId uuid.UUID, deadIds []uuid.UUID, limit int32) ([]*sqlcv1.ClaimServerlessLeasesRow, error) {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()

	l.r.claimCalls = append(l.r.claimCalls, limit)

	dead := map[uuid.UUID]struct{}{}

	for _, id := range deadIds {
		dead[id] = struct{}{}
	}

	out := make([]*sqlcv1.ClaimServerlessLeasesRow, 0)

	for _, unit := range sortedUnits(l.r.leases) {
		if int32(len(out)) >= limit { // #nosec G115 -- bounded by limit
			break
		}

		lease := l.r.leases[unit]

		claimable := lease.ProcessID == nil

		if lease.ProcessID != nil {
			_, claimable = dead[*lease.ProcessID]
		}

		if !claimable {
			continue
		}

		owner := processId
		lease.ProcessID = &owner
		lease.ClaimedAt = pgtype.Timestamptz{Time: l.r.Now(), Valid: true}

		out = append(out, &sqlcv1.ClaimServerlessLeasesRow{TenantID: unit.TenantId, Shard: unit.Shard, EndpointCount: lease.EndpointCount})
	}

	return out, nil
}

func (l *leases) Shed(_ context.Context, processId uuid.UUID, units []Unit) ([]*sqlcv1.ShedServerlessLeasesRow, error) {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()

	l.r.shedCalls = append(l.r.shedCalls, append([]Unit{}, units...))

	out := make([]*sqlcv1.ShedServerlessLeasesRow, 0)

	for _, unit := range units {
		lease, ok := l.r.leases[unit]

		if !ok || lease.ProcessID == nil || *lease.ProcessID != processId {
			continue
		}

		lease.ProcessID = nil
		lease.ClaimedAt = pgtype.Timestamptz{}

		out = append(out, &sqlcv1.ShedServerlessLeasesRow{TenantID: unit.TenantId, Shard: unit.Shard, EndpointCount: lease.EndpointCount})
	}

	return out, nil
}

func (l *leases) ReleaseAll(_ context.Context, processId uuid.UUID) (int64, error) {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()

	l.r.releaseAlls++

	var n int64

	for _, lease := range l.r.leases {
		if lease.ProcessID != nil && *lease.ProcessID == processId {
			lease.ProcessID = nil
			n++
		}
	}

	return n, nil
}

func (l *leases) ListOwned(_ context.Context, processId uuid.UUID) ([]*sqlcv1.V1ServerlessLease, error) {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()

	out := make([]*sqlcv1.V1ServerlessLease, 0)

	for _, unit := range sortedUnits(l.r.leases) {
		lease := l.r.leases[unit]

		if lease.ProcessID != nil && *lease.ProcessID == processId {
			cp := *lease
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (l *leases) CountUnowned(_ context.Context) (*sqlcv1.CountUnownedServerlessLeasesRow, error) {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()

	row := &sqlcv1.CountUnownedServerlessLeasesRow{}

	for _, lease := range l.r.leases {
		if lease.ProcessID == nil {
			row.UnitCount++
			row.EndpointCount += int64(lease.EndpointCount)
		}
	}

	return row, nil
}

func (l *leases) InsertIfAbsent(_ context.Context, unit Unit) error {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()

	if _, ok := l.r.leases[unit]; !ok {
		l.r.leases[unit] = &sqlcv1.V1ServerlessLease{TenantID: unit.TenantId, Shard: unit.Shard}
	}

	return nil
}

func (l *leases) IncrementEndpointCount(_ context.Context, unit Unit, delta int32) error {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()

	lease, ok := l.r.leases[unit]

	if !ok {
		return errors.New("lease not found")
	}

	lease.EndpointCount += delta

	return nil
}
