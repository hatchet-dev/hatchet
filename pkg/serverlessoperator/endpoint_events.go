package serverlessoperator

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// eventBufferSize bounds the events waiting for the writer. A full buffer drops: events
	// are diagnostics, and a slow database must never hold up a poll or a delivery.
	eventBufferSize = 1024

	// eventBatchSize is the most events one insert carries.
	eventBatchSize = 100

	// eventWriteTimeout bounds one batch insert, and the final flush at shutdown.
	eventWriteTimeout = 5 * time.Second

	// failureEventWindow is the least time between two events of one failure kind
	// (HEALTHCHECK_FAILED, CATALOG_REJECTED, DELIVERY_FAILED) of one endpoint. The failures in
	// between are counted on the next event of the kind instead, so a failing endpoint writes
	// one row per window and kind, however often it fails and however its error text varies.
	failureEventWindow = 30 * time.Second

	// endpointEventBudget is the most events of every kind one endpoint writes per
	// endpointEventBudgetPeriod; the rest are counted on the next written event of their
	// kind. 120 an hour is what one failure kind can sustain under failureEventWindow, so an
	// endpoint failing one way (deliveries under load, say) is never cut short; only an
	// endpoint failing several ways at once, or flapping between healthy and unhealthy, runs
	// into it. It bounds one endpoint to 2,880 rows a day.
	endpointEventBudget       = 120
	endpointEventBudgetPeriod = time.Hour

	// eventExcerptBytes caps the response body kept on an event.
	eventExcerptBytes = 1024

	// maxEventMessageBytes caps an event's message.
	maxEventMessageBytes = 4096

	// eventMaxListItems and eventMaxNameBytes bound the workflows and actions an event lists,
	// and every name or identifier it carries; eventMaxURLBytes bounds its URL. A catalog
	// event's details stay within about 32 KiB, and the endpoint row keeps the full action
	// set anyway.
	eventMaxListItems = 50
	eventMaxNameBytes = 256
	eventMaxURLBytes  = 2048
)

// Healthcheck stages, as the API reports them: how far a healthcheck got before it failed.
const (
	stageRequest = "REQUEST"
	stageStatus  = "STATUS"
	stageParse   = "PARSE"
	stageCatalog = "CATALOG"
)

// eventDetails is the details object of an endpoint event. Its JSON names are those of the
// API's V1ServerlessEndpointEventDetails, which decodes the stored object as is.
type eventDetails struct {
	RetryCount        *int32   `json:"retryCount,omitempty"`
	Retryable         *bool    `json:"retryable,omitempty"`
	Stage             string   `json:"stage,omitempty"`
	URL               string   `json:"url,omitempty"`
	ResponseExcerpt   string   `json:"responseExcerpt,omitempty"`
	TaskRunExternalId string   `json:"taskRunExternalId,omitempty"`
	ActionId          string   `json:"actionId,omitempty"`
	Actions           []string `json:"actions,omitempty"`
	Workflows         []string `json:"workflows,omitempty"`
	StatusCode        int      `json:"statusCode,omitempty"`
	DurationMs        int64    `json:"durationMs,omitempty"`
	Suppressed        int      `json:"suppressed,omitempty"`
	Truncated         bool     `json:"truncated,omitempty"`
}

// safeCopy returns a copy of the details for storage: every string made safe by safeText and
// bounded, the lists capped at eventMaxListItems, and Truncated set when anything was cut. The
// caller's details are left alone, so one value may back several events.
func (d *eventDetails) safeCopy() *eventDetails {
	if d == nil {
		return nil
	}

	out := *d

	var cut [8]bool

	out.Stage, cut[0] = safeText(d.Stage, eventMaxNameBytes)
	out.URL, cut[1] = safeText(d.URL, eventMaxURLBytes)
	out.ResponseExcerpt, cut[2] = safeText(d.ResponseExcerpt, eventExcerptBytes)
	out.TaskRunExternalId, cut[3] = safeText(d.TaskRunExternalId, eventMaxNameBytes)
	out.ActionId, cut[4] = safeText(d.ActionId, eventMaxNameBytes)
	out.Actions, cut[5] = safeList(d.Actions)
	out.Workflows, cut[6] = safeList(d.Workflows)

	for _, c := range cut {
		out.Truncated = out.Truncated || c
	}

	return &out
}

