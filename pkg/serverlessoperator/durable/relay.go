package durable

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

const (
	// sendQueueSize is how many frames may wait for the endpoint before the relay gives
	// up on it with CloseBackpressure.
	sendQueueSize = 256

	// missedPongLimit is how many pings may go unanswered before the socket is closed.
	missedPongLimit = 2

	defaultHandshakeTimeout = 30 * time.Second
	defaultPingInterval     = 15 * time.Second
	defaultMaxFrameBytes    = 4 * 1024 * 1024

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
	Channel link.DurableChannel
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

	MaxFrameBytes    int64
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

// relay is the per-socket state. finish records the first terminal outcome and closes stop,
// which every goroutine watches; later outcomes are dropped.
type relay struct {
	p          *Params
	conn       *websocket.Conn
	l          *zerolog.Logger
	sendQ      chan []byte
	stop       chan struct{}
	writerDone chan struct{}
	lastError  atomic.Pointer[string]
	result     Outcome
	once       sync.Once
	wg         sync.WaitGroup
	missed     atomic.Int32
	done       atomic.Bool
	evicted    atomic.Bool
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

	r := &relay{
		p:          &p,
		conn:       conn,
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
		r.result = failed(CloseInternalError, fmt.Sprintf("could not send action frame: %s", err.Error()), true)
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
		r.finish(r.abortOutcome(ctx))
	}

	r.teardown()

	return r.result
}

func buildFirstFrame(p *Params) ([]byte, error) {
	raw, err := MarshalProto(p.Action)

	if err != nil {
		return nil, err
	}

	frame, err := json.Marshal(FirstFrame{
		Action:             raw,
		Namespace:          p.Namespace,
		InvocationCount:    p.Invocation,
		InlineWaitBudgetMs: p.InlineWaitBudgetMs,
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
		return abortOutcome(ctx, p)
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

	return failed(0, fmt.Sprintf("could not open websocket: %s", err.Error()), true)
}

// abortOutcome maps a done delivery context: deadline (the request timeout), engine cancel,
// or operator shutdown.
func abortOutcome(ctx context.Context, p *Params) Outcome {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return failed(CloseTimeout, "durable invocation timed out without a done frame", true)
	}

	if p.Cancelled != nil && p.Cancelled() {
		return Outcome{Kind: KindCancelled, CloseCode: CloseCancelled}
	}

	return Outcome{Kind: KindShutdown, CloseCode: CloseShuttingDown}
}

func (r *relay) abortOutcome(ctx context.Context) Outcome {
	return abortOutcome(ctx, r.p)
}

func (r *relay) finish(out Outcome) {
	r.once.Do(func() {
		r.result = out
		close(r.stop)
	})
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
		r.finish(failed(0, fmt.Sprintf("endpoint frame exceeded the %d byte limit", r.p.MaxFrameBytes), false))

		return
	}

	var netErr net.Error

	if errors.As(err, &netErr) && netErr.Timeout() {
		r.closedWithoutDone("websocket read timed out")
		return
	}

	r.peerClosed.Store(true)
	r.closedWithoutDone(fmt.Sprintf("websocket read failed: %s", err.Error()))
}

// closedWithoutDone is the crash rule: a socket that ends without done is a retryable
// failure, unless the invocation was already evicted (the engine re-invokes it) or the
// engine reported an error for it (non-determinism is permanent).
func (r *relay) closedWithoutDone(msg string) {
	if r.evicted.Load() {
		r.finish(evicted(0, EvictionSourceEndpoint))
		return
	}

	if engineErr := r.lastError.Load(); engineErr != nil {
		r.finish(failed(0, *engineErr, false))
		return
	}

	r.finish(failed(0, msg, true))
}

// handleFrame processes one endpoint frame and reports whether reading should continue.
func (r *relay) handleFrame(data []byte) bool {
	if r.done.Load() {
		r.l.Debug().Str("task_id", r.p.TaskId).Msg("dropping durable frame received after done")
		return true
	}

	var frame InboundFrame

	if err := json.Unmarshal(data, &frame); err != nil {
		r.finish(failed(CloseForbiddenMessage, "endpoint sent a malformed frame", false))
		return false
	}

	switch {
	case len(frame.Done) > 0:
		r.handleDone(frame.Done)
		return false
	case len(frame.Request) > 0:
		return r.handleRequest(frame.Request)
	default:
		r.finish(failed(CloseForbiddenMessage, "endpoint sent a frame with neither request nor done", false))
		return false
	}
}

func (r *relay) handleDone(raw json.RawMessage) {
	r.done.Store(true)

	var done DoneFrame

	if err := json.Unmarshal(raw, &done); err != nil {
		r.finish(failed(CloseForbiddenMessage, "endpoint sent a malformed done frame", false))
		return
	}

	switch {
	case done.Status == StatusEvicted:
		r.evicted.Store(true)
		r.finish(evicted(CloseNormal, EvictionSourceEndpoint))
	case done.Error != nil:
		r.finish(failed(CloseNormal, *done.Error, done.Retry))
	default:
		r.finish(completed(done.Output))
	}
}

// handleRequest stamps, validates and forwards one DurableTaskRequest.
func (r *relay) handleRequest(raw json.RawMessage) bool {
	req := &v1.DurableTaskRequest{}

	if err := UnmarshalProto(raw, req); err != nil {
		r.finish(failed(CloseForbiddenMessage, "endpoint sent an undecodable durable request", false))
		return false
	}

	if err := r.stamp(req); err != nil {
		var mismatch *mismatchError

		if errors.As(err, &mismatch) {
			r.finish(failed(CloseInvocationMismatch, err.Error(), false))
		} else {
			r.finish(failed(CloseForbiddenMessage, err.Error(), false))
		}

		return false
	}

	if err := r.p.Channel.Send(req); err != nil {
		if errors.Is(err, link.ErrRequestInFlight) {
			r.finish(failed(CloseRequestInFlight, "endpoint sent a durable request while another was awaiting its ack", false))
		} else {
			r.finish(failed(CloseInternalError, fmt.Sprintf("engine link failed: %s", err.Error()), true))
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
// non-zero count that differs from the invocation's is a mismatch; link-internal request
// kinds are forbidden.
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
		return errors.New("endpoint sent a link-internal durable request")
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

	return nil
}

// pumpLoop forwards engine responses to the send queue.
func (r *relay) pumpLoop() {
	defer r.wg.Done()

	for {
		resp, err := r.p.Channel.Recv()

		if err != nil {
			if !r.stopped() && !errors.Is(err, link.ErrChannelClosed) {
				r.finish(failed(CloseInternalError, fmt.Sprintf("engine link failed: %s", err.Error()), true))
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
		msg := m.Error.GetErrorMessage()
		r.lastError.Store(&msg)
		frame, err = json.Marshal(ErrorFrame{Error: ErrorBody{Code: errorCode(m.Error.GetErrorType()), Message: msg}})
	case *v1.DurableTaskResponse_EvictionAck:
		r.evicted.Store(true)
		frame, err = responseFrame(resp)
	case *v1.DurableTaskResponse_ServerEvict:
		r.evicted.Store(true)
		frame, err = responseFrame(resp)
	default:
		frame, err = responseFrame(resp)
	}

	if err != nil {
		r.finish(failed(CloseInternalError, err.Error(), true))
		return false
	}

	select {
	case r.sendQ <- frame:
	case <-r.stop:
		return false
	default:
		r.finish(failed(CloseBackpressure, fmt.Sprintf("endpoint fell more than %d frames behind", sendQueueSize), true))
		return false
	}

	if _, isEvict := resp.Message.(*v1.DurableTaskResponse_ServerEvict); isEvict {
		// The eviction frame is queued ahead of the close; writeLoop drains the queue
		// before the close frame goes out.
		r.finish(evicted(CloseEvicted, EvictionSourceServer))
		return false
	}

	return true
}

func responseFrame(resp *v1.DurableTaskResponse) ([]byte, error) {
	raw, err := MarshalProto(resp)

	if err != nil {
		return nil, err
	}

	return json.Marshal(ResponseFrame{Response: raw})
}

func errorCode(t v1.DurableTaskErrorType) string {
	switch t {
	case v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM:
		return ErrorCodeNonDeterminism
	default:
		return "unspecified"
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
			if !r.writeQueued(frame) {
				return
			}
		case <-ticker.C:
			if r.missed.Add(1) > missedPongLimit {
				r.finish(failed(CloseUnresponsive, fmt.Sprintf("endpoint missed %d pings", missedPongLimit), true))
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
			r.closedWithoutDone(fmt.Sprintf("websocket write failed: %s", err.Error()))
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
			if err := r.write(frame); err != nil {
				return
			}
		default:
			return
		}
	}
}
