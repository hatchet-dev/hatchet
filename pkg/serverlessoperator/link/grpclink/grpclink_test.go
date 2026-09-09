//go:build !e2e && !load && !rampup && !integration

package grpclink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// testJWT builds an unsigned JWT with the claims the client loader reads.
func testJWT(t *testing.T, tenantId uuid.UUID) string {
	t.Helper()

	header, _ := json.Marshal(map[string]string{"alg": "none"})
	claims, err := json.Marshal(map[string]any{
		"sub":                    tenantId.String(),
		"server_url":             "https://app.example.test",
		"grpc_broadcast_address": "grpc.example.test:443",
		"exp":                    time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)

	enc := base64.RawURLEncoding

	return enc.EncodeToString(header) + "." + enc.EncodeToString(claims) + ".sig"
}

// writeFile replaces path atomically, through a temporary file and a rename, the way a
// rotated secret or a mounted file is swapped. The exchange's poller stats the file between
// writes, so a truncate-then-write could be observed as an empty (valid, tenantless) file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	tmp := path + ".tmp"
	require.NoError(t, os.WriteFile(tmp, []byte(content), 0o600))
	require.NoError(t, os.Rename(tmp, path))
}

func TestLocalExchangeLoadsAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tenants.yaml")

	tenantA := uuid.New()
	tenantB := uuid.New()

	writeFile(t, filepath.Join(dir, "b.token"), "  token-b\n")
	writeFile(t, path, "tenants:\n  "+tenantA.String()+": { token: token-a }\n  "+tenantB.String()+": { token_file: b.token }\n")

	ex, err := NewLocalExchange(path, WithPollInterval(10*time.Millisecond))
	require.NoError(t, err)
	defer ex.Close()

	tok, err := ex.Token(context.Background(), tenantA)
	require.NoError(t, err)
	assert.Equal(t, "token-a", tok)

	tok, err = ex.Token(context.Background(), tenantB)
	require.NoError(t, err)
	assert.Equal(t, "token-b", tok, "token_file is read relative to the yaml and trimmed")

	_, err = ex.Token(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNoToken)
	assert.ErrorIs(t, err, link.ErrNoToken)

	// Rewriting the file with a newer mtime is picked up by the poller.
	writeFile(t, path, "tenants:\n  "+tenantA.String()+": { token: token-a2 }\n")
	future := time.Now().Add(2 * time.Second)
	require.NoError(t, os.Chtimes(path, future, future))

	require.Eventually(t, func() bool {
		tok, err := ex.Token(context.Background(), tenantA)
		return err == nil && tok == "token-a2"
	}, 2*time.Second, 5*time.Millisecond)

	_, err = ex.Token(context.Background(), tenantB)
	assert.ErrorIs(t, err, ErrNoToken, "a tenant removed from the file is no longer served")

	// A broken rewrite keeps the previous mapping.
	writeFile(t, path, "tenants:\n  not-a-uuid: { token: x }\n")
	later := future.Add(2 * time.Second)
	require.NoError(t, os.Chtimes(path, later, later))

	time.Sleep(50 * time.Millisecond)

	tok, err = ex.Token(context.Background(), tenantA)
	require.NoError(t, err)
	assert.Equal(t, "token-a2", tok)
}

func TestLocalExchangeRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := NewLocalExchange(filepath.Join(dir, "missing.yaml"))
	assert.Error(t, err)

	path := filepath.Join(dir, "empty-entry.yaml")
	writeFile(t, path, "tenants:\n  "+uuid.New().String()+": {}\n")

	_, err = NewLocalExchange(path)
	assert.Error(t, err, "an entry needs token or token_file")
}

func TestStaticExchange(t *testing.T) {
	tenant := uuid.New()
	tok := testJWT(t, tenant)

	ex, err := NewStaticExchange(tok)
	require.NoError(t, err)
	assert.Equal(t, tenant, ex.TenantId())

	got, err := ex.Token(context.Background(), tenant)
	require.NoError(t, err)
	assert.Equal(t, tok, got)

	_, err = ex.Token(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNoToken)

	_, err = NewStaticExchange("not.a.jwt")
	assert.Error(t, err)
}

// fakeSession is the minimal OperatorSession the link needs. Durable listeners are built
// over a fakeDurableStream (durable_test.go) and stopped on Close like the real session does.
// Action deltas and flushes are recorded in order; flushErr, when set, fails the next Flush.
type fakeSession struct {
	client.OperatorSession
	stream    *fakeDurableStream
	listeners []*client.DurableTaskListener
	workerId  string
	tenantId  string
	added     [][]string
	removed   [][]string
	puts      []*v1.CreateWorkflowVersionRequest
	flushErr  error
	flushes   int
	closed    bool
}

