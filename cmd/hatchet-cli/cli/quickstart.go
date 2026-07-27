package cli

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	quickstarts "github.com/hatchet-dev/hatchet-quickstarts"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/templater"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Generate a quickstart Hatchet worker project",
	Long: `Generate a quickstart Hatchet worker project with boilerplate code in your language of choice.

Supports multiple package managers:
  Python: poetry, uv, pip
  TypeScript: npm, pnpm, yarn, bun
  Go: go modules

Use-case templates are selected with --use-case. The default, simple, is a
minimal worker with one workflow. Other use cases, such as scheduled, may
support a subset of languages.`,
	Example: `  # Generate a project interactively (prompts for use case, language, package manager, name, and directory)
  hatchet quickstart

  # Generate a Python project with Poetry
  hatchet quickstart --language python --package-manager poetry

  # Generate a TypeScript project with pnpm
  hatchet quickstart --language typescript --package-manager pnpm --project-name my-worker

  # Generate the scheduled use case in Go
  hatchet quickstart --use-case scheduled --language go

  # Generate a Python project with uv using short flags
  hatchet quickstart -l python -m uv -p my-worker -d ./my-worker`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if at least one profile exists
		profileNames := cli.ListProfiles()
		if len(profileNames) == 0 {
			fmt.Println(noProfilesMessage())
			os.Exit(1)
		}

		// Get flag values
		useCase, _ := cmd.Flags().GetString("use-case")
		language, _ := cmd.Flags().GetString("language")
		packageManager, _ := cmd.Flags().GetString("package-manager")
		projectName, _ := cmd.Flags().GetString("project-name")
		dir, _ := cmd.Flags().GetString("directory")

		templatesFS := quickstarts.TemplatesFS()

		// Use interactive forms only if flags not provided
		if useCase == "" {
			// Commands that pass --language or --package-manager ran
			// without prompts before use cases existed and must keep
			// doing so. They get the simple template.
			if cmd.Flags().Changed("language") || cmd.Flags().Changed("package-manager") {
				useCase = templater.DefaultUseCase
			} else {
				useCase = selectUseCaseForm(templatesFS)
			}
		}

		if language == "" {
			language = selectLanguageForm(templatesFS, useCase)
		}

		// Get package manager
		if packageManager == "" {
			packageManager = selectPackageManagerForm(language)
		}

		selection := templater.Selection{
			UseCase:        useCase,
			Language:       language,
			PackageManager: packageManager,
		}

		if err := templater.Validate(templatesFS, selection); err != nil {
			cli.Logger.Fatalf("%v", err)
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

		postQuickstart, err := GenerateQuickstart(selection, projectName, dir)
		if err != nil {
			cli.Logger.Fatalf("could not generate quickstart: %v", err)
		}

		fmt.Println(quickstartSuccessView(language, projectName, dir, postQuickstart))
	},
}

// GenerateQuickstart generates a quickstart project without interactive forms.
// Returns the post-quickstart content that should be displayed to the user.
func GenerateQuickstart(selection templater.Selection, projectName, dir string) (string, error) {
	templateData := templater.Data{
		Name:           projectName,
		PackageManager: selection.PackageManager,
	}

	templatesFS := quickstarts.TemplatesFS()

	err := templater.ProcessMultiSource(templatesFS, selection, dir, templateData)
	if err != nil {
		return "", fmt.Errorf("could not process templates: %w", err)
	}

	// Process POST_QUICKSTART.md if it exists
	postQuickstart, err := templater.ProcessPostQuickstartMultiSource(templatesFS, selection, templateData)
	if err != nil {
		return "", fmt.Errorf("could not process post-quickstart content: %w", err)
	}

	return postQuickstart, nil
}

func init() {
	rootCmd.AddCommand(quickstartCmd)

	// Add flags for quickstart command
	quickstartCmd.Flags().StringP("use-case", "u", "", "Use case template (default: simple)")
	quickstartCmd.Flags().StringP("language", "l", "", "Programming language (python, typescript, go)")
	quickstartCmd.Flags().StringP("package-manager", "m", "", "Package manager (poetry, uv, pip for Python; npm, pnpm, yarn, bun for TypeScript)")
	quickstartCmd.Flags().StringP("project-name", "p", "", "Name of the project (default: hatchet-worker)")
	quickstartCmd.Flags().StringP("directory", "d", "", "Directory to create the project in (default: ./{project-name})")
}

func selectUseCaseForm(templatesFS fs.FS) string {
	useCases, err := templater.UseCases(templatesFS)
	if err != nil {
		cli.Logger.Fatalf("could not list use cases: %v", err)
	}

	if len(useCases) == 1 {
		return useCases[0]
	}

	options := make([]huh.Option[string], 0, len(useCases))
	for _, useCase := range useCases {
		label := useCase
		if useCase == templater.DefaultUseCase {
			label = useCase + " (default)"
		}
		options = append(options, huh.NewOption(label, useCase))
	}

	useCase := ""

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose a use case").
				Options(options...).
				Value(&useCase),
		),
	).WithTheme(styles.HatchetTheme())

	if err := form.Run(); err != nil {
		cli.Logger.Fatalf("could not run quickstart form: %v", err)
	}

	return useCase
}

var languageLabels = map[string]string{
	"python":     "Python",
	"typescript": "Typescript",
	"go":         "Go",
}

func selectLanguageForm(templatesFS fs.FS, useCase string) string {
	languages, err := templater.LanguagesFor(templatesFS, useCase)
	if err != nil {
		cli.Logger.Fatalf("could not list languages for use case %q: %v", useCase, err)
	}

	if len(languages) == 0 {
		cli.Logger.Fatalf("use case %q has no language templates", useCase)
	}

	if len(languages) == 1 {
		return languages[0]
	}

	options := make([]huh.Option[string], 0, len(languages))
	for _, lang := range languages {
		options = append(options, huh.NewOption(languageLabels[lang], lang))
	}

	language := ""

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose your language").
				Options(options...).
				Value(&language),
		),
	).WithTheme(styles.HatchetTheme())

	if err := form.Run(); err != nil {
		cli.Logger.Fatalf("could not run quickstart form: %v", err)
	}

	return language
}

func selectPackageManagerForm(language string) string {
	packageManager := ""

	var options []huh.Option[string]

	switch language {
	case "python":
		options = []huh.Option[string]{
			huh.NewOption("Poetry (recommended)", "poetry"),
			huh.NewOption("uv", "uv"),
			huh.NewOption("pip", "pip"),
		}
	case "typescript":
		options = []huh.Option[string]{
			huh.NewOption("npm", "npm"),
			huh.NewOption("pnpm", "pnpm"),
			huh.NewOption("Yarn", "yarn"),
			huh.NewOption("Bun", "bun"),
		}
	case "go":
		// Go only has one package manager, return default
		return "go"
	default:
		cli.Logger.Fatalf("unsupported language: %s", language)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose your package manager").
				Options(options...).
				Value(&packageManager),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()

	if err != nil {
		cli.Logger.Fatalf("could not run package manager form: %v", err)
	}

	return packageManager
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
	lines = append(lines, "Visit "+styles.Accent.Render("https://cloud.hatchet.run")+" to create an account, then add your profile:")
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
