//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

const testSecret = "test-signing-secret-0123456789abcdef"

// testClock is a settable clock for the event recorder's rate limits.
type testClock struct {
	t  time.Time
	mu sync.Mutex
}

func newTestClock() *testClock {
	return &testClock{t: time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)}
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.t
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.t = c.t.Add(d)
}

// eventsOfKind returns the recorded events of one kind, in order.
func eventsOfKind(repo *memrepo.Repo, kind sqlcv1.V1ServerlessEndpointEventKind) []repository.ServerlessEndpointEventOpts {
	out := make([]repository.ServerlessEndpointEventOpts, 0)

	for _, ev := range repo.EventRows() {
		if ev.Kind == kind {
			out = append(out, ev)
		}
	}

	return out
}

func detailsOf(t *testing.T, ev repository.ServerlessEndpointEventOpts) *eventDetails {
	t.Helper()

	details, ok := ev.Details.(*eventDetails)
	require.True(t, ok, "event %s carries details", ev.Kind)

	return details
}

func TestHealthcheckErrorStages(t *testing.T) {
	const url = "https://ep.example.test/health"

	sender := newFakeSender()
	id := uuid.New()

	cases := []struct {
		respond    func()
		name       string
		stage      string
		message    string
		statusCode int
	}{
		{
			name:    "no response is the request stage",
			respond: func() {},
			stage:   stageRequest,
			message: "healthcheck request failed: no handler for " + url,
		},
		{
			name:       "a non-2xx answer is the status stage, the body kept out of the message",
			respond:    func() { sender.respond(url, http.StatusUnauthorized, `{"error":"bad signature"}`) },
			stage:      stageStatus,
			message:    "healthcheck returned status 401",
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "a body that is not a healthcheck response is the parse stage",
			respond:    func() { sender.respond(url, http.StatusOK, `<html>hello</html>`) },
			stage:      stageParse,
			statusCode: http.StatusOK,
		},
		{
			name:       "a refused catalog is the catalog stage",
			respond:    func() { sender.respond(url, http.StatusOK, `{"actions":["noverb"]}`) },
			stage:      stageCatalog,
			statusCode: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.respond()

			ep := &cachedEndpoint{id: id}
			_, err := pollHealthcheck(context.Background(), sender, ep, &endpointConfig{healthcheckUrl: url, secret: testSecret}, catalogLimits{})
			require.Error(t, err)

			hcErr := healthcheckStage(err)
			assert.Equal(t, tc.stage, hcErr.stage)
			assert.Equal(t, tc.statusCode, hcErr.statusCode)

			if tc.message != "" {
				assert.Equal(t, tc.message, err.Error())
			}

			if tc.stage == stageStatus {
				assert.Equal(t, `{"error":"bad signature"}`, string(hcErr.body))
				assert.NotContains(t, err.Error(), "bad signature")
			}
		})
	}

	t.Run("a secret that cannot be decrypted fails before the request", func(t *testing.T) {
		ep := &cachedEndpoint{id: id}
		_, err := pollHealthcheck(context.Background(), sender, ep, &endpointConfig{healthcheckUrl: url, secretErr: errors.New("could not decrypt")}, catalogLimits{})
		require.Error(t, err)
		assert.Equal(t, stageRequest, healthcheckStage(err).stage)
	})
}

