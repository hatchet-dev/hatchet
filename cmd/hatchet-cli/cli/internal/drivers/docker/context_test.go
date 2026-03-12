package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDockerHostFromContext(t *testing.T) {
	// save original HOME and DOCKER_HOST, restore after test
	origHome := os.Getenv("HOME")
	origDockerHost := os.Getenv("DOCKER_HOST")
	defer func() {
		os.Setenv("HOME", origHome)
		if origDockerHost != "" {
			os.Setenv("DOCKER_HOST", origDockerHost)
		} else {
			os.Unsetenv("DOCKER_HOST")
		}
	}()

	// ensure DOCKER_HOST is not set for these tests
	os.Unsetenv("DOCKER_HOST")

	t.Run("no docker config", func(t *testing.T) {
		// Use a temp dir with no .docker folder
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)

		result := resolveDockerHostFromContext()
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("empty currentContext", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)

		// create .docker/config.json with no currentContext
		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		configContent := `{"auths": {}}`
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := resolveDockerHostFromContext()
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("default context", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)

		// create .docker/config.json with currentContext = "default"
		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		configContent := `{"auths": {}, "currentContext": "default"}`
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := resolveDockerHostFromContext()
		if result != "" {
			t.Errorf("expected empty string for default context, got %q", result)
		}
	})

	t.Run("non-default context with valid metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)

		contextName := "my-rancher"
		expectedHost := "unix:///tmp/rancher.sock"

		// create .docker/config.json
		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		configContent := `{"auths": {}, "currentContext": "` + contextName + `"}`
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		// create context metadata
		hash := sha256.Sum256([]byte(contextName))
		hashStr := hex.EncodeToString(hash[:])
		metaDir := filepath.Join(dockerDir, "contexts", "meta", hashStr)
		if err := os.MkdirAll(metaDir, 0755); err != nil {
			t.Fatal(err)
		}
		metaContent := `{"Name":"` + contextName + `","Metadata":{},"Endpoints":{"docker":{"Host":"` + expectedHost + `","SkipTLSVerify":false}}}`
		if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(metaContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := resolveDockerHostFromContext()
		if result != expectedHost {
			t.Errorf("expected %q, got %q", expectedHost, result)
		}
	})

	t.Run("non-default context with missing metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)

		// create .docker/config.json with non-default context but no metadata
		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		configContent := `{"auths": {}, "currentContext": "missing-context"}`
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := resolveDockerHostFromContext()
		if result != "" {
			t.Errorf("expected empty string for missing metadata, got %q", result)
		}
	})

	t.Run("invalid config json", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)

		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Invalid JSON
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte("{invalid}"), 0644); err != nil {
			t.Fatal(err)
		}

		result := resolveDockerHostFromContext()
		if result != "" {
			t.Errorf("expected empty string for invalid JSON, got %q", result)
		}
	})

	t.Run("tcp host endpoint", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("HOME", tmpDir)

		contextName := "remote-docker"
		expectedHost := "tcp://192.168.1.100:2375"

		// create .docker/config.json
		dockerDir := filepath.Join(tmpDir, ".docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatal(err)
		}
		configContent := `{"auths": {}, "currentContext": "` + contextName + `"}`
		if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		// create context metadata with TCP endpoint
		hash := sha256.Sum256([]byte(contextName))
		hashStr := hex.EncodeToString(hash[:])
		metaDir := filepath.Join(dockerDir, "contexts", "meta", hashStr)
		if err := os.MkdirAll(metaDir, 0755); err != nil {
			t.Fatal(err)
		}
		metaContent := `{"Name":"` + contextName + `","Metadata":{},"Endpoints":{"docker":{"Host":"` + expectedHost + `","SkipTLSVerify":false}}}`
		if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(metaContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := resolveDockerHostFromContext()
		if result != expectedHost {
			t.Errorf("expected %q, got %q", expectedHost, result)
		}
	})
}
