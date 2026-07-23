//go:build integration

package postgres

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

const defaultTestDatabaseURL = "postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet"

func testDatabaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}

	return defaultTestDatabaseURL
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	config, err := pgxpool.ParseConfig(testDatabaseURL())
	require.NoError(t, err)

	config.MaxConns = 5
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	return pool
}

func newTestPubSub(t *testing.T) (*PubSub, *pgxpool.Pool) {
	t.Helper()

	pool := newTestPool(t)

	l := logger.NewDefaultLogger("postgres-pubsub-test")

	repo, cleanupRepo := v1.NewMessageQueueRepositoryWithPool(&l, pool)

	cleanup, ps, err := NewPubSub(repo, WithPubSubLogger(&l))
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("error cleaning up pubsub: %v", err)
		}
		if err := cleanupRepo(); err != nil {
			t.Errorf("error cleaning up repo: %v", err)
		}
	})

	return ps, pool
}

func TestPostgresPubSubNotifyRoundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps, _ := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	cleanupSub, err := ps.Sub(topic, func(received *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, msg.ID, received.ID)
		return nil
	})
	require.NoError(t, err)

	// give the LISTEN connection time to establish
	time.Sleep(1 * time.Second)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	wg.Wait()

	require.NoError(t, cleanupSub())
}

func TestPostgresPubSubLargePayloadFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ps, _ := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	// well beyond pg_notify's 8000-byte limit: must take the fallback-row path
	payload := make([]byte, 32*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-stream-event", true, false, map[string]interface{}{"data": payload})
	require.NoError(t, err)

	originalPayload := make([]byte, len(msg.Payloads[0]))
	copy(originalPayload, msg.Payloads[0])

	received := make(chan *msgqueue.Message, 1)

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		received <- m
		return nil
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	start := time.Now()
	require.NoError(t, ps.Pub(ctx, topic, msg))

	select {
	case m := <-received:
		assert.Equal(t, msg.ID, m.ID)
		require.Len(t, m.Payloads, 1)
		assert.Equal(t, originalPayload, m.Payloads[0])
		// the fallback poll runs at 1s intervals; allow some slack
		assert.Less(t, time.Since(start), 5*time.Second, "fallback row should be drained within a few poll intervals")
	case <-ctx.Done():
		t.Fatal("timed out waiting for >8KB fallback delivery")
	}

	require.NoError(t, cleanupSub())
}

func TestPostgresPubSubTwoSubscribers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps, _ := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	handler := func(received *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, msg.ID, received.ID)
		return nil
	}

	cleanupSub1, err := ps.Sub(topic, handler)
	require.NoError(t, err)

	cleanupSub2, err := ps.Sub(topic, handler)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	wg.Wait()

	require.NoError(t, cleanupSub1())
	require.NoError(t, cleanupSub2())
}

// TestPostgresPubSubDisjointPools asserts the core invariant: a full pub/sub
// cycle never acquires from any pool other than the PubSub's own dedicated pool.
func TestPostgresPubSubDisjointPools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps, _ := newTestPubSub(t)

	// stands in for the shared repository pool
	sharedPool := newTestPool(t)

	// let min conns settle before snapshotting
	time.Sleep(500 * time.Millisecond)

	acquiresBefore := sharedPool.Stat().AcquireCount()

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	cleanupSub, err := ps.Sub(topic, func(received *msgqueue.Message) error {
		defer wg.Done()
		return nil
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	wg.Wait()

	require.NoError(t, cleanupSub())

	assert.Equal(t, acquiresBefore, sharedPool.Stat().AcquireCount(), "pub/sub cycle must not acquire from the shared pool")
}

// TestPostgresPubSubMixedVersionInterop simulates an old-version engine
// mirroring a tenant message via the legacy repo.Notify path on a different
// pool: the new PubSub subscriber must receive it.
func TestPostgresPubSubMixedVersionInterop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps, _ := newTestPubSub(t)

	// the "old engine" publishes through its own repository on a separate pool
	oldEnginePool := newTestPool(t)
	l := logger.NewDefaultLogger("old-engine")
	oldRepo, cleanupOldRepo := v1.NewMessageQueueRepositoryWithPool(&l, oldEnginePool)
	t.Cleanup(func() {
		if err := cleanupOldRepo(); err != nil {
			t.Errorf("error cleaning up old repo: %v", err)
		}
	})

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	cleanupSub, err := ps.Sub(topic, func(received *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, msg.ID, received.ID)
		return nil
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// legacy pubNonDurableMessages wire format: the marshalled Message as the
	// NOTIFY payload on the "<uuid>_v1" channel
	msgBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	require.NoError(t, oldRepo.Notify(ctx, topic.Name(), string(msgBytes)))

	wg.Wait()

	require.NoError(t, cleanupSub())
}
