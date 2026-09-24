package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/admin"
	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	dispatcherconnect "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts/contractsconnect"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	grpcmiddleware "github.com/hatchet-dev/hatchet/pkg/grpc/middleware"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// These tests drive the server with google.golang.org/grpc clients, which is what every released
// SDK speaks, to hold the wire contract steady: codes, messages, error details, streaming
// semantics, compression, size limits and every TLS mode.

const (
	validToken   = "valid-token"
	limitedToken = "limited-token"
	maxMsgSize   = 64 * 1024

	// the "big-response" worker name gets a response larger than any socket buffer
	bigResponseSize = 24 * 1024 * 1024
	// the "ticking" worker id gets a slow server stream
	tickingMessages = 10
	tickingInterval = 100 * time.Millisecond
)

var (
	testTenantID   = uuid.MustParse("707d0855-80ab-4e1f-a156-f1c4546cbf52")
	validTokenID   = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	limitedTokenID = uuid.MustParse("22222222-2222-4222-8222-222222222222")
)

type tenantRepo struct {
	v1.TenantRepository
}

func (tenantRepo) GetTenantByID(_ context.Context, id uuid.UUID) (*sqlcv1.Tenant, error) {
	if id != testTenantID {
		return nil, pgx.ErrNoRows
	}

	return &sqlcv1.Tenant{ID: id}, nil
}

type fakeRepository struct {
	v1.Repository
}

func (fakeRepository) Tenant() v1.TenantRepository {
	return tenantRepo{}
}

type countingAlerter struct {
	n atomic.Int64
}

func (c *countingAlerter) SendAlert(context.Context, error, map[string]interface{}) {
	c.n.Add(1)
}

// fakeDispatcher implements a handful of Dispatcher methods; the rest stay unimplemented.
type fakeDispatcher struct {
	dispatcherconnect.UnimplementedDispatcherHandler

	mu          sync.Mutex
	lastCtx     context.Context
	lateSender  *rpcstream.Sender[dispatchercontracts.AssignedAction]
	bidiResults chan error
}

func (f *fakeDispatcher) Start() (func() error, error) {
	return func() error { return nil }, nil
}

func (f *fakeDispatcher) Register(ctx context.Context, req *dispatchercontracts.WorkerRegisterRequest) (*dispatchercontracts.WorkerRegisterResponse, error) {
	f.mu.Lock()
	f.lastCtx = ctx
	f.mu.Unlock()

	switch req.WorkerName {
	case "invalid":
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid worker name: %s", req.WorkerName))
	case "plain-error":
		return nil, errors.New("database exploded with secret details")
	case "no-rows":
		return nil, fmt.Errorf("lookup: %w", pgx.ErrNoRows)
	case "legacy-status":
		return nil, status.Error(codes.FailedPrecondition, "legacy status error")
	case "panic":
		panic("handler panicked")
	case "details", "bulk-details":
		// the same construction the admin services use for idempotency key collisions
		var msg proto.Message = &v1contracts.IdempotencyCollisionError{ExistingRunExternalId: "existing-run-id"}

		if req.WorkerName == "bulk-details" {
			msg = &v1contracts.BulkTriggerIdempotencyCollisionError{
				SuccessfulWorkflowRunExternalIds: []string{"new-run-id"},
			}
		}

		detail, err := connect.NewErrorDetail(msg)
		if err != nil {
			return nil, err
		}

		connectErr := connect.NewError(connect.CodeAlreadyExists, errors.New("idempotency key collision"))
		connectErr.AddDetail(detail)

		return nil, connectErr
	case "callback":
		grpcmiddleware.TriggerCallback(ctx)
	case "big-response":
		req.WorkerName = strings.Repeat("x", bigResponseSize)
	}

	tenant := ctx.Value("tenant").(*sqlcv1.Tenant) // nolint:staticcheck

	return &dispatchercontracts.WorkerRegisterResponse{
		TenantId:   tenant.ID.String(),
		WorkerId:   "worker-id",
		WorkerName: req.WorkerName,
	}, nil
}

