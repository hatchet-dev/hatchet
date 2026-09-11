//go:build !e2e && !load && !rampup && !integration

package claimer_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/operator/claimer"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/operatortest"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const eventually = 5 * time.Second

// testKind is the kind the tests' factory builds; the DAG kind needs a repository.
const testKind = sqlcv1.V1OperatorKindHTTPAPI

// fakeOperator records its lifecycle. release, when set, blocks Drain until it is closed.
type fakeOperator struct {
	mu       sync.Mutex
	session  operator.Session
	started  bool
	drained  bool
	startErr error
	release  chan struct{}
}

func (f *fakeOperator) HandleAction(context.Context, *contracts.AssignedAction) error { return nil }

func (f *fakeOperator) Start(_ context.Context, s operator.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.started = true
	f.session = s

	return f.startErr
}

func (f *fakeOperator) Drain(ctx context.Context) {
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
		}
	}

	f.mu.Lock()
	f.drained = true
	f.mu.Unlock()
}

func (f *fakeOperator) isDrained() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.drained
}

// fakeHost opens recording sessions and remembers what it was asked to open.
type fakeHost struct {
	mu       sync.Mutex
	opens    []operator.Identity
	opts     []operator.OpenOpts
	sessions map[uuid.UUID]*operatortest.Session
	openErr  error
}

func newFakeHost() *fakeHost {
	return &fakeHost{sessions: map[uuid.UUID]*operatortest.Session{}}
}

func (h *fakeHost) Open(_ context.Context, id operator.Identity, o operator.OpenOpts) (operator.Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.opens = append(h.opens, id)
	h.opts = append(h.opts, o)

	if h.openErr != nil {
		return nil, h.openErr
	}

	s := operatortest.NewSession(id.TenantId, *id.OperatorId)
	h.sessions[*id.OperatorId] = s

	return s, nil
}

func (h *fakeHost) session(operatorId uuid.UUID) *operatortest.Session {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.sessions[operatorId]
}

func (h *fakeHost) openCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.opens)
}

// fakeClaims answers the claim query with whatever the test set.
type fakeClaims struct {
	mu   sync.Mutex
	rows []*sqlcv1.V1Operator
	err  error
}

func (c *fakeClaims) ClaimOperators(context.Context, uuid.UUID) ([]*sqlcv1.V1Operator, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.rows, c.err
}

func (c *fakeClaims) set(rows ...*sqlcv1.V1Operator) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rows = rows
}

type harness struct {
	*claimer.Claimer

	host      *fakeHost
	claims    *fakeClaims
	operators map[uuid.UUID]*fakeOperator
	mu        sync.Mutex
}

func newHarness(t *testing.T, opts ...claimer.Opt) *harness {
	t.Helper()

	h := &harness{host: newFakeHost(), claims: &fakeClaims{}, operators: map[uuid.UUID]*fakeOperator{}}
	l := zerolog.Nop()

	factory := func(op *sqlcv1.V1Operator) (operator.Operator, operator.OpenOpts, error) {
		h.mu.Lock()
		defer h.mu.Unlock()

		fake, ok := h.operators[op.ID]

		if !ok {
			fake = &fakeOperator{}
			h.operators[op.ID] = fake
		}

		return fake, operator.OpenOpts{SlotConfig: map[string]int32{"durable": 2}}, nil
	}

	c, err := claimer.New(append([]claimer.Opt{
		claimer.WithHost(h.host),
		claimer.WithClaims(h.claims),
		claimer.WithDispatcherId(uuid.New()),
		claimer.WithFactory(testKind, factory),
		claimer.WithLogger(&l),
	}, opts...)...)
	require.NoError(t, err)

	h.Claimer = c

	return h
}

func (h *harness) operator(op *sqlcv1.V1Operator) *fakeOperator {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.operators[op.ID]
}

func row() *sqlcv1.V1Operator {
	return &sqlcv1.V1Operator{ID: uuid.New(), TenantID: uuid.New(), Name: "op", Kind: testKind, Config: []byte(`{}`)}
}

// A newly claimed row is opened through the host by id, with the operator as the handler, and
// started on its session; reconciling the same result again changes nothing.
func TestReconcileHostsClaimedRows(t *testing.T) {
	h := newHarness(t)
	op := row()

	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})

	require.Equal(t, 1, h.host.openCount())
	assert.Equal(t, op.TenantID, h.host.opens[0].TenantId)
	require.NotNil(t, h.host.opens[0].OperatorId)
	assert.Equal(t, op.ID, *h.host.opens[0].OperatorId)
	assert.Equal(t, map[string]int32{"durable": 2}, h.host.opts[0].SlotConfig, "the factory's options are used")

	fake := h.operator(op)
	assert.Equal(t, fake, h.host.opts[0].Handler, "the operator is the session's handler")
	assert.True(t, fake.started)
	assert.Equal(t, h.host.session(op.ID), fake.session)
	assert.Equal(t, 1, h.Running())

	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})
	assert.Equal(t, 1, h.host.openCount(), "a row already hosted is not opened again")

	// a kind with no factory is left alone
	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op, {ID: uuid.New(), Kind: sqlcv1.V1OperatorKindGRPC}})
	assert.Equal(t, 1, h.host.openCount())
	assert.Equal(t, 1, h.Running())
}

