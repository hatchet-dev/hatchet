package local

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

const hatchetModule = "github.com/hatchet-dev/hatchet"

// requiredBinaries are the binaries needed for local mode
var requiredBinaries = []string{
	"hatchet-api",
	"hatchet-engine",
	"hatchet-migrate",
	"hatchet-admin",
}

// DetectRepoRoot walks up from the current directory looking for the hatchet go.mod
func DetectRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %w", err)
	}

	for {
		goMod := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(goMod); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					mod := strings.TrimSpace(strings.TrimPrefix(line, "module"))
					if mod == hatchetModule {
						return dir, nil
					}
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not inside the hatchet repository (looked for go.mod with module %s)", hatchetModule)
}

// BuildAllBinaries builds all required hatchet binaries from source into the
// download cache, keyed by the given version. After this, 'hatchet server start --local'
// will find them via the normal EnsureBinary cache lookup.
func BuildAllBinaries(ctx context.Context, repoRoot, version string) error {
	downloader, err := NewBinaryDownloader()
	if err != nil {
		return fmt.Errorf("failed to create binary downloader: %w", err)
	}

	// Normalize version
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	versionDir := filepath.Join(downloader.cacheDir, version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Building binaries into cache (%s)...", versionDir)))

	for _, name := range requiredBinaries {
		if err := buildBinary(ctx, repoRoot, name, versionDir); err != nil {
			return err
		}
	}

	fmt.Println(styles.SuccessMessage("All binaries built successfully"))
	return nil
}

// buildBinary builds a single hatchet binary from source
func buildBinary(ctx context.Context, repoRoot, name, outputDir string) error {
	outputPath := filepath.Join(outputDir, name)
	pkg := fmt.Sprintf("./cmd/%s", name)

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Building %s...", name)))

	cmd := exec.CommandContext(ctx, "go", "build", "-o", outputPath, pkg)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build %s: %w", name, err)
	}

	fmt.Println(styles.SuccessMessage(fmt.Sprintf("Built %s", name)))
	return nil
}
