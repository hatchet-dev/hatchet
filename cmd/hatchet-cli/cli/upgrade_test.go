package cli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUpgradeVersionAndInstallDetection(t *testing.T) {
	cases := []struct {
		latest, current string
		newer           bool
	}{
		{"v0.114.2", "v0.114.1", true},
		{"v0.114.2", "v0.114.2", false},
		{"v0.114.2", "v0.115.0", false},
		{"v1.0.0", "v0.114.2", true},
	}

	for _, c := range cases {
		got, err := isNewer(c.latest, c.current)
		if err != nil {
			t.Fatalf("isNewer(%s, %s): %v", c.latest, c.current, err)
		}
		if got != c.newer {
			t.Errorf("isNewer(%s, %s) = %v, want %v", c.latest, c.current, got, c.newer)
		}
	}

	if _, err := isNewer("not-a-version", "v0.1.0"); err == nil {
		t.Error("expected error for invalid latest version")
	}

	cask := "# generated\ncask \"hatchet\" do\n  version \"0.114.2\"\n\n  on_macos do\n"
	if got, err := parseCaskVersion(cask); err != nil || got != "v0.114.2" {
		t.Errorf("parseCaskVersion = %q, %v; want v0.114.2", got, err)
	}
	if _, err := parseCaskVersion("cask \"hatchet\" do\nend\n"); err == nil {
		t.Error("expected error for cask without a version")
	}

	ctx := context.Background()
	prefix := t.TempDir()
	if err := os.MkdirAll(filepath.Join(prefix, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prefix, "bin", "brew"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	brew, _, err := brewUpgradeCommand(ctx, filepath.Join(prefix, "Caskroom", "hatchet", "0.114.2", "hatchet"))
	if runtime.GOOS == "darwin" {
		if err != nil || brew == nil || brew.Path != filepath.Join(prefix, "bin", "brew") {
			t.Errorf("Caskroom install: cmd=%v err=%v, want brew at %s", brew, err, filepath.Join(prefix, "bin", "brew"))
		}
		if _, _, err := brewUpgradeCommand(ctx, "/opt/missing/Caskroom/hatchet/0.114.2/hatchet"); err == nil {
			t.Error("expected error when the Homebrew prefix has no brew binary")
		}
	} else if brew != nil || err != nil {
		t.Errorf("non-darwin brew command = %v, %v; want nil, nil", brew, err)
	}
	if brew, _, err := brewUpgradeCommand(ctx, "/home/dev/.local/bin/hatchet"); brew != nil || err != nil {
		t.Errorf("script install path: cmd=%v err=%v; want nil, nil", brew, err)
	}
}
