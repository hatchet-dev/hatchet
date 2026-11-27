package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const (
	// ddlAppName is the application_name of the engine's DDL pool, the one
	// pool that DATABASE_MAX_CONNS does not bound.
	ddlAppName = "server:ddl"

	gateTimeout   = 3 * time.Minute
	readyTimeout  = 2 * time.Minute
	stopTimeout   = 90 * time.Second
	drainTimeout  = 2 * time.Minute
	refillTimeout = 5 * time.Second
)

type runOpts struct {
	engineBin, migrateBin, adminBin string
	databaseURL, rabbitURL          string
	restarts                        int
	logBase                         string
}

func runCmd(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var o runOpts
	var trackName, out string
	fs.StringVar(&o.engineBin, "engine-bin", "", "hatchet-engine binary of the build under test")
	fs.StringVar(&o.migrateBin, "migrate-bin", "", "hatchet-migrate binary of the build under test")
	fs.StringVar(&o.adminBin, "admin-bin", "", "hatchet-admin binary of the build under test")
	fs.StringVar(&trackName, "track", "", "track to run (default: all, in order)")
	fs.IntVar(&o.restarts, "restarts", minRestarts, "SIGTERM restarts per track")
	fs.StringVar(&out, "out", "", "results file, one Result per track")
	fs.StringVar(&o.databaseURL, "database-url", "postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet?sslmode=disable", "Postgres server to create each track's database on")
	fs.StringVar(&o.rabbitURL, "rabbitmq-url", "amqp://user:password@127.0.0.1:5672/", "RabbitMQ server to create each track's vhost on (management API on port 15672)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.engineBin == "" || o.migrateBin == "" || o.adminBin == "" || out == "" || o.restarts < 1 {
		return errors.New("run needs --engine-bin, --migrate-bin, --admin-bin, --out and --restarts >= 1")
	}
	// The admin and migrate binaries run in a scratch directory.
	for _, p := range []*string{&o.engineBin, &o.migrateBin, &o.adminBin} {
		abs, err := filepath.Abs(*p)
		if err != nil {
			return err
		}
		*p = abs
	}
	selected := tracks
	if trackName != "" {
		t, ok := trackByName(trackName)
		if !ok {
			return fmt.Errorf("unknown track %q", trackName)
		}
		selected = []Track{t}
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	o.logBase = strings.TrimSuffix(out, filepath.Ext(out))

	for _, t := range selected {
		res, err := runTrack(o, t)
		if err != nil {
			return fmt.Errorf("track %s: %w", t.Name, err)
		}
		b, err := json.Marshal(res)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
		printSummary(os.Stderr, res)
	}
	return nil
}

// env is a child process's whole environment: nothing of the harness's own
// leaks into it.
type env []string

func (e env) with(kv ...string) env {
	return append(append(env{}, e...), kv...)
}

func runTrack(o runOpts, t Track) (Result, error) {
	ctx := context.Background()
	logf("%s: setting up", t.Name)

	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	name := "restartharness_" + hex.EncodeToString(suffix)

	admin, err := pgx.Connect(ctx, o.databaseURL)
	if err != nil {
		return Result{}, err
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		return Result{}, err
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			logf("%s: could not drop database %s: %v", t.Name, name, err)
		}
	}()
	dbURL, err := withPath(o.databaseURL, name)
	if err != nil {
		return Result{}, err
	}

	// Queue names are static, so a fresh vhost keeps a previous run's
	// messages out of this one.
	if err := rabbitVhost(o.rabbitURL, name, http.MethodPut); err != nil {
		return Result{}, err
	}
	defer func() {
		if err := rabbitVhost(o.rabbitURL, name, http.MethodDelete); err != nil {
			logf("%s: could not delete vhost %s: %v", t.Name, name, err)
		}
	}()
	amqpURL, err := withPath(o.rabbitURL, name)
	if err != nil {
		return Result{}, err
	}

	dir, err := os.MkdirTemp("", "restartharness-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)
	cfgDir := filepath.Join(dir, "generated")

	grpcPort, healthPort := freePort(), freePort()
	base := env{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"TMPDIR=" + os.TempDir(),
		"DATABASE_URL=" + dbURL,
		"DATABASE_MAX_CONNS=" + strconv.Itoa(t.MaxConns),
		"DATABASE_FAIRPOOL_TENANT_MAX_PERCENT=" + strconv.Itoa(t.TenantMaxPercent),
		"DATABASE_FAIRPOOL_SHARED_MAX_PERCENT=" + strconv.Itoa(t.SharedMaxPercent),
		"DATABASE_LOGGER_LEVEL=warn",
		"DATABASE_LOGGER_FORMAT=console",
		"SERVER_MSGQUEUE_KIND=rabbitmq",
		"SERVER_MSGQUEUE_RABBITMQ_URL=" + amqpURL,
		"SERVER_GRPC_PORT=" + strconv.Itoa(grpcPort),
		// Tokens embed the broadcast address, so admin and engine share it.
		"SERVER_GRPC_BROADCAST_ADDRESS=" + fmt.Sprintf("127.0.0.1:%d", grpcPort),
		"SERVER_GRPC_INSECURE=true",
		"SERVER_HEALTHCHECK_PORT=" + strconv.Itoa(healthPort),
		"SERVER_SECURITY_CHECK_ENABLED=false",
		// A task assigned in the last moments before the engine exits can
		// miss its dispatch; this has the next engine reassign it instead of
		// leaving it to time out.
		"SERVER_REASSIGN_UNACKED_TASKS=true",
		"SERVER_AUTH_COOKIE_DOMAIN=localhost",
		"SERVER_AUTH_COOKIE_INSECURE=true",
		"SERVER_LOGGER_LEVEL=warn",
		"SERVER_LOGGER_FORMAT=console",
	}

	shapes := tenantShapes(t)
	ids := make([]string, len(shapes))
	for i := range ids {
		ids[i] = uuid.NewSHA1(uuid.NameSpaceOID, fmt.Appendf(nil, "restartharness-tenant-%d", i)).String()
	}
	seedEnv := func(i int) env {
		return base.with("DEFAULT_TENANT_ID="+ids[i], fmt.Sprintf("DEFAULT_TENANT_SLUG=t%d", i), fmt.Sprintf("DEFAULT_TENANT_NAME=t%d", i))
	}

	if _, err := run(dir, base, o.migrateBin); err != nil {
		return Result{}, err
	}
	if _, err := run(dir, seedEnv(0), o.adminBin, "quickstart", "--skip", "certs", "--generated-config-dir", cfgDir); err != nil {
		return Result{}, err
	}
	tokens := make([]string, len(shapes))
	g := new(errgroup.Group)
	g.SetLimit(8)
	for i := range shapes {
		g.Go(func() error {
			if i > 0 {
				if _, err := run(dir, seedEnv(i), o.adminBin, "--config", cfgDir, "seed"); err != nil {
					return err
				}
			}
			out, err := run(dir, base, o.adminBin, "--config", cfgDir, "token", "create", "--tenant-id", ids[i])
			if err != nil {
				return err
			}
			lines := strings.Fields(strings.TrimSpace(out))
			if len(lines) == 0 {
				return errors.New("token create printed nothing")
			}
			tokens[i] = lines[len(lines)-1]
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return Result{}, err
	}

	engineLog, err := os.Create(o.logBase + "-" + t.Name + "-engine.log")
	if err != nil {
		return Result{}, err
	}
	defer engineLog.Close()
	workerLog, err := os.Create(o.logBase + "-" + t.Name + "-workers.log")
	if err != nil {
		return Result{}, err
	}
	defer workerLog.Close()

	eng := &engine{bin: o.engineBin, args: []string{"--config", cfgDir}, env: base, log: engineLog,
		readyURL: fmt.Sprintf("http://127.0.0.1:%d/ready", healthPort)}
	if err := eng.start(); err != nil {
		return Result{}, err
	}
	defer eng.kill()

	sampleCtx, stopSampling := context.WithCancel(ctx)
	peak := make(chan int, 1)
	go func() { peak <- samplePeakConns(sampleCtx, o.databaseURL, name) }()
	defer stopSampling()

	if err := eng.waitReady(); err != nil {
		return Result{}, err
	}
	logf("%s: engine ready, starting %d workers", t.Name, len(shapes))

	h := &harness{t: t, ops: map[string]*op{}, running: map[runKey]int{}}
	for i, s := range shapes {
		ten, err := h.newTenant(i, s, tokens[i], grpcPort, workerLog)
		if err != nil {
			return Result{}, err
		}
		defer ten.close()
		h.tenants = append(h.tenants, ten)
	}

	if err := h.gate(); err != nil {
		return Result{}, err
	}
	logf("%s: every tenant is scheduling, queueing the backlog", t.Name)

	period := secs(t.PeriodS)
	h.setLoadEnd(time.Now().Add(time.Hour))
	if err := h.queueBacklog(); err != nil {
		return Result{}, err
	}
	t0 := time.Now().Add(time.Second)
	from := t0.Add(secs(t.WarmupS))
	end := from.Add(time.Duration(o.restarts) * period)
	h.setLoadEnd(end)
	for _, ten := range h.tenants {
		go h.probe(ten, t0, end)
	}
	logf("%s: load running, %d restarts every %s after %s warm-up", t.Name, o.restarts, period, secs(t.WarmupS))

	var sigterms []time.Time
	res := Result{Track: t}
	for r := range o.restarts {
		time.Sleep(time.Until(from.Add(time.Duration(r) * period)))
		at := time.Now()
		if err := eng.stop(); err != nil {
			return Result{}, err
		}
		if err := eng.start(); err != nil {
			return Result{}, err
		}
		if err := eng.waitReady(); err != nil {
			return Result{}, err
		}
		down := time.Since(at)
		if down > period {
			return Result{}, fmt.Errorf("restart %d took %.0fs, longer than the %.0fs between restarts", r+1, down.Seconds(), period.Seconds())
		}
		sigterms = append(sigterms, at)
		res.Restarts = append(res.Restarts, Restart{DownS: down.Seconds()})
		logf("%s: restart %d: down %.1fs", t.Name, r+1, down.Seconds())
	}
	time.Sleep(time.Until(end))
	logf("%s: load stopped, draining", t.Name)
	h.drain()

	for _, ten := range h.tenants {
		ten.close()
	}
	eng.kill()
	stopSampling()
	res.PeakConns = <-peak

	h.mu.Lock()
	defer h.mu.Unlock()
	ops := make([]*op, 0, len(h.ops))
	for id, op := range h.ops {
		ops = append(ops, op)
		if op.err == nil && op.starts == 0 {
			res.Lost++
			logf("%s: lost: op %s tenant %d probe %t scheduled %s", t.Name, id, op.tenant, op.probe, op.at.Format(time.RFC3339Nano))
		}
		if op.starts > 1 {
			res.Duplicates++
			logf("%s: duplicate: op %s tenant %d probe %t scheduled %s started %d times, first %s", t.Name, id, op.tenant, op.probe, op.at.Format(time.RFC3339Nano), op.starts, op.started.Format(time.RFC3339Nano))
		}
	}
	res.MaxRunsViolations = h.maxRunsViolations
	res.attribute(len(h.tenants), ops, sigterms, from, end)
	return res, nil
}

// tenantShapes lists each tenant's shape in tenant order.
func tenantShapes(t Track) []TenantShape {
	var out []TenantShape
	for _, s := range []TenantShape{t.Large, t.Small} {
		for range s.Count {
			out = append(out, s)
		}
	}
	return out
}

// run runs one admin or migrate invocation to completion and returns its
// stdout.
func run(dir string, e env, bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = e
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s%s", filepath.Base(bin), strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func withPath(raw, path string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	u.Path = "/" + path
	return u.String(), nil
}

// rabbitVhost creates (PUT, granting the AMQP user full permissions on it) or
// deletes a vhost through the management API on port 15672 of the AMQP host.
func rabbitVhost(amqpURL, vhost, method string) error {
	u, err := url.Parse(amqpURL)
	if err != nil {
		return err
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	api := "http://" + net.JoinHostPort(u.Hostname(), "15672") + "/api"
	reqs := []struct{ path, body string }{{"/vhosts/" + vhost, ""}}
	if method == http.MethodPut {
		reqs = append(reqs, struct{ path, body string }{"/permissions/" + vhost + "/" + url.PathEscape(user), `{"configure":".*","write":".*","read":".*"}`})
	}
	for _, r := range reqs {
		req, err := http.NewRequest(method, api+r.path, strings.NewReader(r.body))
		if err != nil {
			return err
		}
		req.SetBasicAuth(user, pass)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("rabbitmq %s %s: %s: %s", method, r.path, resp.Status, body)
		}
	}
	return nil
}

func freePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// samplePeakConns polls pg_stat_activity once a second until ctx ends and
// returns the most connections to the database it saw, DDL pool aside. Only
// the engine connects to a track's database while it is sampled.
func samplePeakConns(ctx context.Context, serverURL, db string) int {
	conn, err := pgx.Connect(ctx, serverURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())
	peak := 0
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		var n int
		err := conn.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE datname = $1 AND backend_type = 'client backend' AND application_name <> $2", db, ddlAppName).Scan(&n)
		if ctx.Err() != nil {
			return peak
		}
		if err != nil {
			panic(err)
		}
		peak = max(peak, n)
		select {
		case <-ctx.Done():
			return peak
		case <-tick.C:
		}
	}
}

// engine is the hatchet-engine child process, restarted in place.
type engine struct {
	bin      string
	args     []string
	env      env
	log      *os.File
	readyURL string

	cmd    *exec.Cmd
	exited chan struct{}
}

func (e *engine) start() error {
	fmt.Fprintf(e.log, "\n=== restartharness: starting engine at %s ===\n", time.Now().Format(time.RFC3339Nano))
	cmd := exec.Command(e.bin, e.args...) // #nosec G204 -- the engine under test is the harness's input
	cmd.Env = e.env
	cmd.Stdout = e.log
	cmd.Stderr = e.log
	if err := cmd.Start(); err != nil {
		return err
	}
	e.cmd, e.exited = cmd, make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(e.exited)
	}()
	return nil
}

func (e *engine) waitReady() error {
	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(readyTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-e.exited:
			return fmt.Errorf("engine exited before it was ready: %s", e.cmd.ProcessState)
		default:
		}
		if resp, err := client.Get(e.readyURL); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("engine not ready after %s", readyTimeout)
}

// stop SIGTERMs the engine and waits for it to exit. An engine that already
// exited crashed.
func (e *engine) stop() error {
	select {
	case <-e.exited:
		return fmt.Errorf("engine exited unexpectedly: %s", e.cmd.ProcessState)
	default:
	}
	if err := e.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	select {
	case <-e.exited:
		if !e.cmd.ProcessState.Success() {
			logf("engine exited after SIGTERM with %s", e.cmd.ProcessState)
		}
		return nil
	case <-time.After(stopTimeout):
		e.kill()
		return fmt.Errorf("engine did not exit within %s of SIGTERM", stopTimeout)
	}
}

func (e *engine) kill() {
	select {
	case <-e.exited:
	default:
		_ = e.cmd.Process.Kill()
		<-e.exited
	}
}

type taskInput struct {
	Op      string `json:"op"`
	Key     string `json:"key"`
	SleepMs int    `json:"sleep_ms"`
}

type taskOutput struct{}

// op is one trigger the harness sent, keyed by the operation id in its input.
type op struct {
	tenant int
	probe  bool
	at     time.Time // the trigger's scheduled timestamp
	done   bool      // the trigger RPC returned
	err    error     // its result: nil means accepted

	starts  int
	started time.Time // the first start
}

type runKey struct {
	tenant, strategy int
	key              string
}

type harness struct {
	t                 Track
	tenants           []*tenant
	mu                sync.Mutex
	loadEnd           time.Time // backlog refills stop here
	nextOp            int
	ops               map[string]*op
	running           map[runKey]int
	maxRunsViolations int
}

type tenant struct {
	idx   int
	shape TenantShape
	token string
	conn  *grpc.ClientConn
	wf    admincontracts.WorkflowServiceClient
	stop  func() error
	once  sync.Once
}

func (h *harness) setLoadEnd(t time.Time) {
	h.mu.Lock()
	h.loadEnd = t
	h.mu.Unlock()
}

func strategyName(s int) string { return fmt.Sprintf("s%d", s) }

// newTenant starts the tenant's SDK worker and dials its trigger connection.
func (h *harness) newTenant(idx int, shape TenantShape, token string, grpcPort int, logOut io.Writer) (*tenant, error) {
	l := zerolog.New(logOut).Level(zerolog.WarnLevel).With().Timestamp().Int("tenant", idx).Logger()
	client, err := hatchet.NewClient(
		hatchet.WithToken(token),
		hatchet.WithHostPort("127.0.0.1", grpcPort),
		hatchet.WithTLSConfig(nil),
		hatchet.WithClientLogger(&l),
	)
	if err != nil {
		return nil, err
	}
	maxRuns := int32(1)
	limit := hatchet.GroupRoundRobin
	var tasks []hatchet.WorkflowBase
	for s := range shape.Strategies {
		tasks = append(tasks, client.NewStandaloneTask(strategyName(s), h.handler(idx, s),
			hatchet.WithConcurrency(&hatchet.Concurrency{Expression: "input.key", MaxRuns: &maxRuns, LimitStrategy: &limit})))
	}
	// Backlog tasks hold at most one slot per strategy; the rest are for
	// probes, which must never wait for one.
	w, err := client.NewWorker(fmt.Sprintf("w%d", idx), hatchet.WithWorkflows(tasks...), hatchet.WithSlots(shape.Backlogged+100))
	if err != nil {
		return nil, err
	}
	stop, err := w.Start()
	if err != nil {
		return nil, err
	}

	// Triggers go over their own connection: no retry interceptor, and a
	// short reconnect backoff so a trigger fails only while the engine is
	// actually unreachable.
	conn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.Config{BaseDelay: 50 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 500 * time.Millisecond},
			MinConnectTimeout: time.Second,
		}))
	if err != nil {
		return nil, err
	}
	return &tenant{idx: idx, shape: shape, token: token, conn: conn, wf: admincontracts.NewWorkflowServiceClient(conn), stop: stop}, nil
}

