package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/durable"
)

// attemptKey identifies one delivery attempt of a task: the engine retries a task with a
// new retry count and re-invokes a durable task with a new invocation count, and either can
// overlap the previous attempt's cleanup.
type attemptKey struct {
	retry      int32
	invocation int32
}

// noInvocation is the invocation part of a non-durable attempt's key.
const noInvocation = -1

func attemptOf(action *contracts.AssignedAction) attemptKey {
	key := attemptKey{retry: action.RetryCount, invocation: noInvocation}

	if action.DurableTaskInvocationCount != nil {
		key.invocation = *action.DurableTaskInvocationCount
	}

	return key
}

// inflightTask is one delivery in progress. byEngine records that a CANCEL_STEP_RUN, not a
// drain, cancelled it, so the delivery goroutine knows the terminal event was already sent.
type inflightTask struct {
	cancel   context.CancelFunc
	byEngine atomic.Bool
}

// errRegistrationClosed is what HandleAction reports for an action that arrives after the
// registration closed: in process the dispatcher requeues the task, over gRPC the host
// reports a retryable failure, and either way the task goes to a live worker.
var errRegistrationClosed = errors.New("serverless registration is closed")

// errRegistrationNotOpen is what HandleAction reports when the host handed it an action but
// the open that would have produced the session failed.
var errRegistrationNotOpen = errors.New("serverless registration was not opened")

// registration is a tenant's engine session on this process: the operator.Session the host
// opened, the in-flight deliveries keyed by task and attempt, and the action union revision
// it last advertised. It is the session's ActionHandler.
//
// session and events are set once the host's Open returns and ready is closed; a handler
// call that arrives earlier (the in-process host links the initial actions before Open
// returns, so the dispatcher may assign right away) waits on ready.
type registration struct {
	r       *runner
	ts      *tenantState
	session operator.Session
	events  *eventSender
	ready   chan struct{}

	// slots caps the deliveries in flight at the worker's slot count; see HandleAction.
	slots chan struct{}

	inflight map[string]map[attemptKey]*inflightTask
	// advertised is the union the engine holds for this registration (the cache's shared
	// sorted slice, never modified) and advertisedRev its revision: a sync is free while the
	// cache is at the same revision.
	advertised    []string
	advertisedRev uint64
	mu            sync.Mutex
	// syncMu serializes syncActions so two callers (maintenance pass and a poller) never
	// derive and push the same delta twice.
	syncMu sync.Mutex
	active sync.WaitGroup
	closed bool
}

var _ operator.ActionHandler = (*registration)(nil)

// openTimeout bounds one Host.Open, so a hung engine cannot hold a tenant's operations for
// longer than this.
const openTimeout = 30 * time.Second

// sessionOpTimeout bounds one Pause or Close on the session at teardown.
const sessionOpTimeout = 30 * time.Second

// newRegistration builds the registration the host's Open will hand actions to. It is not
// usable until open installs the session.
func newRegistration(r *runner, ts *tenantState, union []string, rev uint64, slotConfig map[string]int32) *registration {
	return &registration{
		r:             r,
		ts:            ts,
		ready:         make(chan struct{}),
		slots:         make(chan struct{}, slotCap(slotConfig)),
		inflight:      map[string]map[attemptKey]*inflightTask{},
		advertised:    union,
		advertisedRev: rev,
	}
}

// open installs the session Open returned, or records that there is none, and releases the
// handler calls waiting on ready.
func (reg *registration) open(session operator.Session) {
	if session != nil {
		reg.session = session
		reg.events = &eventSender{session: session}
	}

	close(reg.ready)
}

