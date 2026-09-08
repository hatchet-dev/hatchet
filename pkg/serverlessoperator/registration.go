package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/durable"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// inflightTask is one delivery in progress. byEngine records that a CANCEL_STEP_RUN, not a
// drain, cancelled it, so the delivery goroutine knows the terminal event was already sent.
type inflightTask struct {
	cancel   context.CancelFunc
	byEngine atomic.Bool
}

// registration is one owned unit's engine registration: the link Registration, the action
// loop that dispatches assigned actions, the in-flight deliveries and the action set it
// last advertised, from which the next sync derives its delta.
type registration struct {
	r          *runner
	ts         *tenantState
	reg        link.Registration
	events     *eventSender
	loopCancel context.CancelFunc
	loopDone   chan struct{}
	inflight   map[string]*inflightTask
	advertised []string
	shard      int32
	mu         sync.Mutex
	// syncMu serializes syncActions so two callers (maintenance pass and a poller) never
	// derive and push the same delta twice.
	syncMu sync.Mutex
	active sync.WaitGroup
	closed bool
}

func (ts *tenantState) registration(shard int32) *registration {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	return ts.regs[shard]
}

// openTimeout bounds one Link.Open. Open runs under the runner lock, so a hung engine must
// not stall lease reconciliation for longer than this.
const openTimeout = 30 * time.Second

// openRegistration connects the unit with the tenant's current action union. Workflows are
// not part of opening: the pollers put them as they learn them, and the engine keeps them.
func (r *runner) openRegistration(ctx context.Context, ts *tenantState, shard int32) error {
	ctx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()

	union := ts.cache.ActionUnion()

	opts := link.OpenOpts{
		Actions:    union,
		SlotConfig: r.slotConfig(),
		Labels:     map[string]interface{}{workerLabelProcess: r.processId.String()},
	}

	reg, err := r.link.Open(ctx, ts.tenantId, int(shard), opts)

	if err != nil {
		if errors.Is(err, link.ErrNoToken) {
			if !ts.noToken {
				r.l.Warn().Str("tenant_id", ts.tenantId.String()).Msg("no token for tenant; endpoints are not polled or registered")
			}

			ts.noToken = true
			r.markNoToken(ctx, ts)

			return err
		}

		return fmt.Errorf("could not open registration for tenant %s shard %d: %w", ts.tenantId, shard, err)
	}

	ts.noToken = false

	reg2 := &registration{
		r:          r,
		ts:         ts,
		reg:        reg,
		events:     &eventSender{reg: reg},
		shard:      shard,
		inflight:   map[string]*inflightTask{},
		advertised: union,
		loopDone:   make(chan struct{}),
	}

	// The action loop outlives ctx (the reconcile call) and ends with the runner's loop
	// context or an explicit drain; loopCancel is set before the registration is published
	// so a concurrent UnitsLost can always drain it.
	loopCtx, cancel := context.WithCancel(r.loopCtx)
	reg2.loopCancel = cancel

	ts.mu.Lock()
	prev := ts.regs[shard]
	ts.regs[shard] = reg2
	ts.mu.Unlock()

	if prev != nil {
		// A stale registration for the shard is closed without draining; its deliveries
		// report through the closed link and the engine retries them.
		go prev.close()
	}

	r.wg.Add(1)

	go func() {
		defer r.wg.Done()
		reg2.run(loopCtx)
	}()

	r.l.Info().
		Str("tenant_id", ts.tenantId.String()).
		Int32("shard", shard).
		Str("worker_id", reg.WorkerId()).
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

// run is the action loop. It ends when the loop context is cancelled (unit lost, shutdown)
// or the link fails permanently, in which case the registration is dropped from the tenant so
// the maintenance loop reopens it.
func (reg *registration) run(ctx context.Context) {
	defer close(reg.loopDone)

	ch, errCh, err := reg.reg.Actions(ctx)

	if err != nil {
		reg.r.l.Error().Err(err).Int32("shard", reg.shard).Msg("could not start serverless action stream")
		reg.failed()

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case action, ok := <-ch:
			if !ok {
				if ctx.Err() == nil {
					reg.failed()
				}

				return
			}

			reg.handle(action)
		case err, ok := <-errCh:
			if !ok {
				// A closed error channel means the stream ended cleanly; the action channel
				// closing decides what happens next.
				errCh = nil
				continue
			}

			if err != nil && ctx.Err() == nil {
				reg.r.l.Error().Err(err).Int32("shard", reg.shard).Msg("serverless action stream failed")
				reg.failed()

				return
			}
		}
	}
}

// failed detaches a broken registration so the maintenance loop opens a fresh one.
func (reg *registration) failed() {
	reg.ts.mu.Lock()

	if reg.ts.regs[reg.shard] == reg {
		delete(reg.ts.regs, reg.shard)
	}

	reg.ts.mu.Unlock()

	reg.r.m.sessionReconnect()

	go reg.close()
}

func (reg *registration) handle(action *contracts.AssignedAction) {
	switch action.ActionType {
	case contracts.ActionType_START_STEP_RUN:
		reg.startDelivery(action)
	case contracts.ActionType_CANCEL_STEP_RUN:
		reg.cancelTask(action)
	default:
		reg.r.l.Warn().
			Str("action_type", action.ActionType.String()).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("serverless registration received unsupported action type")
	}
}

