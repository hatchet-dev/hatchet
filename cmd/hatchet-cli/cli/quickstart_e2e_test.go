//go:build e2e_cli

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	// 7. Verify Dockerfile builds
	t.Log("Verifying Dockerfile builds...")
	if err := testDockerfileBuild(t, projectDir, language, packageManager); err != nil {
		t.Fatalf("Dockerfile build test failed: %v", err)
	}

	t.Logf("Successfully tested %s with %s", language, packageManager)
}

func testWorkerDev(t *testing.T, workerConfig *worker.WorkerConfig, profile *profileconfig.Profile) error {
	// Create a context with timeout (safety net)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Channel to signal when pre-commands complete
	preCmdsComplete := make(chan struct{}, 1)
	errChan := make(chan error, 1)

	// Start the worker process using the CLI implementation in a goroutine
	go func() {
		t.Log("Starting worker process using RunWorkerDev...")
		// Call the actual CLI implementation
		// Note: devConfig.Reload is set to false to avoid file watching in tests
		testDevConfig := workerConfig.Dev
		testDevConfig.Reload = false

		if err := RunWorkerDev(ctx, profile, &testDevConfig, preCmdsComplete); err != nil {
			errChan <- fmt.Errorf("worker process failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Wait for pre-commands to complete (dependency installation)
	select {
	case <-preCmdsComplete:
		t.Log("Pre-commands completed, worker starting...")
	case err := <-errChan:
		if err != nil {
			return err
		}
		return fmt.Errorf("worker exited before pre-commands completed")
	case <-time.After(4 * time.Minute):
		return fmt.Errorf("timeout waiting for pre-commands to complete")
	}

	// Wait 5 seconds for the worker to fully start, then trigger the workflow
	time.Sleep(5 * time.Second)

	// Trigger the "simple" workflow if it exists in the config
	if len(workerConfig.Triggers) > 0 {
		var simpleTrigger *worker.Trigger
		for i := range workerConfig.Triggers {
			if workerConfig.Triggers[i].Name == "simple" {
				simpleTrigger = &workerConfig.Triggers[i]
				break
			}
		}

		if simpleTrigger != nil {
			t.Logf("Triggering workflow using command: %s", simpleTrigger.Command)
			triggerCtx, triggerCancel := context.WithTimeout(ctx, 30*time.Second)
			if err := executeTriggerCommand(triggerCtx, simpleTrigger.Command, profile); err != nil {
				t.Logf("Warning: failed to trigger workflow: %v", err)
			} else {
				t.Log("Successfully triggered workflow")
			}
			triggerCancel()
		}
	}

	// Wait another 10 seconds for the workflow to process
	time.Sleep(10 * time.Second)
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

	t.Log("Worker ran successfully and workflow was triggered")
	return nil
}

func verifyProjectStructure(t *testing.T, projectDir, language, packageManager string) error {
	t.Logf("Verifying project structure for %s/%s", language, packageManager)

	// Common files that should exist
	commonFiles := []string{
		"README.md",
		"hatchet.yaml",
		"Dockerfile",
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

func testDockerfileBuild(t *testing.T, projectDir, language, packageManager string) error {
	// Verify Dockerfile exists
	dockerfilePath := filepath.Join(projectDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile does not exist at %s", dockerfilePath)
	}

	t.Logf("Building Dockerfile for %s/%s...", language, packageManager)

	// Build the Docker image
	// Use a unique tag for each test to avoid conflicts
	imageName := fmt.Sprintf("hatchet-test-%s-%s:latest", language, packageManager)

	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	cmd.Dir = projectDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %v\nOutput: %s", err, output)
	}

	t.Logf("Successfully built Docker image: %s", imageName)

	// Clean up the image after test
	cleanupCmd := exec.Command("docker", "rmi", imageName)
	if err := cleanupCmd.Run(); err != nil {
		t.Logf("Warning: failed to clean up Docker image %s: %v", imageName, err)
	}

	return nil
}

func executeTriggerCommand(ctx context.Context, command string, profile *profileconfig.Profile) error {
	// Use sh -c to execute the command with proper environment
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Set environment variables from profile
	env := os.Environ()
	env = append(env, fmt.Sprintf("HATCHET_CLIENT_TOKEN=%s", profile.Token))
	env = append(env, fmt.Sprintf("HATCHET_CLIENT_TLS_STRATEGY=%s", profile.TLSStrategy))
	if profile.GrpcHostPort != "" {
		env = append(env, fmt.Sprintf("HATCHET_CLIENT_HOST_PORT=%s", profile.GrpcHostPort))
	}
	cmd.Env = env

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
