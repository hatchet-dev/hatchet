package dispatcher

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// These opt-in experiments use the production dispatcher send path over a real local
// HTTP/2 connection. The receive-side conditions are controlled; send latency is measured.
// HATCHET_STREAM_STALL_REPRO_DIR enables them and names the directory for measurements
// and goroutine snapshots. They do not start databases, queues, or an engine instance.
func requireStallRepro(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("HATCHET_STREAM_STALL_REPRO_DIR")
	if dir == "" {
		t.Skip("set HATCHET_STREAM_STALL_REPRO_DIR to run transport experiments")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Baseline mode retains the incident's admission timeout. Verification mode uses
// the dispatcher default so the same transport faults exercise the configured fix.
func stallAdmissionTimeout() time.Duration {
	if os.Getenv("HATCHET_STREAM_STALL_REPRO_VERIFY_FIX") != "" {
		return defaultDispatcherOpts().defaultMaxWorkerLockAcquisitionTime
	}
	return 250 * time.Millisecond
}

type stallSendSample struct {
	Start      time.Time `json:"start"`
	DurationMS float64   `json:"duration_ms"`
	Error      string    `json:"error,omitempty"`
}

type stallRecordingStream struct {
	stream     *connect.ServerStream[contracts.AssignedAction]
	mu         sync.Mutex
	samples    []stallSendSample
	active     atomic.Int64
	beforeSend func()
}

func (s *stallRecordingStream) Send(action *contracts.AssignedAction) error {
	begin := time.Now()
	s.active.Store(begin.UnixNano())
	if s.beforeSend != nil {
		s.beforeSend()
	}
	err := s.stream.Send(action)
	s.active.Store(0)
	sample := stallSendSample{Start: begin, DurationMS: float64(time.Since(begin)) / float64(time.Millisecond)}
	if err != nil {
		sample.Error = err.Error()
	}
	s.mu.Lock()
	s.samples = append(s.samples, sample)
	s.mu.Unlock()
	return err
}

type stallHarness struct {
	worker *subscribedWorker
	stream *stallRecordingStream
}

type stallTCPListener struct {
	net.Listener
	smallBuffer bool
}

func (l stallTCPListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil && l.smallBuffer {
		if tcp, ok := c.(*net.TCPConn); ok {
			err = tcp.SetWriteBuffer(4096)
		}
		if err != nil {
			_ = c.Close()
		}
	}
	return c, err
}

func startStallServer(t *testing.T, smallBuffer bool) (string, <-chan *stallHarness) {
	t.Helper()
	ready := make(chan *stallHarness, 1)
	handler := connect.NewServerStreamHandler("/Dispatcher/ListenV2", func(ctx context.Context, _ *connect.Request[contracts.WorkerListenRequest], stream *connect.ServerStream[contracts.AssignedAction]) error {
		recording := &stallRecordingStream{stream: stream}
		sender := rpcstream.NewSender[contracts.AssignedAction](ctx, recording)
		defer sender.Close()
		ready <- &stallHarness{worker: newGRPCSubscribedWorker(sender, nil, uuid.New(), stallAdmissionTimeout(), nil), stream: recording}
		<-ctx.Done()
		return nil
	}, connect.WithCompressMinBytes(1<<30))
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	server := &http.Server{Protocols: protocols, HTTP2: &http.HTTP2Config{MaxReceiveBufferPerStream: 10 * 1024 * 1024, MaxReceiveBufferPerConnection: 20 * 1024 * 1024}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := rpcstream.WithAbort(r.Context(), func() { _ = http.NewResponseController(w).SetWriteDeadline(time.Now()) })
		handler.ServeHTTP(w, r.WithContext(ctx))
	})}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(stallTCPListener{Listener: lis, smallBuffer: smallBuffer}) }()
	t.Cleanup(func() { _ = server.Close() })
	return lis.Addr().String(), ready
}

func stallAction() *contracts.AssignedAction {
	return &contracts.AssignedAction{
		TenantId:          "11111111-1111-4111-8111-111111111111",
		TaskRunExternalId: "22222222-2222-4222-8222-222222222222",
		WorkflowRunId:     "33333333-3333-4333-8333-333333333333",
		ActionId:          "repro:task", TaskName: "repro-task", ActionPayload: strings.Repeat("x", 515),
	}
}

