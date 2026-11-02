//go:build !darwin

package cli

import (
	"context"
	"os/exec"
)

func brewUpgradeCommand(context.Context, string) (*exec.Cmd, string, error) {
	return nil, "", nil
}
