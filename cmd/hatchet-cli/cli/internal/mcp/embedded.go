package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
)

// defaultEmbeddedAPIURL is where the embedded engine binds its API when the
// default port is free (hatchet-embedded's DefaultAPIPort). This mirrors the
// probe used by `hatchet embedded-ui`.
const defaultEmbeddedAPIURL = "http://localhost:28243"

// handshakeEnv is the environment variable the embedded SDKs export so child
// processes can attach to an already-running embedded engine. It contains the
// sidecar handshake JSON.
const handshakeEnv = "HATCHET_EMBEDDED_HANDSHAKE"

const embeddedProbeTimeout = 3 * time.Second

// EmbeddedDetection describes whether a running embedded Hatchet instance was
// found and whether it can be used as an implicit profile.
type EmbeddedDetection struct {
	// Detected is true when a live embedded API server was found.
	Detected bool

	// APIURL is the embedded API server that answered the probe.
	APIURL string

	// Profile is the implicit connection profile, or nil when the instance was
	// detected but no client token could be discovered.
	Profile *cliconfig.Profile

	// Note explains detection limitations (e.g. a detected instance without a
	// discoverable token).
	Note string
}

// Usable reports whether the detection produced a connectable profile.
func (d *EmbeddedDetection) Usable() bool {
	return d != nil && d.Detected && d.Profile != nil
}

// embeddedHandshake mirrors the handshake JSON written by the hatchet-embedded
// sidecar and exported via HATCHET_EMBEDDED_HANDSHAKE.
type embeddedHandshake struct {
	Token       string `json:"token"`
	TenantID    string `json:"tenant_id"`
	GRPCAddress string `json:"grpc_address"`
	APIURL      string `json:"api_url"`
}

// DetectEmbedded looks for a running embedded Hatchet instance.
//
// The embedded engine leaves no reliable filesystem artifact behind (the
// sidecar handshake file is written to a caller-chosen temp path and deleted
// once read), so detection works as follows, in order:
//
//  1. HATCHET_EMBEDDED_HANDSHAKE: the handshake JSON exported by the embedded
//     SDKs for child processes. Verified live via its API URL.
//  2. HATCHET_CLIENT_TOKEN: the token env var set by the Go in-process embed
//     path; endpoints come from the token's claims. Verified live.
//  3. A probe of the default embedded API port (28243). This confirms an
//     instance is running but yields no token, so the instance is reported as
//     detected-but-unusable.
//
// In every case the API's /api/v1/meta must report embedded: true, so a token
// for a non-embedded deployment can never sneak past the grant checks.
func DetectEmbedded(ctx context.Context) *EmbeddedDetection {
	return detectEmbedded(ctx, defaultEmbeddedAPIURL, os.Getenv)
}

func detectEmbedded(ctx context.Context, defaultAPIURL string, getenv func(string) string) *EmbeddedDetection {
	if raw := getenv(handshakeEnv); raw != "" {
		var handshake embeddedHandshake
		if err := json.Unmarshal([]byte(raw), &handshake); err == nil && handshake.Token != "" && handshake.APIURL != "" {
			if isEmbeddedAPI(ctx, handshake.APIURL) {
				return &EmbeddedDetection{
					Detected: true,
					APIURL:   handshake.APIURL,
					Profile: &cliconfig.Profile{
						TenantId:     handshake.TenantID,
						Name:         EmbeddedProfileName,
						Token:        handshake.Token,
						ApiServerURL: handshake.APIURL,
						GrpcHostPort: handshake.GRPCAddress,
						TLSStrategy:  "none",
					},
				}
			}
		}
	}

	if token := getenv("HATCHET_CLIENT_TOKEN"); token != "" {
		if conf, err := loaderutils.GetConfFromJWT(token); err == nil && conf.ServerURL != "" {
			if isEmbeddedAPI(ctx, conf.ServerURL) {
				return &EmbeddedDetection{
					Detected: true,
					APIURL:   conf.ServerURL,
					Profile: &cliconfig.Profile{
						TenantId:     conf.TenantId,
						Name:         EmbeddedProfileName,
						Token:        token,
						ApiServerURL: conf.ServerURL,
						GrpcHostPort: conf.GrpcBroadcastAddress,
						ExpiresAt:    conf.ExpiresAt,
						TLSStrategy:  "none",
					},
				}
			}
		}
	}

	if isEmbeddedAPI(ctx, defaultAPIURL) {
		return &EmbeddedDetection{
			Detected: true,
			APIURL:   defaultAPIURL,
			Note: "an embedded API server is running but its client token is not discoverable from outside the embedding process; " +
				"export HATCHET_CLIENT_TOKEN (or HATCHET_EMBEDDED_HANDSHAKE) from the process that started it, or configure a profile",
		}
	}

	return &EmbeddedDetection{}
}

// isEmbeddedAPI reports whether apiURL is a live Hatchet API server running in
// embedded mode, via the unauthenticated /api/v1/meta endpoint.
func isEmbeddedAPI(ctx context.Context, apiURL string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, embeddedProbeTimeout)
	defer cancel()

	client, err := rest.NewClientWithResponses(apiURL, rest.WithHTTPClient(&http.Client{Timeout: embeddedProbeTimeout}))
	if err != nil {
		return false
	}

	resp, err := client.MetadataGetWithResponse(probeCtx)
	if err != nil || resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return false
	}

	return resp.JSON200.Embedded != nil && *resp.JSON200.Embedded
}
