package local

import (
	"strings"
	"testing"
)

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "hatchet", true},
		{"valid with underscore", "hatchet_db", true},
		{"valid with numbers", "hatchet123", true},
		{"valid mixed", "my_db_123", true},
		{"valid uppercase", "Hatchet", true},
		{"valid single char", "a", true},
		{"empty string", "", false},
		{"starts with digit", "123hatchet", false},
		{"contains hyphen", "hatchet-db", false},
		{"contains space", "hatchet db", false},
		{"contains semicolon", "hatchet;drop", false},
		{"contains quotes", "hatchet\"test", false},
		{"too long", "a" + string(make([]byte, 63)), false},
		{"max length", string(make([]byte, 63)), false}, // 63 null bytes aren't valid chars
		{"exactly 63 valid chars", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetStateFilePath(t *testing.T) {
	path := GetStateFilePath()
	if path == "" {
		t.Error("GetStateFilePath() returned empty string")
	}
	if !strings.Contains(path, ".hatchet/local/state.json") {
		t.Errorf("GetStateFilePath() = %q, expected to contain .hatchet/local/state.json", path)
	}
}

func TestNewLocalDriver(t *testing.T) {
	driver := NewLocalDriver()
	if driver == nil {
		t.Fatal("NewLocalDriver() returned nil")
	}
	if driver.apiPort != DefaultAPIPort {
		t.Errorf("apiPort = %d, want %d", driver.apiPort, DefaultAPIPort)
	}
	if driver.grpcPort != DefaultGRPCPort {
		t.Errorf("grpcPort = %d, want %d", driver.grpcPort, DefaultGRPCPort)
	}
	if driver.configDir == "" {
		t.Error("configDir is empty")
	}
}

func TestLocalOpts(t *testing.T) {
	opts := &LocalOpts{}

	WithDatabaseURL("postgresql://localhost/test")(opts)
	if opts.DatabaseURL != "postgresql://localhost/test" {
		t.Errorf("DatabaseURL = %q, want postgresql://localhost/test", opts.DatabaseURL)
	}

	WithAPIPort(9000)(opts)
	if opts.APIPort != 9000 {
		t.Errorf("APIPort = %d, want 9000", opts.APIPort)
	}

	WithGRPCPort(9001)(opts)
	if opts.GRPCPort != 9001 {
		t.Errorf("GRPCPort = %d, want 9001", opts.GRPCPort)
	}

	WithHealthcheckPort(9002)(opts)
	if opts.HealthcheckPort != 9002 {
		t.Errorf("HealthcheckPort = %d, want 9002", opts.HealthcheckPort)
	}

	WithProfileName("test-profile")(opts)
	if opts.ProfileName != "test-profile" {
		t.Errorf("ProfileName = %q, want test-profile", opts.ProfileName)
	}
}
