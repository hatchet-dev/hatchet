//go:build e2e

package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
)

const (
	healthcheckPath = "/healthcheck"
	triggerPath     = "/trigger"

	// ackWait bounds how long the durable script waits for an engine ack.
	ackWait = 20 * time.Second
)

// recordedRequest is one request the fake endpoint accepted, for assertions on what the
// operator sent: the namespace, the prefixed action id and workflow name, the invocation.
type recordedRequest struct {
	kind         string
	namespace    string
	endpointId   string
	actionId     string
	workflowName string
	invocation   int32
	retryCount   int32
}

// durableRun is what the durable script observed for one websocket invocation.
type durableRun struct {
	invocation  int32
	retryCount  int32
	memoExisted bool
	evicted     bool
	crashed     bool
}

// durableScript is what the fake does for a durable invocation after the first frame:
// memoize (send memo, use the cached value when memo_already_existed, else compute and
// complete_memo), sleep (send wait_for with a sleep condition; evict when entry_completed
// does not arrive within the inline budget), then done with the output. crashFirstAttempt
// closes the socket without done on the task's first attempt.
type durableScript struct {
	memoKey           string
	memoValue         json.RawMessage
	sleep             time.Duration
	crashFirstAttempt bool
}

// fakeEndpoint is an httptest server standing in for a serverless endpoint: it verifies the
// signature of every request with its secret, answers the healthcheck with its configured
// workflows, echoes non-durable triggers and runs the durable script on websocket upgrades.
type fakeEndpoint struct {
	t          *testing.T
	name       string
	secret     string
	srv        *httptest.Server
	upgrader   websocket.Upgrader
	workflows  []*v1.CreateWorkflowVersionRequest
	script     durableScript
	failStatus int
	failBody   string
	hold       chan struct{}
	endpointId string
	requests   []recordedRequest
	runs       []durableRun
	conns      map[*websocket.Conn]struct{}
	closed     bool
	wg         sync.WaitGroup
	mu         sync.Mutex
}

func newFakeEndpoint(t *testing.T, name string, workflows ...*v1.CreateWorkflowVersionRequest) *fakeEndpoint {
	t.Helper()

	f := &fakeEndpoint{
		t:         t,
		name:      name,
		secret:    uniqueName("secret"),
		workflows: workflows,
		conns:     map[*websocket.Conn]struct{}{},
	}

	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.close)

	return f
}

// close stops the server and every websocket handler still running. Hijacked connections are
// not awaited by httptest, so they are closed here and their handlers awaited explicitly.
func (f *fakeEndpoint) close() {
	f.mu.Lock()
	f.closed = true
	conns := make([]*websocket.Conn, 0, len(f.conns))

	for c := range f.conns {
		conns = append(conns, c)
	}

	f.mu.Unlock()

	for _, c := range conns {
		_ = c.Close()
	}

	f.srv.Close()
	f.wg.Wait()
}

func (f *fakeEndpoint) healthcheckURL() string { return f.srv.URL + healthcheckPath }
func (f *fakeEndpoint) triggerURL() string     { return f.srv.URL + triggerPath }

// attach records the endpoint id the operator will present, so upgrades can verify it.
func (f *fakeEndpoint) attach(id uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.endpointId = id.String()
}

func (f *fakeEndpoint) setWorkflows(workflows ...*v1.CreateWorkflowVersionRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.workflows = workflows
}

func (f *fakeEndpoint) setScript(s durableScript) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.script = s
}

// holdTriggers makes every trigger request from now on wait, after it is recorded, until the
// returned release is called (or the request ends), so a test can observe the operator with
// a delivery in flight.
func (f *fakeEndpoint) holdTriggers() (release func()) {
	hold := make(chan struct{})

	f.mu.Lock()
	f.hold = hold
	f.mu.Unlock()

	var once sync.Once

	return func() {
		once.Do(func() {
			f.mu.Lock()

			if f.hold == hold {
				f.hold = nil
			}

			f.mu.Unlock()
			close(hold)
		})
	}
}

func (f *fakeEndpoint) setFailure(status int, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failStatus = status
	f.failBody = body
}

func (f *fakeEndpoint) record(r recordedRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests = append(f.requests, r)
}

func (f *fakeEndpoint) recordRun(r durableRun) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.runs = append(f.runs, r)
}

func (f *fakeEndpoint) requestsOfKind(kind string) []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := []recordedRequest{}

	for _, r := range f.requests {
		if r.kind == kind {
			out = append(out, r)
		}
	}

	return out
}

