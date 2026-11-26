package middleware

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

func TestStoredTokenCapabilitiesGateCalls(t *testing.T) {
	cases := []struct {
		name    string
		repo    fakeRepository
		message string // empty when the call reaches the handler
	}{
		{"token from before capabilities", fakeRepository{}, ""},
		{"worker", fakeRepository{capabilities: []string{"worker"}}, ""},
		{"any held capability is enough", fakeRepository{capabilities: []string{"events:push", "worker"}}, ""},
		{"missing capability", fakeRepository{capabilities: []string{"events:push"}}, "token does not have a capability that allows /Dispatcher/Register"},
		{"read-only token", fakeRepository{readOnly: true}, "read-only tokens cannot use gRPC"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := startTestServerWithRepo(t, nil, 1000, 1000, c.repo)

			_, err := ts.client().Register(authContext(testToken), &contracts.WorkerRegisterRequest{WorkerName: "w"})
			if c.message == "" {
				require.NoError(t, err)
				require.EqualValues(t, 1, ts.dispatcher.calls.Load())
				return
			}

			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Equal(t, c.message, err.(*connect.Error).Message())
			require.Zero(t, ts.dispatcher.calls.Load(), "a denied call never reaches the handler")
		})
	}
}