func (f *fakeDispatcher) ListenV2(ctx context.Context, req *dispatchercontracts.WorkerListenRequest, stream *connect.ServerStream[dispatchercontracts.AssignedAction]) error {
	sender := rpcstream.NewSender[dispatchercontracts.AssignedAction](ctx, stream)
	defer sender.Close()

	f.mu.Lock()
	f.lateSender = sender
	f.mu.Unlock()

	if req.WorkerId == "ticking" {
		for i := range tickingMessages {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(tickingInterval):
			}

			if err := sender.Send(&dispatchercontracts.AssignedAction{ActionId: fmt.Sprintf("tick-%d", i)}); err != nil {
				return err
			}
		}

		return nil
	}

	if req.WorkerId == "until-cancelled" {
		if err := sender.Send(&dispatchercontracts.AssignedAction{ActionId: "first"}); err != nil {
			return err
		}

		<-ctx.Done()

		return nil
	}

	for i := range 5 {
		if err := sender.Send(&dispatchercontracts.AssignedAction{ActionId: fmt.Sprintf("action-%d", i)}); err != nil {
			return err
		}
	}

	if req.WorkerId == "fail-after" {
		return connect.NewError(connect.CodeNotFound, errors.New("worker not found"))
	}

	return nil
}

func (f *fakeDispatcher) SubscribeToWorkflowRuns(ctx context.Context, stream *connect.BidiStream[dispatchercontracts.SubscribeToWorkflowRunsRequest, dispatchercontracts.WorkflowRunEvent]) error {
	sender := rpcstream.NewSender[dispatchercontracts.WorkflowRunEvent](ctx, stream)
	defer sender.Close()

	for {
		req, err := stream.Receive()

		if err != nil {
			if f.bidiResults != nil {
				f.bidiResults <- err
			}

			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if err := sender.Send(&dispatchercontracts.WorkflowRunEvent{WorkflowRunId: req.WorkflowRunId}); err != nil {
			return err
		}
	}
}

// The server requires every service. These tests only call the dispatcher, so the others are
// typed placeholders whose methods are never reached.
type (
	fakeIngestor     struct{ ingestor.Ingestor }
	fakeAdmin        struct{ admin.AdminService }
	fakeAdminV1      struct{ adminv1.AdminService }
	fakeDispatcherV1 struct {
		v1connect.UnimplementedV1DispatcherHandler
	}
)

func withFakeServices() ServerOpt {
	return func(opts *ServerOpts) {
		opts.ingestor = fakeIngestor{}
		opts.admin = fakeAdmin{}
		opts.adminv1 = fakeAdminV1{}
		opts.dispatcherv1 = fakeDispatcherV1{}
	}
}

type testEnv struct {
	addr     string
	alerter  *countingAlerter
	disp     *fakeDispatcher
	cleanup  func() error
	callback atomic.Value
}

type transport struct {
	name       string
	serverTLS  func(t *testing.T, pki *testPKI) *tls.Config
	clientTLS  func(pki *testPKI, withClientCert bool) *tls.Config
	insecure   bool
	needsmTLS  bool
	clientCert bool
}

func transports() []transport {
	return []transport{
		{name: "insecure-h2c", insecure: true},
		{
			name: "tls",
			serverTLS: func(_ *testing.T, pki *testPKI) *tls.Config {
				// mirrors loaderutils.LoadServerTLSConfig for the "tls" strategy
				return &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pki.serverCert}, ClientAuth: tls.VerifyClientCertIfGiven, ClientCAs: pki.pool}
			},
			clientTLS: func(pki *testPKI, _ bool) *tls.Config {
				return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pki.pool, ServerName: "localhost"}
			},
		},
		{
			name: "mtls",
			serverTLS: func(_ *testing.T, pki *testPKI) *tls.Config {
				// mirrors loaderutils.LoadServerTLSConfig for the "mtls" strategy
				return &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pki.serverCert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pki.pool}
			},
			clientTLS: func(pki *testPKI, withClientCert bool) *tls.Config {
				cfg := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pki.pool, ServerName: "localhost"}

				if withClientCert {
					cfg.Certificates = []tls.Certificate{pki.clientCert}
				}

				return cfg
			},
			needsmTLS:  true,
			clientCert: true,
		},
	}
}

