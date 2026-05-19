package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// imagePullProgress represents the JSON structure from Docker's ImagePull API
type imagePullProgress struct {
	ProgressDetail *imagePullProgressDetail `json:"progressDetail"`
	Status         string                   `json:"status"`
	ID             string                   `json:"id"`
	Progress       string                   `json:"progress"`
}

type imagePullProgressDetail struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

// progressMsg is sent when progress updates
type progressMsg struct {
	status  string
	percent float64
}

// doneMsg is sent when pulling is complete
type doneMsg struct{}

// model holds the bubbletea model for the progress bar
type model struct {
	imageName    string
	currentState string
	progress     progress.Model
	done         bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = min(msg.Width-4, 80)
		return m, nil

	case progressMsg:
		if msg.percent >= 1.0 {
			m.done = true
			return m, tea.Quit
		}
		m.currentState = msg.status
		return m, m.progress.SetPercent(msg.percent)

	case doneMsg:
		m.done = true
		return m, tea.Quit

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.done {
		// Use Hatchet blue for success
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3392FF")).Bold(true)
		return successStyle.Render(fmt.Sprintf("✓ Pulled image %s\n", m.imageName))
	}

	pad := strings.Repeat(" ", 2)
	// Use Hatchet muted cyan for status text
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("#A5C5E9")).Render(m.currentState)
	return "\n" + pad + m.progress.View() + "\n" + pad + status + "\n"
}

// displayImagePullProgress displays progress information while pulling a Docker image.
// It is panic-safe and will not crash if there are any errors parsing the progress stream.
//
// Note: Docker's image pull API doesn't provide total layer count or size upfront.
// Layers are discovered progressively during the pull, so we track the maximum progress
// seen to ensure the progress bar only moves forward, even as new layers are discovered.
func displayImagePullProgress(reader io.Reader, imageName string) {
	// Ensure we recover from any panics to avoid crashing the application
	defer func() {
		if r := recover(); r != nil {
			// Silently recover - progress display is not critical
			_ = r
		}
	}()

	prog := progress.New(
		progress.WithScaledGradient("#3392FF", "#B8D9FF"), // Blue to Cyan
		progress.WithWidth(80),
		progress.WithoutPercentage(),
	)

	m := model{
		progress:     prog,
		imageName:    imageName,
		currentState: fmt.Sprintf("Pulling %s...", imageName),
	}

	p := tea.NewProgram(m)

	// Channel to signal when the goroutine has finished consuming the reader
	done := make(chan struct{})

	// Start parsing in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Silently recover from parsing errors
				_ = r
			}
			p.Send(doneMsg{})
			close(done) // Signal that we're done consuming the reader
		}()

		scanner := bufio.NewScanner(reader)
		layerProgress := make(map[string]*imagePullProgressDetail)
		layerStates := make(map[string]string)
		maxPercent := 0.0 // Track the maximum progress seen to ensure we only move forward

		for scanner.Scan() {
			line := scanner.Bytes()

			var progress imagePullProgress
			if err := json.Unmarshal(line, &progress); err != nil {
				continue
			}

			// Track layer states
			if progress.ID != "" {
				layerStates[progress.ID] = progress.Status

				if progress.ProgressDetail != nil && progress.ProgressDetail.Total > 0 {
					layerProgress[progress.ID] = progress.ProgressDetail
				}
			}

			// Calculate overall progress
			percent, status := calculateProgress(layerProgress, layerStates)

			// Only move forward - never backwards
			if percent > maxPercent {
				maxPercent = percent
				p.Send(progressMsg{
					percent: percent,
					status:  status,
				})
			} else {
				// Still update status even if percentage hasn't increased
				p.Send(progressMsg{
					percent: maxPercent,
					status:  status,
				})
			}

			// Small delay to avoid overwhelming the UI
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Run the program (this blocks until done)
	if _, err := p.Run(); err != nil {
		// Fallback for non-TTY environments: wait for the goroutine to finish
		// consuming the reader before returning to ensure the image pull completes
		<-done
		fmt.Fprintf(os.Stderr, "Pulled image %s\n", imageName)
	} else {
		// Even on success, wait for the goroutine to clean up
		<-done
	}
}

// calculateProgress computes the overall progress percentage and status message
func calculateProgress(layerProgress map[string]*imagePullProgressDetail, layerStates map[string]string) (float64, string) {
	if len(layerStates) == 0 {
		return 0, "Starting..."
	}

	// Count states and track cached layers
	statusCounts := make(map[string]int)
	cachedLayers := 0

	for _, status := range layerStates {
		normalized := normalizeStatus(status)
		statusCounts[normalized]++

		// Track layers that are cached (no download needed)
		if strings.Contains(strings.ToLower(status), "already exists") {
			cachedLayers++
		}
	}

	// Build status message
	var parts []string
	if count := statusCounts["downloading"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d downloading", count))
	}
	if count := statusCounts["extracting"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d extracting", count))
	}
	if count := statusCounts["complete"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d complete", count))
	}

	statusMsg := strings.Join(parts, ", ")
	if statusMsg == "" {
		statusMsg = "Processing layers..."
	}

	// Calculate weighted progress
	// Strategy: Only count layers that need downloading (have byte data or are actively downloading)
	// Ignore cached layers in the denominator since they don't contribute to download time
	var totalBytes int64
	var currentBytes int64
	activeLayerCount := 0

	for _, detail := range layerProgress {
		if detail.Total > 0 {
			totalBytes += detail.Total
			currentBytes += detail.Current
			activeLayerCount++
		}
	}

	var percent float64
	switch {
	case totalBytes > 0:
		// Use byte-based progress for layers being downloaded
		percent = float64(currentBytes) / float64(totalBytes)
	case activeLayerCount == 0 && cachedLayers > 0:
		// All layers are cached - show near complete
		percent = 0.95
	default:
		// Fallback: base on completed non-cached layers
		totalActiveLayers := len(layerStates) - cachedLayers
		completedLayers := statusCounts["complete"] - cachedLayers
		if totalActiveLayers > 0 {
			percent = float64(completedLayers) / float64(totalActiveLayers)
		}
	}

	// Cap at 0.99 until we receive the done message
	if percent >= 1.0 {
		percent = 0.99
	}

	return percent, statusMsg
}

// normalizeStatus converts various Docker status messages to simplified categories
func normalizeStatus(status string) string {
	status = strings.ToLower(status)
	switch {
	case strings.Contains(status, "download"):
		return "downloading"
	case strings.Contains(status, "extract"):
		return "extracting"
	case strings.Contains(status, "pull complete"):
		return "complete"
	case strings.Contains(status, "already exists"):
		return "complete"
	default:
		return "processing"
	}
}