func (t *tenant) close() {
	t.once.Do(func() {
		_ = t.stop()
		_ = t.conn.Close()
	})
}

func (h *harness) handler(tenant, strategy int) func(hatchet.Context, taskInput) (taskOutput, error) {
	return func(_ hatchet.Context, in taskInput) (taskOutput, error) {
		now := time.Now()
		k := runKey{tenant, strategy, in.Key}
		h.mu.Lock()
		o := h.ops[in.Op]
		o.starts++
		first := o.starts == 1
		if first {
			o.started = now
		}
		if h.running[k] >= 1 {
			h.maxRunsViolations++
		}
		h.running[k]++
		// Each backlog start queues one more, so the queue stays Backlog deep.
		refill := !o.probe && first && now.Before(h.loadEnd)
		h.mu.Unlock()

		if refill {
			go h.refill(h.tenants[tenant], strategy) // #nosec G118 -- the refill outlives the task
		}
		time.Sleep(time.Duration(in.SleepMs) * time.Millisecond)

		h.mu.Lock()
		h.running[k]--
		h.mu.Unlock()
		return taskOutput{}, nil
	}
}

// refill queues one more backlog task, trying again until the engine accepts
// it or load ends: a refill lost to a restart would shrink the backlog for
// every later restart.
func (h *harness) refill(t *tenant, strategy int) {
	for {
		h.mu.Lock()
		over := !time.Now().Before(h.loadEnd)
		h.mu.Unlock()
		if over || h.trigger(t, strategy, false, time.Now(), refillTimeout).err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// trigger registers an operation scheduled at at and sends its trigger with
// a deadline of timeout after at. A probe gets a key no other run uses.
func (h *harness) trigger(t *tenant, strategy int, probe bool, at time.Time, timeout time.Duration) *op {
	h.mu.Lock()
	h.nextOp++
	id := strconv.Itoa(h.nextOp)
	o := &op{tenant: t.idx, probe: probe, at: at}
	h.ops[id] = o
	h.mu.Unlock()

	in := taskInput{Op: id, Key: "backlog", SleepMs: h.t.TaskMs}
	if probe {
		in = taskInput{Op: id, Key: "probe-" + id}
	}
	input, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), at.Add(timeout))
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+t.token)
	_, err = t.wf.TriggerWorkflow(ctx, &admincontracts.TriggerWorkflowRequest{Name: strategyName(strategy), Input: string(input)})

	h.mu.Lock()
	o.done, o.err = true, err
	h.mu.Unlock()
	return o
}

