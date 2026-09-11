package durable

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

const (
	// sendQueueSize is how many frames may wait for the endpoint before the relay gives
	// up on it with CloseBackpressure.
	sendQueueSize = 256

	// missedPongLimit is how many pings may go unanswered before the socket is closed.
	missedPongLimit = 2

	defaultHandshakeTimeout      = 30 * time.Second
	defaultPingInterval          = 15 * time.Second
	defaultMaxFrameBytes         = 4 * 1024 * 1024
	defaultMaxUpgradeHeaderBytes = 64 * 1024
	defaultMaxQueuedBytes        = 16 * 1024 * 1024

	// closeWriteTimeout bounds writing the close frame at teardown.
	closeWriteTimeout = 5 * time.Second

	// frameWriteTimeout bounds one frame write; an endpoint that stops reading is closed
	// by the pong watchdog anyway, this keeps a stalled write from outliving it.
	frameWriteTimeout = 30 * time.Second
)

// Kind is the class of a relay outcome.
type Kind int

const (
	// KindCompleted: the endpoint sent done with an output; report COMPLETED.
	KindCompleted Kind = iota + 1
	// KindFailed: report FAILED with Outcome.Error and Outcome.Retry.
	KindFailed
	// KindEvicted: the invocation was evicted by the endpoint or the engine; no terminal
	// event, the engine re-invokes when the awaited entry is satisfied.
	KindEvicted
	// KindCancelled: the engine cancelled the task; the CANCELLED event was already sent.
	KindCancelled
	// KindShutdown: the operator is stopping; no event, the engine re-delivers the task
	// when the worker goes away.
	KindShutdown
)

// Eviction sources for the evictions_total metric.
const (
	EvictionSourceEndpoint = "endpoint"
	EvictionSourceServer   = "server"
)

// Outcome is what one relayed invocation reports back.
type Outcome struct {
	Output         json.RawMessage
	Error          string
	EvictionSource string
	Kind           Kind
	CloseCode      int
	Retry          bool
}

func completed(output json.RawMessage) Outcome {
	if len(output) == 0 {
		output = json.RawMessage("{}")
	}

	return Outcome{Kind: KindCompleted, Output: output, CloseCode: CloseNormal}
}

func failed(closeCode int, msg string, retry bool) Outcome {
	return Outcome{Kind: KindFailed, Error: msg, Retry: retry, CloseCode: closeCode}
}

func evicted(closeCode int, source string) Outcome {
	return Outcome{Kind: KindEvicted, EvictionSource: source, CloseCode: closeCode}
}

// Params describes one invocation to relay. Channel is opened by the caller and closed by
// Run on every exit.
type Params struct {
	Logger  *zerolog.Logger
	Dialer  NetDialer
	Channel operator.DurableChannel
	Action  *contracts.AssignedAction

	// Cancelled reports, once ctx is done, whether the engine cancelled the task (close
	// 4003, no event: the CANCELLED event was already sent) as opposed to the operator
	// shutting down (close 4002, no event). nil means shutting down.
	Cancelled func() bool

	TriggerURL string
	Secret     string
	EndpointId string
	Namespace  string
	TaskId     string

	MaxFrameBytes int64

	// MaxUpgradeHeaderBytes bounds the endpoint's upgrade response head (status line and
	// headers); 64 KiB by default.
	MaxUpgradeHeaderBytes int64

	// MaxQueuedBytes bounds the encoded frames waiting for the endpoint, in addition to the
	// sendQueueSize frame count; crossing it closes the socket with CloseBackpressure. 16 MiB
	// by default. A single frame larger than the budget trips it on its own.
	MaxQueuedBytes int64

	PingInterval     time.Duration
	HandshakeTimeout time.Duration

	InlineWaitBudgetMs int32
	Invocation         int32

	// Insecure allows http trigger URLs (ws) and must only be set together with
	// safeclient's InsecureDestinations.
	Insecure bool

	// Test hooks.
	now       func() time.Time
	nonce     func() (string, error)
	tlsConfig *tls.Config
	writeGate <-chan struct{}
}

