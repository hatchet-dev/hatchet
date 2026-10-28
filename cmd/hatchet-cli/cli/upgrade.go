package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	cliconfig "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

const (
	latestCaskURL       = "https://raw.githubusercontent.com/hatchet-dev/homebrew-hatchet/main/Casks/hatchet.rb"
	releasesPageURL     = "https://github.com/hatchet-dev/hatchet/releases"
	installScriptURL    = "https://install.hatchet.run/install.sh"
	brewCask            = "hatchet-dev/hatchet/hatchet"
	updateCheckInterval = 24 * time.Hour
	updateCheckTimeout  = 2 * time.Second
	noUpdateCheckEnv    = "HATCHET_CLI_NO_UPDATE_CHECK"
)

var (
	upgradeYes       bool
	upgradeCheckOnly bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the hatchet CLI version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(buildInfo())
	},
}

var upgradeCmd = &cobra.Command{
	Use:           "upgrade",
	Short:         "Upgrade the hatchet CLI to the latest released version",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgrade(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.Flags().BoolVarP(&upgradeYes, "yes", "y", false, "Upgrade without asking for confirmation")
	upgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Only check for a new version; exit 1 if one is available")
}

func buildInfo() string {
	goVersion, commit, commitTime := runtime.Version(), "unknown", ""

	if bi, ok := debug.ReadBuildInfo(); ok {
		goVersion = bi.GoVersion
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				commit = s.Value
			case "vcs.time":
				commitTime = s.Value
			}
		}
	}

	if commitTime != "" {
		commit += " (" + commitTime + ")"
	}

	return fmt.Sprintf("hatchet version %s\ngo version: %s\nplatform: %s/%s\ncommit: %s\n",
		Version, goVersion, runtime.GOOS, runtime.GOARCH, commit)
}

func runUpgrade(ctx context.Context) error {
	if Version == devDefaultVersion {
		fmt.Println("This is a development build of hatchet; nothing to upgrade.")
		return nil
	}

	fmt.Println(styles.Muted.Render("Checking for the latest release..."))

	latest, err := fetchLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("could not check the latest version: %w", err)
	}

	newer, err := isNewer(latest, Version)
	if err != nil {
		return err
	}

	if !newer {
		fmt.Printf("hatchet %s is already the latest version.\n", Version)
		return nil
	}

	fmt.Printf("A new version of hatchet is available: %s (you have %s)\n", latest, Version)

	if upgradeCheckOnly {
		os.Exit(1)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate the running binary: %w", err)
	}

	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	if runtime.GOOS == "windows" {
		fmt.Printf("Automatic upgrades are not supported on Windows. Download the latest release from %s\n", releasesPageURL)
		return nil
	}

	var upgrade *exec.Cmd
	var describe string

	method := detectInstallMethod(exe)

	if method == "homebrew" {
		upgrade = exec.CommandContext(ctx, "brew", "upgrade", "--cask", brewCask)
		describe = strings.Join(upgrade.Args, " ")
	} else {
		upgrade = exec.CommandContext(ctx, "bash", "-s", "--", latest)
		upgrade.Env = append(os.Environ(), "INSTALL_DIR="+filepath.Dir(exe))
		describe = fmt.Sprintf("curl -fsSL %s | INSTALL_DIR=%s bash -s -- %s", installScriptURL, filepath.Dir(exe), latest)
	}

	if !upgradeYes {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return errors.New("stdin is not a terminal; re-run with --yes to upgrade without confirmation")
		}

		confirmed := false
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Upgrade hatchet %s → %s?", Version, latest)).
					Description("Will run: " + describe).
					Value(&confirmed),
			),
		).WithTheme(styles.HatchetTheme())

		if err := form.Run(); err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("Upgrade cancelled.")
			return nil
		}
	}

	if method == "script" {
		script, err := fetchInstallScript(ctx)
		if err != nil {
			return err
		}
		upgrade.Stdin = strings.NewReader(script)
	}

	upgrade.Stdout = os.Stdout
	upgrade.Stderr = os.Stderr

	if err := upgrade.Run(); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	return nil
}

func detectInstallMethod(exe string) string {
	if strings.Contains(exe, "/Caskroom/") || strings.Contains(exe, "/Cellar/") {
		return "homebrew"
	}
	return "script"
}

func isNewer(latest, current string) (bool, error) {
	l, err := semver.NewVersion(latest)
	if err != nil {
		return false, fmt.Errorf("invalid latest version %q: %w", latest, err)
	}

	c, err := semver.NewVersion(current)
	if err != nil {
		return false, fmt.Errorf("invalid current version %q: %w", current, err)
	}

	return l.GreaterThan(c), nil
}

var caskVersionRe = regexp.MustCompile(`(?m)^\s*version\s+"([^"]+)"`)

func fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestCaskURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s from %s", resp.Status, latestCaskURL)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	return parseCaskVersion(string(b))
}

func parseCaskVersion(cask string) (string, error) {
	m := caskVersionRe.FindStringSubmatch(cask)
	if m == nil {
		return "", errors.New("could not find a version in the Homebrew cask")
	}

	tag := "v" + strings.TrimPrefix(m[1], "v")

	if _, err := semver.NewVersion(tag); err != nil {
		return "", fmt.Errorf("unexpected cask version %q", m[1])
	}

	return tag, nil
}

func fetchInstallScript(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, installScriptURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not download the install script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s from %s", resp.Status, installScriptURL)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func startUpdateCheck() func() string {
	result := make(chan string, 1)

	if Version == devDefaultVersion ||
		os.Getenv(noUpdateCheckEnv) != "" ||
		!cliconfig.TelemetryEnabled() ||
		!term.IsTerminal(int(os.Stderr.Fd())) {
		close(result)
		return func() string { return "" }
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)

	go func() {
		defer close(result)
		result <- updateNotice(ctx)
	}()

	return func() string {
		defer cancel()
		select {
		case msg := <-result:
			return msg
		case <-ctx.Done():
			return ""
		}
	}
}

func updateNotice(ctx context.Context) string {
	cachePath := filepath.Join(cliconfig.HomeDir, ".hatchet", "latest_version")

	latest := ""
	if info, err := os.Stat(cachePath); err == nil && time.Since(info.ModTime()) < updateCheckInterval {
		if b, err := os.ReadFile(cachePath); err == nil {
			latest = strings.TrimSpace(string(b))
		}
	}

	if latest == "" {
		var err error
		if latest, err = fetchLatestVersion(ctx); err != nil {
			return ""
		}
		_ = os.WriteFile(cachePath, []byte(latest+"\n"), 0600)
	}

	if newer, err := isNewer(latest, Version); err != nil || !newer {
		return ""
	}

	return styles.Muted.Render(fmt.Sprintf(
		"A new version of hatchet is available: %s (you have %s). Run `hatchet upgrade`.", latest, Version))
}