// gate waits until every tenant has had a task start: a new tenant is only
// scheduled once the engine's partition rebalance has picked it up.
func (h *harness) gate() error {
	deadline := time.Now().Add(gateTimeout)
	pending := h.tenants
	for len(pending) > 0 {
		if time.Now().After(deadline) {
			return fmt.Errorf("%d tenant(s) had no task start within %s", len(pending), gateTimeout)
		}
		ops := make([]*op, len(pending))
		var wg sync.WaitGroup
		for i, t := range pending {
			wg.Go(func() { ops[i] = h.trigger(t, 0, true, time.Now(), 5*time.Second) })
		}
		wg.Wait()
		time.Sleep(5 * time.Second)
		var next []*tenant
		h.mu.Lock()
		for i, o := range ops {
			if o.starts == 0 {
				next = append(next, pending[i])
			}
		}
		h.mu.Unlock()
		pending = next
	}
	return nil
}

// queueBacklog queues each strategy's standing backlog, paced below the
// engine's per-token gRPC rate limit.
func (h *harness) queueBacklog() error {
	var mu sync.Mutex
	var failed error
	var wg sync.WaitGroup
	for _, t := range h.tenants {
		for s := range t.shape.Backlogged {
			for range t.shape.Backlog {
				wg.Go(func() {
					if o := h.trigger(t, s, false, time.Now(), 30*time.Second); o.err != nil {
						mu.Lock()
						failed = o.err
						mu.Unlock()
					}
				})
				time.Sleep(2 * time.Millisecond)
			}
		}
	}
	wg.Wait()
	if failed != nil {
		return fmt.Errorf("queueing the backlog: %w", failed)
	}
	return nil
}

