package hostgrpc

import (
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // see below
)

// The legacy pkg/client is deprecated as a whole, so every mention of it is flagged. This host
// still needs two things only it has: the dial that turns a token and the HATCHET_CLIENT_*
// environment into a connection, and the durable task listener the Go SDK worker shares. This
// file is the one place the package is named; the rest of hostgrpc uses these aliases.

type (
	engineClient        = client.Client              //nolint:staticcheck // see above
	durableTaskListener = client.DurableTaskListener //nolint:staticcheck // see above
	pendingAckKey       = client.PendingAckKey       //nolint:staticcheck // see above
	pendingCallbackKey  = client.PendingCallbackKey  //nolint:staticcheck // see above
	nonDeterminismError = client.NonDeterminismError //nolint:staticcheck // see above
)

var newDurableTaskListener = client.NewDurableTaskListener //nolint:staticcheck // see above

// dialEngine builds the engine client for one token, deriving the gRPC address and TLS
// settings from the token's claims and the environment the way the SDK does.
func dialEngine(token string, l *zerolog.Logger) (engineClient, error) {
	return client.New(client.WithToken(token), client.WithLogger(l)) //nolint:staticcheck // see above
}