// openRegistration connects the tenant with its current action union. Workflows are not part
// of opening: the pollers put them as they learn them, and the engine keeps them. Runs under
// ts.opMu.
func (r *runner) openRegistration(ctx context.Context, ts *tenantState) error {
	ctx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()

	union, rev := ts.cache.ActionUnion()
	slotConfig := r.slotConfig()
	reg := newRegistration(r, ts, union, rev, slotConfig)

	session, err := r.host.Open(ctx, operator.Identity{
		TenantId: ts.tenantId,
		Name:     r.cfg.OperatorName,
		Kind:     r.kind,
	}, operator.OpenOpts{
		Handler:    reg,
		Actions:    union,
		SlotConfig: slotConfig,
		Labels:     map[string]interface{}{workerLabelProcess: r.processId.String()},
		WorkerName: r.workerName,
	})

	if err != nil {
		reg.open(nil)

		if errors.Is(err, hostgrpc.ErrNoToken) {
			if !ts.noToken.Load() {
				r.l.Warn().Str("tenant_id", ts.tenantId.String()).Msg("no token for tenant; endpoints are not polled or registered")
			}

			ts.noToken.Store(true)
			r.markNoToken(ctx, ts)

			return err
		}

		return fmt.Errorf("could not open registration for tenant %s: %w", ts.tenantId, err)
	}

	ts.noToken.Store(false)
	reg.open(session)

	if prev := ts.setRegistration(reg); prev != nil {
		// A stale registration for the tenant is closed without draining; its deliveries
		// report through the closed session and the engine retries them.
		go prev.close()
	}

	r.l.Info().
		Str("tenant_id", ts.tenantId.String()).
		Str("worker_id", session.Registration().WorkerId.String()).
		Int("actions", len(union)).
		Msg("serverless registration opened")

	return nil
}

func (r *runner) slotConfig() map[string]int32 {
	return map[string]int32{
		repository.SlotTypeDefault: r.cfg.DefaultSlots,
		repository.SlotTypeDurable: r.cfg.DurableSlots,
	}
}

// slotCap is the most deliveries a registration runs at once: the worker's slot total, which
// is the most the engine assigns it, and at least one.
func slotCap(slotConfig map[string]int32) int {
	total := 0

	for _, n := range slotConfig {
		if n > 0 {
			total += int(n)
		}
	}

	return max(total, 1)
}

// HandleAction implements operator.ActionHandler: a START_STEP_RUN starts a delivery, a
// CANCEL_STEP_RUN interrupts one, anything else has no meaning for a serverless worker and
// is dropped.
//
// Flow control is by blocking, never by refusing: the deliveries in flight are capped at the
// worker's slot count, and a start that finds every slot taken waits on ctx for one to free
// rather than returning an error. The engine never assigns a worker more than its slots, so
// the wait only happens when an assignment overlaps the report that frees its predecessor's
// slot; and an error would be the wrong answer to it anyway, since in process the dispatcher
// would requeue the task and over gRPC the host would report a FAILED event that burns one
// of the task's retries. In process ctx carries the dispatcher's own send timeout, which
// bounds the wait; over gRPC the host's inbox is unbounded and the wait holds its deliver
// loop, which is the backpressure a saturated worker should apply.
func (reg *registration) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	select {
	case <-reg.ready:
	case <-ctx.Done():
		return ctx.Err()
	}

	if reg.session == nil {
		return errRegistrationNotOpen
	}

	switch action.ActionType {
	case contracts.ActionType_START_STEP_RUN:
		return reg.startDelivery(ctx, action)
	case contracts.ActionType_CANCEL_STEP_RUN:
		reg.cancelTask(action)
		return nil
	default:
		reg.r.l.Warn().
			Str("action_type", action.ActionType.String()).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("serverless registration received unsupported action type")

		return nil
	}
}