func (p *Params) withDefaults() {
	if p.MaxFrameBytes <= 0 {
		p.MaxFrameBytes = defaultMaxFrameBytes
	}

	if p.MaxUpgradeHeaderBytes <= 0 {
		p.MaxUpgradeHeaderBytes = defaultMaxUpgradeHeaderBytes
	}

	if p.MaxQueuedBytes <= 0 {
		p.MaxQueuedBytes = defaultMaxQueuedBytes
	}

	if p.PingInterval <= 0 {
		p.PingInterval = defaultPingInterval
	}

	if p.HandshakeTimeout <= 0 {
		p.HandshakeTimeout = defaultHandshakeTimeout
	}

	if p.now == nil {
		p.now = time.Now
	}

	if p.nonce == nil {
		p.nonce = newNonce
	}

	if p.Logger == nil {
		nop := zerolog.Nop()
		p.Logger = &nop
	}
}

func (p *Params) newNonce() (string, error) {
	return p.nonce()
}

// relay is the per-socket state. finish records the first exit, resolves it against the
// phase (see outcome.go) and closes stop, which every goroutine watches; later exits are
// dropped.
type relay struct {
	p          *Params
	conn       *websocket.Conn
	ctx        context.Context
	cancel     context.CancelFunc
	l          *zerolog.Logger
	sendQ      chan []byte
	stop       chan struct{}
	writerDone chan struct{}
	engineErr  atomic.Pointer[string]
	result     Outcome
	once       sync.Once
	wg         sync.WaitGroup
	queued     atomic.Int64
	phase      atomic.Int32
	missed     atomic.Int32
	done       atomic.Bool
	peerClosed atomic.Bool
}

// Run relays one invocation and returns its outcome. ctx is the delivery context: the
// endpoint's request timeout is expected to be applied to it by the caller; cancellation
// maps to KindCancelled or KindShutdown through Params.Cancelled.
func Run(ctx context.Context, p Params) Outcome {
	p.withDefaults()

	if p.Channel == nil || p.Dialer == nil || p.Action == nil {
		return failed(0, "durable relay misconfigured: channel, dialer and action are required", false)
	}

	defer func() {
		_ = p.Channel.Close()
	}()

	first, err := buildFirstFrame(&p)

	if err != nil {
		return failed(0, err.Error(), false)
	}

	conn, err := dial(ctx, &p)

	if err != nil {
		return classifyDialError(ctx, &p, err)
	}

	conn.SetReadLimit(p.MaxFrameBytes)

	// The channel calls of the pump goroutines run on the relay's own context, cancelled by
	// teardown once the channel is closed, so no goroutine outlives Run. It is deliberately
	// not derived from ctx: the delivery context ending is Run's exit (an abort through the
	// select below), and a Recv that returned on it first would look like an engine failure.
	rctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &relay{
		p:          &p,
		conn:       conn,
		ctx:        rctx,
		cancel:     cancel,
		l:          p.Logger,
		sendQ:      make(chan []byte, sendQueueSize),
		stop:       make(chan struct{}),
		writerDone: make(chan struct{}),
	}

	conn.SetPongHandler(func(string) error {
		r.missed.Store(0)
		return nil
	})

	if err := r.write(first); err != nil {
		r.finish(exit{kind: exitClosedWithoutDone, closeCode: CloseInternalError, msg: fmt.Sprintf("could not send action frame: %s", err.Error())})
		_ = conn.Close()

		return r.result
	}

	r.wg.Add(3)

	go r.readLoop()
	go r.pumpLoop()
	go r.writeLoop()

	select {
	case <-r.stop:
	case <-ctx.Done():
		r.finish(abortExit(ctx, &p))
	}

	r.teardown()

	return r.result
}

// buildFirstFrame encodes the first frame: the assigned action (action id and workflow name
// namespaced as registered), the endpoint's namespace, the invocation count and the inline
// wait budget.
func buildFirstFrame(p *Params) ([]byte, error) {
	frame, err := contract.MarshalFrame(&v1.ServerlessDurableFrame{
		Frame: &v1.ServerlessDurableFrame_First{First: &v1.ServerlessFirstFrame{
			Action:             p.Action,
			Namespace:          p.Namespace,
			InvocationCount:    p.Invocation,
			InlineWaitBudgetMs: p.InlineWaitBudgetMs,
		}},
	})

	if err != nil {
		return nil, fmt.Errorf("could not encode action frame: %w", err)
	}

	if int64(len(frame)) > p.MaxFrameBytes {
		return nil, fmt.Errorf("action frame of %d bytes exceeds the %d byte frame limit", len(frame), p.MaxFrameBytes)
	}

	return frame, nil
}

