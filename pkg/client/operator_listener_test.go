package client

import (
	"context"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// fakeOperatorListenStream is one Listen stream as the client sees it. It
// records every request, delivers actions through deliver, acknowledges
// every sequenced delta the way the engine does unless noAck is set, and
// models the ways a stream dies: breakRecv (server hangup, Recv fails with
// Unavailable) and breakSend (transport failure, Send fails too). CloseSend
// ends Recv with EOF the way a server does after the client half-closes.
type fakeOperatorListenStream struct {
	ctx       context.Context
	recvDead  chan struct{}
	responses chan *v1.OperatorListenResponse
	recvErr   error
	requests  []*v1.OperatorListenRequest
	// operatorId is the hatchet-operator-id metadata the stream was opened with
	operatorId string
	id         int
	mu         sync.Mutex
	recvOnce   sync.Once
	sendDead   atomic.Bool
	noAck      atomic.Bool
}

func (s *fakeOperatorListenStream) Send(req *v1.OperatorListenRequest) error {
	if s.sendDead.Load() {
		return status.Error(codes.Unavailable, "send on broken stream")
	}

	s.mu.Lock()
	s.requests = append(s.requests, req)
	s.mu.Unlock()

	if delta := req.GetActions(); delta != nil && delta.Sequence != 0 && !s.noAck.Load() {
		s.responses <- &v1.OperatorListenResponse{
			Message: &v1.OperatorListenResponse_Ack{Ack: &v1.OperatorActionsAck{Sequence: delta.Sequence}},
		}
	}

	return nil
}

func (s *fakeOperatorListenStream) Recv() (*v1.OperatorListenResponse, error) {
	select {
	case resp := <-s.responses:
		return resp, nil
	case <-s.recvDead:
		return nil, s.recvErr
	case <-s.ctx.Done():
		return nil, status.Error(codes.Canceled, s.ctx.Err().Error())
	}
}

func (s *fakeOperatorListenStream) deliver(action *dispatchercontracts.AssignedAction) {
	s.responses <- &v1.OperatorListenResponse{Message: &v1.OperatorListenResponse_Action{Action: action}}
}

// ack acknowledges seq explicitly, for streams created with noAck.
func (s *fakeOperatorListenStream) ack(seq uint64) {
	s.responses <- &v1.OperatorListenResponse{
		Message: &v1.OperatorListenResponse_Ack{Ack: &v1.OperatorActionsAck{Sequence: seq}},
	}
}

func (s *fakeOperatorListenStream) breakRecvWith(err error) {
	s.recvOnce.Do(func() {
		s.recvErr = err
		close(s.recvDead)
	})
}

func (s *fakeOperatorListenStream) breakRecv() {
	s.breakRecvWith(status.Error(codes.Unavailable, "recv on broken stream"))
}

func (s *fakeOperatorListenStream) breakSend() {
	s.sendDead.Store(true)
}

func (s *fakeOperatorListenStream) CloseSend() error {
	s.breakRecvWith(io.EOF)
	return nil
}

func (s *fakeOperatorListenStream) starts() []*v1.OperatorListenStart {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []*v1.OperatorListenStart
	for _, req := range s.requests {
		if r := req.GetStart(); r != nil {
			out = append(out, r)
		}
	}
	return out
}

func (s *fakeOperatorListenStream) heartbeats() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := 0
	for _, req := range s.requests {
		if req.GetHeartbeat() != nil {
			n++
		}
	}
	return n
}

func (s *fakeOperatorListenStream) deltas() []*v1.OperatorActionsDelta {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []*v1.OperatorActionsDelta
	for _, req := range s.requests {
		if d := req.GetActions(); d != nil {
			out = append(out, d)
		}
	}
	return out
}

// firstMessageIsStart reports whether the first message on the stream was start.
func (s *fakeOperatorListenStream) firstMessageIsStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.requests) > 0 && s.requests[0].GetStart() != nil
}