func TestHealthcheckEventsRecordStreakStartsAndTransitions(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{"svc:a"}
	env.addEndpoint(a)

	clock := newTestClock()
	env.r.events.now = clock.now

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})

	// The first success flips an unknown endpoint healthy; re-putting the catalog the row
	// already holds is not a catalog change.
	require.Eventually(t, func() bool {
		return len(eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHY)) == 1
	}, eventually, 10*time.Millisecond)
	assert.Empty(t, eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindCATALOGREGISTERED))

	poller := env.poller(a)
	require.NotNil(t, poller)
	poller.stop()

	env.sender.respond(a.HealthcheckUrl, http.StatusBadGateway, "upstream down")

	for range 5 {
		poller.pollOnce(context.Background())
	}

	// One failure event for the streak, one unhealthy transition after the third failure.
	require.Eventually(t, func() bool {
		return len(eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindUNHEALTHY)) == 1
	}, eventually, 10*time.Millisecond)

	failed := eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED)
	require.Len(t, failed, 1)
	assert.Equal(t, "healthcheck returned status 502", failed[0].Message)
	assert.Equal(t, tenant, failed[0].TenantId)
	assert.Equal(t, a.ID, failed[0].EndpointId)

	details := detailsOf(t, failed[0])
	assert.Equal(t, stageStatus, details.Stage)
	assert.Equal(t, http.StatusBadGateway, details.StatusCode)
	assert.Equal(t, "upstream down", details.ResponseExcerpt)
	assert.Equal(t, a.HealthcheckUrl, details.URL)

	unhealthy := eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindUNHEALTHY)[0]
	assert.Equal(t, "healthcheck returned status 502", unhealthy.Message)
	assert.Equal(t, http.StatusBadGateway, detailsOf(t, unhealthy).StatusCode)

	// A different error within the window is only counted, and the row's error follows it
	// without another UNHEALTHY: the endpoint was unhealthy already.
	env.sender.respond(a.HealthcheckUrl, http.StatusServiceUnavailable, "")
	poller.pollOnce(context.Background())
	poller.pollOnce(context.Background())

	time.Sleep(50 * time.Millisecond)
	assert.Len(t, eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED), 1)
	assert.Equal(t, "healthcheck returned status 503", poller.ts.cache.Config(poller.ep).statusError)

	// Past the window the next different error is recorded, carrying the count.
	clock.advance(failureEventWindow)
	env.sender.respond(a.HealthcheckUrl, http.StatusGatewayTimeout, "")
	poller.pollOnce(context.Background())

	require.Eventually(t, func() bool {
		return len(eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED)) == 2
	}, eventually, 10*time.Millisecond)

	second := eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED)[1]
	assert.Equal(t, "healthcheck returned status 504", second.Message)
	assert.Equal(t, 1, detailsOf(t, second).Suppressed)
	assert.Len(t, eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindUNHEALTHY), 1, "a new error while unhealthy is no transition")

	// Recovery is a transition back.
	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckBody("svc:a"))
	poller.pollOnce(context.Background())

	require.Eventually(t, func() bool {
		return len(eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHY)) == 2
	}, eventually, 10*time.Millisecond)

	// A new streak records its first failure again, even with an error seen before.
	clock.advance(failureEventWindow)
	env.sender.respond(a.HealthcheckUrl, http.StatusServiceUnavailable, "")
	poller.pollOnce(context.Background())

	require.Eventually(t, func() bool {
		return len(eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED)) == 3
	}, eventually, 10*time.Millisecond)
}

func TestCatalogChangeIsRecorded(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})

	poller := env.poller(a)
	require.NotNil(t, poller)
	poller.stop()

	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckBody("svc:a", "svc:b"))
	poller.pollOnce(context.Background())

	require.Eventually(t, func() bool {
		return len(eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindCATALOGREGISTERED)) == 1
	}, eventually, 10*time.Millisecond)

	registered := eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindCATALOGREGISTERED)[0]
	assert.Equal(t, "registered 0 workflows and 2 actions", registered.Message)
	assert.Equal(t, []string{"svc:a", "svc:b"}, detailsOf(t, registered).Actions)

	// The same catalog again is no change.
	poller.pollOnce(context.Background())
	time.Sleep(50 * time.Millisecond)
	assert.Len(t, eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindCATALOGREGISTERED), 1)
}

func TestDeliveryFailureIsRecorded(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)
	env.sender.respond(a.TriggerUrl, http.StatusInternalServerError, "<html>worker threw</html>")

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.host.session(0)
	require.NotNil(t, reg)

	action := startAction("svc:run")
	reg.deliver(t, action)

	require.Eventually(t, func() bool {
		return len(eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindDELIVERYFAILED)) == 1
	}, eventually, 10*time.Millisecond)

	ev := eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindDELIVERYFAILED)[0]
	assert.Equal(t, "endpoint returned status 500 Internal Server Error", ev.Message, "the message is the task's error")

	details := detailsOf(t, ev)
	assert.Equal(t, action.TaskRunExternalId, details.TaskRunExternalId)
	assert.Equal(t, "svc:run", details.ActionId)
	assert.Equal(t, http.StatusInternalServerError, details.StatusCode)
	assert.Equal(t, "<html>worker threw</html>", details.ResponseExcerpt)
	assert.Equal(t, a.TriggerUrl, details.URL)
	require.NotNil(t, details.Retryable)
	assert.True(t, *details.Retryable)
	require.NotNil(t, details.RetryCount)
	assert.Equal(t, int32(1), *details.RetryCount)

	// The task's own error carries no body.
	failed := reg.lastEvent()
	assert.Equal(t, "endpoint returned status 500 Internal Server Error", failed.EventPayload)

	// A success records nothing.
	env.sender.respond(a.TriggerUrl, http.StatusOK, `{}`)
	reg.deliver(t, startAction("svc:run"))
	time.Sleep(50 * time.Millisecond)
	assert.Len(t, eventsOfKind(env.repo, sqlcv1.V1ServerlessEndpointEventKindDELIVERYFAILED), 1)
}

