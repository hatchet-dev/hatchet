package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"

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
	Example: `  # Create a new profile interactively (prompts for token)
  hatchet profile add

  # Create a new profile with a specific name
  hatchet profile add --name production --token <your-token>`,
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

		fmt.Println(profileActionView("added", name))
	},
}

// profileRemoveCmd represents the profile remove command
var profileRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a profile",
	Long:  `Remove an existing profile from the configuration.`,
	Example: `  # Remove a profile interactively (shows selection menu)
  hatchet profile remove

  # Remove a specific profile by name
  hatchet profile remove --name production`,
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

		fmt.Println(profileActionView("removed", name))
	},
}

// profileListCmd represents the profile list command
var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  `List all configured profiles.`,
	Example: `  # List all configured profiles
  hatchet profile list`,
	Run: func(cmd *cobra.Command, args []string) {
		profileNames := cli.ListProfiles()

		fmt.Println(profileListView(profileNames))
	},
}

// profileShowCmd represents the profile show command
var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show details of a specific profile",
	Long:  `Show the details of a specific profile including address and token.`,
	Example: `  # Show profile details interactively (shows selection menu)
  hatchet profile show

  # Show a specific profile with masked token
  hatchet profile show --name production

  # Show a specific profile with full token visible
  hatchet profile show --name production --show-token`,
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

		fmt.Println(profileView(name, profile.Token, profile.ApiServerURL, profile.GrpcHostPort, showToken))
	},
}

// profileUpdateCmd represents the profile update command
var profileUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing profile",
	Long:  `Update the address and/or token of an existing profile.`,
	Example: `  # Update a profile interactively (prompts for profile and token)
  hatchet profile update

  # Update a specific profile with a new token
  hatchet profile update --name production --token <new-token>`,
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

		fmt.Println(profileActionView("updated", name))
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
	).WithTheme(styles.HatchetTheme())

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
		Name:         name,
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
				Title("Select a profile:").
				Options(profileNames...).
				Value(&selectedName),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()

	if err != nil {
		cli.Logger.Fatalf("could not run profile remove form: %v", err)
	}

	return selectedName
}

// profileView renders a profile view with details
func profileView(name, token, apiURL, grpcHost string, showToken bool) string {
	var lines []string

	// Title
	lines = append(lines, styles.Section("Profile: "+name))
	lines = append(lines, "")

	// Token
	tokenValue := token
	if !showToken && len(token) > 4 {
		tokenValue = token[:4] + "****"
	}
	lines = append(lines, styles.KeyValue("Token", tokenValue))

	// URLs
	lines = append(lines, styles.KeyValue("API Server URL", apiURL))
	lines = append(lines, styles.KeyValue("gRPC Host Port", grpcHost))

	return styles.InfoBox.Render(strings.Join(lines, "\n"))
}

// profileListView renders a list of profiles
func profileListView(profiles []string) string {
	if len(profiles) == 0 {
		return styles.InfoMessage("No profiles configured")
	}

	var lines []string
	lines = append(lines, styles.Section("Configured Profiles"))

	for _, profile := range profiles {
		lines = append(lines, styles.ListItem.Render(styles.Accent.Render("• ")+profile))
	}

	return strings.Join(lines, "\n")
}

// profileActionView renders a profile action result (add, update, remove)
func profileActionView(action, profileName string) string {
	var message string
	switch action {
	case "added":
		message = fmt.Sprintf("Profile '%s' added successfully", profileName)
	case "updated":
		message = fmt.Sprintf("Profile '%s' updated successfully", profileName)
	case "removed":
		message = fmt.Sprintf("Profile '%s' removed successfully", profileName)
	default:
		message = fmt.Sprintf("Profile '%s' %s", profileName, action)
	}

	return styles.SuccessMessage(message)
}