// classifyDialError maps a failed upgrade. Policy blocks are configuration errors and are
// not retried; a refused upgrade follows the status rule; anything else is transient.
func classifyDialError(ctx context.Context, p *Params, err error) Outcome {
	if ctx.Err() != nil {
		return resolve(phaseRunning, "", abortExit(ctx, p))
	}

	var upgrade *UpgradeError

	if errors.As(err, &upgrade) {
		return failed(0, upgrade.Error(), upgrade.Retryable())
	}

	if errors.Is(err, safeclient.ErrBlockedDestination) || errors.Is(err, safeclient.ErrBadScheme) || errors.Is(err, safeclient.ErrBadPort) {
		return failed(0, err.Error(), false)
	}

	if errors.Is(err, ErrNoSecret) {
		return failed(0, err.Error(), false)
	}

	if errors.Is(err, ErrUpgradeHeadersTooLarge) {
		return failed(0, fmt.Sprintf("endpoint upgrade response headers exceeded the %d byte limit", p.MaxUpgradeHeaderBytes), false)
	}

	return failed(0, fmt.Sprintf("could not open websocket: %s", transportMessage(p.TriggerURL, err)), true)
}

// transportMessage is the tenant-facing form of a socket or dial error: the endpoint host
// and the stage that failed, through safeclient.PublicError, so a resolved address or a
// request URL never reaches a task error. Policy errors are public already.
func transportMessage(triggerURL string, err error) string {
	host := ""

	if u, parseErr := url.Parse(triggerURL); parseErr == nil {
		host = u.Hostname()
	}

	return safeclient.PublicError(host, err).Error()
}

// abortExit maps a done delivery context: deadline (the request timeout), engine cancel,
// or operator shutdown.
func abortExit(ctx context.Context, p *Params) exit {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return exit{kind: exitTimeout, closeCode: CloseTimeout, msg: "durable invocation timed out without a done frame"}
	}

	if p.Cancelled != nil && p.Cancelled() {
		return exit{kind: exitCancelled, closeCode: CloseCancelled}
	}

	return exit{kind: exitShutdown, closeCode: CloseShuttingDown}
}

// finish records the first exit. The phase is read at this point: transitions happen on
// the pump goroutine before the frame that announces them is queued, so an endpoint that
// reacts to an eviction_ack or error frame is always seen in the phase it reacted to.
func (r *relay) finish(e exit) {
	r.once.Do(func() {
		var engineErr string

		if msg := r.engineErr.Load(); msg != nil {
			engineErr = *msg
		}

		r.result = resolve(phase(r.phase.Load()), engineErr, e)
		close(r.stop)
	})
}

// enterPhase moves the relay out of phaseRunning; the first transition wins.
func (r *relay) enterPhase(ph phase) {
	r.phase.CompareAndSwap(int32(phaseRunning), int32(ph))
}

func (r *relay) stopped() bool {
	select {
	case <-r.stop:
		return true
	default:
		return false
	}
}

// teardown sends the close frame (unless the endpoint closed first), closes the socket and
// the channel, and waits for the loops. The writer is awaited first, bounded, so a queued
// server_evict frame goes out before the close frame.
func (r *relay) teardown() {
	select {
	case <-r.writerDone:
	case <-time.After(closeWriteTimeout):
	}

	if !r.peerClosed.Load() && r.result.CloseCode > 0 {
		reason := r.result.Error

		if len(reason) > 120 {
			reason = reason[:120]
		}

		msg := websocket.FormatCloseMessage(r.result.CloseCode, reason)
		_ = r.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(closeWriteTimeout))
	}

	_ = r.conn.Close()
	_ = r.p.Channel.Close()
	r.cancel()

	r.wg.Wait()
}

// write sends one text frame from the caller's goroutine. Only the first frame is written
// this way; everything else goes through writeLoop.
func (r *relay) write(frame []byte) error {
	if err := r.conn.SetWriteDeadline(time.Now().Add(frameWriteTimeout)); err != nil {
		return err
	}

	return r.conn.WriteMessage(websocket.TextMessage, frame)
}

// readLoop is the socket's only reader. It ends on the first read error or once a terminal
// frame was handled.
func (r *relay) readLoop() {
	defer r.wg.Done()

	for {
		_, data, err := r.conn.ReadMessage()

		if err != nil {
			r.onReadError(err)
			return
		}

		if !r.handleFrame(data) {
			return
		}
	}
}

