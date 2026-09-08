//go:build !e2e && !load && !rampup && !integration

package grpclink

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // OperatorService's client lives in the legacy client package
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// completeMemo is a fire-and-forget request, the kind that fills the shared listener's queue
// fastest when the engine is unreachable.
func completeMemo() *v1.DurableTaskRequest {
	return &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{
		CompleteMemo: &v1.DurableTaskCompleteMemoRequest{Ref: ref("task", 1, 0, 1)},
	}}
}

// A Send blocked on a full shared request queue must return once the invocation's channel
// is closed, so a timed-out or cancelled invocation can tear down and shutdown can complete
// while the engine is unreachable.
func TestFullQueueSendUnblocksOnClose(t *testing.T) {
	session := &fakeSession{workerId: "w"}
	reg := &registration{session: session}

	ch, err := reg.OpenDurable(context.Background(), "task", 1)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = reg.Close()

		if session.stream != nil {
			session.stream.end()
		}
	})

	// Nobody reads the fake stream, so requests back up in the stream buffer, the shared
	// listener's queue and the hub's own queue, until a Send blocks.
	blocked := make(chan error, 1)
	sent := 0

	for {
		done := make(chan error, 1)

		go func() { done <- ch.Send(completeMemo()) }()

		select {
		case err := <-done:
			require.NoError(t, err)
			sent++

			if sent > 10000 {
				t.Fatal("sends never blocked; the queue is unbounded")
			}

			continue
		case <-time.After(100 * time.Millisecond):
			go func() { blocked <- <-done }()
		}

		break
	}

	t.Logf("%d sends queued before one blocked", sent)

	require.NoError(t, ch.Close())

	select {
	case err := <-blocked:
		assert.ErrorIs(t, err, link.ErrChannelClosed, "a blocked Send reports the close")
	case <-time.After(2 * time.Second):
		t.Fatal("a Send blocked on the full request queue survived the channel's Close")
	}

	// A closed channel refuses further sends at once.
	assert.ErrorIs(t, ch.Send(completeMemo()), link.ErrChannelClosed)
}

// A callback registration scheduled just before an invocation closes must not outlive the
// invocation on the shared listener.
func TestLateCallbackDoesNotSurviveClose(t *testing.T) {
	session := &fakeSession{workerId: "w"}
	reg := &registration{session: session}

	t.Cleanup(func() {
		_ = reg.Close()

		if session.stream != nil {
			session.stream.end()
		}
	})

	for i := 0; i < 100; i++ {
		ch, err := reg.OpenDurable(context.Background(), fmt.Sprintf("task-%d", i), 1)
		require.NoError(t, err)

		c := ch.(*durableChannel)

		// The goroutine an ack schedules races the close: whichever order the scheduler
		// picks, no callback may remain.
		go c.awaitEntry(0, 1)

		require.NoError(t, ch.Close())
		c.awaitEntry(0, 2)
	}

	listener := session.listeners[0]

	assert.Eventually(t, func() bool { return listener.PendingCallbackCount() == 0 }, eventually, 5*time.Millisecond,
		"closed invocations left %d late callbacks on the shared listener", listener.PendingCallbackCount())
}

// closableClient is a fake engine client that owns a goroutine, as a gRPC connection does,
// and stops it on Close.
type closableClient struct {
	fakeClient
	closed atomic.Bool
	stop   chan struct{}
}

func newClosableClient(operator *fakeOperatorClient) *closableClient {
	c := &closableClient{fakeClient: fakeClient{operator: operator}, stop: make(chan struct{})}

	go func() { <-c.stop }()

	return c
}

func (c *closableClient) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		close(c.stop)
	}

	return nil
}

// Evicting a tenant's cached client closes it: on release, on token rotation and on an
// authentication failure. The goroutine the client owns must go with it.
func TestEvictedClientsAreClosed(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	tenant := uuid.New()
	exchange := mapExchange{tenant: "tok-1"}

	var built []*closableClient

	lnk := New(exchange, Options{NewClient: func(string) (client.Client, error) { //nolint:staticcheck // see import
		c := newClosableClient(&fakeOperatorClient{tenantId: tenant.String()})
		built = append(built, c)

		return c, nil
	}})

	reg, err := lnk.Open(context.Background(), tenant, link.OpenOpts{})
	require.NoError(t, err)
	require.NoError(t, reg.Close())
	require.Len(t, built, 1)

	// Token rotation replaces the client and closes the previous one.
	exchange[tenant] = "tok-2"
	reg, err = lnk.Open(context.Background(), tenant, link.OpenOpts{})
	require.NoError(t, err)
	require.NoError(t, reg.Close())
	require.Len(t, built, 2)
	assert.True(t, built[0].closed.Load(), "the replaced client is closed")
	assert.False(t, built[1].closed.Load())

	// Releasing the tenant closes the current client.
	lnk.ReleaseTenant(tenant)
	assert.True(t, built[1].closed.Load(), "the released client is closed")
	assert.Empty(t, lnk.clients)
}