func (s *fakeOperatorListenStream) Header() (metadata.MD, error)  { return nil, nil }
func (s *fakeOperatorListenStream) Trailer() metadata.MD          { return nil }
func (s *fakeOperatorListenStream) Context() context.Context      { return s.ctx }
func (s *fakeOperatorListenStream) SendMsg(msg interface{}) error { return nil }
func (s *fakeOperatorListenStream) RecvMsg(msg interface{}) error { return nil }

// fakeOperatorServiceClient answers Register from a queue of responses (the
// last one repeats), hands out numbered fakeOperatorListenStreams from Listen
// and records the unary calls with the outgoing metadata they carried.
type fakeOperatorServiceClient struct {
	registrations       []*v1.OperatorRegisterResponse
	registers           []*v1.OperatorRegisterRequest
	streams             []*fakeOperatorListenStream
	stepEvents          []*dispatchercontracts.StepActionEvent
	stepEventOperatorId []string
	pauses              []*v1.OperatorPauseWorkerRequest
	registerErr         error
	listenErr           error
	pauseErr            error
	mu                  sync.Mutex
	// noAck makes every new stream withhold delta acks
	noAck bool
}

func (f *fakeOperatorServiceClient) Register(ctx context.Context, in *v1.OperatorRegisterRequest, opts ...grpc.CallOption) (*v1.OperatorRegisterResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.registerErr != nil {
		return nil, f.registerErr
	}

	idx := len(f.registers)
	f.registers = append(f.registers, in)

	resp := f.registrations[len(f.registrations)-1]
	if idx < len(f.registrations) {
		resp = f.registrations[idx]
	}

	return resp, nil
}

func (f *fakeOperatorServiceClient) Listen(ctx context.Context, opts ...grpc.CallOption) (v1.OperatorService_ListenClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.listenErr != nil {
		return nil, f.listenErr
	}

	s := &fakeOperatorListenStream{
		id:         len(f.streams),
		ctx:        ctx,
		operatorId: outgoingOperatorId(ctx),
		recvDead:   make(chan struct{}),
		responses:  make(chan *v1.OperatorListenResponse, 1024),
	}
	if f.noAck {
		s.noAck.Store(true)
	}
	f.streams = append(f.streams, s)
	return s, nil
}

func (f *fakeOperatorServiceClient) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stepEvents = append(f.stepEvents, in)
	f.stepEventOperatorId = append(f.stepEventOperatorId, outgoingOperatorId(ctx))
	return &dispatchercontracts.ActionEventResponse{TenantId: "tenant-1", WorkerId: in.WorkerId}, nil
}

func (f *fakeOperatorServiceClient) PauseWorker(ctx context.Context, in *v1.OperatorPauseWorkerRequest, opts ...grpc.CallOption) (*v1.OperatorPauseWorkerResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.pauseErr != nil {
		return nil, f.pauseErr
	}

	f.pauses = append(f.pauses, in)

	return &v1.OperatorPauseWorkerResponse{WorkerId: in.WorkerId, Paused: in.Paused}, nil
}

func (f *fakeOperatorServiceClient) pauseRequests() []*v1.OperatorPauseWorkerRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*v1.OperatorPauseWorkerRequest(nil), f.pauses...)
}

func (f *fakeOperatorServiceClient) DurableTask(ctx context.Context, opts ...grpc.CallOption) (v1.OperatorService_DurableTaskClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeOperatorServiceClient) stream(i int) *fakeOperatorListenStream {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.streams[i]
}

func (f *fakeOperatorServiceClient) streamCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.streams)
}

func (f *fakeOperatorServiceClient) registerRequests() []*v1.OperatorRegisterRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*v1.OperatorRegisterRequest(nil), f.registers...)
}

// fakeAdminServiceClient records PutWorkflow calls and the metadata they carried.
type fakeAdminServiceClient struct {
	v1.AdminServiceClient

	workflows  []*v1.CreateWorkflowVersionRequest
	operatorId []string
	authorized []bool
	mu         sync.Mutex
}