// A row that leaves the claim result is torn down in the host's order: pause, drain, close.
func TestReconcileTearsDownLostRows(t *testing.T) {
	h := newHarness(t)
	op := row()

	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})
	require.Equal(t, 1, h.Running())

	h.Reconcile(t.Context(), nil)
	assert.Zero(t, h.Running())

	session := h.host.session(op.ID)
	fake := h.operator(op)

	require.Eventually(t, session.Closed, eventually, time.Millisecond)
	assert.Equal(t, []string{"pause", "close"}, session.Ops())
	assert.True(t, fake.isDrained())

	// the row coming back is hosted on a new session
	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})
	assert.Equal(t, 2, h.host.openCount())
	assert.NotEqual(t, session, h.host.session(op.ID))
}

// The worker is paused before the operator drains, and the session is closed only after the
// drain finished.
func TestTeardownPausesBeforeDrainAndClosesAfter(t *testing.T) {
	h := newHarness(t)
	op := row()

	h.mu.Lock()
	h.operators[op.ID] = &fakeOperator{release: make(chan struct{})}
	h.mu.Unlock()

	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})
	h.Reconcile(t.Context(), nil)

	session := h.host.session(op.ID)
	fake := h.operator(op)

	require.Eventually(t, func() bool { return len(session.Ops()) == 1 }, eventually, time.Millisecond)
	assert.Equal(t, []string{"pause"}, session.Ops(), "the pause is committed while the drain is still running")
	assert.False(t, session.Closed())
	assert.False(t, fake.isDrained())

	close(fake.release)

	require.Eventually(t, session.Closed, eventually, time.Millisecond)
	assert.Equal(t, []string{"pause", "close"}, session.Ops())
	assert.True(t, fake.isDrained())
}

// A row whose session cannot be opened, or whose operator does not start, is retried on the
// next poll rather than remembered as running.
func TestReconcileFailuresAreRetried(t *testing.T) {
	h := newHarness(t)
	op := row()

	h.host.openErr = errors.New("no tenant")
	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})
	assert.Zero(t, h.Running())

	h.host.openErr = nil
	h.mu.Lock()
	h.operators[op.ID] = &fakeOperator{startErr: errors.New("no workflows")}
	h.mu.Unlock()

	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})
	assert.Zero(t, h.Running())
	assert.True(t, h.host.session(op.ID).Closed(), "the session of an operator that did not start is closed")

	h.mu.Lock()
	h.operators[op.ID] = &fakeOperator{}
	h.mu.Unlock()

	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})
	assert.Equal(t, 1, h.Running())
}

// Start polls at once and then on the interval; a failed claim changes nothing; Stop tears
// every hosted operator down and waits for the teardowns.
func TestStartPollsAndStopTearsDown(t *testing.T) {
	h := newHarness(t, claimer.WithPollInterval(5*time.Millisecond))
	a, b := row(), row()

	h.claims.set(a, b)
	h.Start(t.Context())

	require.Eventually(t, func() bool { return h.Running() == 2 }, eventually, time.Millisecond)

	h.claims.mu.Lock()
	h.claims.err = errors.New("db down")
	h.claims.mu.Unlock()

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, h.Running(), "a failed claim query tears nothing down")

	h.claims.mu.Lock()
	h.claims.err = nil
	h.claims.mu.Unlock()
	h.claims.set(a)

	require.Eventually(t, func() bool { return h.Running() == 1 }, eventually, time.Millisecond)
	require.Eventually(t, h.host.session(b.ID).Closed, eventually, time.Millisecond)

	h.Stop(t.Context())

	assert.Zero(t, h.Running())
	assert.Equal(t, []string{"pause", "close"}, h.host.session(a.ID).Ops())
	assert.True(t, h.operator(a).isDrained())

	// stopping again is a no-op, and nothing is hosted afterwards
	h.Stop(t.Context())
	assert.Equal(t, 2, h.host.openCount())
}

// A drain that outlives the teardown timeout does not hold the session open forever.
func TestTeardownTimeout(t *testing.T) {
	h := newHarness(t, claimer.WithTeardownTimeout(20*time.Millisecond))
	op := row()

	h.mu.Lock()
	h.operators[op.ID] = &fakeOperator{release: make(chan struct{})}
	h.mu.Unlock()

	h.Reconcile(t.Context(), []*sqlcv1.V1Operator{op})

	h.Stop(t.Context())

	assert.True(t, h.host.session(op.ID).Closed(), "the session is closed once the drain timed out")
}
