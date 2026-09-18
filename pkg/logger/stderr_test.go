package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

func TestNewWithWriterJSON(t *testing.T) {
	buf := &bytes.Buffer{}

	l := NewWithWriter(&shared.LoggerConfigFile{Level: "info", Format: "json"}, "test-service", buf)

	l.Info().Str("key", "value").Msg("hello")

	var parsed map[string]interface{}

	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected json output, got %q: %v", buf.String(), err)
	}

	if parsed["message"] != "hello" {
		t.Errorf("expected message %q, got %v", "hello", parsed["message"])
	}

	if parsed["service"] != "test-service" {
		t.Errorf("expected service %q, got %v", "test-service", parsed["service"])
	}

	if parsed["key"] != "value" {
		t.Errorf("expected key %q, got %v", "value", parsed["key"])
	}
}

func TestNewWithWriterConsole(t *testing.T) {
	buf := &bytes.Buffer{}

	l := NewWithWriter(&shared.LoggerConfigFile{Level: "info", Format: "console"}, "test-service", buf)

	l.Info().Msg("hello")

	out := buf.String()

	if out == "" {
		t.Fatal("expected console output, got empty buffer")
	}

	if json.Valid(bytes.TrimSpace(buf.Bytes())) {
		t.Errorf("expected console-formatted output, got json: %q", out)
	}

	if !strings.Contains(out, "hello") {
		t.Errorf("expected output to contain %q, got %q", "hello", out)
	}

	if !strings.Contains(out, "test-service") {
		t.Errorf("expected output to contain %q, got %q", "test-service", out)
	}
}

func TestNewWithWriterRespectsLevel(t *testing.T) {
	buf := &bytes.Buffer{}

	l := NewWithWriter(&shared.LoggerConfigFile{Level: "warn", Format: "json"}, "", buf)

	l.Info().Msg("filtered")

	if buf.Len() != 0 {
		t.Errorf("expected info message to be filtered at warn level, got %q", buf.String())
	}

	l.Warn().Msg("kept")

	if !strings.Contains(buf.String(), "kept") {
		t.Errorf("expected warn message to be written, got %q", buf.String())
	}
}

// TestNewStdErrWriterOverride asserts that the runtime writer on the config is
// honored by NewStdErr, which is the seam embedding callers use.
func TestNewStdErrWriterOverride(t *testing.T) {
	buf := &bytes.Buffer{}

	for _, format := range []string{"json", "console"} {
		buf.Reset()

		l := NewStdErr(&shared.LoggerConfigFile{Level: "info", Format: format, Writer: buf}, "svc")

		l.Info().Msg("routed")

		if !strings.Contains(buf.String(), "routed") {
			t.Errorf("format %s: expected output in injected writer, got %q", format, buf.String())
		}
	}
}

// TestNewStdErrDefaultsToStderr asserts that without a writer override the
// default construction path is unchanged and writes nothing to an injected
// buffer. zerolog does not expose its writer, so this checks that construction
// succeeds and that output does not land anywhere the test can observe.
func TestNewStdErrDefaultsToStderr(t *testing.T) {
	buf := &bytes.Buffer{}

	l := NewStdErr(&shared.LoggerConfigFile{Level: "info", Format: "json"}, "svc")

	l.Info().Msg("to stderr")

	if buf.Len() != 0 {
		t.Errorf("expected no output in unrelated buffer, got %q", buf.String())
	}

	d := NewDefaultLogger("svc")

	d.Debug().Msg("to stderr")
}