func (f *fakeAdminServiceClient) PutWorkflow(ctx context.Context, in *v1.CreateWorkflowVersionRequest, opts ...grpc.CallOption) (*v1.CreateWorkflowVersionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.workflows = append(f.workflows, in)
	f.operatorId = append(f.operatorId, outgoingOperatorId(ctx))
	md, _ := metadata.FromOutgoingContext(ctx)
	f.authorized = append(f.authorized, len(md.Get("authorization")) > 0)
	return &v1.CreateWorkflowVersionResponse{Id: "wfv-1", WorkflowId: "wf-1"}, nil
}

func outgoingOperatorId(ctx context.Context) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(operatorIdMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func registeredAs(workerId string, resumed bool) *v1.OperatorRegisterResponse {
	return &v1.OperatorRegisterResponse{
		TenantId:   "tenant-1",
		OperatorId: "operator-1",
		WorkerId:   workerId,
		Resumed:    resumed,
	}
}

func newTestOperatorSession(t *testing.T, client *fakeOperatorServiceClient, resume bool) (*operatorSession, *fakeAdminServiceClient) {
	t.Helper()

	logger := zerolog.Nop()
	admin := &fakeAdminServiceClient{}
	s := newOperatorSession(
		client,
		admin,
		newContextLoader("token", nil),
		&logger,
		&v1.OperatorRegisterRequest{
			Name:       "test-operator",
			SlotConfig: map[string]int32{"default": 10},
		},
		resume,
	)
	s.heartbeatInterval = 5 * time.Millisecond
	s.actions.interval = 5 * time.Millisecond
	disableStreamBackoff(t, s.stream)
	return s, admin
}

func connectTestOperatorSession(t *testing.T, client *fakeOperatorServiceClient, resume bool) (*operatorSession, *fakeAdminServiceClient) {
	t.Helper()

	s, admin := newTestOperatorSession(t, client, resume)
	require.NoError(t, s.connect(context.Background()))
	t.Cleanup(func() { _ = s.Close() })
	return s, admin
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal(msg)
}

func flushed(t *testing.T, s *operatorSession) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, s.Flush(ctx))
}

// deltaIds flattens deltas into sorted add and remove id lists.
func deltaIds(deltas []*v1.OperatorActionsDelta) (adds, removes []string) {
	for _, d := range deltas {
		adds = append(adds, d.Add...)
		removes = append(removes, d.Remove...)
	}
	sort.Strings(adds)
	sort.Strings(removes)
	return adds, removes
}

func TestOperatorSessionRegistersThenStartsBeforeReceivingActions(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := connectTestOperatorSession(t, client, true)

	registers := client.registerRequests()
	require.Len(t, registers, 1)
	assert.Equal(t, "test-operator", registers[0].Name)
	assert.Nil(t, registers[0].WorkerId, "first connect must not resume a worker")
	assert.Equal(t, map[string]int32{"default": 10}, registers[0].SlotConfig)

	stream := client.stream(0)
	assert.Equal(t, "operator-1", stream.operatorId, "Listen carries the operator id from Register")
	assert.True(t, stream.firstMessageIsStart(), "the first message on the stream must be start")
	require.Len(t, stream.starts(), 1)
	assert.Equal(t, "worker-1", stream.starts()[0].WorkerId)
	assert.Empty(t, stream.deltas(), "an empty desired set replays nothing")

	reg := s.Registration()
	assert.Equal(t, "tenant-1", reg.TenantId)
	assert.Equal(t, "operator-1", reg.OperatorId)
	assert.Equal(t, "worker-1", reg.WorkerId)
	assert.False(t, reg.Resumed)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actions, errCh, err := s.Actions(ctx)
	require.NoError(t, err)

	_, _, err = s.Actions(ctx)
	require.ErrorIs(t, err, errOperatorActionsStarted)

	stream.deliver(&dispatchercontracts.AssignedAction{
		ActionType: dispatchercontracts.ActionType_START_STEP_RUN,
		ActionId:   "svc:one",
	})

	select {
	case action := <-actions:
		require.NotNil(t, action)
		assert.Equal(t, "svc:one", action.ActionId)
	case err := <-errCh:
		t.Fatalf("unexpected terminal error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for action")
	}

	waitFor(t, func() bool { return stream.heartbeats() >= 1 }, "no heartbeat was sent")
}