func (r *relay) onReadError(err error) {
	if r.stopped() {
		return
	}

	var closeErr *websocket.CloseError

	if errors.As(err, &closeErr) {
		r.peerClosed.Store(true)
		r.closedWithoutDone(fmt.Sprintf("endpoint closed the websocket (code %d) without a done frame", closeErr.Code))

		return
	}

	if errors.Is(err, websocket.ErrReadLimit) {
		// The library already sent close 1009.
		r.peerClosed.Store(true)
		r.finish(exit{kind: exitProtocolViolation, msg: fmt.Sprintf("endpoint frame exceeded the %d byte limit", r.p.MaxFrameBytes)})

		return
	}

	var netErr net.Error

	if errors.As(err, &netErr) && netErr.Timeout() {
		r.closedWithoutDone("websocket read timed out")
		return
	}

	r.peerClosed.Store(true)
	r.closedWithoutDone(fmt.Sprintf("websocket read failed: %s", transportMessage(r.p.TriggerURL, err)))
}

// closedWithoutDone is the crash rule: a socket that ends without done is a retryable
// failure, unless the phase says otherwise (see resolve).
func (r *relay) closedWithoutDone(msg string) {
	r.finish(exit{kind: exitClosedWithoutDone, msg: msg})
}

// handleFrame processes one endpoint frame and reports whether reading should continue. An
// endpoint may only send request and done frames; anything else is a protocol violation.
func (r *relay) handleFrame(data []byte) bool {
	if r.done.Load() {
		r.l.Debug().Str("task_id", r.p.TaskId).Msg("dropping durable frame received after done")
		return true
	}

	frame, err := contract.UnmarshalFrame(data)

	if err != nil {
		r.violation(CloseForbiddenMessage, "endpoint sent a malformed frame")
		return false
	}

	switch f := frame.Frame.(type) {
	case *v1.ServerlessDurableFrame_Done:
		r.handleDone(f.Done)
		return false
	case *v1.ServerlessDurableFrame_Request:
		return r.handleRequest(f.Request)
	default:
		r.violation(CloseForbiddenMessage, "endpoint sent a frame with neither request nor done")
		return false
	}
}

func (r *relay) violation(closeCode int, msg string) {
	r.finish(exit{kind: exitProtocolViolation, closeCode: closeCode, msg: msg})
}

// handleDone maps the terminal frame: status evicted first, then error, then output. What
// the frame means depends on the phase: done evicted is only valid after an eviction_ack
// (see resolve).
func (r *relay) handleDone(done *v1.ServerlessDoneFrame) {
	r.done.Store(true)

	switch {
	case done.GetStatus() == contract.DoneStatusEvicted:
		r.finish(exit{kind: exitDoneEvicted, closeCode: CloseNormal})
	case done.Error != nil:
		r.finish(exit{kind: exitDoneError, closeCode: CloseNormal, msg: done.GetError(), retry: done.GetRetry()})
	default:
		output := []byte(done.GetOutput())

		if len(output) > 0 && !json.Valid(output) {
			r.violation(CloseForbiddenMessage, "endpoint sent a done frame whose output is not JSON")
			return
		}

		r.finish(exit{kind: exitDoneOutput, closeCode: CloseNormal, output: output})
	}
}

// handleRequest stamps, validates, confines and forwards one DurableTaskRequest.
func (r *relay) handleRequest(req *v1.DurableTaskRequest) bool {
	if err := r.stamp(req); err != nil {
		var mismatch *mismatchError

		if errors.As(err, &mismatch) {
			r.violation(CloseInvocationMismatch, err.Error())
		} else {
			r.violation(CloseForbiddenMessage, err.Error())
		}

		return false
	}

	if err := r.p.Channel.Send(r.ctx, req); err != nil {
		if errors.Is(err, operator.ErrRequestInFlight) {
			r.violation(CloseRequestInFlight, "endpoint sent a durable request while another was awaiting its ack")
		} else {
			r.linkFailed(err)
		}

		return false
	}

	return true
}

type mismatchError struct {
	msg string
}

func (e *mismatchError) Error() string {
	return e.msg
}