func safeList(items []string) ([]string, bool) {
	if items == nil {
		return nil, false
	}

	cut := len(items) > eventMaxListItems

	if cut {
		items = items[:eventMaxListItems]
	}

	out := make([]string, len(items))

	for i, item := range items {
		var c bool

		out[i], c = safeText(item, eventMaxNameBytes)
		cut = cut || c
	}

	return out, cut
}

// eventWindow is one endpoint's rate limit state for one event kind.
type eventWindow struct {
	lastWritten time.Time
	suppressed  int
}

// endpointEvents is one endpoint's rate limit state: a window per kind, and the budget it
// has spent across every kind since budgetStart.
type endpointEvents struct {
	kinds          map[sqlcv1.V1ServerlessEndpointEventKind]*eventWindow
	budgetStart    time.Time
	lastSeen       time.Time
	budgetUsed     int
	overBudgetSeen bool
}

// windowedKinds are the failure kinds limited to one event per failureEventWindow.
var windowedKinds = map[sqlcv1.V1ServerlessEndpointEventKind]bool{
	sqlcv1.V1ServerlessEndpointEventKindHEALTHCHECKFAILED: true,
	sqlcv1.V1ServerlessEndpointEventKindCATALOGREJECTED:   true,
	sqlcv1.V1ServerlessEndpointEventKindDELIVERYFAILED:    true,
}

// eventRecorder writes endpoint events off the caller's path: record only enqueues, and a
// single writer goroutine batches the queue into inserts.
type eventRecorder struct {
	repo      repository.ServerlessEndpointEventRepository
	l         *zerolog.Logger
	now       func() time.Time
	queue     chan repository.ServerlessEndpointEventOpts
	stopCh    chan struct{}
	done      chan struct{}
	endpoints map[uuid.UUID]*endpointEvents
	dropped   atomic.Int64
	// stopCtx bounds the flush at shutdown; set by stop before stopCh is closed.
	stopCtx  context.Context
	stopOnce sync.Once
	mu       sync.Mutex
}

func newEventRecorder(repo repository.ServerlessEndpointEventRepository, l *zerolog.Logger) *eventRecorder {
	rec := &eventRecorder{
		repo:      repo,
		l:         l,
		now:       time.Now,
		queue:     make(chan repository.ServerlessEndpointEventOpts, eventBufferSize),
		stopCh:    make(chan struct{}),
		done:      make(chan struct{}),
		endpoints: map[uuid.UUID]*endpointEvents{},
	}

	go rec.run()

	return rec
}

// record enqueues an event, unless the endpoint's rate limits hold it back (see admit) or
// the queue is full. A held back event is counted as suppressed on the endpoint's next event
// of the same kind. The message and details are stored through safeText.
func (rec *eventRecorder) record(ep *cachedEndpoint, kind sqlcv1.V1ServerlessEndpointEventKind, message string, details *eventDetails) {
	if rec == nil {
		return
	}

	suppressed, ok := rec.admit(ep, kind)

	if !ok {
		return
	}

	details = details.safeCopy()

	if suppressed > 0 {
		if details == nil {
			details = &eventDetails{}
		}

		details.Suppressed = suppressed
	}

	// A failure's message may be the endpoint's own error text.
	message, _ = safeText(message, maxEventMessageBytes)

	ev := repository.ServerlessEndpointEventOpts{
		CreatedAt:  rec.now(),
		Kind:       kind,
		Message:    message,
		TenantId:   ep.tenantId,
		EndpointId: ep.id,
	}

	if details != nil {
		ev.Details = details
	}

	select {
	case rec.queue <- ev:
	default:
		if rec.dropped.Add(1) == 1 {
			rec.l.Warn().Str("endpoint_id", ep.id.String()).Msg("serverless endpoint event queue full; dropping events")
		}
	}
}

// deliveryFailed records a failed delivery, at most one per failureEventWindow.
func (rec *eventRecorder) deliveryFailed(ep *cachedEndpoint, message string, details *eventDetails) {
	rec.record(ep, sqlcv1.V1ServerlessEndpointEventKindDELIVERYFAILED, message, details)
}