func (f *fakeEndpoint) durableRuns() []durableRun {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]durableRun{}, f.runs...)
}

func (f *fakeEndpoint) serve(w http.ResponseWriter, r *http.Request) {
	if websocket.IsWebSocketUpgrade(r) {
		f.serveUpgrade(w, r)
		return
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !signature.Verify(string(body), f.secret, r.Header.Get(contract.SignatureHeader)) {
		f.t.Errorf("fake %s: bad signature on %s", f.name, r.URL.Path)
		http.Error(w, "bad signature", http.StatusUnauthorized)

		return
	}

	switch r.URL.Path {
	case healthcheckPath:
		f.serveHealthcheck(w, body)
	case triggerPath:
		f.serveTrigger(w, r, body)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeEndpoint) serveHealthcheck(w http.ResponseWriter, body []byte) {
	req := &v1.ServerlessHealthcheckRequest{}

	if err := contract.Unmarshal(body, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.record(recordedRequest{kind: "healthcheck", namespace: req.Namespace, endpointId: req.EndpointId})

	f.mu.Lock()
	workflows := f.workflows
	f.mu.Unlock()

	out, err := contract.Marshal(&v1.ServerlessHealthcheckResponse{
		Workflows: workflows,
		Durable:   &v1.ServerlessDurableSupport{Supported: true},
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func (f *fakeEndpoint) serveTrigger(w http.ResponseWriter, r *http.Request, body []byte) {
	env := &v1.ServerlessTriggerRequest{}

	if err := contract.Unmarshal(body, env); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	action := env.GetAction()

	if action == nil {
		http.Error(w, "trigger request without an action", http.StatusBadRequest)
		return
	}

	f.record(recordedRequest{
		kind:         "trigger",
		namespace:    env.Namespace,
		endpointId:   env.EndpointId,
		actionId:     action.ActionId,
		workflowName: action.JobName,
		retryCount:   action.RetryCount,
	})

	f.mu.Lock()
	failStatus, failBody, hold := f.failStatus, f.failBody, f.hold
	f.mu.Unlock()

	if hold != nil {
		select {
		case <-hold:
		case <-r.Context().Done():
			return
		}
	}

	if failStatus != 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(failStatus)
		_, _ = w.Write([]byte(failBody))

		return
	}

	var payload struct {
		Input json.RawMessage `json:"input"`
	}

	if err := json.Unmarshal([]byte(action.ActionPayload), &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"input": payload.Input, "endpoint": f.name})
}

// serveUpgrade verifies the signed upgrade headers, accepts the socket and runs the durable
// script.
func (f *fakeEndpoint) serveUpgrade(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	endpointId := f.endpointId
	closed := f.closed
	f.mu.Unlock()

	if closed {
		http.Error(w, "closed", http.StatusServiceUnavailable)
		return
	}

	if endpointId != "" && r.Header.Get(contract.EndpointIdHeader) != endpointId {
		f.t.Errorf("fake %s: upgrade carried endpoint id %q, want %q", f.name, r.Header.Get(contract.EndpointIdHeader), endpointId)
		http.Error(w, "wrong endpoint", http.StatusForbidden)

		return
	}

	ts, err := strconv.ParseInt(r.Header.Get(contract.TimestampHeader), 10, 64)

	if err != nil || time.Since(time.Unix(ts, 0)) > contract.UpgradeMaxAge {
		f.t.Errorf("fake %s: upgrade timestamp %q rejected", f.name, r.Header.Get(contract.TimestampHeader))
		http.Error(w, "stale timestamp", http.StatusUnauthorized)

		return
	}

	payload := contract.UpgradeSigningPayload(
		r.Header.Get(contract.TimestampHeader),
		r.Header.Get(contract.NonceHeader),
		r.Header.Get(contract.TaskIdHeader),
		r.Header.Get(contract.InvocationHeader),
	)

	if r.Header.Get(contract.NonceHeader) == "" || !signature.Verify(payload, f.secret, r.Header.Get(contract.SignatureHeader)) {
		f.t.Errorf("fake %s: bad upgrade signature", f.name)
		http.Error(w, "bad signature", http.StatusUnauthorized)

		return
	}

	conn, err := f.upgrader.Upgrade(w, r, nil)

	if err != nil {
		return
	}

	f.mu.Lock()
	f.conns[conn] = struct{}{}
	f.wg.Add(1)
	f.mu.Unlock()

	defer func() {
		_ = conn.Close()

		f.mu.Lock()
		delete(f.conns, conn)
		f.mu.Unlock()

		f.wg.Done()
	}()

	f.runDurable(conn)
}

// runDurable reads the first frame and executes the script on the socket.
func (f *fakeEndpoint) runDurable(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(ackWait))

	_, data, err := conn.ReadMessage()

	if err != nil {
		f.t.Errorf("fake %s: could not read first frame: %v", f.name, err)
		return
	}

	frame, err := contract.UnmarshalFrame(data)

	if err != nil {
		f.t.Errorf("fake %s: malformed first frame: %v", f.name, err)
		return
	}

	first := frame.GetFirst()

	if first == nil || first.GetAction() == nil {
		f.t.Errorf("fake %s: first frame carried no action: %s", f.name, string(data))
		return
	}

	action := first.GetAction()

	f.record(recordedRequest{
		kind:         "upgrade",
		namespace:    first.Namespace,
		actionId:     action.ActionId,
		workflowName: action.JobName,
		invocation:   first.InvocationCount,
		retryCount:   action.RetryCount,
	})

	f.mu.Lock()
	script := f.script
	f.mu.Unlock()

	run := durableRun{invocation: first.InvocationCount, retryCount: action.RetryCount}
	defer func() { f.recordRun(run) }()

	if script.crashFirstAttempt && action.RetryCount == 0 {
		// A crash: the socket ends without a done frame.
		run.crashed = true
		return
	}

	s := newDurableSession(f, conn)
	output := map[string]any{"invocation": first.InvocationCount}

	if script.memoKey != "" {
		value, existed, err := s.memo(script.memoKey, script.memoValue)

		if err != nil {
			f.t.Errorf("fake %s: memo failed: %v", f.name, err)
			return
		}

		run.memoExisted = existed
		output["memo"] = value
	}

	if script.sleep > 0 {
		satisfied, err := s.sleep(script.sleep, time.Duration(first.InlineWaitBudgetMs)*time.Millisecond)

		if err != nil {
			f.t.Errorf("fake %s: sleep failed: %v", f.name, err)
			return
		}

		if !satisfied {
			if err := s.evict(); err != nil {
				f.t.Errorf("fake %s: eviction failed: %v", f.name, err)
				return
			}

			run.evicted = true

			s.done(&v1.ServerlessDoneFrame{Status: contract.DoneStatusEvicted})

			return
		}
	}

	raw, err := json.Marshal(output)

	if err != nil {
		f.t.Errorf("fake %s: could not encode output: %v", f.name, err)
		return
	}

	encoded := string(raw)
	s.done(&v1.ServerlessDoneFrame{Output: &encoded})
}

// durableSession is the endpoint side of one durable websocket: it sends requests as the
// endpoint SDK would and reads the engine's responses back. Frames are read by one goroutine
// into a channel so waiting with a timeout (the inline budget) never sets a read deadline,
// which gorilla treats as a permanent read error.
type durableSession struct {
	f      *fakeEndpoint
	conn   *websocket.Conn
	frames chan inboundFrame
	seq    int64
}

type inboundFrame struct {
	data []byte
	err  error
}

func newDurableSession(f *fakeEndpoint, conn *websocket.Conn) *durableSession {
	s := &durableSession{f: f, conn: conn, frames: make(chan inboundFrame, 64)}

	f.wg.Add(1)

	go func() {
		defer f.wg.Done()
		defer close(s.frames)

		_ = conn.SetReadDeadline(time.Time{})

		for {
			_, data, err := conn.ReadMessage()

			if err != nil {
				s.frames <- inboundFrame{err: err}
				return
			}

			s.frames <- inboundFrame{data: data}
		}
	}()

	return s
}

func (s *durableSession) send(req *v1.DurableTaskRequest) error {
	s.seq++
	id := s.seq

	frame, err := contract.MarshalFrame(&v1.ServerlessDurableFrame{
		Frame: &v1.ServerlessDurableFrame_Request{Request: req},
		Id:    &id,
	})

	if err != nil {
		return err
	}

	_ = s.conn.SetWriteDeadline(time.Now().Add(ackWait))

	return s.conn.WriteMessage(websocket.TextMessage, frame)
}

// errReadTimeout reports that no frame arrived within the wait.
var errReadTimeout = errors.New("read timed out")

// recv reads the next engine response, or errReadTimeout when none arrives within wait.
func (s *durableSession) recv(wait time.Duration) (*v1.DurableTaskResponse, error) {
	var data []byte

	select {
	case frame, ok := <-s.frames:
		if !ok {
			return nil, errors.New("socket closed")
		}

		if frame.err != nil {
			return nil, frame.err
		}

		data = frame.data
	case <-time.After(wait):
		return nil, errReadTimeout
	}

	frame, err := contract.UnmarshalFrame(data)

	if err != nil {
		return nil, err
	}

	if engineErr := frame.GetError(); engineErr != nil {
		return nil, fmt.Errorf("engine error %s: %s", engineErr.Code, engineErr.Message)
	}

	resp := frame.GetResponse()

	if resp == nil {
		return nil, fmt.Errorf("unexpected frame %s", string(data))
	}

	return resp, nil
}

// memo runs the memo step: value is the cached payload when the memo already existed, the
// computed value (persisted with complete_memo) otherwise.
func (s *durableSession) memo(key string, value json.RawMessage) (json.RawMessage, bool, error) {
	err := s.send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte(key)},
	}})

	if err != nil {
		return nil, false, err
	}

	resp, err := s.recv(ackWait)

	if err != nil {
		return nil, false, err
	}

	ack := resp.GetMemoAck()

	if ack == nil || ack.GetRef() == nil {
		return nil, false, fmt.Errorf("expected memo_ack, got %s", resp.String())
	}

	if ack.GetMemoAlreadyExisted() {
		return json.RawMessage(ack.GetMemoResultPayload()), true, nil
	}

	err = s.send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{
		CompleteMemo: &v1.DurableTaskCompleteMemoRequest{Ref: ack.GetRef(), MemoKey: []byte(key), Payload: value},
	}})

	if err != nil {
		return nil, false, err
	}

	return value, false, nil
}