type testJWTManager struct{}

func (testJWTManager) GenerateTenantToken(context.Context, uuid.UUID, string, bool, *time.Time) (*token.Token, error) {
	return nil, errors.New("not implemented")
}

func (testJWTManager) ValidateTenantToken(_ context.Context, tok string) (uuid.UUID, uuid.UUID, error) {
	switch tok {
	case validToken:
		return testTenantID, validTokenID, nil
	case limitedToken:
		return testTenantID, limitedTokenID, nil
	case "unknown-tenant":
		return uuid.New(), uuid.New(), nil
	default:
		return uuid.Nil, uuid.Nil, errors.New("invalid token")
	}
}

func startTestServer(t *testing.T, tr transport, pki *testPKI, rateLimit float64, extra ...ServerOpt) *testEnv {
	t.Helper()

	l := zerolog.Nop()
	env := &testEnv{alerter: &countingAlerter{}, disp: &fakeDispatcher{}}

	sc := &server.ServerConfig{
		Layer: &database.Layer{V1: fakeRepository{}},
	}
	sc.Logger = &l
	sc.Auth.JWTManager = testJWTManager{}
	sc.Runtime.GRPCMaxMsgSize = maxMsgSize
	sc.Runtime.GRPCStaticStreamWindowSize = 10 * 1024 * 1024
	sc.Runtime.GRPCRateLimit = rateLimit

	sc.AddGRPCUnaryInterceptor(grpcmiddleware.CallbackInterceptor(&l, func(ctx context.Context) error {
		env.callback.Store(ctx.Value(constants.GRPCMethodKey))
		return nil
	}))

	port := freePort(t)
	env.addr = fmt.Sprintf("localhost:%d", port)

	opts := []ServerOpt{
		WithConfig(sc),
		WithLogger(&l),
		WithAlerter(env.alerter),
		WithDispatcher(env.disp),
		withFakeServices(),
		WithPort(port),
		WithBindAddress("127.0.0.1"),
		WithShutdownTimeout(2 * time.Second),
	}

	opts = append(opts, extra...)

	if tr.insecure {
		opts = append(opts, WithInsecure(), WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	} else {
		opts = append(opts, WithTLSConfig(tr.serverTLS(t, pki)))
	}

	s, err := NewServer(opts...)
	require.NoError(t, err)

	cleanup, err := s.Start()
	require.NoError(t, err)

	env.cleanup = cleanup

	t.Cleanup(func() { _ = cleanup() })

	return env
}

func (e *testEnv) dial(t *testing.T, tr transport, pki *testPKI, withClientCert bool, extra ...grpcgo.DialOption) *grpcgo.ClientConn {
	t.Helper()

	var creds credentials.TransportCredentials

	if tr.insecure {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(tr.clientTLS(pki, withClientCert))
	}

	// same dial options as pkg/client
	dialOpts := append([]grpcgo.DialOption{
		grpcgo.WithTransportCredentials(creds),
		grpcgo.WithKeepaliveParams(keepalive.ClientParameters{Time: 10 * time.Second, Timeout: 60 * time.Second, PermitWithoutStream: true}),
	}, extra...)

	conn, err := grpcgo.NewClient(e.addr, dialOpts...)
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func authCtx(t *testing.T, tok string, extra ...string) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	pairs := append([]string{"authorization", "Bearer " + tok}, extra...)

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(pairs...))
}