func awaitStallHarness(t *testing.T, ready <-chan *stallHarness) *stallHarness {
	t.Helper()
	select {
	case h := <-ready:
		return h
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not start")
		return nil
	}
}

func stallSnapshot(t *testing.T, dir, name string) {
	t.Helper()
	buf := make([]byte, 8<<20)
	n := runtime.Stack(buf, true)
	if err := os.WriteFile(filepath.Join(dir, name+".stacks.txt"), buf[:n], 0644); err != nil {
		t.Error(err)
	}
}

func awaitStallCount(t *testing.T, counter *atomic.Int64, want int64) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if counter.Load() >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("receiver counted %d, want at least %d", counter.Load(), want)
}

type stallBulkResult struct {
	Success     int       `json:"success"`
	LockTimeout int       `json:"lock_timeout"`
	OtherErrors []string  `json:"other_errors,omitempty"`
	CallerMS    []float64 `json:"caller_ms"`
}

func stallBulk(ctx context.Context, h *stallHarness, action *contracts.AssignedAction, count int) <-chan stallBulkResult {
	done := make(chan stallBulkResult, 1)
	go func() {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var result stallBulkResult
		for range count {
			wg.Go(func() {
				begin := time.Now()
				err := h.worker.sendToWorker(ctx, action)
				mu.Lock()
				defer mu.Unlock()
				result.CallerMS = append(result.CallerMS, float64(time.Since(begin))/float64(time.Millisecond))
				switch {
				case err == nil:
					result.Success++
				case errors.Is(err, errFlowControlActive):
					result.LockTimeout++
				default:
					result.OtherErrors = append(result.OtherErrors, err.Error())
				}
			})
		}
		wg.Wait()
		done <- result
	}()
	return done
}

