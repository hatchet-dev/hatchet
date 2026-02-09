package cli

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
	Aliases: []string{"profiles", "prof"},
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

# Set a profile as default (interactive)
hatchet profile set-default

# Set a profile as default (non-interactive)
hatchet profile set-default --name [name]

# Unset the default profile
hatchet profile unset-default
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
		tlsStrategy, _ := cmd.Flags().GetString("tls-strategy")

		if token == "" {
			token = getApiTokenForm()
		}

		profile, err := getProfileFromToken(cmd, token, name, tlsStrategy)

		if err != nil {
			cli.Logger.Fatalf("could not get profile from token: %v", err)
		}

		// If name is not provided via flag, prompt for it with tenant name as default
		if name == "" {
			candidateName := profile.Name

			nameForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title(fmt.Sprintf("Enter a name for this profile (default: %s)", candidateName)).
						Placeholder(candidateName).
						Value(&name),
				),
			).WithTheme(styles.HatchetTheme())

			err = nameForm.Run()
			if err != nil {
				cli.Logger.Fatalf("could not run name input form: %v", err)
			}

			// If user leaves it empty, use the candidate name
			if name == "" {
				name = candidateName
			}
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
			name = selectProfileForm(false)

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
			name = selectProfileForm(false)
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
		tlsStrategy, _ := cmd.Flags().GetString("tls-strategy")

		if name == "" {
			name = selectProfileForm(false)
		}

		if token == "" {
			token = getApiTokenForm()
		}

		profile, err := getProfileFromToken(cmd, token, name, tlsStrategy)

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

// profileSetDefaultCmd represents the profile set-default command
var profileSetDefaultCmd = &cobra.Command{
	Use:   "set-default",
	Short: "Set a profile as the default",
	Long:  `Set a profile as the default. The default profile will be automatically used when no profile is specified.`,
	Example: `  # Set a profile as default interactively (shows selection menu)
  hatchet profile set-default

  # Set a specific profile as default
  hatchet profile set-default --name production`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")

		if name == "" {
			name = selectProfileForm(false)

			if name == "" {
				cli.Logger.Info("No profile selected, exiting")
				return
			}
		}

		err := cli.SetDefaultProfile(name)
		if err != nil {
			cli.Logger.Fatalf("could not set default profile: %v", err)
		}

		fmt.Println(profileActionView("set-as-default", name))
	},
}

// profileUnsetDefaultCmd represents the profile unset-default command
var profileUnsetDefaultCmd = &cobra.Command{
	Use:   "unset-default",
	Short: "Unset the default profile",
	Long:  `Unset the default profile. After unsetting, you will be prompted to select a profile when running commands.`,
	Example: `  # Unset the default profile
  hatchet profile unset-default`,
	Run: func(cmd *cobra.Command, args []string) {
		currentDefault := cli.GetDefaultProfile()

		if currentDefault == "" {
			fmt.Println(styles.InfoMessage("No default profile is currently set"))
			return
		}

		err := cli.ClearDefaultProfile()
		if err != nil {
			cli.Logger.Fatalf("could not unset default profile: %v", err)
		}

		fmt.Println(profileActionView("unset-default", currentDefault))
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
	profileCmd.AddCommand(profileSetDefaultCmd)
	profileCmd.AddCommand(profileUnsetDefaultCmd)

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

	// Add flags to profile set-default command
	profileSetDefaultCmd.Flags().StringP("name", "n", "", "Name of the profile to set as default (prompted if not provided)")
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

	return parseToken(resp)
}

// parseToken extracts the token from various formats that users might paste
func parseToken(input string) string {
	input = strings.TrimSpace(input)

	// Remove "export " prefix if present
	input = strings.TrimPrefix(input, "export ")
	input = strings.TrimSpace(input)

	// Check for HATCHET_CLIENT_TOKEN= or HATCHET_CLIENT_TOKEN: prefix
	if strings.HasPrefix(input, "HATCHET_CLIENT_TOKEN=") {
		input = strings.TrimPrefix(input, "HATCHET_CLIENT_TOKEN=")
	} else if strings.HasPrefix(input, "HATCHET_CLIENT_TOKEN:") {
		input = strings.TrimPrefix(input, "HATCHET_CLIENT_TOKEN:")
	}

	input = strings.TrimSpace(input)

	// Remove surrounding quotes if present
	if len(input) >= 2 {
		if (strings.HasPrefix(input, "\"") && strings.HasSuffix(input, "\"")) ||
			(strings.HasPrefix(input, "'") && strings.HasSuffix(input, "'")) {
			input = input[1 : len(input)-1]
		}
	}

	return strings.TrimSpace(input)
}

// addProfileFromToken prompts for an API token and creates a profile, returning the profile name
func addProfileFromToken(cmd *cobra.Command) (string, error) {
	var token string

	tokenForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your Hatchet API token").
				Placeholder("API Token").
				Value(&token).
				EchoMode(huh.EchoModePassword),
		),
	).WithTheme(styles.HatchetTheme())

	err := tokenForm.Run()
	if err != nil {
		return "", fmt.Errorf("could not run token input form: %w", err)
	}

	if token == "" {
		return "", fmt.Errorf("no token provided")
	}

	// Get profile details from token
	profile, err := getProfileFromToken(cmd, token, "", "")
	if err != nil {
		return "", fmt.Errorf("could not get profile from token: %w", err)
	}

	// Prompt for custom name or use tenant name
	name := profile.Name

	nameForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Enter a name for this profile (default: %s)", name)).
				Placeholder(name).
				Value(&name),
		),
	).WithTheme(styles.HatchetTheme())

	err = nameForm.Run()
	if err != nil {
		return "", fmt.Errorf("could not run name input form: %w", err)
	}

	if name == "" {
		name = profile.Name
	}

	// Save the profile
	err = cli.AddProfile(name, profile)
	if err != nil {
		return "", fmt.Errorf("could not add profile: %w", err)
	}

	return name, nil
}