// admit decides whether the endpoint writes an event of kind now. A failure kind written
// within failureEventWindow, or any kind once the endpoint has spent endpointEventBudget in
// the current period, is only counted. An admitted event returns the count of its kind held
// back since the last one written, and resets it.
func (rec *eventRecorder) admit(ep *cachedEndpoint, kind sqlcv1.V1ServerlessEndpointEventKind) (int, bool) {
	now := rec.now()

	rec.mu.Lock()

	st, ok := rec.endpoints[ep.id]

	if !ok {
		rec.pruneEndpointsLocked(now)
		st = &endpointEvents{kinds: map[sqlcv1.V1ServerlessEndpointEventKind]*eventWindow{}}
		rec.endpoints[ep.id] = st
	}

	st.lastSeen = now

	w, ok := st.kinds[kind]

	if !ok {
		w = &eventWindow{}
		st.kinds[kind] = w
	}

	if windowedKinds[kind] && !w.lastWritten.IsZero() && now.Sub(w.lastWritten) < failureEventWindow {
		w.suppressed++
		rec.mu.Unlock()

		return 0, false
	}

	if now.Sub(st.budgetStart) >= endpointEventBudgetPeriod {
		st.budgetStart = now
		st.budgetUsed = 0
		st.overBudgetSeen = false
	}

	if st.budgetUsed >= endpointEventBudget {
		w.suppressed++
		first := !st.overBudgetSeen
		st.overBudgetSeen = true
		rec.mu.Unlock()

		if first {
			rec.l.Warn().Str("endpoint_id", ep.id.String()).Str("tenant_id", ep.tenantId.String()).
				Msg("serverless endpoint over its event budget; counting further events until the period ends")
		}

		return 0, false
	}

	st.budgetUsed++

	suppressed := w.suppressed
	w.suppressed = 0
	w.lastWritten = now

	rec.mu.Unlock()

	return suppressed, true
}

// pruneEndpointsLocked forgets endpoints without an event for a whole budget period, so the
// map stays the size of the endpoints with recent events rather than of every endpoint that
// ever had one. A forgotten endpoint's suppressed counts are lost, which only drops counts
// for windows and a budget period that have ended.
func (rec *eventRecorder) pruneEndpointsLocked(now time.Time) {
	if len(rec.endpoints) < 1024 {
		return
	}

	for id, st := range rec.endpoints {
		if now.Sub(st.lastSeen) >= endpointEventBudgetPeriod {
			delete(rec.endpoints, id)
		}
	}
}

func (rec *eventRecorder) run() {
	defer close(rec.done)

	for {
		// A stop takes precedence over queued events, so they are flushed under its deadline.
		select {
		case <-rec.stopCh:
			rec.flush(rec.stopCtx)
			return
		default:
		}

		select {
		case <-rec.stopCh:
			rec.flush(rec.stopCtx)
			return
		case ev := <-rec.queue:
			rec.write(context.Background(), rec.drain([]repository.ServerlessEndpointEventOpts{ev}))
		}
	}
}

// drain adds whatever is already queued to batch, up to eventBatchSize.
func (rec *eventRecorder) drain(batch []repository.ServerlessEndpointEventOpts) []repository.ServerlessEndpointEventOpts {
	for len(batch) < eventBatchSize {
		select {
		case ev := <-rec.queue:
			batch = append(batch, ev)
		default:
			return batch
		}
	}

	return batch
}

// flush writes what is queued at shutdown, without waiting for more. Once ctx ends, what is
// still queued is dropped: a slow database must not hold the process past its deadline.
func (rec *eventRecorder) flush(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			if n := len(rec.queue); n > 0 {
				rec.l.Warn().Int("events", n).Msg("shutdown deadline passed; dropping queued serverless endpoint events")
			}

			return
		}

		batch := rec.drain(nil)

		if len(batch) == 0 {
			return
		}

		rec.write(ctx, batch)
	}
}

