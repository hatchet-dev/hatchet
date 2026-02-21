package cli

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

//go:embed all:skill-assets
var skillAssets embed.FS

var (
	skillsInstallDir   string
	skillsInstallForce bool
)

// skillsCmd represents the skills parent command
var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage Hatchet agent skills for AI coding agents",
	Long: `Hatchet agent skills are reference documents that teach AI coding agents
how to use the Hatchet CLI to manage workflows, workers, and runs.

Install the skill package into your project to give agents step-by-step
instructions for common Hatchet operations.`,
	Example: `  # Install skills interactively
  hatchet skills install

  # Install to a custom directory
  hatchet skills install --dir ./my-project`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// skillsInstallCmd represents the skills install subcommand
var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Hatchet CLI agent skill package into your project",
	Long: `Install Hatchet CLI agent skills into your project.

Creates the skill directory structure under {dir}/skills/hatchet-cli/ and
appends a reference section to the project AGENTS.md file.`,
	Example: `  # Install to current directory (creates ./skills/hatchet-cli/)
  hatchet skills install

  # Install to a custom base directory
  hatchet skills install --dir ./my-project

  # Skip confirmation prompts
  hatchet skills install --force`,
	Run: func(cmd *cobra.Command, args []string) {
		runSkillsInstall()
	},
}