func TestGRPCClientCompatibility(t *testing.T) {
	pki := newTestPKI(t)

	for _, tr := range transports() {
		t.Run(tr.name, func(t *testing.T) {
			env := startTestServer(t, tr, pki, 0)
			conn := env.dial(t, tr, pki, tr.clientCert)
			client := dispatchercontracts.NewDispatcherClient(conn)

			t.Run("unary ok carries tenant from auth", func(t *testing.T) {
				res, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
				require.NoError(t, err)
				assert.Equal(t, testTenantID.String(), res.TenantId)
				assert.Equal(t, "w", res.WorkerName)
			})

			t.Run("gzip request and response", func(t *testing.T) {
				res, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: strings.Repeat("w", 4096)}, grpcgo.UseCompressor("gzip"))
				require.NoError(t, err)
				assert.Len(t, res.WorkerName, 4096)
			})

			t.Run("auth failures are unauthenticated with the historical message", func(t *testing.T) {
				for name, ctx := range map[string]context.Context{
					"no header":      context.Background(),
					"bad token":      authCtx(t, "nope"),
					"unknown tenant": authCtx(t, "unknown-tenant"),
					"wrong scheme":   metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Basic "+validToken)),
					"no scheme":      metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", validToken)),
				} {
					_, err := client.Register(ctx, &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
					require.Error(t, err, name)
					assert.Equal(t, codes.Unauthenticated, status.Code(err), name)
					assert.Equal(t, "invalid auth token", status.Convert(err).Message(), name)
				}

				// the scheme is case-insensitive: SDKs send both Bearer and bearer
				_, err := client.Register(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "bearer "+validToken)), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
				require.NoError(t, err)
			})

			t.Run("streams are authenticated too", func(t *testing.T) {
				stream, err := client.ListenV2(authCtx(t, "nope"), &dispatchercontracts.WorkerListenRequest{WorkerId: "w"})
				require.NoError(t, err)

				_, err = stream.Recv()
				assert.Equal(t, codes.Unauthenticated, status.Code(err))
			})

			t.Run("coded errors keep code and message", func(t *testing.T) {
				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "invalid"})
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				assert.Equal(t, "invalid worker name: invalid", status.Convert(err).Message())
			})

			t.Run("uncoded errors become internal, are alerted and do not leak", func(t *testing.T) {
				before := env.alerter.n.Load()

				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "plain-error"})
				assert.Equal(t, codes.Internal, status.Code(err))
				assert.Equal(t, "An internal error occurred.", status.Convert(err).Message())
				assert.Equal(t, before+1, env.alerter.n.Load())
			})

			t.Run("missing rows become not found without an alert", func(t *testing.T) {
				before := env.alerter.n.Load()

				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "no-rows"})
				assert.Equal(t, codes.NotFound, status.Code(err))
				assert.Equal(t, "not found", status.Convert(err).Message())
				assert.Equal(t, before, env.alerter.n.Load())
			})

			t.Run("status errors from extensions keep their code", func(t *testing.T) {
				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "legacy-status"})
				assert.Equal(t, codes.FailedPrecondition, status.Code(err))
				assert.Equal(t, "legacy status error", status.Convert(err).Message())
			})

			t.Run("panics become internal and the server survives", func(t *testing.T) {
				before := env.alerter.n.Load()

				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "panic"})
				assert.Equal(t, codes.Internal, status.Code(err))
				assert.Equal(t, "An internal error occurred", status.Convert(err).Message())
				assert.Equal(t, before+1, env.alerter.n.Load())

				_, err = client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
				require.NoError(t, err)
			})

			t.Run("idempotency collision details decode as the SDKs decode them", func(t *testing.T) {
				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "details"})
				st := status.Convert(err)
				assert.Equal(t, codes.AlreadyExists, st.Code())
				assert.Equal(t, "idempotency key collision", st.Message())
				require.Len(t, st.Details(), 1)

				single, ok := st.Details()[0].(*v1contracts.IdempotencyCollisionError)
				require.True(t, ok, "detail was %T", st.Details()[0])
				assert.Equal(t, "existing-run-id", single.ExistingRunExternalId)

				_, err = client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "bulk-details"})
				st = status.Convert(err)
				assert.Equal(t, codes.AlreadyExists, st.Code())
				require.Len(t, st.Details(), 1)

				bulk, ok := st.Details()[0].(*v1contracts.BulkTriggerIdempotencyCollisionError)
				require.True(t, ok, "detail was %T", st.Details()[0])
				assert.Equal(t, []string{"new-run-id"}, bulk.SuccessfulWorkflowRunExternalIds)
			})

			t.Run("unimplemented method and unknown service", func(t *testing.T) {
				_, err := client.GetVersion(authCtx(t, validToken), &dispatchercontracts.GetVersionRequest{})
				assert.Equal(t, codes.Unimplemented, status.Code(err))

				err = conn.Invoke(authCtx(t, validToken), "/NoSuchService/Push", &dispatchercontracts.GetVersionRequest{}, &dispatchercontracts.GetVersionResponse{})
				assert.Equal(t, codes.Unimplemented, status.Code(err))
			})

			t.Run("oversized request is resource exhausted", func(t *testing.T) {
				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: strings.Repeat("w", maxMsgSize+1)})
				assert.Equal(t, codes.ResourceExhausted, status.Code(err))
			})

			t.Run("deadline and source metadata reach the handler", func(t *testing.T) {
				_, err := client.Register(authCtx(t, validToken, analytics.SourceMetadataKey, "cli"), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
				require.NoError(t, err)

				env.disp.mu.Lock()
				handlerCtx := env.disp.lastCtx
				env.disp.mu.Unlock()

				_, hasDeadline := handlerCtx.Deadline()
				assert.True(t, hasDeadline, "grpc-timeout should become a context deadline")
				assert.Equal(t, analytics.Source("cli"), handlerCtx.Value(analytics.SourceKey))
				assert.Equal(t, validTokenID, handlerCtx.Value(analytics.APITokenIDKey))
				assert.Equal(t, testTenantID, handlerCtx.Value(analytics.TenantIDKey))
			})

			t.Run("extension unary interceptors run with the full method name", func(t *testing.T) {
				_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "callback"})
				require.NoError(t, err)
				assert.Equal(t, "/Dispatcher/Register", env.callback.Load())
			})

			t.Run("server stream delivers in order then ends cleanly", func(t *testing.T) {
				stream, err := client.ListenV2(authCtx(t, validToken), &dispatchercontracts.WorkerListenRequest{WorkerId: "w"})
				require.NoError(t, err)

				for i := range 5 {
					msg, err := stream.Recv()
					require.NoError(t, err)
					assert.Equal(t, fmt.Sprintf("action-%d", i), msg.ActionId)
				}

				_, err = stream.Recv()
				assert.ErrorIs(t, err, io.EOF)
			})

			t.Run("server stream error after messages arrives as a trailer status", func(t *testing.T) {
				stream, err := client.ListenV2(authCtx(t, validToken), &dispatchercontracts.WorkerListenRequest{WorkerId: "fail-after"})
				require.NoError(t, err)

				for range 5 {
					_, err := stream.Recv()
					require.NoError(t, err)
				}

				_, err = stream.Recv()
				assert.Equal(t, codes.NotFound, status.Code(err))
				assert.Equal(t, "worker not found", status.Convert(err).Message())
			})

			t.Run("send after the handler returned is an error, not a panic", func(t *testing.T) {
				ctx, cancel := context.WithCancel(authCtx(t, validToken))

				stream, err := client.ListenV2(ctx, &dispatchercontracts.WorkerListenRequest{WorkerId: "until-cancelled"})
				require.NoError(t, err)

				_, err = stream.Recv()
				require.NoError(t, err)

				env.disp.mu.Lock()
				sender := env.disp.lateSender
				env.disp.mu.Unlock()

				cancel()

				_, err = stream.Recv()
				assert.Equal(t, codes.Canceled, status.Code(err))

				require.Eventually(t, func() bool {
					return errors.Is(sender.Send(&dispatchercontracts.AssignedAction{ActionId: "late"}), rpcstream.ErrClosed)
				}, 5*time.Second, 10*time.Millisecond)
			})

			t.Run("bidi stream echoes and sees half close as EOF", func(t *testing.T) {
				env.disp.bidiResults = make(chan error, 1)

				stream, err := client.SubscribeToWorkflowRuns(authCtx(t, validToken))
				require.NoError(t, err)

				for i := range 3 {
					id := fmt.Sprintf("run-%d", i)
					require.NoError(t, stream.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: id}))

					msg, err := stream.Recv()
					require.NoError(t, err)
					assert.Equal(t, id, msg.WorkflowRunId)
				}

				require.NoError(t, stream.CloseSend())

				_, err = stream.Recv()
				assert.ErrorIs(t, err, io.EOF)
				assert.ErrorIs(t, <-env.disp.bidiResults, io.EOF)
			})

			t.Run("bidi stream sees client cancellation as canceled", func(t *testing.T) {
				env.disp.bidiResults = make(chan error, 1)

				ctx, cancel := context.WithCancel(authCtx(t, validToken))

				stream, err := client.SubscribeToWorkflowRuns(ctx)
				require.NoError(t, err)
				require.NoError(t, stream.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run"}))

				_, err = stream.Recv()
				require.NoError(t, err)

				cancel()

				select {
				case err := <-env.disp.bidiResults:
					assert.Equal(t, connect.CodeCanceled, connect.CodeOf(err), "got %v", err)
				case <-time.After(5 * time.Second):
					t.Fatal("handler did not observe the cancellation")
				}
			})

			if tr.needsmTLS {
				t.Run("mtls rejects a client without a certificate", func(t *testing.T) {
					noCert := env.dial(t, tr, pki, false)

					_, err := dispatchercontracts.NewDispatcherClient(noCert).Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
					assert.Equal(t, codes.Unavailable, status.Code(err))
				})
			}

			t.Run("connect and grpc-web protocols are served too, over HTTP/2", func(t *testing.T) {
				httpClient := newHTTPClient(tr, pki)
				scheme := "https"

				if tr.insecure {
					scheme = "http"
				}

				for name, opts := range map[string][]connect.ClientOption{
					"connect":  nil,
					"grpc-web": {connect.WithGRPCWeb()},
				} {
					c := dispatcherconnect.NewDispatcherClient(httpClient, scheme+"://"+env.addr, opts...)

					ctx, callInfo := connect.NewClientContext(context.Background())
					callInfo.RequestHeader().Set("Authorization", "Bearer "+validToken)

					res, err := c.Register(ctx, &dispatchercontracts.WorkerRegisterRequest{WorkerName: name})
					require.NoError(t, err, name)
					assert.Equal(t, name, res.WorkerName)
				}
			})
		})
	}
}