func TestDeliveryFailedEventsAreRateLimited(t *testing.T) {
	repo := memrepo.New()
	l := zerolog.Nop()

	rec := newEventRecorder(repo.Events(), &l)
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	rec.now = func() time.Time { return now }

	ep := &cachedEndpoint{id: uuid.New(), tenantId: uuid.New()}
	other := &cachedEndpoint{id: uuid.New(), tenantId: ep.tenantId}

	rec.deliveryFailed(ep, "first", &eventDetails{})
	rec.deliveryFailed(ep, "second", &eventDetails{})
	rec.deliveryFailed(ep, "third", &eventDetails{})
	rec.deliveryFailed(other, "other endpoint", &eventDetails{})

	now = now.Add(failureEventWindow)
	rec.deliveryFailed(ep, "after the window", &eventDetails{})

	rec.stop(context.Background())

	rows := repo.EventRows()
	require.Len(t, rows, 3)

	assert.Equal(t, "first", rows[0].Message)
	assert.Zero(t, detailsOf(t, rows[0]).Suppressed)

	assert.Equal(t, "other endpoint", rows[1].Message, "each endpoint has its own window")

	assert.Equal(t, "after the window", rows[2].Message)
	assert.Equal(t, 2, detailsOf(t, rows[2]).Suppressed, "the failures inside the window are counted")
}

func TestEventRecorderDropsWhenFull(t *testing.T) {
	repo := memrepo.New()
	l := zerolog.Nop()

	// A recorder whose writer never runs: the queue fills and further events drop instead
	// of blocking the caller.
	rec := &eventRecorder{
		repo:      repo.Events(),
		l:         &l,
		now:       time.Now,
		queue:     make(chan repository.ServerlessEndpointEventOpts, 2),
		endpoints: map[uuid.UUID]*endpointEvents{},
	}

	ep := &cachedEndpoint{id: uuid.New(), tenantId: uuid.New()}

	for range 5 {
		rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindHEALTHY, "healthy", nil)
	}

	assert.Len(t, rec.queue, 2)
	assert.Equal(t, int64(3), rec.dropped.Load())
}

func TestExcerptIsSafeForJSONB(t *testing.T) {
	assert.Equal(t, `a\u0000b`, excerpt([]byte("a\x00b"), 10))
	assert.Equal(t, "abc", excerpt([]byte("abcdef"), 3))
	// A rune cut at the limit is dropped rather than stored half.
	assert.Equal(t, "a", excerpt([]byte("aé"), 2))
}

func TestSafeText(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		want  string
		limit int
		cut   bool
	}{
		{name: "plain text is kept", in: "status 500\n\tat handler", want: "status 500\n\tat handler", limit: 100},
		{name: "NUL is visible", in: "bad\x00name", want: `bad\u0000name`, limit: 100},
		{name: "invalid UTF-8 is a replacement character", in: "a\xffb", want: "a\uFFFDb", limit: 100},
		{name: "C0 controls, DEL and C1 controls are visible", in: "\x1b[32mgreen\b\r\x7f\u009b", want: `\x1b[32mgreen\x08\x0d\x7f\u009b`, limit: 100},
		{name: "an escape is never split", in: "ab\x00", want: "ab", limit: 6, cut: true},
		{name: "the limit counts the escaped text", in: "\x01\x02\x03", want: `\x01\x02`, limit: 9, cut: true},
		{name: "a rune split by the limit is dropped", in: "aé", want: "a", limit: 2, cut: true},
		{name: "an invalid trailing byte within the limit is visible", in: "a\xc3", want: "a\uFFFD", limit: 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, cut := safeText(tc.in, tc.limit)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.cut, cut)
			assert.LessOrEqual(t, len(got), tc.limit)
			assert.True(t, utf8.ValidString(got))
			assert.NotContains(t, got, "\x00")

			again, _ := safeText(got, tc.limit)
			assert.Equal(t, got, again, "safe text stays as it is")
		})
	}
}

