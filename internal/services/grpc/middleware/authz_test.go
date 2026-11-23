package middleware

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

func TestGRPCAuthZ(t *testing.T) {
	l := zerolog.Nop()
	a, err := NewAuthZ(&l)
	require.NoError(t, err)

	tenant := uuid.New()
	token := func(readOnly bool, caps ...string) context.Context {
		return pkgauthz.WithPrincipal(context.Background(), pkgauthz.TokenPrincipal{
			Kind: pkgauthz.TenantAPIToken, TenantID: &tenant, ReadOnly: readOnly, Capabilities: caps,
		})
	}

	cases := []struct {
		name      string
		ctx       context.Context
		procedure string
		code      connect.Code
		message   string
	}{
		{"all reaches anything", token(false, pkgauthz.AllCapabilities), "/v1.V1Streams/Reset", 0, ""},
		{"worker can listen", token(false, "worker"), "/Dispatcher/ListenV2", 0, ""},
		{"worker inherits triggering", token(false, "worker"), "/v1.AdminService/TriggerWorkflowRun", 0, ""},
		{"worker cannot cancel", token(false, "worker"), "/v1.AdminService/CancelTasks", connect.CodePermissionDenied, "token does not have a capability that allows /v1.AdminService/CancelTasks"},
		{"any held capability is enough", token(false, "events:push", "runs:manage"), "/v1.AdminService/CancelTasks", 0, ""},
		{"read-only is denied even with all", token(true, pkgauthz.AllCapabilities), "/Dispatcher/ListenV2", connect.CodePermissionDenied, "read-only tokens cannot use gRPC"},
		{"no principal is an internal error", context.Background(), "/Dispatcher/ListenV2", connect.CodeInternal, "An internal error occurred."},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := a.Authorize(c.ctx, c.procedure)
			if c.code == 0 {
				require.NoError(t, err)
				return
			}

			require.Equal(t, c.code, connect.CodeOf(err))
			require.Equal(t, c.message, err.(*connect.Error).Message())
		})
	}
}

func TestGateRejectsMissingCapabilityBeforeTheHandler(t *testing.T) {
	ts := startTestServerWithRepo(t, nil, 1000, 1000, fakeRepository{capabilities: []string{"events:push"}})

	_, err := ts.client().Register(authContext(testToken), &contracts.WorkerRegisterRequest{WorkerName: "w"})
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Zero(t, ts.dispatcher.calls.Load())
}