// startDelivery takes a slot, records the attempt and delivers it. A second start for an
// attempt already in flight is a duplicate assignment and is ignored; a new attempt of a
// task whose previous attempt is still cleaning up gets its own record.
func (reg *registration) startDelivery(ctx context.Context, action *contracts.AssignedAction) error {
	select {
	case reg.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	release := func() { <-reg.slots }
	key := attemptOf(action)

	reg.mu.Lock()

	if reg.closed {
		reg.mu.Unlock()
		release()

		return errRegistrationClosed
	}

	attempts, ok := reg.inflight[action.TaskRunExternalId]

	if !ok {
		attempts = map[attemptKey]*inflightTask{}
		reg.inflight[action.TaskRunExternalId] = attempts
	}

	if _, dup := attempts[key]; dup {
		reg.mu.Unlock()
		release()
		reg.r.l.Warn().
			Str("task_run_external_id", action.TaskRunExternalId).
			Int32("retry", key.retry).
			Int32("invocation", key.invocation).
			Msg("duplicate assignment for an attempt already in flight; ignored")

		return nil
	}

	dctx, cancel := context.WithCancel(reg.r.deliveryCtx)
	task := &inflightTask{cancel: cancel}
	attempts[key] = task
	reg.active.Add(1)
	reg.mu.Unlock()

	reg.r.wg.Add(1)

	go func() {
		defer reg.r.wg.Done()
		defer release()
		defer reg.active.Done()
		defer reg.finish(action.TaskRunExternalId, key, task)

		reg.deliver(dctx, task, action)
	}()

	return nil
}

// finish removes the attempt's own record only: a newer attempt of the same task keeps its
// record and stays cancellable.
func (reg *registration) finish(taskRunExternalId string, key attemptKey, task *inflightTask) {
	reg.mu.Lock()

	if attempts, ok := reg.inflight[taskRunExternalId]; ok && attempts[key] == task {
		delete(attempts, key)

		if len(attempts) == 0 {
			delete(reg.inflight, taskRunExternalId)
		}
	}

	reg.mu.Unlock()

	task.cancel()
}

// cancelTask interrupts the in-flight delivery of the attempt the cancel names, or of every
// attempt of the task when the named one is not in flight, and reports CANCELLED; the
// delivery goroutine then stays quiet.
func (reg *registration) cancelTask(action *contracts.AssignedAction) {
	key := attemptOf(action)

	reg.mu.Lock()

	var targets []*inflightTask

	if attempts, ok := reg.inflight[action.TaskRunExternalId]; ok {
		if task, ok := attempts[key]; ok {
			targets = append(targets, task)
		} else {
			for _, task := range attempts {
				targets = append(targets, task)
			}
		}
	}

	reg.mu.Unlock()

	for _, task := range targets {
		task.byEngine.Store(true)
		task.cancel()
	}

	if err := reg.events.cancelled(action); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task cancelled")
	}
}

// deliver routes the action to its endpoint by namespace, reports STARTED, delivers, and
// reports the outcome.
func (reg *registration) deliver(ctx context.Context, task *inflightTask, action *contracts.AssignedAction) {
	start := time.Now()

	ep, cfg, err := reg.ts.cache.Route(ctx, action.ActionId)

	if err != nil {
		reg.r.m.routingMiss()
		reg.r.m.delivered("routing_miss", time.Since(start))
		reg.reportFailure(action, err.Error(), true)

		return
	}

	if action.DurableTaskInvocationCount != nil {
		reg.deliverDurable(ctx, task, action, ep, cfg, start)
		return
	}

	if err := reg.events.started(action); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task started")
	}

	out := deliverAction(ctx, reg.r.sender, ep, cfg, action)

	reg.r.m.delivered(out.result, time.Since(start))

	if out.status == contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED {
		reg.reportAborted(task, action)
		return
	}

	if err := reg.events.report(action, out); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task outcome")
	}
}

