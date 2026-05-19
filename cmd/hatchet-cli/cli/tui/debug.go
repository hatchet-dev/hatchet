package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DebugLog represents a single debug log entry
type DebugLog struct {
	Timestamp time.Time
	Message   string
}

// DebugLogger is a fixed-size ring buffer for debug logs with file writing support
type DebugLogger struct {
	fileInput        string
	statusMessage    string
	logs             []DebugLog
	capacity         int
	index            int
	size             int
	mu               sync.RWMutex
	promptingFile    bool
	confirmOverwrite bool
}

// NewDebugLogger creates a new debug logger with the specified capacity
func NewDebugLogger(capacity int) *DebugLogger {
	return &DebugLogger{
		logs:             make([]DebugLog, capacity),
		capacity:         capacity,
		index:            0,
		size:             0,
		promptingFile:    false,
		fileInput:        "",
		confirmOverwrite: false,
		statusMessage:    "",
	}
}

// Log adds a new log entry to the ring buffer
func (d *DebugLogger) Log(format string, args ...interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logs[d.index] = DebugLog{
		Timestamp: time.Now(),
		Message:   fmt.Sprintf(format, args...),
	}

	d.index = (d.index + 1) % d.capacity
	if d.size < d.capacity {
		d.size++
	}
}

// GetLogs returns all logs in chronological order
func (d *DebugLogger) GetLogs() []DebugLog {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.size == 0 {
		return []DebugLog{}
	}

	result := make([]DebugLog, d.size)

	if d.size < d.capacity {
		// Buffer not full yet, logs are from 0 to index-1
		copy(result, d.logs[:d.size])
	} else {
		// Buffer is full, logs wrap around
		// Copy from index to end (older logs)
		n := copy(result, d.logs[d.index:])
		// Copy from start to index (newer logs)
		copy(result[n:], d.logs[:d.index])
	}

	return result
}

// Clear removes all logs
func (d *DebugLogger) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.index = 0
	d.size = 0
}

// Size returns the current number of logs
func (d *DebugLogger) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.size
}

// Capacity returns the maximum capacity
func (d *DebugLogger) Capacity() int {
	return d.capacity
}

// IsPromptingFile returns whether the logger is prompting for a filename
func (d *DebugLogger) IsPromptingFile() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.promptingFile
}

// GetStatusMessage returns the current status message
func (d *DebugLogger) GetStatusMessage() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.statusMessage
}

// GetFileInput returns the current file input
func (d *DebugLogger) GetFileInput() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.fileInput
}

// IsConfirmingOverwrite returns whether confirming file overwrite
func (d *DebugLogger) IsConfirmingOverwrite() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.confirmOverwrite
}

// StartFilePrompt initiates the file writing prompt
func (d *DebugLogger) StartFilePrompt() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.promptingFile = true
	d.fileInput = ""
	d.statusMessage = ""
}

// CancelFilePrompt cancels the file writing prompt
func (d *DebugLogger) CancelFilePrompt() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.promptingFile = false
	d.fileInput = ""
	d.confirmOverwrite = false
	d.statusMessage = "Write cancelled"
}

// HandleFileInput handles keyboard input for filename
func (d *DebugLogger) HandleFileInput(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch key {
	case "backspace":
		if len(d.fileInput) > 0 {
			d.fileInput = d.fileInput[:len(d.fileInput)-1]
		}
	default:
		// Add character to filename
		if len(key) == 1 {
			d.fileInput += key
		}
	}
}

// GetLogsAsString returns all logs formatted as a string
func (d *DebugLogger) GetLogsAsString() string {
	logs := d.GetLogs()
	var b strings.Builder

	for _, log := range logs {
		timestamp := log.Timestamp.Format("2006-01-02 15:04:05.000")
		b.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, log.Message))
	}

	return b.String()
}

// debugFileCheckMsg is sent after checking if file exists
type debugFileCheckMsg struct{}

// CheckAndWriteFile checks if file exists and writes or prompts for confirmation
func (d *DebugLogger) CheckAndWriteFile() tea.Cmd {
	return func() tea.Msg {
		d.mu.Lock()
		filename := d.fileInput
		d.mu.Unlock()

		// Check if file exists
		if _, err := os.Stat(filename); err == nil {
			// File exists, prompt for overwrite
			d.mu.Lock()
			d.confirmOverwrite = true
			d.mu.Unlock()
			return debugFileCheckMsg{} // Return message to trigger re-render
		}

		// File doesn't exist, write directly
		return d.WriteToFile()()
	}
}

// debugFileWriteMsg is sent after writing file
type debugFileWriteMsg struct{}

// WriteToFile writes the logs to the specified file
func (d *DebugLogger) WriteToFile() tea.Cmd {
	return func() tea.Msg {
		// Get the content FIRST, before acquiring any locks for file operations
		// This avoids deadlock since GetLogsAsString() acquires its own lock
		content := d.GetLogsAsString()

		d.mu.Lock()
		filename := d.fileInput
		d.mu.Unlock()

		err := os.WriteFile(filename, []byte(content), 0600)

		d.mu.Lock()
		defer d.mu.Unlock()

		if err != nil {
			d.statusMessage = fmt.Sprintf("Error writing file: %v", err)
		} else {
			d.statusMessage = fmt.Sprintf("Successfully wrote to %s", filename)
		}
		d.promptingFile = false
		d.confirmOverwrite = false
		d.fileInput = ""

		return debugFileWriteMsg{} // Return message to trigger re-render
	}
}

// ConfirmOverwrite confirms overwriting an existing file
func (d *DebugLogger) ConfirmOverwrite() tea.Cmd {
	return d.WriteToFile()
}

// ClearStatusMessage clears the status message
func (d *DebugLogger) ClearStatusMessage() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.statusMessage = ""
}