func writeStallResult(t *testing.T, dir, name string, h *stallHarness, bulk stallBulkResult, extra map[string]any) {
	t.Helper()
	h.stream.mu.Lock()
	samples := append([]stallSendSample(nil), h.stream.samples...)
	h.stream.mu.Unlock()
	maxMS := 0.0
	slow := 0
	for _, sample := range samples {
		if sample.DurationMS > maxMS {
			maxMS = sample.DurationMS
		}
		if sample.DurationMS > 50 {
			slow++
		}
	}
	data := map[string]any{"name": name, "bulk": bulk, "sends": samples, "max_transport_send_ms": maxMS, "transport_sends_over_50ms": slow, "admission_timeout_ms": h.worker.sendLock.Timeout.Milliseconds(), "details": extra}
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), encoded, 0644); err != nil {
		t.Fatal(err)
	}
	journal, err := os.OpenFile(filepath.Join(dir, "measurements.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(journal).Encode(data); err != nil {
		_ = journal.Close()
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: successes=%d lock_timeouts=%d other_errors=%v transport_max=%.2fms slow_sends=%d details=%v", name, bulk.Success, bulk.LockTimeout, bulk.OtherErrors, maxMS, slow, extra)
}

func TestStreamStallReproReceiverPause(t *testing.T) {
	dir := requireStallRepro(t)
	action := stallAction()
	wireBytes := proto.Size(action) + 5
	for _, cfg := range []struct {
		name           string
		window         int32
		prefill        int
		drainInitially bool
		wantFailures   bool
	}{
		{"paused_fresh_window", 65535, 0, false, false},
		{"paused_depleted_window", 65535, 65535/wireBytes - 24, false, true},
		{"paused_larger_window", 4 << 20, 65535/wireBytes - 24, false, false},
		{"actively_reading_small_window", 65535, 65535/wireBytes - 24, true, false},
	} {
		t.Run(cfg.name, func(t *testing.T) {
			addr, ready := startStallServer(t, false)
			conn, err := grpcgo.NewClient(addr, grpcgo.WithTransportCredentials(insecure.NewCredentials()), grpcgo.WithStaticStreamWindowSize(cfg.window), grpcgo.WithStaticConnWindowSize(4<<20))
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			client, err := contracts.NewDispatcherClient(conn).ListenV2(ctx, &contracts.WorkerListenRequest{WorkerId: "repro"})
			if err != nil {
				t.Fatal(err)
			}
			h := awaitStallHarness(t, ready)
			var received atomic.Int64
			drain := func() {
				go func() {
					for {
						if _, err := client.Recv(); err != nil {
							return
						}
						received.Add(1)
					}
				}()
			}
			if cfg.drainInitially {
				drain()
			}
			for range cfg.prefill {
				if err := h.worker.sendToWorker(ctx, action); err != nil {
					t.Fatal(err)
				}
			}
			begin := time.Now()
			done := stallBulk(ctx, h, action, 50)
			time.Sleep(100 * time.Millisecond)
			stallSnapshot(t, dir, cfg.name)
			if remaining := 500*time.Millisecond - time.Since(begin); remaining > 0 {
				time.Sleep(remaining)
			}
			if !cfg.drainInitially {
				drain()
			}
			result := <-done
			awaitStallCount(t, &received, int64(cfg.prefill+result.Success))
			writeStallResult(t, dir, cfg.name, h, result, map[string]any{"payload_bytes": 515, "grpc_message_bytes": wireBytes, "window_bytes": cfg.window, "prefill_messages": cfg.prefill, "resume_after_ms": 500, "actively_reading": cfg.drainInitially, "received_at_result": received.Load()})
			if len(result.OtherErrors) > 0 {
				t.Fatal(result.OtherErrors)
			}
			wantSuccess, wantTimeouts := 50, 0
			if cfg.wantFailures && stallAdmissionTimeout() < 500*time.Millisecond {
				wantSuccess, wantTimeouts = 25, 25
			}
			if result.Success != wantSuccess || result.LockTimeout != wantTimeouts {
				t.Fatalf("want %d successful / %d failed sends, got %+v", wantSuccess, wantTimeouts, result)
			}
			if !cfg.wantFailures && result.Success != 50 {
				t.Fatalf("want 50 successes, got %+v", result)
			}
		})
	}
}

type stallRawPeer struct {
	conn       net.Conn
	framer     *http2.Framer
	writeMu    sync.Mutex
	readGate   chan struct{}
	resumeOnce sync.Once
	dataBytes  atomic.Int64
	lastData   atomic.Int64
}

func newStallRawPeer(t *testing.T, addr string, window uint32, pauseReads bool) *stallRawPeer {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if pauseReads {
		if err := conn.(*net.TCPConn).SetReadBuffer(4096); err != nil {
			t.Fatal(err)
		}
	}
	p := &stallRawPeer{conn: conn, framer: http2.NewFramer(conn, conn), readGate: make(chan struct{})}
	if _, err := conn.Write([]byte(http2.ClientPreface)); err != nil {
		t.Fatal(err)
	}
	if err := p.framer.WriteSettings(http2.Setting{ID: http2.SettingInitialWindowSize, Val: window}); err != nil {
		t.Fatal(err)
	}
	if err := p.framer.WriteWindowUpdate(0, 16*1024*1024-65535); err != nil {
		t.Fatal(err)
	}
	var headers bytes.Buffer
	encoder := hpack.NewEncoder(&headers)
	for _, h := range []hpack.HeaderField{{Name: ":method", Value: "POST"}, {Name: ":scheme", Value: "http"}, {Name: ":authority", Value: addr}, {Name: ":path", Value: "/Dispatcher/ListenV2"}, {Name: "content-type", Value: "application/grpc"}, {Name: "te", Value: "trailers"}} {
		if err := encoder.WriteField(h); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.framer.WriteHeaders(http2.HeadersFrameParam{StreamID: 1, BlockFragment: headers.Bytes(), EndHeaders: true}); err != nil {
		t.Fatal(err)
	}
	request, err := proto.Marshal(&contracts.WorkerListenRequest{WorkerId: "repro"})
	if err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, 5, len(request)+5)
	binary.BigEndian.PutUint32(frame[1:], uint32(len(request)))
	frame = append(frame, request...)
	if err := p.framer.WriteData(1, true, frame); err != nil {
		t.Fatal(err)
	}
	if !pauseReads {
		p.resume()
	}
	go func() {
		<-p.readGate
		for {
			frame, err := p.framer.ReadFrame()
			if err != nil {
				return
			}
			switch f := frame.(type) {
			case *http2.DataFrame:
				p.dataBytes.Add(int64(len(f.Data())))
				p.lastData.Store(time.Now().UnixNano())
			case *http2.SettingsFrame:
				if !f.IsAck() {
					p.writeMu.Lock()
					_ = p.framer.WriteSettingsAck()
					p.writeMu.Unlock()
				}
			case *http2.PingFrame:
				if !f.IsAck() {
					p.writeMu.Lock()
					_ = p.framer.WritePing(true, f.Data)
					p.writeMu.Unlock()
				}
			}
		}
	}()
	// Unblock the reader on every failure path before closing its connection.
	t.Cleanup(p.resume)
	return p
}

func (p *stallRawPeer) resume() { p.resumeOnce.Do(func() { close(p.readGate) }) }
func (p *stallRawPeer) addCredit(n uint32) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return p.framer.WriteWindowUpdate(1, n)
}

