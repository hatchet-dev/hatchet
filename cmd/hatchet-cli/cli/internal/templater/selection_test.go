package templater

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	quickstarts "github.com/hatchet-dev/hatchet-quickstarts"
)

func TestUseCases(t *testing.T) {
	useCases, err := UseCases(quickstarts.TemplatesFS())
	if err != nil {
		t.Fatalf("UseCases returned an error: %v", err)
	}

	if len(useCases) == 0 || useCases[0] != DefaultUseCase {
		t.Fatalf("expected %q first, got %v", DefaultUseCase, useCases)
	}

	found := false
	for _, useCase := range useCases {
		if useCase == "scheduled" {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected use case %q in %v", "scheduled", useCases)
	}
}

func TestLanguagesFor(t *testing.T) {
	fsys := quickstarts.TemplatesFS()

	simple, err := LanguagesFor(fsys, DefaultUseCase)
	if err != nil {
		t.Fatalf("LanguagesFor(simple) returned an error: %v", err)
	}

	if !reflect.DeepEqual(simple, []string{"python", "typescript", "go"}) {
		t.Errorf("expected all three languages for simple, got %v", simple)
	}

	scheduled, err := LanguagesFor(fsys, "scheduled")
	if err != nil {
		t.Fatalf("LanguagesFor(scheduled) returned an error: %v", err)
	}

	if !reflect.DeepEqual(scheduled, []string{"go"}) {
		t.Errorf("expected only go for scheduled, got %v", scheduled)
	}
}

func TestValidate(t *testing.T) {
	fsys := quickstarts.TemplatesFS()

	valid := []Selection{
		{UseCase: "simple", Language: "python", PackageManager: "poetry"},
		{UseCase: "", Language: "typescript", PackageManager: "pnpm"},
		{UseCase: "scheduled", Language: "go", PackageManager: "go"},
	}

	for _, sel := range valid {
		if err := Validate(fsys, sel); err != nil {
			t.Errorf("expected %+v to validate, got: %v", sel, err)
		}
	}

	invalid := []struct {
		sel     Selection
		wantErr string
	}{
		{Selection{UseCase: "nonexistent", Language: "go", PackageManager: "go"}, "unknown use case"},
		{Selection{UseCase: "scheduled", Language: "python", PackageManager: "poetry"}, "does not support language"},
		{Selection{UseCase: "simple", Language: "rust", PackageManager: "cargo"}, "invalid language"},
		{Selection{UseCase: "simple", Language: "go", PackageManager: "npm"}, "invalid package manager"},
	}

	for _, tc := range invalid {
		err := Validate(fsys, tc.sel)
		if err == nil {
			t.Errorf("expected %+v to fail validation", tc.sel)
			continue
		}

		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("expected error for %+v to contain %q, got: %v", tc.sel, tc.wantErr, err)
		}
	}
}

func TestProcessMultiSourceScheduledGo(t *testing.T) {
	dstDir := filepath.Join(t.TempDir(), "project")
	sel := Selection{UseCase: "scheduled", Language: "go", PackageManager: "go"}
	data := Data{Name: "test-project", PackageManager: "go"}

	if err := ProcessMultiSource(quickstarts.TemplatesFS(), sel, dstDir, data); err != nil {
		t.Fatalf("ProcessMultiSource failed: %v", err)
	}

	expected := []string{
		"cmd/worker/main.go",
		"cmd/run/main.go",
		"client/client.go",
		"workflows/scheduled_workflow.go",
		"go.mod",
		"hatchet.yaml",
	}

	for _, file := range expected {
		if _, err := os.Stat(filepath.Join(dstDir, file)); err != nil {
			t.Errorf("expected file %s: %v", file, err)
		}
	}

	if _, err := os.Stat(filepath.Join(dstDir, "POST_QUICKSTART.md")); err == nil {
		t.Error("POST_QUICKSTART.md must not be copied into the project")
	}

	goMod, err := os.ReadFile(filepath.Join(dstDir, "go.mod"))
	if err != nil {
		t.Fatalf("could not read generated go.mod: %v", err)
	}

	// The go templates use a fixed module path, so the project name is not in
	// go.mod. Assert the hatchet SDK requirement instead.
	if !strings.Contains(string(goMod), "github.com/hatchet-dev/hatchet ") {
		t.Errorf("expected go.mod to require the hatchet SDK, got:\n%s", goMod)
	}
}

func TestProcessMultiSourceSimpleGoUnchanged(t *testing.T) {
	dstDir := filepath.Join(t.TempDir(), "project")
	sel := Selection{UseCase: "", Language: "go", PackageManager: "go"}
	data := Data{Name: "test-project", PackageManager: "go"}

	if err := ProcessMultiSource(quickstarts.TemplatesFS(), sel, dstDir, data); err != nil {
		t.Fatalf("ProcessMultiSource failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dstDir, "workflows/first_workflow.go")); err != nil {
		t.Errorf("expected the default go template's workflow file: %v", err)
	}
}

func TestProcessPostQuickstartMultiSourceScheduled(t *testing.T) {
	sel := Selection{UseCase: "scheduled", Language: "go", PackageManager: "go"}
	data := Data{Name: "test-project", PackageManager: "go"}

	content, err := ProcessPostQuickstartMultiSource(quickstarts.TemplatesFS(), sel, data)
	if err != nil {
		t.Fatalf("ProcessPostQuickstartMultiSource failed: %v", err)
	}

	if !strings.Contains(content, "manual-run") {
		t.Errorf("expected the scheduled post-quickstart content to mention the manual-run trigger, got:\n%s", content)
	}
}
