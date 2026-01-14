package cli

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/templater"
)

//go:embed all:templates/*
var content embed.FS

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Generate a quickstart Hatchet worker project",
	Long:  `Generate a quickstart Hatchet worker project with boilerplate code in your language of choice.`,
	Example: `  # Generate a project interactively (prompts for language, name, and directory)
  hatchet quickstart

  # Generate a Python project with default settings
  hatchet quickstart --language python

  # Generate a TypeScript project with custom name and directory
  hatchet quickstart --language typescript --project-name my-worker --directory ./workers/my-worker

  # Generate a Go project with short flags
  hatchet quickstart -l go -p my-worker -d ./my-worker`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if at least one profile exists
		profileNames := cli.ListProfiles()
		if len(profileNames) == 0 {
			fmt.Println(noProfilesMessage())
			os.Exit(1)
		}

		// Get flag values
		language, _ := cmd.Flags().GetString("language")
		projectName, _ := cmd.Flags().GetString("project-name")
		dir, _ := cmd.Flags().GetString("directory")

		// Use interactive forms only if flags not provided
		if language == "" {
			language = selectLanguageForm()
		}

		// Validate language
		validLanguages := map[string]bool{"python": true, "typescript": true, "go": true}

		if !validLanguages[language] {
			cli.Logger.Fatalf("invalid language: %s (must be one of: python, typescript, go)", language)
		}

		if projectName == "" {
			projectName = selectNameForm()
		}

		if dir == "" {
			dir = selectDirectoryForm(projectName)
		} else {
			// If directory was provided via flag, check if it exists
			if _, err := os.Stat(dir); err == nil {
				cli.Logger.Fatalf("directory %s already exists - please choose a different directory or delete the existing one", dir)
			}
		}

		templateData := templater.Data{
			Name: projectName,
		}

		err := templater.Process(content, fmt.Sprintf("templates/%s", language), dir, templateData)

		if err != nil {
			cli.Logger.Fatalf("could not process templates: %v", err)
		}

		// Process POST_QUICKSTART.md if it exists
		postQuickstart, err := templater.ProcessPostQuickstart(content, fmt.Sprintf("templates/%s", language), templateData)
		if err != nil {
			cli.Logger.Fatalf("could not process post-quickstart content: %v", err)
		}

		fmt.Println(quickstartSuccessView(language, projectName, dir, postQuickstart))
	},
}

func init() {
	rootCmd.AddCommand(quickstartCmd)

	// Add flags for quickstart command
	quickstartCmd.Flags().StringP("language", "l", "", "Programming language (python, typescript, go)")
	quickstartCmd.Flags().StringP("project-name", "p", "", "Name of the project (default: hatchet-worker)")
	quickstartCmd.Flags().StringP("directory", "d", "", "Directory to create the project in (default: ./{project-name})")
}

func selectLanguageForm() string {
	language := ""

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose your language").
				Options(
					huh.NewOption("Python", "python"),
					huh.NewOption("Typescript", "typescript"),
					huh.NewOption("Go", "go"),
				).
				Value(&language),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()

	if err != nil {
		cli.Logger.Fatalf("could not run quickstart form: %v", err)
	}

	return language
}

func selectNameForm() string {
	name := ""

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter a name for your project (default hatchet-worker)").
				Placeholder("hatchet-worker").
				Value(&name),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()

	if err != nil {
		cli.Logger.Fatalf("could not run quickstart form: %v", err)
	}

	if name == "" {
		name = "hatchet-worker"
	}

	return name
}

func selectDirectoryForm(name string) string {
	directory := ""

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Enter a directory to create your project in (default ./%s)", name)).
				Placeholder(fmt.Sprintf("./%s", name)).
				Value(&directory).
				Validate(func(s string) error {
					// Use default if empty
					targetDir := s
					if targetDir == "" {
						targetDir = fmt.Sprintf("./%s", name)
					}

					// Check if directory exists
					if _, err := os.Stat(targetDir); err == nil {
						return fmt.Errorf("directory already exists - please choose a different name or delete the existing directory")
					}

					return nil
				}),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()

	if err != nil {
		cli.Logger.Fatalf("could not run quickstart form: %v", err)
	}

	if directory == "" {
		directory = fmt.Sprintf("./%s", name)
	}

	return directory
}

// renderCodeBlocks processes a string and renders code blocks wrapped in ```sh
func renderCodeBlocks(content string) string {
	// Simple code block rendering for ```sh blocks
	lines := strings.Split(content, "\n")
	var result []string
	inCodeBlock := false
	var codeLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "```sh") || strings.HasPrefix(line, "```bash") {
			inCodeBlock = true
			codeLines = []string{}
			continue
		}
		if inCodeBlock && strings.HasPrefix(line, "```") {
			// End of code block - render it
			inCodeBlock = false
			for _, codeLine := range codeLines {
				result = append(result, styles.Code.Render("   "+codeLine))
			}
			continue
		}
		if inCodeBlock {
			codeLines = append(codeLines, line)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// noProfilesMessage returns a formatted message when no profiles are configured
func noProfilesMessage() string {
	var lines []string

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ErrorColor)

	lines = append(lines, headerStyle.Render("No Hatchet profiles configured"))
	lines = append(lines, "")
	lines = append(lines, "To use the quickstart command, you need to connect to a Hatchet instance first.")
	lines = append(lines, "")
	lines = append(lines, headerStyle.Render("Option 1: Sign up for Hatchet Cloud"))
	lines = append(lines, "")
	lines = append(lines, "Visit "+styles.Accent.Render("https://cloud.onhatchet.run")+" to create an account, then add your profile:")
	lines = append(lines, styles.Code.Render("   hatchet profile add"))
	lines = append(lines, "")
	lines = append(lines, headerStyle.Render("Option 2: Start a local Hatchet server"))
	lines = append(lines, "")
	lines = append(lines, "Run a local instance with Docker:")
	lines = append(lines, styles.Code.Render("   hatchet server start"))

	return styles.InfoBox.Render(strings.Join(lines, "\n"))
}

// quickstartSuccessView renders the success message with next steps
func quickstartSuccessView(language, projectName, dir, postQuickstart string) string {
	var lines []string

	lines = append(lines, styles.SuccessMessage(fmt.Sprintf("Successfully created %s project: %s", language, projectName)))
	lines = append(lines, "")
	lines = append(lines, styles.KeyValue("Location", dir))
	lines = append(lines, "")
	lines = append(lines, styles.Section("Next Steps"))
	lines = append(lines, "")
	lines = append(lines, styles.Accent.Render("1.")+" Navigate to your project:")
	lines = append(lines, styles.Code.Render("   cd "+dir))
	lines = append(lines, "")
	lines = append(lines, styles.Accent.Render("2.")+" Start the worker in development mode:")
	lines = append(lines, styles.Code.Render("   hatchet worker dev"))

	// Add POST_QUICKSTART.md content as step 3 if it exists
	if postQuickstart != "" {
		lines = append(lines, "")
		lines = append(lines, styles.Accent.Render("3.")+" "+strings.TrimSpace(renderCodeBlocks(postQuickstart)))
	} else {
		lines = append(lines, "")
		lines = append(lines, styles.Muted.Render("The worker will automatically connect to your Hatchet instance."))
	}

	return styles.SuccessBox.Render(strings.Join(lines, "\n"))
}
