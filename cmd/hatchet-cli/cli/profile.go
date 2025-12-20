package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// profileCmd represents the profile command
var profileCmd = &cobra.Command{
	Use:     "profile",
	Aliases: []string{"profiles"},
	Short:   "Manage profiles for different Hatchet environments",
	Long:    `Manage profiles that store connection information (address and token) for different Hatchet environments.`,
	Example: `
# Create a new profile (interactive)
hatchet profile add

# Create a new profile (non-interactive)
hatchet profile add --name [name] --token [token]

# Remove a profile (interactive)
hatchet profile remove

# Remove a profile (non-interactive)
hatchet profile remove --name [name]

# List all profiles
hatchet profile list

# Show details of a specific profile
hatchet profile show --name [name] [--show-token]

# Update an existing profile (interactive)
hatchet profile update

# Update an existing profile (non-interactive)
hatchet profile update --name [name] --token [token]
`,
}

// profileAddCmd represents the profile add command
var profileAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new profile",
	Long:  `Add a new profile with an address and token for a Hatchet environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		token, _ := cmd.Flags().GetString("token")

		if token == "" {
			token = getApiTokenForm()
		}

		profile, err := getProfileFromToken(cmd, token, name)

		if err != nil {
			cli.Logger.Fatalf("could not get profile from token: %v", err)
		}

		err = cli.AddProfile(name, profile)

		if err != nil {
			cli.Logger.Fatalf("could not add profile: %v", err)
		}

		cli.Logger.Infof("Profile '%s' added successfully", name)
	},
}

// profileRemoveCmd represents the profile remove command
var profileRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a profile",
	Long:  `Remove an existing profile from the configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")

		// if there's no name, list profiles and ask user to select one
		if name == "" {
			name = selectProfileForm()

			// if still no name, exit
			if name == "" {
				cli.Logger.Info("No profile selected, exiting")
				return
			}
		}

		err := cli.RemoveProfile(name)
		if err != nil {
			cli.Logger.Fatalf("could not remove profile: %v", err)
		}

		cli.Logger.Infof("Profile '%s' removed successfully", name)
	},
}

// profileListCmd represents the profile list command
var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  `List all configured profiles.`,
	Run: func(cmd *cobra.Command, args []string) {
		profiles := cli.ListProfiles()

		if len(profiles) == 0 {
			cli.Logger.Info("No profiles configured")
			return
		}

		cli.Logger.Info("Configured profiles:", "profiles", profiles)
	},
}

// profileShowCmd represents the profile show command
var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show details of a specific profile",
	Long:  `Show the details of a specific profile including address and token.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		showToken, _ := cmd.Flags().GetBool("show-token")

		if name == "" {
			name = selectProfileForm()
		}

		profile, err := cli.GetProfile(name)
		if err != nil {
			cli.Logger.Fatalf("could not get profile: %v", err)
		}

		fmt.Printf("Profile: %s\n", name)
		if showToken {
			fmt.Printf("  Token: %s\n", profile.Token)
		} else {
			maskedToken := "****"
			if len(profile.Token) > 4 {
				maskedToken = profile.Token[:4] + "****"
			}
			fmt.Printf("  Token: %s\n", maskedToken)
		}

		fmt.Printf("  API Server URL: %s\n", profile.ApiServerURL)
		fmt.Printf("  gRPC Host Port: %s\n", profile.GrpcHostPort)
	},
}

// profileUpdateCmd represents the profile update command
var profileUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing profile",
	Long:  `Update the address and/or token of an existing profile.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		token, _ := cmd.Flags().GetString("token")

		if name == "" {
			name = selectProfileForm()
		}

		if token == "" {
			token = getApiTokenForm()
		}

		profile, err := getProfileFromToken(cmd, token, name)

		if err != nil {
			cli.Logger.Fatalf("could not get profile from token: %v", err)
		}

		err = cli.UpdateProfile(name, profile)

		if err != nil {
			cli.Logger.Fatalf("could not update profile: %v", err)
		}

		cli.Logger.Infof("Profile '%s' updated successfully", name)
	},
}

func init() {
	// Add profile command to root
	rootCmd.AddCommand(profileCmd)

	// Add subcommands to profile
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileRemoveCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileUpdateCmd)

	// Add flags to profile add command
	profileAddCmd.Flags().StringP("token", "t", "", "Authentication token (prompted if not provided)")
	profileAddCmd.Flags().StringP("name", "n", "", "Name of the profile (defaults to tenant name)")

	// Add flags to profile remove command
	profileRemoveCmd.Flags().StringP("name", "n", "", "Name of the profile to remove (prompted if not provided)")

	// Add flags to profile show command
	profileShowCmd.Flags().Bool("show-token", false, "Show the full token (default: masked)")

	// Add flags to profile update command
	profileUpdateCmd.Flags().StringP("token", "t", "", "Authentication token (prompted if not provided)")
	profileUpdateCmd.Flags().StringP("name", "n", "", "Name of the profile to update (prompted if not provided)")
}

func getApiTokenForm() string {
	// ask for a token
	var resp string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your Hatchet API token").
				Placeholder("API Token").
				Value(&resp).
				EchoMode(huh.EchoModePassword),
		),
	)

	err := form.Run()

	if err != nil {
		cli.Logger.Fatalf("could not run profile add form: %v", err)
	}

	return resp
}

func getProfileFromToken(cmd *cobra.Command, token, nameOverride string) (*cliconfig.Profile, error) {
	name := nameOverride
	parsedTokenConf, err := loaderutils.GetConfFromJWT(token)

	if err != nil {
		cli.Logger.Fatalf("invalid token provided: %v", err)
	}

	// make sure client can connect with token
	nopLogger := zerolog.Nop()
	client, err := client.New(client.WithToken(token), client.WithLogger(&nopLogger)) // TODO: improve logging here

	if err != nil {
		cli.Logger.Fatalf("could not create client with provided token: %v", err)
	}

	tenant, err := client.API().TenantGetWithResponse(cmd.Context(), uuid.MustParse(client.TenantId()))

	if err != nil {
		cli.Logger.Fatalf("could not connect to Hatchet server with provided token: %v", err)
	} else if tenant.JSON200 == nil {
		cli.Logger.Warnf("got a %d response from Hatchet server running at %s", tenant.StatusCode(), parsedTokenConf.ServerURL)
		cli.Logger.Warnf("response body: %s", string(tenant.Body))
		cli.Logger.Fatalf("could not connect to Hatchet server with provided token: invalid response. Ensure the Hatchet server is running at %s", parsedTokenConf.ServerURL)
	}

	if name == "" {
		name = tenant.JSON200.Name
	}

	return &cliconfig.Profile{
		TenantId:     client.TenantId(),
		Name:         tenant.JSON200.Name,
		Token:        token,
		ApiServerURL: parsedTokenConf.ServerURL,
		GrpcHostPort: parsedTokenConf.GrpcBroadcastAddress,
		ExpiresAt:    parsedTokenConf.ExpiresAt,
	}, nil
}

func selectProfileForm() string {
	profiles := cli.GetProfiles()

	if len(profiles) == 0 {
		cli.Logger.Info("No profiles configured")
		return ""
	}

	profileNames := make([]huh.Option[string], 0, len(profiles))
	for profileName := range profiles {
		profileNames = append(profileNames, huh.Option[string]{
			Key:   profileName,
			Value: profileName,
		})
	}

	var selectedName string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a profile to remove").
				Options(profileNames...).
				Value(&selectedName),
		),
	)

	err := form.Run()

	if err != nil {
		cli.Logger.Fatalf("could not run profile remove form: %v", err)
	}

	return selectedName
}
