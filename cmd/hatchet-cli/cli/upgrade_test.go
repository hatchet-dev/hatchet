package cli

import "testing"

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

	if got := detectInstallMethod("/opt/homebrew/Caskroom/hatchet/0.114.2/hatchet"); got != "homebrew" {
		t.Errorf("Caskroom path detected as %q", got)
	}
	if got := detectInstallMethod("/usr/local/Cellar/hatchet/0.114.2/bin/hatchet"); got != "homebrew" {
		t.Errorf("Cellar path detected as %q", got)
	}
	if got := detectInstallMethod("/Users/dev/.local/bin/hatchet"); got != "script" {
		t.Errorf("script path detected as %q", got)
	}
}