func TestOperatorSessionConnectFailsWhenRegisterFails(t *testing.T) {
	client := &fakeOperatorServiceClient{
		registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)},
		registerErr:   status.Error(codes.InvalidArgument, "bad name"),
	}
	s, _ := newTestOperatorSession(t, client, true)
	t.Cleanup(func() { _ = s.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.stream.connectSync(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not register operator")
	assert.Zero(t, client.streamCount(), "Listen is not opened when Register fails")
	assert.Empty(t, s.Registration().WorkerId)
}

func TestOperatorSessionReconnectResumesPreviousWorkerWithoutReplay(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{
		registeredAs("worker-1", false),
		registeredAs("worker-1", true),
	}}
	s, _ := connectTestOperatorSession(t, client, true)

	s.AddActions("svc:one", "svc:two")
	flushed(t, s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actions, errCh, err := s.Actions(ctx)
	require.NoError(t, err)

	client.stream(0).breakRecv()

	waitFor(t, func() bool { return client.streamCount() >= 2 }, "listener did not reconnect after recv failure")

	registers := client.registerRequests()
	require.Len(t, registers, 2)
	require.NotNil(t, registers[1].WorkerId)
	assert.Equal(t, "worker-1", *registers[1].WorkerId, "reconnect must resume the previous worker id")

	second := client.stream(1)
	waitFor(t, func() bool { return len(second.starts()) == 1 }, "second stream did not receive a start message")
	assert.Equal(t, "worker-1", second.starts()[0].WorkerId)

	waitFor(t, func() bool { return s.Registration().Resumed }, "registration was not updated from the reconnect handshake")

	second.deliver(&dispatchercontracts.AssignedAction{ActionId: "after-reconnect"})

	select {
	case action := <-actions:
		assert.Equal(t, "after-reconnect", action.ActionId)
	case err := <-errCh:
		t.Fatalf("unexpected terminal error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for action on reconnected stream")
	}

	assert.Empty(t, second.deltas(), "a resumed worker keeps its action set, nothing is replayed")
}

func TestOperatorSessionReconnectWithoutResumeReplaysDesiredSet(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{
		registeredAs("worker-1", false),
		registeredAs("worker-2", false),
	}}
	s, _ := connectTestOperatorSession(t, client, true)
	s.actions.maxChunk = 2

	s.AddActions("svc:one", "svc:two", "svc:three", "svc:gone")
	s.RemoveActions("svc:gone")
	flushed(t, s)

	// the previous worker is gone on the engine side: Register answers with a
	// new worker and resumed=false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, err := s.Actions(ctx)
	require.NoError(t, err)

	client.stream(0).breakRecv()

	waitFor(t, func() bool { return client.streamCount() >= 2 && len(client.stream(1).starts()) == 1 }, "listener did not reconnect")

	second := client.stream(1)
	assert.Equal(t, "worker-2", second.starts()[0].WorkerId)
	assert.True(t, second.firstMessageIsStart(), "start precedes the replay")

	waitFor(t, func() bool { return len(second.deltas()) == 2 }, "desired set was not replayed in chunks")
	adds, removes := deltaIds(second.deltas())
	assert.Equal(t, []string{"svc:one", "svc:three", "svc:two"}, adds, "the whole desired set is replayed")
	assert.Empty(t, removes)
	for _, d := range second.deltas() {
		assert.LessOrEqual(t, len(d.Add), 2, "replay respects the chunk size")
	}

	assert.Equal(t, "worker-2", s.Registration().WorkerId)
}

func TestOperatorSessionReconnectWithoutResumeOptionOmitsWorkerId(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{
		registeredAs("worker-1", false),
		registeredAs("worker-2", false),
	}}
	s, _ := connectTestOperatorSession(t, client, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, err := s.Actions(ctx)
	require.NoError(t, err)

	client.stream(0).breakRecv()

	waitFor(t, func() bool { return len(client.registerRequests()) == 2 }, "listener did not re-register")
	assert.Nil(t, client.registerRequests()[1].WorkerId)
}

func TestOperatorSessionHeartbeatFailureTriggersReconnect(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{
		registeredAs("worker-1", false),
		registeredAs("worker-1", true),
	}}
	s, _ := connectTestOperatorSession(t, client, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, errCh, err := s.Actions(ctx)
	require.NoError(t, err)

	// The transport dies for sends only: Recv keeps blocking, so the only way
	// the session notices is the next heartbeat.
	client.stream(0).breakSend()

	waitFor(t, func() bool { return client.streamCount() >= 2 }, "heartbeat failure did not open a new stream")
	second := client.stream(1)
	waitFor(t, func() bool { return second.heartbeats() >= 1 }, "no heartbeat was sent on the reconnected stream")

	registers := client.registerRequests()
	require.Len(t, registers, 2)
	require.NotNil(t, registers[1].WorkerId)
	assert.Equal(t, "worker-1", *registers[1].WorkerId)

	select {
	case err := <-errCh:
		t.Fatalf("unexpected terminal error: %v", err)
	default:
	}
}

func TestOperatorSessionDeltasCoalesceAndChunk(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := connectTestOperatorSession(t, client, true)
	s.actions.interval = 50 * time.Millisecond
	s.actions.maxChunk = 3

	// add then remove inside the window cancels out; remove then add of an
	// existing action cancels out too
	s.AddActions("svc:keep", "svc:cancel", "")
	s.RemoveActions("svc:cancel", "svc:unknown")
	flushed(t, s)

	deltas := client.stream(0).deltas()
	require.Len(t, deltas, 1)
	assert.Equal(t, []string{"svc:keep"}, deltas[0].Add)
	assert.Empty(t, deltas[0].Remove)

	s.RemoveActions("svc:keep")
	s.AddActions("svc:keep")
	flushed(t, s)
	assert.Len(t, client.stream(0).deltas(), 1, "a remove cancelled by a re-add sends nothing")

	// six ids with a chunk size of three arrive as two messages, sent as
	// soon as a full chunk is pending
	s.AddActions("a:1", "a:2", "a:3", "a:4", "a:5")
	s.RemoveActions("svc:keep")
	flushed(t, s)

	deltas = client.stream(0).deltas()[1:]
	require.Len(t, deltas, 2)
	for _, d := range deltas {
		assert.LessOrEqual(t, len(d.Add)+len(d.Remove), 3)
	}
	adds, removes := deltaIds(deltas)
	assert.Equal(t, []string{"a:1", "a:2", "a:3", "a:4", "a:5"}, adds)
	assert.Equal(t, []string{"svc:keep"}, removes)

	assert.ElementsMatch(t, []string{"a:1", "a:2", "a:3", "a:4", "a:5"}, s.actions.desiredSet())
}

func TestOperatorSessionDeltasAreSentThroughRetrySend(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{
		registeredAs("worker-1", false),
		registeredAs("worker-1", true),
	}}
	s, _ := connectTestOperatorSession(t, client, true)

	client.stream(0).breakSend()

	s.AddActions("svc:one")
	flushed(t, s)

	require.GreaterOrEqual(t, client.streamCount(), 2, "a failed delta send reconnects")
	second := client.stream(1)
	require.Len(t, second.deltas(), 1, "the delta is retried on the new stream")
	assert.Equal(t, []string{"svc:one"}, second.deltas()[0].Add)
}

