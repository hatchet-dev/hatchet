package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func brewUpgradeCommand(ctx context.Context, exe string) (*exec.Cmd, string, error) {
	prefix := ""
	for _, dir := range []string{"/Caskroom/", "/Cellar/"} {
		if i := strings.Index(exe, dir); i >= 0 {
			prefix = exe[:i]
			break
		}
	}
	if prefix == "" {
		return nil, "", nil
	}

	brew := filepath.Join(prefix, "bin", "brew")
	if _, err := os.Stat(brew); err != nil {
		return nil, "", fmt.Errorf("hatchet was installed by Homebrew at %s but %s is missing", exe, brew)
	}

	cmd := exec.CommandContext(ctx, brew, "upgrade", "--cask", "hatchet-dev/hatchet/hatchet")

	return cmd, strings.Join(cmd.Args, " "), nil
}