func (f *fakeSession) Registration() client.OperatorRegistration {
	return client.OperatorRegistration{WorkerId: f.workerId, TenantId: f.tenantId}
}

func (f *fakeSession) AddActions(ids ...string) {
	f.added = append(f.added, append([]string{}, ids...))
}

func (f *fakeSession) RemoveActions(ids ...string) {
	f.removed = append(f.removed, append([]string{}, ids...))
}

func (f *fakeSession) Flush(context.Context) error {
	f.flushes++

	if f.flushErr != nil {
		err := f.flushErr
		f.flushErr = nil

		return err
	}

	return nil
}

func (f *fakeSession) PutWorkflow(_ context.Context, wf *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, []string, error) {
	f.puts = append(f.puts, wf)

	actions := make([]string, 0, len(wf.Tasks))

	for _, task := range wf.Tasks {
		actions = append(actions, task.Action)
	}

	return &v1.CreateWorkflowVersionResponse{Id: "v"}, actions, nil
}

func (f *fakeSession) NewDurableTaskListener(opts ...client.DurableTaskListenerOpt) *client.DurableTaskListener {
	if f.stream == nil {
		f.stream = newFakeDurableStream()
	}

	stream := f.stream

	listener := client.NewDurableTaskListener(f.workerId, func(context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
		return stream, nil
	}, nil, opts...)

	f.listeners = append(f.listeners, listener)

	return listener
}

func (f *fakeSession) Close(_ ...client.CloseOpt) error {
	f.closed = true

	for _, l := range f.listeners {
		l.Stop()
	}

	return nil
}

// fakeOperatorClient connects sessions the engine authenticated as tenantId.
type fakeOperatorClient struct {
	connectErr error
	flushErr   error
	tenantId   string
	requests   []*client.ConnectOperatorRequest
	sessions   []*fakeSession
}

func (f *fakeOperatorClient) Connect(_ context.Context, req *client.ConnectOperatorRequest) (client.OperatorSession, error) {
	f.requests = append(f.requests, req)

	if f.connectErr != nil {
		err := f.connectErr
		f.connectErr = nil

		return nil, err
	}

	s := &fakeSession{workerId: "w" + req.Name, tenantId: f.tenantId, flushErr: f.flushErr}
	f.flushErr = nil
	f.sessions = append(f.sessions, s)

	return s, nil
}

type fakeClient struct {
	client.Client
	operator *fakeOperatorClient
	token    string
}

func (f *fakeClient) Operator() client.OperatorClient {
	return f.operator
}

func (f *fakeClient) Close() error { return nil }

type mapExchange map[uuid.UUID]string

func (m mapExchange) Token(_ context.Context, tenantId uuid.UUID) (string, error) {
	tok, ok := m[tenantId]

	if !ok {
		return "", ErrNoToken
	}

	return tok, nil
}