func TestStreamStallReproDelayedWindowUpdate(t *testing.T) {
	dir := requireStallRepro(t)
	action := stallAction()
	wireBytes := proto.Size(action) + 5
	for _, ample := range []bool{false, true} {
		name := "delayed_window_update"
		window := uint32(24 * wireBytes)
		if ample {
			name = "delayed_window_update_larger_window"
			window = 1 << 20
		}
		t.Run(name, func(t *testing.T) {
			addr, ready := startStallServer(t, false)
			peer := newStallRawPeer(t, addr, window, false)
			h := awaitStallHarness(t, ready)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			begin := time.Now()
			done := stallBulk(ctx, h, action, 50)
			time.Sleep(100 * time.Millisecond)
			bytesBeforeUpdate := peer.dataBytes.Load()
			lastDataMS := float64(peer.lastData.Load()-begin.UnixNano()) / 1e6
			stallSnapshot(t, dir, name)
			if left := 500*time.Millisecond - time.Since(begin); left > 0 {
				time.Sleep(left)
			}
			if err := peer.addCredit(1 << 20); err != nil {
				t.Fatal(err)
			}
			result := <-done
			awaitStallCount(t, &peer.dataBytes, int64(result.Success*wireBytes))
			writeStallResult(t, dir, name, h, result, map[string]any{"initial_stream_window": window, "data_read_before_update": bytesBeforeUpdate, "last_data_before_update_ms": lastDataMS, "credit_delay_ms": 500, "application_reading": true})
			if len(result.OtherErrors) > 0 {
				t.Fatal(result.OtherErrors)
			}
			wantSuccess, wantTimeouts := 50, 0
			if !ample && stallAdmissionTimeout() < 500*time.Millisecond {
				wantSuccess, wantTimeouts = 25, 25
			}
			if !ample && (result.Success != wantSuccess || result.LockTimeout != wantTimeouts || bytesBeforeUpdate != int64(window)) {
				t.Fatalf("unexpected outcome: %+v bytes=%d", result, bytesBeforeUpdate)
			}
			if ample && result.Success != 50 {
				t.Fatalf("unexpected control outcome: %+v", result)
			}
		})
	}
}