func getProfileFromToken(cmd *cobra.Command, token, nameOverride, tlsOverride string) (*cliconfig.Profile, error) {
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

	// Determine TLS strategy based on port or override
	var tlsStrategy string
	if tlsOverride != "" {
		tlsStrategy = tlsOverride
	} else {
		tlsStrategy = determineTLSStrategy(parsedTokenConf.GrpcBroadcastAddress)
	}

	return &cliconfig.Profile{
		TenantId:     client.TenantId(),
		Name:         name,
		Token:        token,
		ApiServerURL: parsedTokenConf.ServerURL,
		GrpcHostPort: parsedTokenConf.GrpcBroadcastAddress,
		ExpiresAt:    parsedTokenConf.ExpiresAt,
		TLSStrategy:  tlsStrategy,
	}, nil
}

func determineTLSStrategy(grpcHostPort string) string {
	// Try to auto-detect TLS by probing the endpoint
	strategy, err := probeTLSEndpoint(grpcHostPort)
	if err != nil {
		cli.Logger.Warnf("could not auto-detect TLS for %s: %v", grpcHostPort, err)
		cli.Logger.Info("falling back to user prompt")

		// Fall back to asking the user
		var useTLS bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Does the gRPC endpoint %s use TLS?", grpcHostPort)).
					Description("Auto-detection failed.").
					Value(&useTLS),
			),
		).WithTheme(styles.HatchetTheme())

		err := form.Run()
		if err != nil {
			cli.Logger.Fatalf("could not run TLS confirmation form: %v", err)
		}

		if useTLS {
			return "tls"
		}
		return "none"
	}

	return strategy
}

// probeTLSEndpoint attempts to detect if an endpoint uses TLS by probing it
func probeTLSEndpoint(hostPort string) (string, error) {
	// Parse the host:port to ensure it's valid
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", fmt.Errorf("invalid host:port format: %w", err)
	}

	// Create a context with timeout for the probe
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try TLS connection first (most common case)
	tlsConfig := &tls.Config{
		// we just want to see if TLS is spoken, not validate the cert
		InsecureSkipVerify: true, // nolint: gosec
		ServerName:         host,
	}

	// Attempt TLS dial
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", hostPort, tlsConfig)
	if err == nil {
		// TLS connection succeeded
		conn.Close()
		return "tls", nil
	}

	dialNoTLS := func() (string, error) {
		plainConn, plainErr := dialer.DialContext(ctx, "tcp", hostPort)
		if plainErr == nil {
			plainConn.Close()
			return "none", nil
		}
		return "", fmt.Errorf("endpoint not reachable: %w", plainErr)
	}

	// Check if the error is a RecordHeaderError - this means the server sent non-TLS data
	var recordHeaderErr tls.RecordHeaderError
	if errors.As(err, &recordHeaderErr) {
		return dialNoTLS()
	}

	// Check for EOF, which commonly occurs when connecting to a non-TLS server with TLS
	if errors.Is(err, io.EOF) {
		return dialNoTLS()
	}

	// If it's neither a RecordHeaderError nor EOF, it's likely a connection error
	return "", fmt.Errorf("could not connect to endpoint: %w", err)
}

func selectProfileForm(useDefault bool) string {
	profiles := cli.GetProfiles()

	if len(profiles) == 0 {
		cli.Logger.Info("No profiles configured")
		return ""
	}

	// Get and sort profile names for stable ordering
	names := make([]string, 0, len(profiles))
	for profileName := range profiles {
		names = append(names, profileName)
	}
	sort.Strings(names)

	// If useDefault is true, try to return the default profile without showing the form
	if useDefault {
		defaultProfile := cli.GetDefaultProfile()
		// Verify the default profile is still in the list
		if defaultProfile != "" && slices.Contains(names, defaultProfile) {
			return defaultProfile
		}
	}

	// Create options in sorted order
	profileNames := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		profileNames = append(profileNames, huh.Option[string]{
			Key:   name,
			Value: name,
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

	defaultProfile := cli.GetDefaultProfile()

	var lines []string
	// Use Primary.Bold instead of Section to avoid the MarginBottom spacing
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.PrimaryColor)
	lines = append(lines, headerStyle.Render("Configured Profiles"))
	for _, profile := range profiles {
		profileDisplay := profile
		if profile == defaultProfile {
			profileDisplay = profile + " " + styles.Muted.Render("(default)")
		}
		lines = append(lines, styles.ListItem.Render(styles.Accent.Render("• ")+profileDisplay))
	}

	return strings.Join(lines, "\n")
}

// profileActionView renders a profile action result (add, update, remove, set-as-default, unset-default)
func profileActionView(action, profileName string) string {
	var message string
	switch action {
	case "added":
		message = fmt.Sprintf("Profile '%s' added successfully", profileName)
	case "updated":
		message = fmt.Sprintf("Profile '%s' updated successfully", profileName)
	case "removed":
		message = fmt.Sprintf("Profile '%s' removed successfully", profileName)
	case "set-as-default":
		message = fmt.Sprintf("Profile '%s' set as default", profileName)
	case "unset-default":
		message = fmt.Sprintf("Default profile '%s' unset successfully", profileName)
	default:
		message = fmt.Sprintf("Profile '%s' %s", profileName, action)
	}

	return styles.SuccessMessage(message)
}