func TestLinkOpenCachesClientPerTenant(t *testing.T) {
	tenant := uuid.New()
	exchange := mapExchange{tenant: "tok-1"}

	var built []*fakeClient

	lnk := New(exchange, Options{
		OperatorName: "serverless",
		NewClient: func(token string) (client.Client, error) {
			c := &fakeClient{token: token, operator: &fakeOperatorClient{tenantId: tenant.String()}}
			built = append(built, c)

			return c, nil
		},
	})

	opts := link.OpenOpts{Actions: []string{"ns_svc:run"}, SlotConfig: map[string]int32{"default": 1}, Labels: map[string]interface{}{"k": "v"}}

	reg, err := lnk.Open(context.Background(), tenant, opts)
	require.NoError(t, err)
	assert.Equal(t, "wserverless", reg.WorkerId())

	reg2, err := lnk.Open(context.Background(), tenant, opts)
	require.NoError(t, err)

	require.Len(t, built, 1, "one client per tenant")
	require.Len(t, built[0].operator.requests, 2)

	req := built[0].operator.requests[1]
	assert.Equal(t, "serverless", req.Name)
	assert.Equal(t, opts.SlotConfig, req.SlotConfig)
	assert.Equal(t, "v", req.Labels["k"])
	assert.Len(t, req.Labels, 1, "only the core's labels are sent")

	// the initial action set is streamed and flushed before Open returns
	session := built[0].operator.sessions[1]
	assert.Equal(t, [][]string{opts.Actions}, session.added)
	assert.Equal(t, 1, session.flushes)

	// deltas and puts pass straight through to the session
	require.NoError(t, reg2.AddActions(context.Background(), []string{"ns_svc:other"}))
	require.NoError(t, reg2.RemoveActions(context.Background(), []string{"ns_svc:run"}))
	require.NoError(t, reg2.Flush(context.Background()))
	assert.Equal(t, [][]string{opts.Actions, {"ns_svc:other"}}, session.added)
	assert.Equal(t, [][]string{{"ns_svc:run"}}, session.removed)
	assert.Equal(t, 2, session.flushes)

	wf := &v1.CreateWorkflowVersionRequest{Name: "ns_wf", Tasks: []*v1.CreateTaskOpts{{ReadableId: "t", Action: "ns_svc:put"}}}
	derived, err := reg2.PutWorkflow(context.Background(), wf)
	require.NoError(t, err)
	assert.Equal(t, []string{"ns_svc:put"}, derived)
	require.Len(t, session.puts, 1)
	assert.Same(t, wf, session.puts[0])
	assert.Equal(t, 2, session.flushes, "a put does not touch the action set")

	ch, err := reg.OpenDurable(context.Background(), "task", 0)
	require.NoError(t, err)
	require.NoError(t, ch.Close())

	require.NoError(t, reg2.Close())
	assert.True(t, session.closed)

	// A rotated token rebuilds the client; a released tenant is evicted.
	exchange[tenant] = "tok-2"

	_, err = lnk.Open(context.Background(), tenant, opts)
	require.NoError(t, err)
	require.Len(t, built, 2)
	assert.Equal(t, "tok-2", built[1].token)

	lnk.ReleaseTenant(tenant)

	_, err = lnk.Open(context.Background(), tenant, opts)
	require.NoError(t, err)
	assert.Len(t, built, 3)

	_, err = lnk.Open(context.Background(), uuid.New(), opts)
	assert.ErrorIs(t, err, link.ErrNoToken)
	assert.Len(t, built, 3, "no client is built without a token")
}

func TestLinkRetriesOnceOnUnauthenticated(t *testing.T) {
	tenant := uuid.New()
	exchange := mapExchange{tenant: "tok"}

	var built []*fakeClient

	lnk := New(exchange, Options{
		NewClient: func(token string) (client.Client, error) {
			op := &fakeOperatorClient{tenantId: tenant.String()}

			if len(built) == 0 {
				op.connectErr = status.Error(codes.Unauthenticated, "expired")
			}

			c := &fakeClient{token: token, operator: op}
			built = append(built, c)

			return c, nil
		},
	})

	reg, err := lnk.Open(context.Background(), tenant, link.OpenOpts{})
	require.NoError(t, err)
	assert.NotNil(t, reg)
	assert.Len(t, built, 2, "the cached client is dropped and rebuilt after Unauthenticated")

	// Other errors are not retried.
	lnk2 := New(exchange, Options{
		NewClient: func(token string) (client.Client, error) {
			return &fakeClient{operator: &fakeOperatorClient{connectErr: errors.New("boom")}}, nil
		},
	})

	_, err = lnk2.Open(context.Background(), tenant, link.OpenOpts{})
	assert.Error(t, err)
}

// A registration whose initial actions the engine refused is closed rather than handed to
// the core with an empty action set; an empty initial set is not flushed at all.
func TestLinkOpenFlushesInitialActions(t *testing.T) {
	tenant := uuid.New()
	exchange := mapExchange{tenant: "tok"}

	op := &fakeOperatorClient{tenantId: tenant.String(), flushErr: errors.New("invalid action")}

	lnk := New(exchange, Options{
		NewClient: func(token string) (client.Client, error) {
			return &fakeClient{token: token, operator: op}, nil
		},
	})

	_, err := lnk.Open(context.Background(), tenant, link.OpenOpts{Actions: []string{"bad"}})
	require.ErrorContains(t, err, "invalid action")
	require.Len(t, op.sessions, 1)
	assert.True(t, op.sessions[0].closed, "the session is closed when the initial flush fails")

	reg, err := lnk.Open(context.Background(), tenant, link.OpenOpts{})
	require.NoError(t, err)
	assert.Empty(t, op.sessions[1].added)
	assert.Equal(t, 0, op.sessions[1].flushes)
	require.NoError(t, reg.Close())
}
