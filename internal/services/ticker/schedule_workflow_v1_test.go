//go:build !e2e && !load && !rampup && !integration

package ticker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func setupTickerTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:15.6",
		postgres.WithDatabase("hatchet"),
		postgres.WithUsername("hatchet"),
		postgres.WithPassword("hatchet"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	t.Setenv("DATABASE_URL", connStr)
	migrate.RunMigrations(ctx)

	config, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		container.Terminate(ctx)
	})

	return pool
}

type mockMQ struct{ sends atomic.Int64 }

func (m *mockMQ) SendMessage(_ context.Context, _ msgqueue.Queue, _ *msgqueue.Message) error {
	m.sends.Add(1)
	return nil
}
func (m *mockMQ) Clone() (func() error, msgqueue.MessageQueue, error) { return nil, nil, nil }
func (m *mockMQ) SetQOS(_ int)                                        {}
func (m *mockMQ) Subscribe(_ msgqueue.Queue, _, _ msgqueue.AckHook) (func() error, error) {
	return nil, nil
}
func (m *mockMQ) RegisterTenant(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockMQ) IsReady() bool                                       { return true }

type mockRepo struct {
	v1.Repository
	idempotency v1.IdempotencyRepository
}

func (r *mockRepo) Idempotency() v1.IdempotencyRepository            { return r.idempotency }
func (r *mockRepo) WorkflowSchedules() v1.WorkflowScheduleRepository { return &noopSchedules{} }

type noopSchedules struct{ v1.WorkflowScheduleRepository }

func (n *noopSchedules) DeleteScheduledWorkflow(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func TestRunScheduledWorkflow_ConcurrentCallersOnlyTriggerOnce(t *testing.T) {
	pool := setupTickerTestPool(t)

	logger := zerolog.Nop()
	repo := &mockRepo{idempotency: v1.NewIdempotencyRepository(pool)}
	mq := &mockMQ{}

	tenantID := uuid.New()
	opts := v1.RunScheduledWorkflowV1Opts{
		ID:           uuid.New(),
		WorkflowName: "test-workflow",
		TriggerAt:    time.Now(),
	}

	const n = 20
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			RunScheduledWorkflow(context.Background(), &logger, mq, repo, tenantID, opts)
		}()
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), mq.sends.Load(), "exactly one message should be sent across all concurrent callers")
}