// sleep runs the wait_for step with a sleep condition, as the Go SDK's durable SleepFor does,
// and reports whether entry_completed arrived within the inline budget.
func (s *durableSession) sleep(d, budget time.Duration) (bool, error) {
	pb := condition.SleepCondition(d).ToPB(v1.Action_CREATE)

	err := s.send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WaitFor{
		WaitFor: &v1.DurableTaskWaitForRequest{
			WaitForConditions: &v1.DurableEventListenerConditions{SleepConditions: pb.SleepConditions},
		},
	}})

	if err != nil {
		return false, err
	}

	resp, err := s.recv(ackWait)

	if err != nil {
		return false, err
	}

	if resp.GetWaitForAck() == nil || resp.GetWaitForAck().GetRef() == nil {
		return false, fmt.Errorf("expected wait_for_ack, got %s", resp.String())
	}

	resp, err = s.recv(budget)

	if errors.Is(err, errReadTimeout) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if resp.GetEntryCompleted() == nil {
		return false, fmt.Errorf("expected entry_completed, got %s", resp.String())
	}

	return true, nil
}

func (s *durableSession) evict() error {
	err := s.send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_EvictInvocation{
		EvictInvocation: &v1.DurableTaskEvictInvocationRequest{},
	}})

	if err != nil {
		return err
	}

	resp, err := s.recv(ackWait)

	if err != nil {
		return err
	}

	if resp.GetEvictionAck() == nil {
		return fmt.Errorf("expected eviction_ack, got %s", resp.String())
	}

	return nil
}

// done sends the terminal frame and waits for the relay's close frame.
func (s *durableSession) done(done *v1.ServerlessDoneFrame) {
	out, err := contract.MarshalFrame(&v1.ServerlessDurableFrame{
		Frame: &v1.ServerlessDurableFrame_Done{Done: done},
	})

	if err != nil {
		s.f.t.Errorf("fake %s: could not encode done frame: %v", s.f.name, err)
		return
	}

	_ = s.conn.SetWriteDeadline(time.Now().Add(ackWait))

	if err := s.conn.WriteMessage(websocket.TextMessage, out); err != nil {
		s.f.t.Errorf("fake %s: could not send done frame: %v", s.f.name, err)
		return
	}

	// The relay answers done with a close frame; wait for the socket to end, bounded.
	deadline := time.After(ackWait)

	for {
		select {
		case frame, ok := <-s.frames:
			if !ok || frame.err != nil {
				return
			}
		case <-deadline:
			return
		}
	}
}