// stamp overwrites the task id and invocation count on the request. A non-empty id or a
// non-zero count that differs from the invocation's is a mismatch; the request kinds the host
// owns (register_worker, worker_status) are forbidden.
func (r *relay) stamp(req *v1.DurableTaskRequest) error {
	var (
		id  *string
		inv *int32
	)

	switch m := req.Message.(type) {
	case *v1.DurableTaskRequest_Memo:
		id, inv = &m.Memo.DurableTaskExternalId, &m.Memo.InvocationCount
	case *v1.DurableTaskRequest_TriggerRuns:
		id, inv = &m.TriggerRuns.DurableTaskExternalId, &m.TriggerRuns.InvocationCount
	case *v1.DurableTaskRequest_WaitFor:
		id, inv = &m.WaitFor.DurableTaskExternalId, &m.WaitFor.InvocationCount
	case *v1.DurableTaskRequest_EvictInvocation:
		id, inv = &m.EvictInvocation.DurableTaskExternalId, &m.EvictInvocation.InvocationCount
	case *v1.DurableTaskRequest_CompleteMemo:
		if m.CompleteMemo.Ref == nil {
			return errors.New("endpoint sent complete_memo without a ref")
		}

		id, inv = &m.CompleteMemo.Ref.DurableTaskExternalId, &m.CompleteMemo.Ref.InvocationCount
	case *v1.DurableTaskRequest_RegisterWorker, *v1.DurableTaskRequest_WorkerStatus:
		return errors.New("endpoint sent a host-internal durable request")
	default:
		return errors.New("endpoint sent an unknown durable request")
	}

	if *id != "" && *id != r.p.TaskId {
		return &mismatchError{msg: "endpoint request carried another task id"}
	}

	if *inv != 0 && *inv != r.p.Invocation {
		return &mismatchError{msg: fmt.Sprintf("endpoint request carried invocation %d, expected %d", *inv, r.p.Invocation)}
	}

	*id = r.p.TaskId
	*inv = r.p.Invocation

	r.confine(req)

	return nil
}

// confine is the namespace boundary of the relay, the same in either host: the resources an
// endpoint names in a nested request are prefixed with its namespace the way the operator
// prefixed what it registered (contract.ApplyNamespace), so a durable task can only
// trigger workflows and wait for user events of its own namespace. Names that already carry
// the prefix are left alone. Everything the engine generates is untouched: log entry refs,
// readable data keys and or-group ids are labels of this task's own log, sleep conditions
// name no resource, event scopes are matched within the namespaced key, and memo keys are
// private to the task.
func (r *relay) confine(req *v1.DurableTaskRequest) {
	ns := r.p.Namespace

	switch m := req.Message.(type) {
	case *v1.DurableTaskRequest_TriggerRuns:
		for _, opt := range m.TriggerRuns.GetTriggerOpts() {
			if opt != nil {
				opt.Name = contract.ApplyNamespace(ns, opt.Name)
			}
		}
	case *v1.DurableTaskRequest_WaitFor:
		for _, cond := range m.WaitFor.GetWaitForConditions().GetUserEventConditions() {
			if cond != nil {
				cond.UserEventKey = contract.ApplyNamespace(ns, cond.UserEventKey)
			}
		}
	}
}

// pumpLoop forwards engine responses to the send queue. A Recv that fails while the relay is
// live ends it as an engine-side failure, which includes the engine ending the invocation on
// its own (operator.ErrSessionEnded); a closed channel is the relay's own teardown.
func (r *relay) pumpLoop() {
	defer r.wg.Done()

	for {
		resp, err := r.p.Channel.Recv(r.ctx)

		if err != nil {
			if !r.stopped() && !errors.Is(err, operator.ErrChannelClosed) {
				r.linkFailed(err)
			}

			return
		}

		if !r.forward(resp) {
			return
		}
	}
}