func TestStreamStallReproSocketBackpressure(t *testing.T) {
	dir := requireStallRepro(t)
	addr, ready := startStallServer(t, true)
	peer := newStallRawPeer(t, addr, 16<<20, true)
	h := awaitStallHarness(t, ready)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	action := stallAction()
	var stop atomic.Bool
	prefillDone := make(chan error, 1)
	go func() {
		for i := 0; i < 10000 && !stop.Load(); i++ {
			if err := h.worker.sendToWorker(ctx, action); err != nil {
				prefillDone <- err
				return
			}
		}
		prefillDone <- nil
	}()
	var blockedAt int64
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		active := h.stream.active.Load()
		if active != 0 && time.Since(time.Unix(0, active)) > 200*time.Millisecond {
			blockedAt = active
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if blockedAt == 0 {
		peer.resume()
		t.Fatal("did not saturate TCP buffers")
	}
	stop.Store(true)
	h.stream.mu.Lock()
	completedBefore := len(h.stream.samples)
	h.stream.mu.Unlock()
	// The active prefill send is the first member; siblings encounter its held send lock.
	done := stallBulk(ctx, h, action, 49)
	stallSnapshot(t, dir, "socket_backpressure")
	if left := 500*time.Millisecond - time.Since(time.Unix(0, blockedAt)); left > 0 {
		time.Sleep(left)
	}
	peer.resume()
	result := <-done
	if err := <-prefillDone; err != nil {
		t.Fatal(err)
	}
	result.Success++
	writeStallResult(t, dir, "socket_backpressure", h, result, map[string]any{"initial_stream_and_connection_window": 16 << 20, "prefill_completed": completedBefore, "prefill_grpc_bytes": completedBefore * (proto.Size(action) + 5), "tcp_write_buffer_requested": 4096, "tcp_read_buffer_requested": 4096, "socket_resume_after_active_ms": 500})
	wantTimeouts := 49
	if stallAdmissionTimeout() >= 500*time.Millisecond {
		wantTimeouts = 0
	}
	if result.LockTimeout != wantTimeouts || result.Success != 50-wantTimeouts || len(result.OtherErrors) > 0 {
		t.Fatalf("unexpected outcome: %+v", result)
	}
	if (completedBefore+1)*(proto.Size(action)+5) >= 16<<20 {
		t.Fatal("HTTP/2 window was not excluded as the bottleneck")
	}
	// Prove that the same amount of traffic fits in the available HTTP/2 credit once
	// the peer consumes socket data, without sending any WINDOW_UPDATE frames.
	h.stream.mu.Lock()
	h.stream.samples = nil
	h.stream.mu.Unlock()
	control := <-stallBulk(ctx, h, action, 50)
	awaitStallCount(t, &peer.dataBytes, int64((completedBefore+result.Success+control.Success)*(proto.Size(action)+5)))
	writeStallResult(t, dir, "socket_reading_control", h, control, map[string]any{"peer_reading": true, "additional_window_updates": 0})
	if control.Success != 50 {
		t.Fatalf("unexpected control outcome: %+v", control)
	}
}

var stallCPUSink atomic.Uint64

func stallBurnCPU(iterations int) {
	x := uint64(123456789)
	for range iterations {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
	}
	stallCPUSink.Store(x)
}

func TestStreamStallReproScheduling(t *testing.T) {
	dir := requireStallRepro(t)
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	addr, ready := startStallServer(t, false)
	peer := newStallRawPeer(t, addr, 4<<20, false)
	h := awaitStallHarness(t, ready)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	action := stallAction()
	baseline := <-stallBulk(ctx, h, action, 50)
	writeStallResult(t, dir, "scheduling_baseline", h, baseline, map[string]any{"gomaxprocs": 1})
	if baseline.Success != 50 {
		t.Fatalf("baseline failed: %+v", baseline)
	}
	h.stream.mu.Lock()
	h.stream.samples = nil
	h.stream.mu.Unlock()
	calibration := time.Now()
	stallBurnCPU(1000000)
	iterations := int(float64(1000000) * float64(15*time.Millisecond) / float64(time.Since(calibration)))
	gate := make(chan struct{})
	var wg sync.WaitGroup
	for range 64 {
		wg.Go(func() { <-gate; stallBurnCPU(iterations) })
	}
	var once sync.Once
	h.stream.beforeSend = func() { once.Do(func() { close(gate) }) }
	traceFile, err := os.Create(filepath.Join(dir, "scheduling.trace"))
	if err != nil {
		t.Fatal(err)
	}
	defer traceFile.Close()
	if err := trace.Start(traceFile); err != nil {
		t.Fatal(err)
	}
	done := stallBulk(ctx, h, action, 50)
	time.Sleep(100 * time.Millisecond)
	stallSnapshot(t, dir, "scheduling")
	result := <-done
	wg.Wait()
	trace.Stop()
	awaitStallCount(t, &peer.dataBytes, int64((baseline.Success+result.Success)*(proto.Size(action)+5)))
	writeStallResult(t, dir, "scheduling", h, result, map[string]any{"gomaxprocs": 1, "cpu_competitors": 64, "calibrated_cpu_ms_each": 15, "iterations_each": iterations, "receiver_window_bytes": 4 << 20, "peer_data_bytes": peer.dataBytes.Load()})
	if len(result.OtherErrors) > 0 {
		t.Fatal(result.OtherErrors)
	}
}