// The default factory builds real SDK clients over gRPC connections; releasing a tenant must
// leave none of their goroutines behind. Closing them requires the client to expose its
// connection through Close.
func TestReleasedRealClientsLeakNoGoroutines(t *testing.T) {
	lnk := New(nil, Options{})

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	for i := 0; i < 3; i++ {
		tenant := uuid.New()
		_, err := lnk.clientFor(tenant, testJWT(t, tenant))
		require.NoError(t, err)
		lnk.ReleaseTenant(tenant)
	}

	assert.Empty(t, lnk.clients)
}

// tenantSession reports the tenant the engine authenticated, which may differ from the one
// the link asked the exchange for.
type tenantSession struct {
	*fakeSession
	tenantId string
}

func (s *tenantSession) Registration() client.OperatorRegistration { //nolint:staticcheck // see import
	return client.OperatorRegistration{TenantId: s.tenantId, WorkerId: s.workerId} //nolint:staticcheck // see import
}

type tenantOperator struct{ session *tenantSession }

func (o *tenantOperator) Connect(context.Context, *client.ConnectOperatorRequest) (client.OperatorSession, error) { //nolint:staticcheck // see import
	return o.session, nil
}

type tenantClient struct {
	client.Client //nolint:staticcheck // see import
	operator      *tenantOperator
}

func (c *tenantClient) Operator() client.OperatorClient { return c.operator } //nolint:staticcheck // see import

func (c *tenantClient) Close() error { return nil }

// A registration the engine authenticated as another tenant than the one the unit belongs to
// is refused before any action, workflow or delivery touches it, and so is a token whose
// tenant claim names another tenant.
func TestOpenRefusesMismatchedTenant(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	session := &tenantSession{fakeSession: &fakeSession{workerId: uuid.NewString()}, tenantId: b.String()}
	c := &tenantClient{operator: &tenantOperator{session: session}}

	lnk := New(mapExchange{a: testJWT(t, a)}, Options{NewClient: func(string) (client.Client, error) { return c, nil }}) //nolint:staticcheck // see import

	_, err := lnk.Open(context.Background(), a, link.OpenOpts{Actions: []string{"review:run"}})
	require.Error(t, err, "the engine authenticated tenant B for a unit of tenant A")
	assert.Contains(t, err.Error(), b.String())
	assert.Empty(t, session.added, "no actions were sent on the mismatched registration")
	assert.True(t, session.closed, "the mismatched session is closed")

	// A token whose claim names another tenant is refused before connecting.
	session2 := &tenantSession{fakeSession: &fakeSession{workerId: uuid.NewString()}, tenantId: a.String()}
	connects := 0
	lnk2 := New(mapExchange{a: testJWT(t, b)}, Options{NewClient: func(string) (client.Client, error) { //nolint:staticcheck // see import
		connects++
		return &tenantClient{operator: &tenantOperator{session: session2}}, nil
	}})

	_, err = lnk2.Open(context.Background(), a, link.OpenOpts{})
	require.Error(t, err, "the exchange returned tenant B's token for tenant A")
	assert.Equal(t, 0, connects, "no client is built for a token of the wrong tenant")
}

// Rotating the contents of a referenced token_file, without touching the YAML, is observed by
// the exchange's poller.
func TestLocalExchangeObservesReferencedTokenFile(t *testing.T) {
	dir := t.TempDir()
	tenant := uuid.New()
	yamlPath := filepath.Join(dir, "tenants.yaml")
	tokenPath := filepath.Join(dir, "tenant.token")

	require.NoError(t, os.WriteFile(tokenPath, []byte("old-token"), 0o600))
	require.NoError(t, os.WriteFile(yamlPath, []byte("tenants:\n  "+tenant.String()+": { token_file: tenant.token }\n"), 0o600))

	e, err := NewLocalExchange(yamlPath, WithPollInterval(time.Hour))
	require.NoError(t, err)
	defer e.Close()

	// Filesystems may round mtimes; make the rotation land on a later second.
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, os.WriteFile(tokenPath, []byte("new-token"), 0o600))

	changed, err := e.reloadIfChanged()
	require.NoError(t, err)
	assert.True(t, changed, "referenced token file changed but the YAML did not; the rotation was not observed")

	tok, err := e.Token(context.Background(), tenant)
	require.NoError(t, err)
	assert.Equal(t, "new-token", tok)

	// Nothing changed: no reload.
	changed, err = e.reloadIfChanged()
	require.NoError(t, err)
	assert.False(t, changed)
}

// An ordinary invocation lifecycle (open, ack-bearing request, ack, close, registration close)
// leaves no goroutine behind: every goroutine a channel starts is joined by its Close.
func TestDurableChannelLifecycleLeaksNoGoroutines(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	session := &fakeSession{workerId: "w"}
	reg := &registration{session: session}

	for i := 0; i < 20; i++ {
		ch, err := reg.OpenDurable(context.Background(), fmt.Sprintf("task-%d", i), 1)
		require.NoError(t, err)

		require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{Memo: &v1.DurableTaskMemoRequest{}}}))
		session.stream.next(t)

		session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
			MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: ref(fmt.Sprintf("task-%d", i), 1, 0, 1)},
		}}

		require.NotNil(t, recvOne(t, ch).GetMemoAck())
		require.NoError(t, ch.Close())
	}

	require.NoError(t, reg.Close())
	session.stream.end()
}
