//go:build e2e

package e2e

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
)

// operatorProcess is one out-of-process operator instance run inside the test binary: the
// core over the gRPC host with the shared token registry, its own pool and a random process
// id.
type operatorProcess struct {
	id     uuid.UUID
	cancel context.CancelFunc
	done   chan error
	crash  *crashSwitch
	pool   *pgxpool.Pool
	stopFn func() error
	once   sync.Once
	err    error
}

// processConfig mirrors the in-engine timings from TestMain.
func processConfig() serverlessoperator.Config {
	return serverlessoperator.Config{
		LinkName:                  serverlessoperator.DefaultLinkName,
		DefaultSlots:              serverlessoperator.DefaultDefaultSlots,
		DurableSlots:              serverlessoperator.DefaultDurableSlots,
		LeaseTTL:                  leaseTTL,
		HeartbeatInterval:         tick,
		RebalanceInterval:         tick,
		RoutingRefreshInterval:    tick,
		RoutingFullReloadInterval: fullReload,
		DrainTimeout:              drainTimeout,
		HealthcheckTimeout:        5 * time.Second,
		HealthPort:                0,
	}
}

// startProcess starts an instance and registers its graceful stop with the test.
func startProcess(t *testing.T, e *testEnv) *operatorProcess {
	t.Helper()

	id := uuid.New()
	l := e.l.With().Str("process", id.String()[:8]).Logger()

	pool, err := pgxpool.New(e.ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)

	repo, cleanupRepo := repository.NewServerlessRepositoryFromPool(pool, &l)

	sender, err := safeclient.New(safeclient.Config{
		AllowEmptyInfraCIDRs: true,
		InsecureDestinations: true,
	}, &l)
	require.NoError(t, err)

	hostname, _ := os.Hostname()

	p := &operatorProcess{
		id:    id,
		done:  make(chan error, 1),
		crash: &crashSwitch{},
		pool:  pool,
	}

	host, err := hostgrpc.New(hostgrpc.WithTokenSource(tokens), hostgrpc.WithLogger(&l))
	require.NoError(t, err)

	p.stopFn = func() error {
		host.Close()
		err := cleanupRepo()
		pool.Close()

		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	deps := serverlessoperator.Deps{
		Repo:       crashableRepo{ServerlessRepository: repo, sw: p.crash},
		Host:       host,
		Encryption: e.enc,
		Sender:     sender,
		Logger:     &l,
		Version:    "e2e",
		Hostname:   hostname,
		ProcessId:  id,
		Config:     processConfig(),
	}

	go func() {
		p.done <- serverlessoperator.Run(ctx, deps)
	}()

	t.Cleanup(func() {
		require.NoError(t, p.stop())
	})

	e.waitLive(id)

	return p
}

// stop shuts the instance down gracefully (release leases, close registrations, delete the
// process row) and returns Run's error. Safe to call more than once.
func (p *operatorProcess) stop() error {
	p.once.Do(func() {
		p.cancel()
		p.err = <-p.done

		if err := p.stopFn(); err != nil && p.err == nil {
			p.err = err
		}
	})

	return p.err
}

// crashNow simulates the process dying: from now on it can neither heartbeat nor release or
// shed its leases nor delete its process row, so its lease rows keep its id and its process
// row expires after the TTL, exactly what other processes see after a real crash. The Go side
// is then torn down so the test binary leaks nothing.
func (p *operatorProcess) crashNow() {
	p.crash.crashed.Store(true)
	_ = p.stop()
}

var errCrashed = errors.New("process crashed")

type crashSwitch struct {
	crashed atomic.Bool
}

// crashableRepo delegates to the real repository until the switch flips, after which the
// writes a dead process could not make fail.
type crashableRepo struct {
	repository.ServerlessRepository
	sw *crashSwitch
}

func (r crashableRepo) Processes() repository.ServerlessProcessRepository {
	return crashableProcesses{ServerlessProcessRepository: r.ServerlessRepository.Processes(), sw: r.sw}
}

func (r crashableRepo) Leases() repository.ServerlessLeaseRepository {
	return crashableLeases{ServerlessLeaseRepository: r.ServerlessRepository.Leases(), sw: r.sw}
}

type crashableProcesses struct {
	repository.ServerlessProcessRepository
	sw *crashSwitch
}

func (p crashableProcesses) Upsert(ctx context.Context, opts repository.UpsertServerlessProcessOpts) error {
	if p.sw.crashed.Load() {
		return errCrashed
	}

	return p.ServerlessProcessRepository.Upsert(ctx, opts)
}

func (p crashableProcesses) Delete(ctx context.Context, processId uuid.UUID) error {
	if p.sw.crashed.Load() {
		return errCrashed
	}

	return p.ServerlessProcessRepository.Delete(ctx, processId)
}

type crashableLeases struct {
	repository.ServerlessLeaseRepository
	sw *crashSwitch
}

func (l crashableLeases) ReleaseAll(ctx context.Context, processId uuid.UUID) (int64, error) {
	if l.sw.crashed.Load() {
		return 0, errCrashed
	}

	return l.ServerlessLeaseRepository.ReleaseAll(ctx, processId)
}

func (l crashableLeases) Shed(ctx context.Context, processId uuid.UUID, units []repository.ServerlessUnit) ([]*sqlcv1.ShedServerlessLeasesRow, error) {
	if l.sw.crashed.Load() {
		return nil, errCrashed
	}

	return l.ServerlessLeaseRepository.Shed(ctx, processId, units)
}
