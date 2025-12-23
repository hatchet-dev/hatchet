package cli

import (
	"embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/templater"

	"github.com/charmbracelet/huh"
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
		}

		err := templater.Process(content, fmt.Sprintf("templates/%s", language), dir, templater.Data{
			Name: projectName,
		})

		if err != nil {
			cli.Logger.Fatalf("could not process templates: %v", err)
		}

		fmt.Println(quickstartSuccessView(language, projectName, dir))
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

	err := huh.NewInput().
		Title("Enter a name for your project (default hatchet-worker)").
		Placeholder("hatchet-worker").
		Value(&name).WithTheme(styles.HatchetTheme()).Run()

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

	err := huh.NewInput().
		Title(fmt.Sprintf("Enter a directory to create your project in (default ./%s)", name)).
		Placeholder(fmt.Sprintf("./%s", name)).
		Value(&directory).WithTheme(styles.HatchetTheme()).Run()

	if err != nil {
		cli.Logger.Fatalf("could not run quickstart form: %v", err)
	}

	if directory == "" {
		directory = fmt.Sprintf("./%s", name)
	}

	return directory
}

// quickstartSuccessView renders the success message with next steps
func quickstartSuccessView(language, projectName, dir string) string {
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
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("The worker will automatically connect to your Hatchet instance."))

	return styles.SuccessBox.Render(strings.Join(lines, "\n"))
}
