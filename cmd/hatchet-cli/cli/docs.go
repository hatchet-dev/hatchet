package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

const defaultMCPURL = "https://docs.hatchet.run/api/mcp"
const docsBaseURL = "https://docs.hatchet.run"

var mcpURL string

// docsCmd represents the docs command
var docsCmd = &cobra.Command{
	Use:     "docs",
	Aliases: []string{"doc"},
	Short:   "Hatchet documentation for AI editors and coding agents",
	Long: `Hatchet documentation is optimized for LLMs and available as:
  • MCP server:    ` + defaultMCPURL + `
  • llms.txt:      ` + docsBaseURL + `/llms.txt
  • Full docs:     ` + docsBaseURL + `/llms-full.txt

Use "hatchet docs install" to configure your AI editor.`,
	Example: `  # Interactive — pick your editor
  hatchet docs install

  # Configure for Cursor
  hatchet docs install cursor

  # Configure for Claude Code
  hatchet docs install claude-code

  # Use a custom MCP URL (self-hosted)
  hatchet docs install cursor --url https://my-hatchet.example.com/api/mcp`,
	Run: func(cmd *cobra.Command, args []string) {
		printAllOptions()
	},
}

// docsInstallCmd represents the docs install command
var docsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Hatchet docs into an AI editor",
	Long: `Configure Hatchet documentation as an MCP (Model Context Protocol) server
for AI editors like Cursor and Claude Code.`,
	Example: `  # Interactive — pick your editor
  hatchet docs install

  # Configure for Cursor
  hatchet docs install cursor

  # Configure for Claude Code
  hatchet docs install claude-code

  # Configure for Claude Code
  hatchet docs install claude-code`,
	Run: func(cmd *cobra.Command, args []string) {
		// Interactive mode: let user pick their editor
		var editor string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Which AI editor do you want to configure?").
					Options(
						huh.NewOption("Cursor", "cursor"),
						huh.NewOption("Claude Code", "claude-code"),
					).
					Value(&editor),
			),
		).WithTheme(styles.HatchetTheme())

		err := form.Run()
		if err != nil {
			os.Exit(1)
		}

		switch editor {
		case "cursor":
			runDocsCursor()
		case "claude-code":
			runDocsClaudeCode()
		}
	},
}

// ---------------------------------------------------------------------------
// Subcommands of `docs install`
// ---------------------------------------------------------------------------

var docsInstallCursorCmd = &cobra.Command{
	Use:   "cursor",
	Short: "Configure Hatchet docs for Cursor IDE",
	Long: `Set up Hatchet documentation as an MCP server in Cursor.

This creates a .cursor/rules/hatchet-docs.mdc file in your project that
configures the Hatchet MCP docs server, and prints the one-click deeplink.`,
	Run: func(cmd *cobra.Command, args []string) {
		runDocsCursor()
	},
}