func TestEventDetailsAreSafeAndBounded(t *testing.T) {
	repo := memrepo.New()
	l := zerolog.Nop()
	rec := newEventRecorder(repo.Events(), &l)

	ep := &cachedEndpoint{id: uuid.New(), tenantId: uuid.New()}

	workflows := []string{"bad\x00name", strings.Repeat("w", 2*eventMaxNameBytes)}

	for i := range eventMaxListItems {
		workflows = append(workflows, fmt.Sprintf("wf-%d", i))
	}

	details := &eventDetails{
		Stage:     stageCatalog,
		URL:       "https://ep.example.test/\x1bhealth",
		ActionId:  "svc:\x00run",
		Workflows: workflows,
		Actions:   []string{"svc:\xffrun"},
	}

	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindCATALOGREJECTED, "engine rejected workflow bad\x00name\x1b[2K", details)
	rec.stop(context.Background())

	rows := repo.EventRows()
	require.Len(t, rows, 1)

	assert.Equal(t, `engine rejected workflow bad\u0000name\x1b[2K`, rows[0].Message)

	stored := detailsOf(t, rows[0])
	assert.NotSame(t, details, stored, "the caller's details are not changed")
	assert.Equal(t, "bad\x00name", details.Workflows[0])

	assert.True(t, stored.Truncated)
	require.Len(t, stored.Workflows, eventMaxListItems)
	assert.Equal(t, `bad\u0000name`, stored.Workflows[0])
	assert.Len(t, stored.Workflows[1], eventMaxNameBytes)
	assert.Equal(t, `https://ep.example.test/\x1bhealth`, stored.URL)
	assert.Equal(t, `svc:\u0000run`, stored.ActionId)
	assert.Equal(t, []string{"svc:\uFFFDrun"}, stored.Actions)

	// Nothing JSONB refuses reaches the row: the stored JSON decodes without a NUL.
	encoded, err := json.Marshal(stored)
	require.NoError(t, err)

	var decoded any
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.NotContains(t, fmt.Sprint(decoded), "\x00")
}

func TestFailureEventsAreLimitedPerKind(t *testing.T) {
	repo := memrepo.New()
	l := zerolog.Nop()

	rec := newEventRecorder(repo.Events(), &l)
	clock := newTestClock()
	rec.now = clock.now

	ep := &cachedEndpoint{id: uuid.New(), tenantId: uuid.New()}

	// A new error text on every poll is still one event per window and kind.
	for i := range 5 {
		rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED, fmt.Sprintf("healthcheck returned status %d", 500+i), &eventDetails{})
		rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindCATALOGREJECTED, fmt.Sprintf("engine rejected workflow w%d", i), &eventDetails{})
	}

	// Kinds outside the failure kinds are not windowed.
	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindUNHEALTHY, "down", nil)
	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindHEALTHY, "endpoint is healthy", nil)

	clock.advance(failureEventWindow)
	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED, "healthcheck returned status 599", &eventDetails{})

	rec.stop(context.Background())

	failed := eventsOfKind(repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED)
	require.Len(t, failed, 2)
	assert.Equal(t, "healthcheck returned status 500", failed[0].Message)
	assert.Equal(t, 4, detailsOf(t, failed[1]).Suppressed)

	rejected := eventsOfKind(repo, sqlcv1.V1ServerlessEndpointEventKindCATALOGREJECTED)
	require.Len(t, rejected, 1, "each kind has its own window")

	assert.Len(t, eventsOfKind(repo, sqlcv1.V1ServerlessEndpointEventKindUNHEALTHY), 1)
	assert.Len(t, eventsOfKind(repo, sqlcv1.V1ServerlessEndpointEventKindHEALTHY), 1)
}

func TestEndpointEventBudget(t *testing.T) {
	repo := memrepo.New()
	l := zerolog.Nop()

	rec := newEventRecorder(repo.Events(), &l)
	clock := newTestClock()
	rec.now = clock.now

	ep := &cachedEndpoint{id: uuid.New(), tenantId: uuid.New()}
	other := &cachedEndpoint{id: uuid.New(), tenantId: ep.tenantId}

	for range endpointEventBudget + 10 {
		rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindCATALOGREGISTERED, "registered", nil)
	}

	rec.record(other, sqlcv1.V1ServerlessEndpointEventKindCATALOGREGISTERED, "other endpoint", nil)

	// The budget renews with the period; the next event carries what was held back.
	clock.advance(endpointEventBudgetPeriod)
	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindCATALOGREGISTERED, "next period", nil)

	rec.stop(context.Background())

	rows := repo.EventRows()
	require.Len(t, rows, endpointEventBudget+2)
	assert.Equal(t, "other endpoint", rows[endpointEventBudget].Message, "each endpoint has its own budget")

	last := rows[len(rows)-1]
	assert.Equal(t, "next period", last.Message)
	assert.Equal(t, 10, detailsOf(t, last).Suppressed)
}