// responseEncodings records the grpc-encoding of every response a client receives.
type responseEncodings struct {
	mu   sync.Mutex
	seen []string
}

func (r *responseEncodings) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (r *responseEncodings) HandleRPC(_ context.Context, s stats.RPCStats) {
	if h, ok := s.(*stats.InHeader); ok {
		r.mu.Lock()
		r.seen = append(r.seen, h.Compression)
		r.mu.Unlock()
	}
}

func (r *responseEncodings) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (r *responseEncodings) HandleConn(context.Context, stats.ConnStats) {}

// grpc-go clients advertise gzip on every call. The response must be compressed only when the
// request was, as it always has been, so that assigned actions are not gzipped for workers that
// never asked for compression.
func TestResponseCompressionFollowsTheRequest(t *testing.T) {
	tr := transports()[0]
	env := startTestServer(t, tr, nil, 0)

	encodings := &responseEncodings{}
	client := dispatchercontracts.NewDispatcherClient(env.dial(t, tr, nil, false, grpcgo.WithStatsHandler(encodings)))
	req := &dispatchercontracts.WorkerRegisterRequest{WorkerName: strings.Repeat("w", 4096)}

	_, err := client.Register(authCtx(t, validToken), req)
	require.NoError(t, err)

	_, err = client.Register(authCtx(t, validToken), req, grpcgo.UseCompressor("gzip"))
	require.NoError(t, err)

	encodings.mu.Lock()
	defer encodings.mu.Unlock()

	assert.Equal(t, []string{"", "gzip"}, encodings.seen)
}