var docsInstallClaudeCodeCmd = &cobra.Command{
	Use:   "claude-code",
	Short: "Configure Hatchet docs for Claude Code",
	Long:  `Set up Hatchet documentation as an MCP server in Claude Code.`,
	Run: func(cmd *cobra.Command, args []string) {
		runDocsClaudeCode()
	},
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

func runDocsCursor() {
	url := getMCPURL()

	fmt.Println(styles.Title("Hatchet Docs → Cursor"))
	fmt.Println()

	// 1. Write .cursor/rules/hatchet-docs.mdc
	rulesDir := filepath.Join(".", ".cursor", "rules")
	rulesFile := filepath.Join(rulesDir, "hatchet-docs.mdc")

	ruleContent := fmt.Sprintf(`---
description: Hatchet documentation MCP server
globs:
alwaysApply: true
---

When working with Hatchet (task queues, workflows, durable execution), use the
Hatchet MCP docs server for accurate, up-to-date API reference and examples.

MCP server URL: %s

Use the search_docs tool to find relevant documentation pages, or get_full_docs
for comprehensive context. Documentation covers Python, TypeScript, and Go SDKs.
`, url)

	if err := os.MkdirAll(rulesDir, 0o755); err == nil {
		if err := os.WriteFile(rulesFile, []byte(ruleContent), 0o644); err == nil {
			fmt.Println(styles.SuccessMessage("Created " + rulesFile))
		} else {
			fmt.Printf("  ⚠ Could not write %s: %v\n", rulesFile, err)
		}
	} else {
		fmt.Printf("  ⚠ Could not create %s: %v\n", rulesDir, err)
	}

	// 2. Print the MCP deeplink
	fmt.Println()
	deeplink := cursorMCPDeeplink(url)
	fmt.Println(styles.Section("One-click install"))
	fmt.Println(styles.InfoMessage("Open this link in your browser to install the MCP server in Cursor:"))
	fmt.Println()
	fmt.Println("  " + styles.URL(deeplink))
	fmt.Println()

	// 3. Offer to open in browser
	if promptOpenBrowser() {
		openBrowser(deeplink)
	}
}

func runDocsClaudeCode() {
	url := getMCPURL()

	fmt.Println(styles.Title("Hatchet Docs → Claude Code"))
	fmt.Println()

	claudeCmd := fmt.Sprintf("claude mcp add --transport http hatchet-docs %s", url)

	// Try to run claude mcp add directly
	if _, err := exec.LookPath("claude"); err == nil {
		fmt.Println(styles.InfoMessage("Found claude CLI. Adding MCP server..."))
		fmt.Println()

		cmd := exec.Command("claude", "mcp", "add", "--transport", "http", "hatchet-docs", url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err == nil {
			fmt.Println()
			fmt.Println(styles.SuccessMessage("Hatchet docs MCP server added to Claude Code"))
			return
		}

		fmt.Printf("  ⚠ Command failed. You can run it manually:\n\n")
	} else {
		fmt.Println(styles.InfoMessage("Claude CLI not found on PATH. Run this command manually:"))
		fmt.Println()
	}

	fmt.Println(styles.Code.Render(claudeCmd))
	fmt.Println()
}

func printAllOptions() {
	url := getMCPURL()

	fmt.Println(styles.Title("Hatchet Docs for AI Editors"))
	fmt.Println()

	// MCP Server
	fmt.Println(styles.Section("MCP Server"))
	fmt.Println(styles.KeyValue("URL", url))
	fmt.Println()

	// Cursor
	fmt.Println(styles.Section("Cursor"))
	deeplink := cursorMCPDeeplink(url)
	fmt.Println(styles.KeyValue("Deeplink", deeplink))
	fmt.Println(styles.KeyValue("Or run", "hatchet docs install cursor"))
	fmt.Println()

	// Claude Code
	fmt.Println(styles.Section("Claude Code"))
	fmt.Println(styles.Code.Render(fmt.Sprintf("claude mcp add --transport http hatchet-docs %s", url)))
	fmt.Println()

	// llms.txt
	fmt.Println(styles.Section("LLM-Friendly Docs (llms.txt)"))
	fmt.Println(styles.KeyValue("Index", docsBaseURL+"/llms.txt"))
	fmt.Println(styles.KeyValue("Full docs", docsBaseURL+"/llms-full.txt"))
	fmt.Println()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func getMCPURL() string {
	if mcpURL != "" {
		return mcpURL
	}
	return defaultMCPURL
}

func cursorMCPDeeplink(url string) string {
	config := map[string]interface{}{
		"command": "npx",
		"args":    []string{"-y", "mcp-remote", url},
	}
	configJSON, _ := json.Marshal(config)
	encoded := base64.StdEncoding.EncodeToString(configJSON)
	return fmt.Sprintf("cursor://anysphere.cursor-deeplink/mcp/install?name=hatchet-docs&config=%s", encoded)
}

func promptOpenBrowser() bool {
	var open bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Open in browser?").
				Value(&open),
		),
	).WithTheme(styles.HatchetTheme())

	if err := form.Run(); err != nil {
		return false
	}
	return open
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("  ⚠ Could not open browser: %v\n", err)
		fmt.Println("  Copy the link above and paste it in your browser.")
	}
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsInstallCmd)
	docsInstallCmd.AddCommand(docsInstallCursorCmd)
	docsInstallCmd.AddCommand(docsInstallClaudeCodeCmd)

	// Add --url flag to install and its subcommands
	for _, cmd := range []*cobra.Command{docsInstallCmd, docsInstallCursorCmd, docsInstallClaudeCodeCmd} {
		cmd.Flags().StringVar(&mcpURL, "url", "", "Custom MCP server URL (default: "+defaultMCPURL+")")
	}
}
