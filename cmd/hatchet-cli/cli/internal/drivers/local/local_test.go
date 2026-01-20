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

	// Test new options
	WithEmbeddedPostgres(true)(opts)
	if !opts.EmbeddedPostgres {
		t.Error("EmbeddedPostgres should be true")
	}

	WithPostgresPort(5434)(opts)
	if opts.PostgresPort != 5434 {
		t.Errorf("PostgresPort = %d, want 5434", opts.PostgresPort)
	}

	WithExecutionMode(ExecutionModeSubprocess)(opts)
	if opts.ExecutionMode != ExecutionModeSubprocess {
		t.Errorf("ExecutionMode = %q, want %q", opts.ExecutionMode, ExecutionModeSubprocess)
	}

	WithBinaryVersion("v0.73.10")(opts)
	if opts.BinaryVersion != "v0.73.10" {
		t.Errorf("BinaryVersion = %q, want v0.73.10", opts.BinaryVersion)
	}
}

func TestNewEmbeddedPostgres(t *testing.T) {
	cfg := PostgresConfig{
		Port: 5434,
	}

	ep, err := NewEmbeddedPostgres(cfg)
	if err != nil {
		t.Fatalf("NewEmbeddedPostgres() error = %v", err)
	}

	if ep.Port() != 5434 {
		t.Errorf("Port() = %d, want 5434", ep.Port())
	}

	if ep.DataPath() == "" {
		t.Error("DataPath() returned empty string")
	}

	if ep.BinPath() == "" {
		t.Error("BinPath() returned empty string")
	}

	if ep.IsRunning() {
		t.Error("IsRunning() should be false before Start()")
	}
}

func TestEmbeddedPostgresConnectionURL(t *testing.T) {
	cfg := PostgresConfig{
		Port: 5433,
	}

	ep, err := NewEmbeddedPostgres(cfg)
	if err != nil {
		t.Fatalf("NewEmbeddedPostgres() error = %v", err)
	}

	url := ep.ConnectionURL("hatchet")
	expected := "postgresql://postgres:postgres@localhost:5433/hatchet?sslmode=disable"
	if url != expected {
		t.Errorf("ConnectionURL() = %q, want %q", url, expected)
	}

	// Test with empty database name
	url = ep.ConnectionURL("")
	expected = "postgresql://postgres:postgres@localhost:5433/hatchet?sslmode=disable"
	if url != expected {
		t.Errorf("ConnectionURL(\"\") = %q, want %q", url, expected)
	}
}

func TestNewBinaryDownloader(t *testing.T) {
	bd, err := NewBinaryDownloader()
	if err != nil {
		t.Fatalf("NewBinaryDownloader() error = %v", err)
	}

	if bd.CacheDir() == "" {
		t.Error("CacheDir() returned empty string")
	}

	if !strings.Contains(bd.CacheDir(), "hatchet") {
		t.Errorf("CacheDir() = %q, expected to contain 'hatchet'", bd.CacheDir())
	}
}

func TestBinaryDownloaderGetArchiveName(t *testing.T) {
	bd, err := NewBinaryDownloader()
	if err != nil {
		t.Fatalf("NewBinaryDownloader() error = %v", err)
	}

	name := bd.getArchiveName("hatchet-api", "v0.73.10")

	// Check the name follows the expected format
	if !strings.HasPrefix(name, "hatchet-api_v0.73.10_") {
		t.Errorf("getArchiveName() = %q, expected to start with 'hatchet-api_v0.73.10_'", name)
	}

	if !strings.HasSuffix(name, ".tar.gz") {
		t.Errorf("getArchiveName() = %q, expected to end with '.tar.gz'", name)
	}
}

func TestNewSubprocessManager(t *testing.T) {
	sm := NewSubprocessManager()
	if sm == nil {
		t.Fatal("NewSubprocessManager() returned nil")
	}

	if sm.IsRunning() {
		t.Error("IsRunning() should be false for new manager")
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"linux", "Linux"},
		{"darwin", "Darwin"},
		{"windows", "Windows"},
		{"", ""},
		{"L", "L"},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capitalizeFirst(tt.input)
			if result != tt.expected {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