// deliverDurable relays a durable invocation over the endpoint websocket: report STARTED,
// open the invocation's channel through the session and run the relay, which owns the
// socket and the channel until the endpoint's done frame or a failure. The endpoint's request
// timeout bounds the whole invocation, the host's handshake included.
func (reg *registration) deliverDurable(ctx context.Context, task *inflightTask, action *contracts.AssignedAction, ep *cachedEndpoint, cfg *endpointConfig, start time.Time) {
	invocation := *action.DurableTaskInvocationCount

	dialer, ok := reg.r.sender.(durable.NetDialer)

	if !ok {
		reg.r.m.delivered("failed", time.Since(start))
		reg.reportFailure(action, "durable delivery requires a request sender that dials under the SSRF policy", false)

		return
	}

	if cfg.secretErr != nil {
		reg.r.m.delivered("failed", time.Since(start))
		reg.reportFailure(action, cfg.secretErr.Error(), false)

		return
	}

	taskId, err := uuid.Parse(action.TaskRunExternalId)

	if err != nil {
		reg.r.m.delivered("failed", time.Since(start))
		reg.reportFailure(action, fmt.Sprintf("durable task id %q is not a uuid", action.TaskRunExternalId), false)

		return
	}

	if err := reg.events.started(action); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task started")
	}

	timeout := requestTimeout(cfg)

	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch, err := reg.session.OpenDurable(rctx, taskId, invocation)

	if err != nil {
		result := "retryable"

		if errors.Is(err, operator.ErrNotSupported) {
			result = "durable_unsupported"
		}

		reg.r.m.delivered(result, time.Since(start))
		reg.reportFailure(action, err.Error(), true)

		return
	}

	insecure := false

	if ir, ok := reg.r.sender.(interface{ InsecureDestinations() bool }); ok {
		insecure = ir.InsecureDestinations()
	}

	reg.r.m.wsOpened()

	out := durable.Run(rctx, durable.Params{
		Logger:                reg.r.l,
		Dialer:                dialer,
		Channel:               ch,
		Action:                action,
		Cancelled:             task.byEngine.Load,
		TriggerURL:            cfg.triggerUrl,
		Secret:                cfg.secret,
		EndpointId:            ep.id.String(),
		Namespace:             ep.namespace.String(),
		TaskId:                action.TaskRunExternalId,
		MaxFrameBytes:         reg.r.cfg.WSMaxFrameBytes,
		MaxUpgradeHeaderBytes: reg.r.cfg.WSMaxUpgradeHeaderBytes,
		MaxQueuedBytes:        reg.r.cfg.WSMaxQueuedBytes,
		PingInterval:          reg.r.cfg.WSPingInterval,
		InlineWaitBudgetMs:    cfg.inlineWaitBudgetMs,
		Invocation:            invocation,
		Insecure:              insecure,
	})

	reg.r.m.wsClosed()

	if out.Kind == durable.KindEvicted {
		reg.r.m.evicted(out.EvictionSource)
	}

	o, report := durableOutcome(out)

	reg.r.m.delivered(o.result, time.Since(start))

	reg.r.l.Debug().
		Str("task_run_external_id", action.TaskRunExternalId).
		Int32("invocation", invocation).
		Str("result", o.result).
		Int("close_code", out.CloseCode).
		Msg("durable invocation relayed")

	if !report {
		return
	}

	if err := reg.events.report(action, o); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task outcome")
	}
}

// reportAborted handles a delivery whose context ended: silent after an engine cancel (the
// CANCELLED event was already sent), a retryable failure after a drain timeout.
func (reg *registration) reportAborted(task *inflightTask, action *contracts.AssignedAction) {
	if task.byEngine.Load() {
		return
	}

	reg.reportFailure(action, "delivery aborted: operator shutting down", true)
}

func (reg *registration) reportFailure(action *contracts.AssignedAction, msg string, retry bool) {
	if err := reg.events.failed(action, msg, retry); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task failure")
	}
}

// inFlight counts every delivery in progress, every attempt included.
func (reg *registration) inFlight() int {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	n := 0

	for _, attempts := range reg.inflight {
		n += len(attempts)
	}

	return n
}