func TestRateLimitIsPerTokenAndResourceExhausted(t *testing.T) {
	tr := transports()[0]
	env := startTestServer(t, tr, nil, 1)
	client := dispatchercontracts.NewDispatcherClient(env.dial(t, tr, nil, false))

	// dispatcher calls get ten times the configured rate as burst
	var limited error

	for range 50 {
		if _, err := client.Register(authCtx(t, limitedToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"}); err != nil {
			limited = err
			break
		}
	}

	require.Error(t, limited)
	assert.Equal(t, codes.ResourceExhausted, status.Code(limited))
	// the message format is go-grpc-middleware's ratelimit interceptor's, which callers may match on
	assert.Equal(t, "/Dispatcher/Register is rejected by grpc_ratelimit middleware, please retry later. rpc error: code = ResourceExhausted desc = dispatcher rate limit exceeded", status.Convert(limited).Message())

	// another token has its own bucket
	_, err := client.Register(authCtx(t, validToken), &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
	require.NoError(t, err)
}

func TestShutdownForcesOpenStreamsClosedAfterTimeout(t *testing.T) {
	tr := transports()[0]
	env := startTestServer(t, tr, nil, 0)
	client := dispatchercontracts.NewDispatcherClient(env.dial(t, tr, nil, false))

	stream, err := client.ListenV2(authCtx(t, validToken), &dispatchercontracts.WorkerListenRequest{WorkerId: "until-cancelled"})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.NoError(t, err)

	start := time.Now()
	require.NoError(t, env.cleanup())
	assert.Less(t, time.Since(start), 5*time.Second)

	_, err = stream.Recv()
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

// TestIdleConnectionSurvivesClientKeepalive exists because a connection that has finished its
// preface or handshake must not keep the header read deadline: if it did, every long-lived
// stream on it would die when the timeout expires, on a transport that only pings. SDKs ping
// every ten seconds and hold streams open for hours.
func TestIdleConnectionSurvivesClientKeepalive(t *testing.T) {
	if testing.Short() {
		t.Skip("waits past the header timeout")
	}

	pki := newTestPKI(t)

	for _, tr := range transports() {
		t.Run(tr.name, func(t *testing.T) {
			// each transport waits past the header timeout; wait once, not three times
			t.Parallel()

			env := startTestServer(t, tr, pki, 0)
			client := dispatchercontracts.NewDispatcherClient(env.dial(t, tr, pki, tr.clientCert))

			// no deadline on the stream itself: it has to outlive the header timeout
			streamCtx, cancelStream := context.WithCancel(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+validToken)))
			defer cancelStream()

			stream, err := client.ListenV2(streamCtx, &dispatchercontracts.WorkerListenRequest{WorkerId: "until-cancelled"})
			require.NoError(t, err)

			_, err = stream.Recv()
			require.NoError(t, err)

			// the handler sends nothing more until the call is cancelled, so this Recv only
			// returns if the stream or its connection is torn down
			streamEnded := make(chan error, 1)

			go func() {
				_, err := stream.Recv()
				streamEnded <- err
			}()

			select {
			case err := <-streamEnded:
				t.Fatalf("stream ended while idle: %v", err)
			case <-time.After(readHeaderTimeout + 5*time.Second):
			}

			ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+validToken)), 5*time.Second)
			defer cancel()

			_, err = client.Register(ctx, &dispatchercontracts.WorkerRegisterRequest{WorkerName: "w"})
			require.NoError(t, err)
		})
	}
}

func newHTTPClient(tr transport, pki *testPKI) *http.Client {
	if tr.insecure {
		protocols := new(http.Protocols)
		protocols.SetUnencryptedHTTP2(true)

		return &http.Client{Transport: &http.Transport{Protocols: protocols}}
	}

	return &http.Client{Transport: &http.Transport{TLSClientConfig: tr.clientTLS(pki, tr.clientCert), ForceAttemptHTTP2: true}}
}

func freePort(t *testing.T) int {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer lis.Close()

	return lis.Addr().(*net.TCPAddr).Port
}

type testPKI struct {
	pool       *x509.CertPool
	serverCert tls.Certificate
	clientCert tls.Certificate
}

func newTestPKI(t *testing.T) *testPKI {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	leaf := func(serial int64, usage x509.ExtKeyUsage) tls.Certificate {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		template := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: "localhost"},
			DNSNames:     []string{"localhost"},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		}

		der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
		require.NoError(t, err)

		return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	}

	return &testPKI{
		pool:       pool,
		serverCert: leaf(2, x509.ExtKeyUsageServerAuth),
		clientCert: leaf(3, x509.ExtKeyUsageClientAuth),
	}
}