func TestOperatorSessionFlushReportsLastSendError(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := connectTestOperatorSession(t, client, true)

	client.stream(0).breakSend()
	client.mu.Lock()
	client.listenErr = status.Error(codes.Unavailable, "engine down")
	client.mu.Unlock()

	s.AddActions("svc:one")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.Flush(ctx)
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err), err)

	// a later successful send clears the error
	client.mu.Lock()
	client.listenErr = nil
	client.mu.Unlock()

	s.AddActions("svc:two")
	flushed(t, s)
	assert.Contains(t, s.actions.desiredSet(), "svc:one", "a delta that could not be sent stays in the desired set for the next replay")
}

func TestOperatorSessionFlushHonoursContext(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := connectTestOperatorSession(t, client, true)
	s.actions.interval = time.Hour

	s.AddActions("svc:one")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	assert.ErrorIs(t, s.Flush(ctx), context.DeadlineExceeded)
}

func TestOperatorSessionCloseDrainsPendingDeltas(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := connectTestOperatorSession(t, client, true)
	s.actions.interval = 100 * time.Millisecond

	actions, errCh, err := s.Actions(context.Background())
	require.NoError(t, err)

	durable := s.NewDurableTaskListener(WithReconnectInterval(time.Millisecond))
	durable.Start(context.Background())
	waitFor(t, func() bool { return durable.IsRunning() }, "durable listener did not start")

	waitFor(t, func() bool { return client.stream(0).heartbeats() >= 1 }, "no heartbeat before close")

	s.AddActions("svc:late")

	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()

	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return")
	}

	deltas := client.stream(0).deltas()
	require.Len(t, deltas, 1, "Close flushes the pending delta before tearing down")
	assert.Equal(t, []string{"svc:late"}, deltas[0].Add)

	// Close waits for the loops, so both channels are closed by now.
	_, open := <-actions
	assert.False(t, open, "action channel still open after Close")
	select {
	case err, open := <-errCh:
		assert.False(t, open, "unexpected error after Close: %v", err)
	default:
		t.Fatal("error channel still open after Close")
	}

	waitFor(t, func() bool { return !durable.IsRunning() }, "durable listener still running after Close")

	require.NoError(t, s.Close(), "Close is idempotent")

	_, _, err = s.Actions(context.Background())
	require.ErrorIs(t, err, errListenerClosed)

	s.AddActions("svc:after-close")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.ErrorIs(t, s.Flush(ctx), errListenerClosed)

	heartbeatsAtClose := client.stream(0).heartbeats()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, heartbeatsAtClose, client.stream(0).heartbeats(), "heartbeat loop outlived Close")
}