func (reg *registration) startDelivery(action *contracts.AssignedAction) {
	reg.mu.Lock()

	if reg.closed {
		reg.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(reg.r.deliveryCtx)
	task := &inflightTask{cancel: cancel}
	reg.inflight[action.TaskRunExternalId] = task
	reg.active.Add(1)
	reg.mu.Unlock()

	reg.r.wg.Add(1)

	go func() {
		defer reg.r.wg.Done()
		defer reg.active.Done()
		defer reg.finish(action.TaskRunExternalId, cancel)

		reg.deliver(ctx, task, action)
	}()
}

func (reg *registration) finish(taskRunExternalId string, cancel context.CancelFunc) {
	reg.mu.Lock()
	delete(reg.inflight, taskRunExternalId)
	reg.mu.Unlock()

	cancel()
}

// cancelTask interrupts the in-flight delivery, if any, and reports CANCELLED; the
// delivery goroutine then stays quiet.
func (reg *registration) cancelTask(action *contracts.AssignedAction) {
	reg.mu.Lock()
	task, ok := reg.inflight[action.TaskRunExternalId]
	reg.mu.Unlock()

	if ok {
		task.byEngine.Store(true)
		task.cancel()
	}

	if err := reg.events.cancelled(action); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task cancelled")
	}
}

// deliver routes the action to its endpoint by namespace, takes an endpoint slot, reports
// STARTED, delivers, and reports the outcome.
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
// open the invocation's channel through the registration and run the relay, which owns the
// socket and the channel until the endpoint's done frame or a failure.
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

	if err := reg.events.started(action); err != nil {
		reg.r.l.Error().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report task started")
	}

	ch, err := reg.reg.OpenDurable(ctx, action.TaskRunExternalId, invocation)

	if err != nil {
		result := "retryable"

		if errors.Is(err, link.ErrDurableNotSupported) {
			result = "durable_unsupported"
		}

		reg.r.m.delivered(result, time.Since(start))
		reg.reportFailure(action, err.Error(), true)

		return
	}

	timeout := requestTimeout(cfg)

	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	insecure := false

	if ir, ok := reg.r.sender.(interface{ InsecureDestinations() bool }); ok {
		insecure = ir.InsecureDestinations()
	}

	reg.r.m.wsOpened()

	out := durable.Run(rctx, durable.Params{
		Logger:             reg.r.l,
		Dialer:             dialer,
		Channel:            ch,
		Action:             action,
		Cancelled:          task.byEngine.Load,
		TriggerURL:         cfg.triggerUrl,
		Secret:             cfg.secret,
		EndpointId:         ep.id.String(),
		Namespace:          ep.namespace.String(),
		TaskId:             action.TaskRunExternalId,
		MaxFrameBytes:      reg.r.cfg.WSMaxFrameBytes,
		PingInterval:       reg.r.cfg.WSPingInterval,
		InlineWaitBudgetMs: cfg.inlineWaitBudgetMs,
		Invocation:         invocation,
		Insecure:           insecure,
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

func (reg *registration) inFlight() int {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	return len(reg.inflight)
}

// syncActions pushes the difference between the last advertised set and union as add and
// remove deltas, then flushes, so the engine sees the union. A failed push leaves advertised
// unchanged and the next sync retries the same delta; both deltas are idempotent on the
// engine, so a retry after a partial push is harmless.
func (reg *registration) syncActions(ctx context.Context, union []string) error {
	reg.syncMu.Lock()
	defer reg.syncMu.Unlock()

	reg.mu.Lock()
	prev := reg.advertised
	reg.mu.Unlock()

	added, removed := diffActions(prev, union)

	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	if len(added) > 0 {
		if err := reg.reg.AddActions(ctx, added); err != nil {
			return fmt.Errorf("could not add actions: %w", err)
		}
	}

	if len(removed) > 0 {
		if err := reg.reg.RemoveActions(ctx, removed); err != nil {
			return fmt.Errorf("could not remove actions: %w", err)
		}
	}

	if err := reg.reg.Flush(ctx); err != nil {
		return fmt.Errorf("could not flush actions: %w", err)
	}

	reg.mu.Lock()
	reg.advertised = union
	reg.mu.Unlock()

	reg.r.l.Debug().
		Str("tenant_id", reg.ts.tenantId.String()).
		Int32("shard", reg.shard).
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

// drain stops reading actions and waits for in-flight deliveries up to timeout, then cancels
// whatever is left.
func (reg *registration) drain(timeout time.Duration) {
	reg.loopCancel()
	<-reg.loopDone

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

	for _, task := range reg.inflight {
		tasks = append(tasks, task)
	}

	reg.mu.Unlock()

	for _, task := range tasks {
		task.cancel()
	}

	<-done
}

func (reg *registration) close() {
	reg.mu.Lock()

	if reg.closed {
		reg.mu.Unlock()
		return
	}

	reg.closed = true
	reg.mu.Unlock()

	if reg.loopCancel != nil {
		reg.loopCancel()
	}

	if err := reg.reg.Close(); err != nil {
		reg.r.l.Warn().Err(err).Int32("shard", reg.shard).Msg("could not close serverless registration")
	}
}
