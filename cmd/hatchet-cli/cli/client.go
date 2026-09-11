package cli

import (
	"fmt"
	"net"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
	clientconfig "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

// NewClientFromProfile creates a new Hatchet client from a profile configuration.
// It properly handles TLS settings, host/port, and authentication based on the profile.
func NewClientFromProfile(profile *profileconfig.Profile, logger *zerolog.Logger) (client.Client, error) { //nolint:staticcheck
	tlsStrategy := profile.TLSStrategy
	if tlsStrategy == "" {
		tlsStrategy = "tls"
	}

	// Construct a ClientConfigFile from the profile
	configFile := &clientconfig.ClientConfigFile{
		TenantId:  profile.TenantId,
		Token:     profile.Token,
		HostPort:  profile.GrpcHostPort,
		ServerURL: profile.ApiServerURL,
		TLS: clientconfig.ClientTLSConfigFile{
			Base: shared.TLSConfigFile{
				TLSStrategy: tlsStrategy,
			},
		},
	}

	// Build the gRPC TLS config from the profile alone. The profile is the
	// source of truth for a CLI connection, so we inject this explicitly to keep
	// ambient HATCHET_CLIENT_TLS_* env vars (e.g. a local mkcert root CA) from
	// silently overriding it and breaking TLS against unrelated endpoints.
	tlsServerName := profile.GrpcHostPort
	if host, _, err := net.SplitHostPort(profile.GrpcHostPort); err == nil {
		tlsServerName = host
	}
	tlsConfig, err := loaderutils.LoadClientTLSConfig(&configFile.TLS, tlsServerName)
	if err != nil {
		return nil, err
	}

	// Create client with the config file and logger
	return client.NewFromConfigFile( //nolint:staticcheck
		configFile,
		client.WithLogger(logger),       //nolint:staticcheck
		client.WithTLSConfig(tlsConfig), //nolint:staticcheck
		client.WithGRPCHeaders(map[string]string{
			analytics.SourceMetadataKey: string(analytics.SourceCLI),
		}),
	)
}

// newClientFromProfileOnly creates a Hatchet client whose connection settings
// come exclusively from the profile, falling back to the claims embedded in
// its token for fields the profile leaves empty. It never reads
// HATCHET_CLIENT_* environment variables and never panics on a malformed
// token: unlike NewClientFromProfile, construction bypasses the SDK's
// env-binding config loader entirely.
//
// The MCP server uses this factory so the profile the user granted stays
// authoritative for the token, tenant, and endpoints. The interactive CLI
// commands keep NewClientFromProfile, where environment overrides are part of
// the established UX.
func newClientFromProfileOnly(profile *profileconfig.Profile, logger *zerolog.Logger) (client.Client, error) { //nolint:staticcheck
	cfg, err := clientConfigFromProfile(profile)
	if err != nil {
		return nil, err
	}

	hatchetClient, err := client.NewFromConfig( //nolint:staticcheck
		cfg,
		client.WithLogger(logger),
		client.WithGRPCHeaders(map[string]string{
			analytics.SourceMetadataKey: string(analytics.SourceCLI),
		}),
	)
	if err != nil {
		return nil, err
	}

	// Fail closed if the effective client identity ever drifts from the
	// resolved configuration (the config above is built from the profile
	// alone, so this guards against regressions in client construction).
	if hatchetClient.TenantId() != cfg.TenantId {
		return nil, fmt.Errorf("client tenant does not match the resolved profile")
	}

	return hatchetClient, nil
}

// clientConfigFromProfile resolves a profile into a complete client
// configuration without consulting the environment. Missing connection fields
// fall back to the claims in the profile's own token, mirroring the SDK
// loader's resolution order minus its environment layer.
func clientConfigFromProfile(profile *profileconfig.Profile) (*clientconfig.ClientConfig, error) {
	if profile.Token == "" {
		return nil, fmt.Errorf("the profile has no API token")
	}

	// Pre-validate the token: this returns an error for malformed tokens where
	// the SDK's config loader would panic.
	tokenConf, err := loaderutils.GetConfFromJWT(profile.Token)
	if err != nil {
		return nil, fmt.Errorf("the profile's stored API token is not valid: %w", err)
	}

	tenantID := profile.TenantId
	if tenantID == "" {
		tenantID = tokenConf.TenantId
	}

	serverURL := profile.ApiServerURL
	if serverURL == "" {
		serverURL = tokenConf.ServerURL
	}

	grpcHostPort := profile.GrpcHostPort
	if grpcHostPort == "" {
		grpcHostPort = tokenConf.GrpcBroadcastAddress
	}

	if serverURL == "" || grpcHostPort == "" {
		return nil, fmt.Errorf("the profile does not specify the server URL or gRPC address")
	}

	tlsStrategy := profile.TLSStrategy
	if tlsStrategy == "" {
		tlsStrategy = "tls"
	}

	tlsServerName := grpcHostPort
	if host, _, splitErr := net.SplitHostPort(grpcHostPort); splitErr == nil {
		tlsServerName = host
	}

	tlsConfig, err := loaderutils.LoadClientTLSConfig(&clientconfig.ClientTLSConfigFile{
		Base: shared.TLSConfigFile{TLSStrategy: tlsStrategy},
	}, tlsServerName)
	if err != nil {
		return nil, err
	}

	return &clientconfig.ClientConfig{
		TenantId:             tenantID,
		Token:                profile.Token,
		ServerURL:            serverURL,
		GRPCBroadcastAddress: grpcHostPort,
		TLSConfig:            tlsConfig,
		Logger:               shared.LoggerConfigFile{Level: "warn", Format: "text"},
	}, nil
}

// clientCmdConfig holds CLI flag values used to create a Hatchet client.
type clientCmdConfig struct {
	Profile string
}

// readClientCmdConfig reads client-related flags from a command into a config struct.
func readClientCmdConfig(cmd *cobra.Command) clientCmdConfig {
	profile, _ := cmd.Flags().GetString("profile")
	return clientCmdConfig{Profile: profile}
}

// clientFromCmd selects a profile and returns a Hatchet client.
// The profile is read from the --profile flag if present, otherwise selected interactively.
func clientFromCmd(cmd *cobra.Command) (string, client.Client) { //nolint:staticcheck
	cfg := readClientCmdConfig(cmd)

	var selectedProfile string
	if cfg.Profile != "" {
		selectedProfile = cfg.Profile
	} else {
		selectedProfile = selectProfileForm(true)
		if selectedProfile == "" {
			selectedProfile = handleNoProfiles(cmd)
			if selectedProfile == "" {
				cli.Logger.Fatal("no profile selected or created")
			}
		}
	}

	profile, err := cli.Profiles.GetProfile(selectedProfile)
	if err != nil {
		cli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
	}

	nopLogger := zerolog.Nop()
	hatchetClient, err := NewClientFromProfile(profile, &nopLogger)
	if err != nil {
		cli.Logger.Fatalf("could not create Hatchet client: %v", err)
	}

	return selectedProfile, hatchetClient
}

// clientTenantUUID parses and returns the tenant UUID from the client configuration.
func clientTenantUUID(hatchetClient client.Client) openapi_types.UUID { //nolint:staticcheck
	parsed, err := uuid.Parse(hatchetClient.TenantId())
	if err != nil {
		cli.Logger.Fatalf("invalid tenant ID: %v", err)
	}
	return parsed
}
