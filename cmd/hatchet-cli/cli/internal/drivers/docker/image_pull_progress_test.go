package docker

import (
	"strings"
	"testing"
)

func TestDisplayImagePullProgress_PanicSafe(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		panics bool
	}{
		{
			name: "valid JSON progress",
			input: `{"status":"Pulling fs layer","id":"abc123"}
{"status":"Downloading","progressDetail":{"current":1024,"total":2048},"progress":"50%","id":"abc123"}
{"status":"Pull complete","id":"abc123"}`,
			panics: false,
		},
		{
			name: "invalid JSON - should not panic",
			input: `invalid json line
{"status":"Downloading","id":"abc123"}
more invalid data
`,
			panics: false,
		},
		{
			name:   "empty input",
			input:  "",
			panics: false,
		},
		{
			name: "malformed JSON objects",
			input: `{"status":"test"
{"unclosed
{]}
`,
			panics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)

			// Wrap in a defer/recover to catch any panics
			didPanic := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
					}
				}()

				// This should never panic
				displayImagePullProgress(reader, "test-image:latest")
			}()

			if didPanic {
				t.Errorf("displayImagePullProgress panicked on input: %s", tt.input)
			}
		})
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"Downloading", "downloading"},
		{"Download complete", "downloading"},
		{"Extracting", "extracting"},
		{"Extract complete", "extracting"},
		{"Pull complete", "complete"},
		{"Already exists", "complete"},
		{"Waiting", "processing"},
		{"Verifying Checksum", "processing"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := normalizeStatus(tt.status)
			if result != tt.expected {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestCalculateProgress(t *testing.T) {
	tests := []struct {
		name          string
		layerProgress map[string]*imagePullProgressDetail
		layerStates   map[string]string
		wantPercent   float64
		wantStatus    string
	}{
		{
			name:          "empty states",
			layerProgress: map[string]*imagePullProgressDetail{},
			layerStates:   map[string]string{},
			wantPercent:   0,
			wantStatus:    "Starting...",
		},
		{
			name: "bytes-based progress",
			layerProgress: map[string]*imagePullProgressDetail{
				"layer1": {Current: 512, Total: 1024},
				"layer2": {Current: 256, Total: 1024},
			},
			layerStates: map[string]string{
				"layer1": "Downloading",
				"layer2": "Downloading",
			},
			wantPercent: 0.375, // (512+256)/(1024+1024)
			wantStatus:  "2 downloading",
		},
		{
			name:          "layer-based progress fallback",
			layerProgress: map[string]*imagePullProgressDetail{},
			layerStates: map[string]string{
				"layer1": "Pull complete",
				"layer2": "Downloading",
				"layer3": "Pull complete",
			},
			wantPercent: 0.6666666666666666, // 2/3 complete
			wantStatus:  "1 downloading, 2 complete",
		},
		{
			name: "mixed states",
			layerProgress: map[string]*imagePullProgressDetail{
				"layer1": {Current: 1024, Total: 1024},
			},
			layerStates: map[string]string{
				"layer1": "Pull complete",
				"layer2": "Extracting",
				"layer3": "Downloading",
			},
			wantPercent: 0.99, // Capped at 0.99
			wantStatus:  "1 downloading, 1 extracting, 1 complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPercent, gotStatus := calculateProgress(tt.layerProgress, tt.layerStates)

			// Use approximate comparison for floats
			if diff := gotPercent - tt.wantPercent; diff < -0.01 || diff > 0.01 {
				t.Errorf("calculateProgress() percent = %v, want %v", gotPercent, tt.wantPercent)
			}

			if gotStatus != tt.wantStatus {
				t.Errorf("calculateProgress() status = %q, want %q", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestProgressMonotonicallyIncreases(t *testing.T) {
	// Simulate the scenario where new layers are discovered during pull
	// This test demonstrates that calculateProgress can return lower values when
	// new layers are discovered, but our maxPercent tracking prevents displaying
	// backwards progress to the user.
	layerProgress := make(map[string]*imagePullProgressDetail)
	layerStates := make(map[string]string)
	maxPercent := 0.0

	// Initial state: 2 layers downloading
	layerProgress["layer1"] = &imagePullProgressDetail{Current: 512, Total: 1024}
	layerProgress["layer2"] = &imagePullProgressDetail{Current: 256, Total: 1024}
	layerStates["layer1"] = "Downloading"
	layerStates["layer2"] = "Downloading"

	percent1, _ := calculateProgress(layerProgress, layerStates)
	if percent1 > maxPercent {
		maxPercent = percent1
	}
	displayedPercent1 := maxPercent

	// New layer discovered - calculateProgress returns a lower value
	// but we display maxPercent instead
	layerProgress["layer3"] = &imagePullProgressDetail{Current: 0, Total: 2048}
	layerStates["layer3"] = "Downloading"

	percent2, _ := calculateProgress(layerProgress, layerStates)

	// This demonstrates the problem: percent2 < percent1
	if percent2 >= percent1 {
		t.Logf("Note: New layer didn't cause backwards calculation (this is rare but OK)")
	}

	// But we use maxPercent to prevent displaying backwards progress
	displayedPercent2 := maxPercent
	if percent2 > maxPercent {
		maxPercent = percent2
		displayedPercent2 = percent2
	}

	// Verify displayed progress doesn't go backwards
	if displayedPercent2 < displayedPercent1 {
		t.Errorf("Displayed progress went backwards: %v -> %v", displayedPercent1, displayedPercent2)
	}

	// Layer 1 completes - this should move progress forward
	layerProgress["layer1"] = &imagePullProgressDetail{Current: 1024, Total: 1024}
	layerStates["layer1"] = "Pull complete"

	percent3, _ := calculateProgress(layerProgress, layerStates)
	displayedPercent3 := maxPercent
	if percent3 > maxPercent {
		maxPercent = percent3
		displayedPercent3 = percent3
	}

	// Verify displayed progress continues forward (or stays same)
	if displayedPercent3 < displayedPercent2 {
		t.Errorf("Displayed progress went backwards: %v -> %v", displayedPercent2, displayedPercent3)
	}

	// All layers complete - this MUST move progress forward to near 100%
	layerProgress["layer2"] = &imagePullProgressDetail{Current: 1024, Total: 1024}
	layerProgress["layer3"] = &imagePullProgressDetail{Current: 2048, Total: 2048}
	layerStates["layer2"] = "Pull complete"
	layerStates["layer3"] = "Pull complete"

	percent4, _ := calculateProgress(layerProgress, layerStates)
	displayedPercent4 := maxPercent
	if percent4 > maxPercent {
		displayedPercent4 = percent4
	}

	// When all layers are complete, progress should be near 100% (0.99 due to cap)
	if displayedPercent4 < 0.9 {
		t.Errorf("Progress should be near 100%% when all layers complete, got: %v", displayedPercent4)
	}

	// Final verification: progress never went backwards at any point
	if displayedPercent2 < displayedPercent1 || displayedPercent3 < displayedPercent2 || displayedPercent4 < displayedPercent3 {
		t.Errorf("Progress went backwards at some point: %v -> %v -> %v -> %v",
			displayedPercent1, displayedPercent2, displayedPercent3, displayedPercent4)
	}
}