func init() {
	rootCmd.AddCommand(skillsCmd)
	skillsCmd.AddCommand(skillsInstallCmd)

	skillsInstallCmd.Flags().StringVarP(&skillsInstallDir, "dir", "d", ".", "Target base directory (skill installs under {dir}/skills/hatchet-cli/)")
	skillsInstallCmd.Flags().BoolVarP(&skillsInstallForce, "force", "f", false, "Skip confirmation prompts")
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

func runSkillsInstall() {
	// 1. Resolve paths
	baseDir, err := filepath.Abs(skillsInstallDir)
	if err != nil {
		cli.Logger.Fatalf("could not resolve directory: %v", err)
	}

	skillDir := filepath.Join(baseDir, "skills", "hatchet-cli")
	agentsFile := filepath.Join(baseDir, "AGENTS.md")

	// 2. Print header and summary
	fmt.Println(styles.Title("Hatchet Agent Skills Install"))
	fmt.Println()
	fmt.Println(styles.InfoMessage("The following will be created:"))
	fmt.Println()
	fmt.Printf("  %s\n", skillDir+"/SKILL.md")
	fmt.Printf("  %s\n", skillDir+"/AGENTS.md")
	fmt.Printf("  %s\n", skillDir+"/CLAUDE.md  (symlink → AGENTS.md)")
	fmt.Printf("  %s\n", skillDir+"/references/setup-cli.md")
	fmt.Printf("  %s\n", skillDir+"/references/start-worker.md")
	fmt.Printf("  %s\n", skillDir+"/references/trigger-and-watch.md")
	fmt.Printf("  %s\n", skillDir+"/references/debug-run.md")
	fmt.Printf("  %s\n", skillDir+"/references/replay-run.md")
	fmt.Println()
	fmt.Printf("  %s  (appended)\n", agentsFile)
	fmt.Println()

	// 3. Check if skill directory already exists
	if _, statErr := os.Stat(skillDir); statErr == nil {
		// Directory exists
		if !skillsInstallForce {
			fmt.Println(styles.InfoMessage("Skill directory already exists: " + skillDir))
			fmt.Println()
			var overwrite bool
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Overwrite existing skill files?").
						Value(&overwrite),
				),
			).WithTheme(styles.HatchetTheme())
			if formErr := form.Run(); formErr != nil || !overwrite {
				fmt.Println(styles.InfoMessage("Install cancelled."))
				return
			}
			fmt.Println()
		}
	}

	// 4. Check AGENTS.md for existing marker
	appendToAgents := true
	if data, readErr := os.ReadFile(agentsFile); readErr == nil {
		if strings.Contains(string(data), "<!-- hatchet-skills:start -->") {
			fmt.Println(styles.InfoMessage("AGENTS.md already contains a hatchet-skills section."))
			fmt.Println()
			if !skillsInstallForce {
				var appendAgain bool
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title("Append another hatchet-skills section to AGENTS.md?").
							Value(&appendAgain),
					),
				).WithTheme(styles.HatchetTheme())
				if formErr := form.Run(); formErr != nil || !appendAgain {
					appendToAgents = false
					fmt.Println()
				} else {
					fmt.Println()
				}
			} else {
				appendToAgents = false
			}
		}
	}

	// 5. Confirm overall installation (unless --force)
	if !skillsInstallForce {
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Proceed with installation?").
					Value(&confirm),
			),
		).WithTheme(styles.HatchetTheme())
		if formErr := form.Run(); formErr != nil || !confirm {
			fmt.Println(styles.InfoMessage("Install cancelled."))
			return
		}
		fmt.Println()
	}

	// 6. Create skill directory structure
	if mkdirErr := os.MkdirAll(filepath.Join(skillDir, "references"), 0o755); mkdirErr != nil {
		cli.Logger.Fatalf("could not create skill directory: %v", mkdirErr)
	}

	// Walk embedded skill-assets/hatchet-cli/ and write each file
	srcRoot := "skill-assets/hatchet-cli"
	err = fs.WalkDir(skillAssets, srcRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		// Skip agents-entry.md — it's a template, not installed directly
		if d.Name() == "agents-entry.md" {
			return nil
		}

		// Compute destination path
		rel, relErr := filepath.Rel(srcRoot, path)
		if relErr != nil {
			return relErr
		}
		dest := filepath.Join(skillDir, rel)

		// Ensure parent directory exists
		if mkdirErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkdirErr != nil {
			return mkdirErr
		}

		fileData, readErr := skillAssets.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(dest, fileData, 0o600)
	})
	if err != nil {
		cli.Logger.Fatalf("could not write skill files: %v", err)
	}

	// Generate AGENTS.md for the skill (SKILL.md body without YAML frontmatter)
	skillMDData, readErr := skillAssets.ReadFile(srcRoot + "/SKILL.md")
	if readErr != nil {
		cli.Logger.Fatalf("could not read SKILL.md: %v", readErr)
	}
	skillAgentsContent := stripFrontmatter(string(skillMDData))
	if writeErr := os.WriteFile(filepath.Join(skillDir, "AGENTS.md"), []byte(skillAgentsContent), 0o600); writeErr != nil {
		cli.Logger.Fatalf("could not write skill AGENTS.md: %v", writeErr)
	}

	// Create CLAUDE.md symlink pointing to AGENTS.md
	claudeMDPath := filepath.Join(skillDir, "CLAUDE.md")
	// Remove existing symlink/file if present (we already confirmed overwrite above)
	_ = os.Remove(claudeMDPath)
	if symlinkErr := os.Symlink("AGENTS.md", claudeMDPath); symlinkErr != nil {
		fmt.Printf("  ⚠ Could not create CLAUDE.md symlink: %v\n", symlinkErr)
	}

	fmt.Println(styles.SuccessMessage("Skill installed to " + skillDir))

	// 7. Append to project AGENTS.md
	if appendToAgents {
		entryData, readErr := skillAssets.ReadFile(srcRoot + "/agents-entry.md")
		if readErr != nil {
			cli.Logger.Fatalf("could not read agents-entry.md template: %v", readErr)
		}

		// Compute relative path from base dir to skill dir for use in the template
		relSkillDir, relErr := filepath.Rel(baseDir, skillDir)
		if relErr != nil {
			relSkillDir = skillDir
		}

		entry := strings.ReplaceAll(string(entryData), "{SKILL_DIR}", relSkillDir)

		if _, statErr := os.Stat(agentsFile); os.IsNotExist(statErr) {
			// Create AGENTS.md with the entry
			if writeErr := os.WriteFile(agentsFile, []byte(entry), 0o600); writeErr != nil {
				cli.Logger.Fatalf("could not create AGENTS.md: %v", writeErr)
			}
		} else {
			// Append to existing AGENTS.md
			f, openErr := os.OpenFile(agentsFile, os.O_APPEND|os.O_WRONLY, 0o600)
			if openErr != nil {
				cli.Logger.Fatalf("could not open AGENTS.md for appending: %v", openErr)
			}
			defer f.Close()
			if _, writeErr := f.WriteString(entry); writeErr != nil {
				cli.Logger.Fatalf("could not append to AGENTS.md: %v", writeErr)
			}
		}

		fmt.Println(styles.SuccessMessage("AGENTS.md updated"))
	}

	fmt.Println()
	fmt.Println(styles.Section("Next steps"))
	fmt.Println()
	fmt.Println("  • Run " + styles.Code.Render("hatchet docs install") + " to add the Hatchet MCP server to your AI editor")
	fmt.Println("  • Commit " + styles.Code.Render("skills/") + " and " + styles.Code.Render("AGENTS.md") + " to version control")
	fmt.Println()
}

// stripFrontmatter removes YAML frontmatter (content between leading --- delimiters)
// and returns the remaining body.
func stripFrontmatter(content string) string {
	content = strings.TrimLeft(content, "\r\n")
	if !strings.HasPrefix(content, "---") {
		return content
	}
	// Find the closing ---
	rest := content[3:]
	_, body, found := strings.Cut(rest, "\n---")
	if !found {
		return content
	}
	body = strings.TrimLeft(body, "\r\n")
	return body
}
