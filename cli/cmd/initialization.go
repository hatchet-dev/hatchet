package cfg

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/hatchet-dev/hatchet/cli/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func initialize() *cobra.Command {
	initCmd := &cobra.Command{
		Use:     "initialize",
		Short:   "init the hatchet cfg.",
		Long:    "init provision the hatchet configuration file.",
		Example: "hatchet init",
		Aliases: []string{"i", "init"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get values from flags
			engineMode := viper.GetString("engine-mode")
			token := viper.GetString("token")
			namespace := viper.GetString("namespace")
			dbURL := viper.GetString("db-url")

			hostname, err := os.Hostname()
			if err != nil {
				hostname = ""
			}

			// Create form fields based on what's not provided via flags
			var groups []*huh.Group

			// If engine mode not provided, add selection
			if engineMode == "" {
				groups = append(groups, huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select execution environment").
						Options(
							huh.NewOption("Local", "local"),
							huh.NewOption("Cloud", "cloud"),
						).
						Value(&engineMode),
				))
			}

			// Dynamically create next group based on engine mode
			var modeGroup *huh.Group
			modeFields := make([]huh.Field, 0)

			// For cloud mode, ensure we have token and namespace
			if engineMode == "cloud" {
				if token == "" {
					modeFields = append(modeFields,
						huh.NewInput().
							Title("Enter your authentication token").
							Description("Generate a token at https://cloud.onhatchet.dev/settings/tokens").
							EchoMode(huh.EchoModePassword).
							Value(&token).
							Validate(func(str string) error {
								if str == "" {
									return fmt.Errorf("token cannot be empty")
								}
								return nil
							}),
					)
				}

				modeFields = append(modeFields,
					huh.NewInput().
						Title("Enter namespace").
						Placeholder(hostname).
						Value(&namespace),
				)
			}

			// For local mode, optionally prompt for database URL
			if engineMode == "local" && dbURL == "" {
				modeFields = append(modeFields,
					huh.NewInput().
						Title("Enter PostgreSQL connection string (optional)").
						Value(&dbURL),
				)
			}

			if len(modeFields) > 0 {
				modeGroup = huh.NewGroup(modeFields...)
				groups = append(groups, modeGroup)
			}

			// If we have any groups to show, run the form
			if len(groups) > 0 {
				form := huh.NewForm(groups...)
				err := form.Run()
				if err != nil {
					return fmt.Errorf("form input failed: %w", err)
				}
			}

			// Show a spinner while initializing
			err = spinner.New().
				Action(func() {
					// TODO: actually initialize hatchet
					time.Sleep(1 * time.Second)

					config.WriteProfiles(
						&config.ProfilesConfigFile{
							Profiles: []config.ProfileConfigFile{
								{
									Name:       "default",
									EngineMode: config.EngineMode(engineMode),
									Token:      token,
									IsDefault:  true,
									Namespace:  namespace,
								},
							},
						}, nil)

				}).
				Title("Initializing Hatchet configuration...").
				Run()

			if err != nil {
				return fmt.Errorf("spinner failed: %w", err)
			}

			fmt.Printf("\n✓ Hatchet initialized successfully in %s mode\n", engineMode)
			return nil
		},
	}

	// Flags (same as original)
	initCmd.Flags().String("engine-mode", "", "Execution environment [local|cloud]")
	initCmd.Flags().String("project-root", "", "Project root directory (auto-detects: go.mod, project.toml, requirements.txt, package.json)")
	initCmd.Flags().String("name", "", "Project name (optional,defaults to tenant name)")
	initCmd.Flags().String("profile-path", "", "Path to the profiles configuration file (optional, defaults to ~/.hatchet/profiles.yaml)")
	initCmd.Flags().String("token", "", "Authentication token (required if engine-mode is cloud)")
	initCmd.Flags().String("namespace", "", "Environment namespace (defaults to local machine name) (required if engine-mode is cloud)")
	initCmd.Flags().String("db-url", "", "PostgreSQL connection string (optional)")

	return initCmd
}
