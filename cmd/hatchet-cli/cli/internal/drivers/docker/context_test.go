package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	dockercontext "github.com/docker/go-sdk/context"
)

func TestDockerContextResolution(t *testing.T) {
	t.Run("no config falls back to default host", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		t.Setenv("USERPROFILE", tmpDir)
		t.Setenv("DOCKER_HOST", "")
		t.Setenv("DOCKER_CONTEXT", "")

		driver, err := NewDockerDriver(context.Background())
		if err != nil {
			t.Fatalf("expected driver to initialize, got error: %v", err)
		}
		defer driver.apiClient.Close()

		host := driver.apiClient.DaemonHost()
		if host == "" {
			t.Error("expected default Docker host, got empty string")
		}
	})

	t.Run("default context returns default host", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		t.Setenv("USERPROFILE", tmpDir)
		t.Setenv("DOCKER_HOST", "")
		t.Setenv("DOCKER_CONTEXT", "")

		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}

		driver, err := NewDockerDriver(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer driver.apiClient.Close()

		host := driver.apiClient.DaemonHost()
		if host == "" {
			t.Error("expected default Docker host, got empty string")
		}
	})

	t.Run("non-default context resolves host", func(t *testing.T) {
		dockercontext.SetupTestDockerContexts(t, 1, 1)
		t.Setenv("DOCKER_HOST", "")
		t.Setenv("DOCKER_CONTEXT", "")

		driver, err := NewDockerDriver(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer driver.apiClient.Close()

		host := driver.apiClient.DaemonHost()
		if host != "tcp://127.0.0.1:1" {
			t.Errorf("expected tcp://127.0.0.1:1, got %q", host)
		}
	})

	t.Run("DOCKER_HOST takes precedence", func(t *testing.T) {
		dockercontext.SetupTestDockerContexts(t, 1, 1)
		t.Setenv("DOCKER_HOST", "tcp://192.168.1.100:2375")

		driver, err := NewDockerDriver(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer driver.apiClient.Close()

		host := driver.apiClient.DaemonHost()
		if host != "tcp://192.168.1.100:2375" {
			t.Errorf("expected tcp://192.168.1.100:2375, got %q", host)
		}
	})

	t.Run("DOCKER_CONTEXT overrides config", func(t *testing.T) {
		dockercontext.SetupTestDockerContexts(t, 1, 2)
		t.Setenv("DOCKER_HOST", "")
		t.Setenv("DOCKER_CONTEXT", "context2")

		driver, err := NewDockerDriver(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer driver.apiClient.Close()

		host := driver.apiClient.DaemonHost()
		if host != "tcp://127.0.0.1:2" {
			t.Errorf("expected tcp://127.0.0.1:2, got %q", host)
		}
	})

	t.Run("missing context falls back to default host", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		t.Setenv("USERPROFILE", tmpDir)
		t.Setenv("DOCKER_HOST", "")
		t.Setenv("DOCKER_CONTEXT", "")

		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(`{"currentContext": "nonexistent"}`), 0644); err != nil {
			t.Fatal(err)
		}

		driver, err := NewDockerDriver(context.Background())
		if err != nil {
			t.Fatalf("expected driver to initialize, got error: %v", err)
		}
		defer driver.apiClient.Close()

		host := driver.apiClient.DaemonHost()
		if host == "" {
			t.Error("expected default Docker host, got empty string")
		}
	})

	t.Run("invalid config falls back to default host", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		t.Setenv("USERPROFILE", tmpDir)
		t.Setenv("DOCKER_HOST", "")
		t.Setenv("DOCKER_CONTEXT", "")

		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte("{invalid}"), 0644); err != nil {
			t.Fatal(err)
		}

		driver, err := NewDockerDriver(context.Background())
		if err != nil {
			t.Fatalf("expected driver to initialize, got error: %v", err)
		}
		defer driver.apiClient.Close()

		host := driver.apiClient.DaemonHost()
		if host == "" {
			t.Error("expected default Docker host, got empty string")
		}
	})
}