// forward encodes one response as a frame and queues it. It reports whether the pump
// should keep going: a server eviction and a full queue both end the relay.
func (r *relay) forward(resp *v1.DurableTaskResponse) bool {
	var (
		frame []byte
		err   error
	)

	switch m := resp.Message.(type) {
	case *v1.DurableTaskResponse_Error:
		// The phase moves before the frame is queued so the endpoint's reaction to it is
		// resolved in the errored phase.
		msg := m.Error.GetErrorMessage()
		r.engineErr.Store(&msg)
		r.enterPhase(phaseEngineErrored)
		frame, err = contract.MarshalFrame(&v1.ServerlessDurableFrame{
			Frame: &v1.ServerlessDurableFrame_Error{Error: &v1.ServerlessErrorFrame{
				Code:    errorCode(m.Error.GetErrorType()),
				Message: msg,
			}},
		})
	case *v1.DurableTaskResponse_EvictionAck:
		r.enterPhase(phaseEvictionAcked)
		frame, err = responseFrame(resp)
	default:
		frame, err = responseFrame(resp)
	}

	if err != nil {
		r.linkFailed(err)
		return false
	}

	// The byte budget is charged before the frame is queued and released by the writer once
	// the frame left the queue, so it bounds what the relay retains for a slow endpoint.
	if r.queued.Add(int64(len(frame))) > r.p.MaxQueuedBytes {
		r.finish(exit{kind: exitBackpressure, closeCode: CloseBackpressure, msg: fmt.Sprintf("endpoint fell more than %d bytes behind", r.p.MaxQueuedBytes)})
		return false
	}

	select {
	case r.sendQ <- frame:
	case <-r.stop:
		return false
	default:
		r.finish(exit{kind: exitBackpressure, closeCode: CloseBackpressure, msg: fmt.Sprintf("endpoint fell more than %d frames behind", sendQueueSize)})
		return false
	}

	if _, isEvict := resp.Message.(*v1.DurableTaskResponse_ServerEvict); isEvict {
		// The eviction frame is queued ahead of the close; writeLoop drains the queue
		// before the close frame goes out.
		r.finish(exit{kind: exitServerEvict, closeCode: CloseEvicted})
		return false
	}

	return true
}

// linkFailed ends the relay on an engine-side failure. The detail stays in the operator
// log: the endpoint sees the close reason and the tenant sees the task error, and neither
// should carry engine addresses or transport internals.
func (r *relay) linkFailed(err error) {
	r.l.Warn().Err(err).Str("task_id", r.p.TaskId).Int32("invocation", r.p.Invocation).Msg("durable engine link failed")
	r.finish(exit{kind: exitLinkFailure, closeCode: CloseInternalError, msg: "engine link failed"})
}

func responseFrame(resp *v1.DurableTaskResponse) ([]byte, error) {
	return contract.MarshalFrame(&v1.ServerlessDurableFrame{
		Frame: &v1.ServerlessDurableFrame_Response{Response: resp},
	})
}

func errorCode(t v1.DurableTaskErrorType) string {
	switch t {
	case v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM:
		return contract.ErrorCodeNonDeterminism
	default:
		return contract.ErrorCodeUnspecified
	}
}

// writeLoop is the socket's only frame writer: queued frames, pings on PingInterval, and
// after stop, whatever is still queued so a server_evict frame precedes the close.
func (r *relay) writeLoop() {
	defer r.wg.Done()
	defer close(r.writerDone)

	ticker := time.NewTicker(r.p.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stop:
			r.drainQueue()
			return
		case frame := <-r.sendQ:
			ok := r.writeQueued(frame)
			r.queued.Add(-int64(len(frame)))

			if !ok {
				return
			}
		case <-ticker.C:
			if r.missed.Add(1) > missedPongLimit {
				r.finish(exit{kind: exitUnresponsive, closeCode: CloseUnresponsive, msg: fmt.Sprintf("endpoint missed %d pings", missedPongLimit)})
				return
			}

			if err := r.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(closeWriteTimeout)); err != nil {
				if !r.stopped() {
					r.closedWithoutDone(fmt.Sprintf("websocket ping failed: %s", err.Error()))
				}

				return
			}
		}
	}
}

func (r *relay) writeQueued(frame []byte) bool {
	if r.p.writeGate != nil {
		select {
		case <-r.p.writeGate:
		case <-r.stop:
			r.drainQueue()
			return false
		}
	}

	if err := r.write(frame); err != nil {
		if !r.stopped() {
			r.closedWithoutDone(fmt.Sprintf("websocket write failed: %s", transportMessage(r.p.TriggerURL, err)))
		}

		return false
	}

	return true
}

// drainQueue writes what is queued at stop time when the relay is closing on its own
// terms (an eviction to forward); on a failure the queue is dropped.
func (r *relay) drainQueue() {
	if r.result.Kind != KindEvicted || r.peerClosed.Load() {
		return
	}

	for {
		select {
		case frame := <-r.sendQ:
			err := r.write(frame)
			r.queued.Add(-int64(len(frame)))

			if err != nil {
				return
			}
		default:
			return
		}
	}
}
