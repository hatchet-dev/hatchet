package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
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
// last advertised.
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
	active     sync.WaitGroup
	closed     bool
}

func (ts *tenantState) registration(shard int32) *registration {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	return ts.regs[shard]
}

// openTimeout bounds one Link.Open. Open runs under the runner lock, so a hung engine must
// not stall lease reconciliation for longer than this.
const openTimeout = 30 * time.Second

// openRegistration connects the unit with the tenant's known workflows and action union. A
// connect rejected because of a workflow is retried without workflows, which are then put one
// by one so a single bad endpoint marks itself unhealthy instead of blocking the tenant.
func (r *runner) openRegistration(ctx context.Context, ts *tenantState, shard int32) error {
	ctx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()

	union := ts.cache.ActionUnion()

	opts := link.OpenOpts{
		Workflows:  ts.cache.Workflows(),
		Actions:    union,
		SlotConfig: r.slotConfig(),
		Labels:     map[string]interface{}{workerLabelProcess: r.processId.String()},
	}

	reg, err := r.link.Open(ctx, ts.tenantId, int(shard), opts)

	var rejected []*v1.CreateWorkflowVersionRequest

	if err != nil && !errors.Is(err, link.ErrNoToken) && len(opts.Workflows) > 0 {
		r.l.Warn().Err(err).Str("tenant_id", ts.tenantId.String()).Msg("registration open with workflows failed; retrying without them")

		rejected = opts.Workflows
		opts.Workflows = nil
		reg, err = r.link.Open(ctx, ts.tenantId, int(shard), opts)
	}

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

	for _, wf := range rejected {
		if err := reg2.putWorkflow(ctx, wf, union); err != nil {
			r.markWorkflowRejected(ctx, ts, wf, err)
		}
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

// markWorkflowRejected flags the endpoint whose workflow the engine refused.
func (r *runner) markWorkflowRejected(ctx context.Context, ts *tenantState, wf *v1.CreateWorkflowVersionRequest, err error) {
	ns, ok := ParseNamespace(wf.Name)

	if !ok {
		return
	}

	for _, ep := range ts.ownedEndpoints() {
		if ep.namespace == ns {
			r.writeStatus(ctx, ts, ep, false, fmt.Sprintf("engine rejected workflow %s: %s", wf.Name, err.Error()))
		}
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

	// TODO(phase 5): durable actions open a DurableChannel through the registration and are
	// relayed over the endpoint websocket. Until then they fail retryably so the engine keeps
	// them.
	if action.DurableTaskInvocationCount != nil {
		reg.r.m.delivered("durable_unsupported", time.Since(start))
		reg.reportFailure(action, "durable delivery not implemented", true)

		return
	}

	if err := ep.limiter.acquire(ctx); err != nil {
		reg.reportAborted(task, action)
		return
	}

	defer ep.limiter.release()

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

// putWorkflow registers one workflow with the full action set and remembers the set as
// advertised, since the engine applies it in the same call.
func (reg *registration) putWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest, fullActions []string) error {
	if err := reg.reg.PutWorkflow(ctx, wf, fullActions); err != nil {
		return err
	}

	reg.mu.Lock()
	reg.advertised = fullActions
	reg.mu.Unlock()

	return nil
}

// syncActions calls UpdateActions when the union differs from what was last advertised.
func (reg *registration) syncActions(ctx context.Context, union []string) error {
	reg.mu.Lock()
	same := stringsEqual(reg.advertised, union)
	reg.mu.Unlock()

	if same {
		return nil
	}

	if err := reg.reg.UpdateActions(ctx, union); err != nil {
		return err
	}

	reg.mu.Lock()
	reg.advertised = union
	reg.mu.Unlock()

	return nil
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