func TestEventRecorderForgetsIdleEndpoints(t *testing.T) {
	repo := memrepo.New()
	l := zerolog.Nop()

	rec := newEventRecorder(repo.Events(), &l)
	defer rec.stop(context.Background())

	clock := newTestClock()
	rec.now = clock.now

	for range 1024 {
		rec.record(&cachedEndpoint{id: uuid.New()}, sqlcv1.V1ServerlessEndpointEventKindHEALTHY, "healthy", nil)
	}

	clock.advance(endpointEventBudgetPeriod)
	rec.record(&cachedEndpoint{id: uuid.New()}, sqlcv1.V1ServerlessEndpointEventKindHEALTHY, "healthy", nil)

	rec.mu.Lock()
	defer rec.mu.Unlock()

	assert.Len(t, rec.endpoints, 1)
}

// rowFailingEvents fails any insert that carries an event with a message starting "bad", the
// way Postgres fails a whole statement on one value it refuses.
type rowFailingEvents struct {
	inserted []repository.ServerlessEndpointEventOpts
	calls    int
}

func (r *rowFailingEvents) Insert(_ context.Context, rows []repository.ServerlessEndpointEventOpts) error {
	r.calls++

	for _, row := range rows {
		if strings.HasPrefix(row.Message, "bad") {
			return fmt.Errorf("could not insert: %w", &pgconn.PgError{Code: "22P05", Message: "unsupported Unicode escape sequence"})
		}
	}

	r.inserted = append(r.inserted, rows...)

	return nil
}

func (r *rowFailingEvents) List(context.Context, uuid.UUID, uuid.UUID, time.Time, *time.Time, int64) ([]*sqlcv1.V1ServerlessEndpointEvent, error) {
	return nil, nil
}

func TestEventWriteIsolatesRefusedRows(t *testing.T) {
	l := zerolog.Nop()
	events := &rowFailingEvents{}
	rec := &eventRecorder{repo: events, l: &l}

	rec.write(context.Background(), []repository.ServerlessEndpointEventOpts{
		{Message: "one", TenantId: uuid.New()},
		{Message: "bad row", TenantId: uuid.New()},
		{Message: "two", TenantId: uuid.New()},
	})

	require.Len(t, events.inserted, 2)
	assert.Equal(t, "one", events.inserted[0].Message)
	assert.Equal(t, "two", events.inserted[1].Message)
	assert.Equal(t, 4, events.calls, "the batch, then each row")
}

// blockingEventRepo holds every Insert until released, and remembers each batch it saw.
type blockingEventRepo struct {
	repository.ServerlessEndpointEventRepository
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	batches [][]repository.ServerlessEndpointEventOpts
}

func (b *blockingEventRepo) Insert(ctx context.Context, events []repository.ServerlessEndpointEventOpts) error {
	b.mu.Lock()
	b.batches = append(b.batches, events)
	b.mu.Unlock()

	b.started <- struct{}{}

	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *blockingEventRepo) seen() [][]repository.ServerlessEndpointEventOpts {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.batches
}

// Events still queued when the shutdown deadline has passed are dropped rather than written
// one slow statement at a time.
func TestEventRecorderDropsQueuedEventsPastShutdownDeadline(t *testing.T) {
	repo := &blockingEventRepo{started: make(chan struct{}, 1), release: make(chan struct{})}
	l := zerolog.Nop()

	rec := newEventRecorder(repo, &l)
	ep := &cachedEndpoint{id: uuid.New(), tenantId: uuid.New()}

	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindHEALTHY, "first", nil)
	<-repo.started // the writer is inside Insert with the first event

	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindUNHEALTHY, "second", nil)
	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindHEALTHY, "third", nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stopped := make(chan struct{})

	go func() {
		rec.stop(ctx)
		close(stopped)
	}()

	<-rec.stopCh // stop has asked the writer to finish
	close(repo.release)

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not return once the shutdown deadline had passed")
	}

	batches := repo.seen()
	require.Len(t, batches, 1)
	assert.Equal(t, "first", batches[0][0].Message)
}