// syncActions brings the registration to the cache's union revision: the delta since the
// advertised revision comes from the cache's log, or from a diff against the full union when
// the log no longer reaches back, and is pushed as add and remove deltas, then flushed, so the
// engine sees the union. A failed push leaves the advertised revision unchanged and the next
// sync retries the same delta; both deltas are idempotent on the engine, so a retry after a
// partial push is harmless. A registration at the current revision returns at once.
func (reg *registration) syncActions(ctx context.Context, cache *routingCache) error {
	reg.mu.Lock()
	prevRev := reg.advertisedRev
	reg.mu.Unlock()

	if cache.Revision() == prevRev {
		return nil
	}

	reg.syncMu.Lock()
	defer reg.syncMu.Unlock()

	reg.mu.Lock()
	prev, prevRev := reg.advertised, reg.advertisedRev
	reg.mu.Unlock()

	union, rev := cache.ActionUnion()

	if rev == prevRev {
		return nil
	}

	added, removed, ok := cache.DeltasSince(prevRev)

	if !ok {
		added, removed = diffActions(prev, union)
	}

	if len(added) == 0 && len(removed) == 0 {
		reg.mu.Lock()
		reg.advertised, reg.advertisedRev = union, rev
		reg.mu.Unlock()

		return nil
	}

	if len(added) > 0 {
		if err := reg.session.AddActions(ctx, added); err != nil {
			return fmt.Errorf("could not add actions: %w", err)
		}
	}

	if len(removed) > 0 {
		if err := reg.session.RemoveActions(ctx, removed); err != nil {
			return fmt.Errorf("could not remove actions: %w", err)
		}
	}

	if err := reg.session.Flush(ctx); err != nil {
		return fmt.Errorf("could not flush actions: %w", err)
	}

	reg.mu.Lock()
	reg.advertised, reg.advertisedRev = union, rev
	reg.mu.Unlock()

	reg.r.l.Debug().
		Str("tenant_id", reg.ts.tenantId.String()).
		Int("added", len(added)).
		Int("removed", len(removed)).
		Msg("serverless registration actions synced")

	return nil
}

// diffActions returns the ids in want but not in have (added) and in have but not in want
// (removed), each in the order of the list they come from.
func diffActions(have, want []string) (added, removed []string) {
	haveSet := make(map[string]struct{}, len(have))
	wantSet := make(map[string]struct{}, len(want))

	for _, id := range have {
		haveSet[id] = struct{}{}
	}

	for _, id := range want {
		wantSet[id] = struct{}{}

		if _, ok := haveSet[id]; !ok {
			added = append(added, id)
		}
	}

	for _, id := range have {
		if _, ok := wantSet[id]; !ok {
			removed = append(removed, id)
		}
	}

	return added, removed
}

// teardown is the orderly end of a registration: pause the worker so the engine assigns it
// nothing further, drain the deliveries in flight up to timeout, then close the session,
// which deactivates the worker. The pause is committed before the drain starts, so a task
// assigned during the drain window goes to another worker instead of into a delivery that
// the drain timeout would abort.
func (reg *registration) teardown(timeout time.Duration) {
	reg.pause()
	reg.drain(timeout)
	reg.close()
}

// pause asks the engine to stop assigning to the worker and returns once it has. A pause the
// session refuses (already closed) is logged and the drain proceeds without it.
func (reg *registration) pause() {
	ctx, cancel := context.WithTimeout(context.Background(), sessionOpTimeout)
	defer cancel()

	if err := reg.session.Pause(ctx); err != nil && !errors.Is(err, operator.ErrSessionClosed) {
		reg.r.l.Warn().Err(err).Str("tenant_id", reg.ts.tenantId.String()).Msg("could not pause serverless worker before draining")
	}
}

// drain waits for in-flight deliveries up to timeout, then cancels whatever is left.
func (reg *registration) drain(timeout time.Duration) {
	done := make(chan struct{})

	go func() {
		reg.active.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
	}

	reg.mu.Lock()
	tasks := make([]*inflightTask, 0, len(reg.inflight))

	for _, attempts := range reg.inflight {
		for _, task := range attempts {
			tasks = append(tasks, task)
		}
	}

	reg.mu.Unlock()

	for _, task := range tasks {
		task.cancel()
	}

	<-done
}

// close ends the session; later actions are refused. It runs once.
func (reg *registration) close() {
	reg.mu.Lock()

	if reg.closed {
		reg.mu.Unlock()
		return
	}

	reg.closed = true
	reg.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), sessionOpTimeout)
	defer cancel()

	if err := reg.session.Close(ctx); err != nil {
		reg.r.l.Warn().Err(err).Str("tenant_id", reg.ts.tenantId.String()).Msg("could not close serverless registration")
	}
}
