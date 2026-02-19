package cli

import (
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
	clientconfig "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

// NewClientFromProfile creates a new Hatchet client from a profile configuration.
// It properly handles TLS settings, host/port, and authentication based on the profile.
func NewClientFromProfile(profile *profileconfig.Profile, logger *zerolog.Logger) (client.Client, error) { //nolint:staticcheck
	// Construct a ClientConfigFile from the profile
	configFile := &clientconfig.ClientConfigFile{
		TenantId:  profile.TenantId,
		Token:     profile.Token,
		HostPort:  profile.GrpcHostPort,
		ServerURL: profile.ApiServerURL,
		TLS: clientconfig.ClientTLSConfigFile{
			Base: shared.TLSConfigFile{
				TLSStrategy: profile.TLSStrategy,
			},
		},
	}

	// Create client with the config file and logger
	return client.NewFromConfigFile(configFile, client.WithLogger(logger)) //nolint:staticcheck
}

// clientFromCmd selects a profile and returns a Hatchet client.
// The profile is read from the --profile flag if present, otherwise selected interactively.
func clientFromCmd(cmd *cobra.Command) (string, client.Client) { //nolint:staticcheck
	profileFlag, _ := cmd.Flags().GetString("profile")

	var selectedProfile string
	if profileFlag != "" {
		selectedProfile = profileFlag
	} else {
		selectedProfile = selectProfileForm(true)
		if selectedProfile == "" {
			selectedProfile = handleNoProfiles(cmd)
			if selectedProfile == "" {
				cli.Logger.Fatal("no profile selected or created")
			}
		}
	}

	profile, err := cli.GetProfile(selectedProfile)
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