// probe offers the tenant's probes at fixed timestamps from t0 until end,
// never waiting for an earlier one. Tenants are phase-shifted so their probes
// interleave.
func (h *harness) probe(t *tenant, t0, end time.Time) {
	interval := secs(1 / t.shape.ProbesPerS)
	phase := interval * time.Duration(t.idx) / time.Duration(len(h.tenants))
	for k := 0; ; k++ {
		at := t0.Add(phase + time.Duration(k)*interval)
		if !at.Before(end) {
			return
		}
		time.Sleep(time.Until(at))
		go h.trigger(t, k%t.shape.Strategies, true, at, secs(h.t.APIDeadlineS))
	}
}

// drain waits until every trigger has returned and every accepted one has
// started, or drainTimeout passes. Halfway through it names what it still
// waits for.
func (h *harness) drain() {
	start := time.Now()
	named := false
	for time.Since(start) < drainTimeout {
		h.mu.Lock()
		var waiting []string
		for id, o := range h.ops {
			if !o.done || (o.err == nil && o.starts == 0) {
				waiting = append(waiting, fmt.Sprintf("%s (tenant %d, scheduled %s)", id, o.tenant, o.at.Format(time.RFC3339Nano)))
			}
		}
		h.mu.Unlock()
		if len(waiting) == 0 {
			return
		}
		if !named && time.Since(start) > drainTimeout/2 {
			named = true
			logf("still waiting for %d operation(s): %s", len(waiting), strings.Join(waiting[:min(len(waiting), 10)], ", "))
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func secs(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s restartharness: %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

func printSummary(w io.Writer, r Result) {
	lost := r.lostS()
	var sum float64
	parts := make([]string, len(lost))
	for i, l := range lost {
		sum += l
		parts[i] = fmt.Sprintf("%.2f (down %.1fs)", l, r.Restarts[i].DownS)
	}
	steady := 0.0
	if r.Steady.Offered > 0 {
		steady = 100 * float64(r.Steady.Bad) / float64(r.Steady.Offered)
	}
	fmt.Fprintf(w, "\n== %s\nlost seconds per restart: %s\nmean lost seconds: %.2f\nsteady bad: %.3f%% of %d, peak conns %d/%d\nerrors: %+v\n",
		r.Track.Name, strings.Join(parts, ", "), sum/float64(max(len(lost), 1)), steady, r.Steady.Offered, r.PeakConns, r.Track.MaxConns, r.Errors)
	if v := r.violations(); len(v) > 0 {
		fmt.Fprintf(w, "INVALID: %s\n", strings.Join(v, "; "))
	} else {
		fmt.Fprintln(w, "valid")
	}
}