func TestOperatorSessionActionsCancelClosesChannels(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := connectTestOperatorSession(t, client, true)

	ctx, cancel := context.WithCancel(context.Background())
	actions, errCh, err := s.Actions(ctx)
	require.NoError(t, err)

	cancel()

	select {
	case _, open := <-actions:
		assert.False(t, open)
	case <-time.After(5 * time.Second):
		t.Fatal("action channel not closed after ctx cancel")
	}
	select {
	case err, open := <-errCh:
		assert.False(t, open, "cancellation is a clean exit, got %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("error channel not closed after ctx cancel")
	}
}

func TestOperatorSessionUnaryCallsCarryOperatorIdAndWorkerId(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, admin := connectTestOperatorSession(t, client, true)

	ctx := context.Background()

	resp, err := s.SendStepActionEvent(ctx, &dispatchercontracts.StepActionEvent{
		ActionId:  "svc:one",
		EventType: dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
	})
	require.NoError(t, err)
	assert.Equal(t, "worker-1", resp.WorkerId)
	require.Len(t, client.stepEvents, 1)
	assert.Equal(t, "worker-1", client.stepEvents[0].WorkerId, "empty worker id is filled from the registration")
	assert.Equal(t, "operator-1", client.stepEventOperatorId[0])

	_, err = s.SendStepActionEvent(ctx, &dispatchercontracts.StepActionEvent{WorkerId: "explicit"})
	require.NoError(t, err)
	assert.Equal(t, "explicit", client.stepEvents[1].WorkerId, "explicit worker id is preserved")

	wf := &v1.CreateWorkflowVersionRequest{
		Name:          "wf-put",
		Tasks:         []*v1.CreateTaskOpts{{ReadableId: "a", Action: "MyService:Run"}, {ReadableId: "b", Action: "myService:run"}},
		OnFailureTask: &v1.CreateTaskOpts{ReadableId: "fail", Action: "myService:OnFailure"},
	}
	putResp, actions, err := s.PutWorkflow(ctx, wf)
	require.NoError(t, err)
	assert.Equal(t, "wfv-1", putResp.Id)
	assert.Equal(t, []string{"myService:run", "myService:onfailure"}, actions, "derived actions are normalized and deduplicated")
	require.Len(t, admin.workflows, 1)
	assert.Same(t, wf, admin.workflows[0])
	assert.True(t, admin.authorized[0], "the admin call carries the bearer token")
	assert.Empty(t, admin.operatorId[0], "the admin call carries no operator metadata")
	assert.Empty(t, s.actions.desiredSet(), "PutWorkflow does not touch the action set")

	_, _, err = s.PutWorkflow(ctx, &v1.CreateWorkflowVersionRequest{Name: "bad", Tasks: []*v1.CreateTaskOpts{{ReadableId: "a"}}})
	require.Error(t, err)
	assert.Len(t, admin.workflows, 1, "a workflow whose actions cannot be derived is not put")
}

func TestActionsForWorkflowErrors(t *testing.T) {
	cases := []struct {
		wf   *v1.CreateWorkflowVersionRequest
		name string
	}{
		{wf: nil, name: "nil workflow"},
		{name: "nil task", wf: &v1.CreateWorkflowVersionRequest{Name: "wf", Tasks: []*v1.CreateTaskOpts{nil}}},
		{name: "missing action", wf: &v1.CreateWorkflowVersionRequest{Name: "wf", Tasks: []*v1.CreateTaskOpts{{ReadableId: "a"}}}},
		{name: "malformed action", wf: &v1.CreateWorkflowVersionRequest{Name: "wf", Tasks: []*v1.CreateTaskOpts{{ReadableId: "a", Action: "noverb"}}}},
		{name: "malformed on-failure action", wf: &v1.CreateWorkflowVersionRequest{Name: "wf", OnFailureTask: &v1.CreateTaskOpts{ReadableId: "f", Action: "a:b:c:d"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := actionsForWorkflow(tc.wf)
			require.Error(t, err)
		})
	}
}

func TestOperatorDurableTaskClientRewritesRegisterWorkerId(t *testing.T) {
	var sent *v1.DurableTaskRequest
	inner := &fakeOperatorDurableStream{sendFn: func(req *v1.DurableTaskRequest) error {
		sent = req
		return nil
	}}

	adapter := &operatorDurableTaskClient{
		OperatorService_DurableTaskClient: inner,
		workerId:                          func() string { return "worker-now" },
	}

	require.NoError(t, adapter.Send(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1.DurableTaskRequestRegisterWorker{WorkerId: "worker-then"},
		},
	}))
	assert.Equal(t, "worker-now", sent.GetRegisterWorker().WorkerId)

	memo := &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{Memo: &v1.DurableTaskMemoRequest{}}}
	require.NoError(t, adapter.Send(memo))
	assert.Same(t, memo, sent, "non-register messages pass through untouched")
}

type fakeOperatorDurableStream struct {
	v1.OperatorService_DurableTaskClient
	sendFn func(req *v1.DurableTaskRequest) error
}

func (f *fakeOperatorDurableStream) Send(req *v1.DurableTaskRequest) error {
	return f.sendFn(req)
}