// write inserts a batch with eventWriteTimeout within ctx.
func (rec *eventRecorder) write(ctx context.Context, batch []repository.ServerlessEndpointEventOpts) {
	ctx, cancel := context.WithTimeout(ctx, eventWriteTimeout)
	defer cancel()

	err := rec.repo.Insert(ctx, batch)

	if err == nil {
		return
	}

	if len(batch) == 1 || !repository.IsServerlessEventRowError(err) {
		rec.l.Warn().Err(err).Int("events", len(batch)).Msg("could not write serverless endpoint events")
		return
	}

	// The database refused a value of some event, which fails the whole statement. The events
	// are written one by one within the same deadline, so only the events at fault are lost
	// and not those of every other endpoint and tenant in the batch.
	for i, ev := range batch {
		if ctx.Err() != nil {
			rec.l.Warn().Err(ctx.Err()).Int("events", len(batch)-i).Msg("could not write serverless endpoint events")
			return
		}

		if err := rec.repo.Insert(ctx, batch[i:i+1]); err != nil {
			rec.l.Warn().Err(err).
				Str("endpoint_id", ev.EndpointId.String()).
				Str("tenant_id", ev.TenantId.String()).
				Str("kind", string(ev.Kind)).
				Msg("could not write serverless endpoint event")
		}
	}
}

// stop flushes the queue within ctx and stops the writer. Events still queued when ctx
// ends, or recorded afterwards, are dropped.
func (rec *eventRecorder) stop(ctx context.Context) {
	if rec == nil {
		return
	}

	rec.stopOnce.Do(func() {
		rec.stopCtx = ctx
		close(rec.stopCh)
	})

	<-rec.done
}

// excerpt is the start of body as text, at most limit bytes, made safe by safeText.
func excerpt(body []byte, limit int) string {
	// One byte past the limit is kept so safeText sees the cut and drops a rune it split.
	if len(body) > limit+1 {
		body = body[:limit+1]
	}

	out, _ := safeText(string(body), limit)

	return out
}

const hexDigits = "0123456789abcdef"

// safeText returns s as text that Postgres stores in JSONB and that prints as it reads, with
// whatever was unsafe left visible rather than removed: invalid UTF-8 becomes U+FFFD, NUL
// (which JSONB refuses) the text \u0000, the other C0 controls but \n and \t, and DEL, the
// text \xNN, and the C1 controls the text \u00NN. An endpoint's text then cannot move a
// terminal's cursor, erase what precedes it or restyle it, wherever it is shown. The result
// is at most limit bytes, cut between runes and escapes; cut reports whether anything was
// left out.
func safeText(s string, limit int) (out string, cut bool) {
	// Nothing gets shorter, so input past the limit can never fit.
	if len(s) > limit {
		s = s[:limit+1]
		cut = true
	}

	if !cut && isPlainText(s) {
		return s, false
	}

	var b strings.Builder

	b.Grow(min(len(s), limit))

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])

		var piece string

		switch {
		case r == utf8.RuneError && size == 1 && cut && !utf8.FullRuneInString(s[i:]):
			// A rune the cut split.
			return b.String(), true
		case r == utf8.RuneError && size == 1:
			piece = string(utf8.RuneError)
		case r == 0:
			piece = `\u0000`
		case r < 0x20 && r != '\n' && r != '\t', r == 0x7f:
			piece = `\x` + string(hexDigits[r>>4]) + string(hexDigits[r&0xf])
		case r >= 0x80 && r <= 0x9f:
			piece = `\u00` + string(hexDigits[r>>4]) + string(hexDigits[r&0xf])
		default:
			piece = s[i : i+size]
		}

		if b.Len()+len(piece) > limit {
			return b.String(), true
		}

		b.WriteString(piece)
		i += size
	}

	return b.String(), cut
}

// isPlainText reports whether s is valid UTF-8 without anything safeText escapes.
func isPlainText(s string) bool {
	for _, r := range s {
		if r == utf8.RuneError || r < 0x20 && r != '\n' && r != '\t' || r >= 0x7f && r <= 0x9f {
			return false
		}
	}

	return true
}

// workflowNames lists the names of the workflows a healthcheck declared, for catalog events.
func workflowNames(res *healthcheckResult) []string {
	names := make([]string, 0, len(res.workflows))

	for _, wf := range res.workflows {
		names = append(names, wf.Name)
	}

	return names
}
