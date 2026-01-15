//go:build e2e_cli

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	cliconfig "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/worker"
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// Test matrix of all language and package manager combinations
var templateTests = []struct {
	language       string
	packageManager string
}{
	// Python
	{"python", "poetry"},
	{"python", "uv"},
	{"python", "pip"},

	// TypeScript
	{"typescript", "npm"},
	{"typescript", "pnpm"},
	{"typescript", "yarn"},
	{"typescript", "bun"},

	// Go
	{"go", "go"},
}

func TestQuickstartTemplates(t *testing.T) {
	for _, tt := range templateTests {
		t.Run(fmt.Sprintf("%s_%s", tt.language, tt.packageManager), func(t *testing.T) {
			testTemplate(t, tt.language, tt.packageManager)
		})
	}
}

func testTemplate(t *testing.T, language, packageManager string) {
	// 1. Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	projectName := "test-project"

	t.Logf("Testing %s with %s in %s", language, packageManager, projectDir)

	// 2. Generate quickstart project using the CLI implementation
	_, err := GenerateQuickstart(language, packageManager, projectName, projectDir)
	if err != nil {
		t.Fatalf("quickstart generation failed: %v", err)
	}

	t.Logf("Project generated successfully")

	// 3. Verify project structure
	if err := verifyProjectStructure(t, projectDir, language, packageManager); err != nil {
		t.Fatalf("Project structure verification failed: %v", err)
	}

	// 4. Change to project directory and load worker config
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	workerConfig, err := worker.LoadWorkerConfig()
	if err != nil {
		t.Fatalf("Failed to load worker config: %v", err)
	}

	if workerConfig == nil {
		t.Fatal("Worker config is nil")
	}

	// 5. Get the local profile (created by hatchet server start)
	profile, err := cliconfig.GetProfile("local")
	if err != nil {
		t.Fatalf("Failed to get local profile: %v", err)
	}

	// 6. Start worker in dev mode using the CLI implementation and ensure it runs for 15 seconds without error
	t.Log("Starting worker dev mode...")
	if err := testWorkerDev(t, workerConfig, profile); err != nil {
		t.Fatalf("Worker dev test failed: %v", err)
	}

	t.Logf("Successfully tested %s with %s", language, packageManager)
}

func testWorkerDev(t *testing.T, workerConfig *worker.WorkerConfig, profile *profileconfig.Profile) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Start the worker process using the CLI implementation in a goroutine
	errChan := make(chan error, 1)
	go func() {
		t.Log("Starting worker process using RunWorkerDev...")
		// Call the actual CLI implementation
		// Note: devConfig.Reload is set to false to avoid file watching in tests
		testDevConfig := workerConfig.Dev
		testDevConfig.Reload = false

		if err := RunWorkerDev(ctx, profile, &testDevConfig); err != nil {
			errChan <- fmt.Errorf("worker process failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Wait for 15 seconds, then cancel the context
	time.Sleep(15 * time.Second)
	cancel()

	// Check if any error occurred
	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-time.After(2 * time.Second):
		// Worker process should have stopped by now
		t.Log("Worker process cleanup timeout - continuing anyway")
	}

	t.Log("Worker ran successfully for 15+ seconds")
	return nil
}

func verifyProjectStructure(t *testing.T, projectDir, language, packageManager string) error {
	t.Logf("Verifying project structure for %s/%s", language, packageManager)

	// Common files that should exist
	commonFiles := []string{
		"README.md",
		"hatchet.yaml",
	}

	for _, file := range commonFiles {
		path := filepath.Join(projectDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("expected file %s does not exist", file)
		}
	}

	// Language-specific files
	switch language {
	case "python":
		pythonFiles := []string{
			"src/hatchet_client.py",
			"src/run.py",
			"src/worker.py",
			"src/workflows/first_workflow.py",
		}
		for _, file := range pythonFiles {
			path := filepath.Join(projectDir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("expected Python file %s does not exist", file)
			}
		}

		// Check for package manager specific files
		switch packageManager {
		case "poetry":
			if _, err := os.Stat(filepath.Join(projectDir, "pyproject.toml")); os.IsNotExist(err) {
				return fmt.Errorf("expected pyproject.toml for poetry")
			}
		case "uv":
			if _, err := os.Stat(filepath.Join(projectDir, "pyproject.toml")); os.IsNotExist(err) {
				return fmt.Errorf("expected pyproject.toml for uv")
			}
		case "pip":
			if _, err := os.Stat(filepath.Join(projectDir, "requirements.txt")); os.IsNotExist(err) {
				return fmt.Errorf("expected requirements.txt for pip")
			}
		}

	case "typescript":
		tsFiles := []string{
			"src/hatchet-client.ts",
			"src/run.ts",
			"src/worker.ts",
			"src/workflows/first-workflow.ts",
			"tsconfig.json",
			"package.json",
		}
		for _, file := range tsFiles {
			path := filepath.Join(projectDir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("expected TypeScript file %s does not exist", file)
			}
		}

	case "go":
		goFiles := []string{
			"cmd/worker/main.go",
			"cmd/run/main.go",
			"client/client.go",
			"workflows/first_workflow.go",
			"go.mod",
		}
		for _, file := range goFiles {
			path := filepath.Join(projectDir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("expected Go file %s does not exist", file)
			}
		}
	}

	return nil
}
