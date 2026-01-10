---
name: build-tui-view
description: Provides instructions for building Hatchet TUI views in the Hatchet CLI.
version: 1.0
last_updated: 2026-01-09
self_updating: true
---

**📝 SELF-UPDATING DOCUMENT**: This skill automatically updates itself when inaccuracies are discovered or new patterns are learned. Always verify information against the actual codebase and update this file when needed.

## Overview

This skill provides instructions for creating and maintaining Terminal User Interface (TUI) views in the Hatchet CLI using bubbletea and lipgloss. The TUI system uses a modular view architecture where individual views are isolated in separate files within the `views/` directory.

**IMPORTANT**: Always start by finding the corresponding view in the frontend application to understand the structure, columns, and API calls.

## Self-Updating Skill Instructions

**CRITICAL - READ FIRST**: This skill document is designed to be continuously improved and kept accurate.

### When to Update This Skill

You MUST update this skill file in the following situations:

1. **Discovering Inaccuracies**
   - When you find incorrect file paths or directory structures
   - When code examples don't compile or don't match actual implementations
   - When API signatures have changed
   - When referenced files don't exist at specified locations

2. **Learning New Patterns**
   - When implementing a new view and discovering better approaches
   - When the user teaches you new conventions or patterns
   - When you find reusable patterns that should be documented
   - When you discover common pitfalls that should be warned about

3. **Finding Missing Information**
   - When you need information that isn't documented here
   - When new components or utilities are added to the codebase
   - When new bubbletea/lipgloss patterns are adopted

4. **User Corrections**
   - When the user corrects any information in this document
   - When the user provides updated approaches or conventions
   - When the user points out outdated information

### How to Update This Skill

When updating this skill:

1. **Verify Before Adding**: Always verify paths, code, and API signatures against the actual codebase before adding to this document
2. **Use Read/Glob/Grep**: Check the actual files to ensure accuracy
3. **Test Code Examples**: Ensure code examples compile and follow current patterns
4. **Be Specific**: Include exact file paths, function signatures, and working code examples
5. **Update Immediately**: Make updates as soon as inaccuracies are discovered, not at the end of a session
6. **Preserve Structure**: Maintain the existing document structure and formatting
7. **Add Context**: When adding new sections, explain why the pattern is recommended

### Verification Checklist

Before using information from this skill, verify:

- [ ] File paths exist and are correct
- [ ] Code examples match current implementations
- [ ] API signatures match the generated REST client
- [ ] Reusable components are correctly referenced
- [ ] Directory structures are accurate

### Self-Correction Process

If you discover an inaccuracy while working:

1. Immediately note the issue
2. Verify the correct information by reading the actual files
3. Update this skill document with the correction
4. Continue with the user's task using the corrected information

**Remember**: This skill should be a living document that grows more accurate and comprehensive with each use.

## Project Context

- **Framework**: [bubbletea](https://github.com/charmbracelet/bubbletea) (TUI framework)
- **Styling**: [lipgloss](https://github.com/charmbracelet/lipgloss) (style definitions)
- **TUI Command Location**: `cmd/hatchet-cli/cli/tui.go`
- **Views Location**: `cmd/hatchet-cli/cli/tui/` directory
- **Theme**: Pre-defined Hatchet theme in `cmd/hatchet-cli/cli/internal/styles/styles.go`
- **Frontend Reference**: `frontend/app/src/pages/main/v1/` directory

## Finding Frontend Prior Art

**CRITICAL FIRST STEP**: Before implementing any TUI view, locate the corresponding frontend view to understand:

1. Column structure and names
2. API endpoints and query parameters
3. Data types and fields used
4. Filtering and sorting logic

### Process for Finding Frontend Reference:

1. **Locate the Frontend View**

   ```bash
   # Navigate to frontend pages
   cd frontend/app/src/pages/main/v1/

   # Find views related to your feature (e.g., workflow-runs, tasks, events)
   ls -la
   ```

2. **Study the Column Definitions**

   - Look for files like `{feature}-columns.tsx`
   - Note the column keys, titles, and accessors
   - Example: `frontend/app/src/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns.tsx`

   ```typescript
   export const TaskRunColumn = {
     taskName: "Task Name",
     status: "Status",
     workflow: "Workflow",
     createdAt: "Created At",
     startedAt: "Started At",
     duration: "Duration",
   };
   ```

3. **Identify the Data Hook**

   - Look for `use-{feature}.tsx` files in the `hooks/` directory
   - These contain the API query logic
   - Example: `frontend/app/src/pages/main/v1/workflow-runs-v1/hooks/use-runs.tsx`

4. **Find the API Query**

   - Check `frontend/app/src/lib/api/queries.ts` for the query definition
   - Note the endpoint name and parameters
   - Example:

   ```typescript
   v1WorkflowRuns: {
     list: (tenant: string, query: V2ListWorkflowRunsQuery) => ({
       queryKey: ['v1:workflow-run:list', tenant, query],
       queryFn: async () => (await api.v1WorkflowRunList(tenant, query)).data,
     }),
   }
   ```

5. **Map to Go REST Client**

   - The frontend `api.v1WorkflowRunList()` maps to Go's `client.API().V1WorkflowRunListWithResponse()`
   - Frontend query parameters map to Go struct parameters
   - Example mapping:

   ```typescript
   // Frontend
   api.v1WorkflowRunList(tenantId, {
     offset: 0,
     limit: 100,
     since: createdAfter,
     only_tasks: true,
   })

   // Go equivalent
   client.API().V1WorkflowRunListWithResponse(
     ctx,
     client.TenantId(),
     &rest.V1WorkflowRunListParams{
       Offset: int64Ptr(0),
       Limit: int64Ptr(100),
       Since: &since,
       OnlyTasks: true,
     },
   )
   ```

### Example: Implementing Tasks View from Frontend Reference

1. **Frontend Structure**: `frontend/app/src/pages/main/v1/workflow-runs-v1/`

   - Columns: `task-runs-columns.tsx`
   - Hook: `use-runs.tsx`
   - Table: `runs-table.tsx`

2. **Extract Column Names**:

   ```typescript
   taskName, status, workflow, createdAt, startedAt, duration;
   ```

3. **Identify API Call**:

   ```typescript
   queries.v1WorkflowRuns.list(tenantId, {
     offset,
     limit,
     statuses,
     workflow_ids,
     since,
     until,
     only_tasks: true,
   });
   ```

4. **Implement in TUI**:

   ```go
   // Create matching columns
   columns := []table.Column{
     {Title: "Task Name", Width: 30},
     {Title: "Status", Width: 12},
     {Title: "Workflow", Width: 25},
     {Title: "Created At", Width: 16},
     {Title: "Started At", Width: 16},
     {Title: "Duration", Width: 12},
   }

   // Call matching API endpoint
   response, err := client.API().V1WorkflowRunListWithResponse(
     ctx,
     client.TenantId(),
     &rest.V1WorkflowRunListParams{
       Offset: int64Ptr(0),
       Limit: int64Ptr(100),
       Since: &since,
       OnlyTasks: true,
     },
   )
   ```

## Reusable Components (CRITICAL - READ FIRST)

**IMPORTANT**: All TUI views MUST use the standardized reusable components defined in `view.go` to ensure consistency across the application. **DO NOT** copy-paste header/footer styling code.

### Header Component

**Always use `RenderHeader()` for view headers:**

```go
header := RenderHeader("Your View Title", v.Ctx.ProfileName, v.Width)
```

**Features:**

- Consistent styling with Hatchet accent colors
- Includes the logo (text-based: "HATCHET TUI") on the right
- Shows profile name
- Bordered bottom edge
- Responsive to terminal width

**❌ NEVER do this:**

```go
// Bad: Copy-pasting header styles
headerStyle := lipgloss.NewStyle().
    Bold(true).
    Foreground(styles.AccentColor).
    BorderStyle(lipgloss.NormalBorder()).
    // ... more styling
header := headerStyle.Render(fmt.Sprintf("My View - Profile: %s", profile))
```

**✅ ALWAYS do this:**

```go
// Good: Use the reusable component
header := RenderHeader("My View", v.Ctx.ProfileName, v.Width)
```

### Instructions Component

Use `RenderInstructions()` to display contextual help text:

```go
instructions := RenderInstructions(
    "Your instructions here  •  Use bullets to separate items",
    v.Width,
)
```

**Features:**

- Muted color styling for reduced visual noise
- Automatically handles width constraints
- Consistent padding
- Uses bullet separators (•)

### Footer Component

**Always use `RenderFooter()` for navigation/control hints:**

```go
footer := RenderFooter([]string{
    "↑/↓: Navigate",
    "Enter: Select",
    "Esc: Cancel",
    "q: Quit",
}, v.Width)
```

**Features:**

- Consistent styling with top border
- Automatically joins control items with bullets (•)
- Muted color for non-intrusive display
- Responsive to terminal width

## Standard View Structure

**Every view should follow this consistent structure:**

```go
func (v *YourView) View() string {
    var b strings.Builder

    // 1. Header (always) - USE REUSABLE COMPONENT
    header := RenderHeader("View Title", v.Ctx.ProfileName, v.Width)
    b.WriteString(header)
    b.WriteString("\n\n")

    // 2. Instructions (when helpful) - USE REUSABLE COMPONENT
    instructions := RenderInstructions("Your instructions", v.Width)
    b.WriteString(instructions)
    b.WriteString("\n\n")

    // 3. Main content
    // ... your view-specific content ...

    // 4. Footer (always) - USE REUSABLE COMPONENT
    footer := RenderFooter([]string{
        "control1: Action1",
        "control2: Action2",
    }, v.Width)
    b.WriteString(footer)

    return b.String()
}
```

## Architecture

### Root TUI Model (`tui.go`)

The root TUI command is responsible for:

1. Profile selection and validation
2. Initializing the Hatchet client
3. Creating the view context
4. Managing the current view
5. Delegating updates to views

### View System (`views/` directory)

Each view is a separate file that implements the `View` interface:

- `view.go` - Base view interface, context, and **reusable components**
- `{viewname}.go` - Individual view implementations (e.g., `tasks.go`)

## Core Principles

### 1. File Structure

#### TUI Command File

- File: `cmd/hatchet-cli/cli/tui.go`
- Purpose: Command setup, profile selection, client initialization, view management

#### View Files

- Location: `cmd/hatchet-cli/cli/tui/`
- Files:
  - `view.go` - View interface and base types
  - `{viewname}.go` - Individual view implementations

### 2. View Interface

All views must implement this interface (defined in `views/view.go`):

```go
package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

// ViewContext contains the shared context passed to all views
type ViewContext struct {
	// Profile name for display
	ProfileName string

	// Hatchet client for API calls
	Client client.Client

	// Terminal dimensions
	Width  int
	Height int
}

// View represents a TUI view component
type View interface {
	// Init initializes the view and returns any initial commands
	Init() tea.Cmd

	// Update handles messages and updates the view state
	Update(msg tea.Msg) (View, tea.Cmd)

	// View renders the view to a string
	View() string

	// SetSize updates the view dimensions
	SetSize(width, height int)
}
```

### 3. Base Model Pattern

Use `BaseModel` for common view fields:

```go
// BaseModel contains common fields for all views
type BaseModel struct {
	Ctx    ViewContext
	Width  int
	Height int
	Err    error
}

// Your view embeds BaseModel
type YourView struct {
	BaseModel
	// Your view-specific fields
	table table.Model
	items []YourDataType
}
```

### 4. Creating a New View

#### Step 1: Create View File

Create `cmd/hatchet-cli/cli/tui/{viewname}.go`:

```go
package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

type YourView struct {
	BaseModel
	// View-specific fields
}

// NewYourView creates a new instance of your view
func NewYourView(ctx ViewContext) *YourView {
	v := &YourView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
	}

	// Initialize view components

	return v
}

func (v *YourView) Init() tea.Cmd {
	return nil
}

func (v *YourView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Refresh logic
			return v, nil
		}
	}

	// Update sub-components
	return v, cmd
}

func (v *YourView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// Build your view
	return "Your view content"
}

func (v *YourView) SetSize(width, height int) {
	v.BaseModel.SetSize(width, height)
	// Update view-specific components
}
```

#### Step 2: Use View in TUI

The root TUI model manages views:

```go
// In tui.go
func newTUIModel(profileName string, hatchetClient client.Client) tuiModel {
	ctx := views.ViewContext{
		ProfileName: profileName,
		Client:      hatchetClient,
	}

	// Initialize with your view
	currentView := views.NewYourView(ctx)

	return tuiModel{
		currentView: currentView,
	}
}
```

### 5. Client Initialization Pattern

Always initialize the Hatchet client in `tui.go`:

```go
import (
	"github.com/rs/zerolog"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

// In the cobra command Run function
profile, err := cli.GetProfile(selectedProfile)
if err != nil {
	cli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
}

// Initialize Hatchet client
nopLogger := zerolog.Nop()
hatchetClient, err := client.New(
	client.WithToken(profile.Token),
	client.WithLogger(&nopLogger),
)
if err != nil {
	cli.Logger.Fatalf("could not create Hatchet client: %v", err)
}
```

### 6. Accessing the Client in Views

The Hatchet client is available through the view context:

```go
func (v *YourView) fetchData() tea.Cmd {
	return func() tea.Msg {
		// Access the client
		client := v.Ctx.Client

		// Make API calls
		// response, err := client.API().SomeEndpoint(...)

		return yourDataMsg{
			data: data,
			err:  err,
		}
	}
}
```

### 7. Hatchet Theme Integration

**CRITICAL**: NEVER hardcode colors or styles in view files. Always use the pre-defined Hatchet theme colors and utilities from `cmd/hatchet-cli/cli/internal/styles`.

#### Available Theme Colors

```go
import "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"

// Primary theme colors:
// - styles.AccentColor
// - styles.PrimaryColor
// - styles.SuccessColor
// - styles.HighlightColor
// - styles.MutedColor
// - styles.Blue, styles.Cyan, styles.Magenta

// Status colors (matching frontend badge variants):
// - styles.StatusSuccessColor / styles.StatusSuccessBg
// - styles.StatusFailedColor / styles.StatusFailedBg
// - styles.StatusInProgressColor / styles.StatusInProgressBg
// - styles.StatusQueuedColor / styles.StatusQueuedBg
// - styles.StatusCancelledColor / styles.StatusCancelledBg
// - styles.ErrorColor

// Available styles:
// - styles.H1, styles.H2
// - styles.Bold, styles.Italic
// - styles.Primary, styles.Accent, styles.Success
// - styles.Code
// - styles.Box, styles.InfoBox, styles.SuccessBox
```

#### Status Rendering

**Per-Cell Coloring in Tables**: Use the custom `TableWithStyleFunc` wrapper to enable per-cell styling.

For status rendering in tables:

```go
// Create table with StyleFunc support
t := NewTableWithStyleFunc(
    table.WithColumns(columns),
    table.WithFocused(true),
    table.WithHeight(20),
)

// Set StyleFunc for per-cell styling
t.SetStyleFunc(func(row, col int) lipgloss.Style {
    // Column 1 is the status column
    if col == 1 && row < len(v.tasks) {
        statusStyle := styles.GetV1TaskStatusStyle(v.tasks[row].Status)
        return lipgloss.NewStyle().Foreground(statusStyle.Foreground)
    }
    return lipgloss.NewStyle()
})

// In updateTableRows, use plain text (StyleFunc applies colors)
statusStyle := styles.GetV1TaskStatusStyle(task.Status)
status := statusStyle.Text  // "Succeeded", "Failed", etc.
```

For non-table contexts (headers, footers, standalone text):

```go
// Render V1TaskStatus with proper colors
status := styles.RenderV1TaskStatus(task.Status)

// Render error messages
errorMsg := styles.RenderError(fmt.Sprintf("Error: %v", err))
```

**Why custom TableWithStyleFunc?**

- Standard bubbles table doesn't support per-cell or per-column styling
- `TableWithStyleFunc` wraps bubbles table and adds StyleFunc support
- StyleFunc allows dynamic cell styling based on row/column index
- Located in `cmd/hatchet-cli/cli/tui/table_custom.go`
- Maintains bubbles table interactivity (cursor, selection, keyboard nav)

#### Table Styling

```go
s := table.DefaultStyles()
s.Header = s.Header.
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(styles.AccentColor).
    BorderBottom(true).
    Bold(true).
    Foreground(styles.AccentColor)
s.Selected = s.Selected.
    Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
    Background(styles.Blue).
    Bold(true)
```

**Note**: Use `lipgloss.AdaptiveColor` even for basic colors like white/black to support light/dark terminals.

#### Adding New Status Colors

If you need to add new status colors:

1. Add the color constants to `cmd/hatchet-cli/cli/internal/styles/styles.go`
2. Create or update the utility function in `cmd/hatchet-cli/cli/internal/styles/status.go`
3. Reference the frontend badge variants in `frontend/app/src/components/v1/ui/badge.tsx` for color values
4. Use adaptive colors for light/dark terminal support

### 8. Standard Keyboard Controls

Use consistent key mappings across all views to provide a predictable user experience.

#### Global Controls (handled in tui.go)

- `q` or `ctrl+c`: Quit the TUI

#### View-Specific Controls

Implement these in individual views:

- **Navigation**: `↑/↓` or arrow keys for list navigation
- **Selection**: `Enter` to select/confirm
- **Tab Navigation**: `Tab`/`Shift+Tab` for form fields
- **Cancel**: `Esc` to go back/cancel
- **Refresh**: `r` to manually refresh data
- **Filter**: `f` to open filter modal (where applicable)
- **Debug**: `d` to toggle debug view (see Debug Logging section)
- **Clear**: `c` to clear debug logs (when in debug view)
- **Tab Views**: `1`, `2`, `3`, etc. or `tab`/`shift+tab` for switching tabs

**Important**: Always document keyboard controls in the footer using `RenderFooter()`

### 9. Layout Components

**CRITICAL**: Use the reusable components from `view.go` for headers, instructions, and footers. See "Reusable Components" section above.

#### Header

**✅ Use the reusable component:**

```go
header := RenderHeader("View Title", v.Ctx.ProfileName, v.Width)
```

**❌ DO NOT manually create headers:**

```go
// Bad: Don't do this
headerStyle := lipgloss.NewStyle().
    Bold(true).
    Foreground(styles.AccentColor).
    // ... (this violates DRY principle)
```

#### Footer

**✅ Use the reusable component:**

```go
footer := RenderFooter([]string{
    "↑/↓: Navigate",
    "r: Refresh",
    "q: Quit",
}, v.Width)
```

**❌ DO NOT manually create footers:**

```go
// Bad: Don't do this
footerStyle := lipgloss.NewStyle().
    Foreground(styles.MutedColor).
    // ... (this violates DRY principle)
```

#### Instructions

**✅ Use the reusable component:**

```go
instructions := RenderInstructions("Your helpful instructions here", v.Width)
```

#### Stats Bar

Custom stats bars are fine for view-specific metrics:

```go
statsStyle := lipgloss.NewStyle().
    Foreground(styles.MutedColor).
    Padding(0, 1)

stats := statsStyle.Render(fmt.Sprintf(
    "Total: %d  |  Status1: %d  |  Status2: %d",
    total, status1Count, status2Count,
))
```

### 10. Data Integration

#### REST API Types

Use generated REST types from:

```go
import "github.com/hatchet-dev/hatchet/pkg/client/rest"
```

Common types:

- `rest.V1TaskSummary`
- `rest.V1TaskSummaryList`
- `rest.WorkflowRun`
- `rest.StepRun`
- `rest.APIResourceMeta`

#### Async Data Fetching Pattern

```go
// Define custom message types in your view file
type yourDataMsg struct {
    items []YourDataType
    err   error
}

// Create fetch command
func (v *YourView) fetchData() tea.Cmd {
    return func() tea.Msg {
        // Use v.Ctx.Client to make API calls
        // Return yourDataMsg
    }
}

// Handle in Update
case yourDataMsg:
    v.loading = false
    if msg.err != nil {
        v.HandleError(msg.err)
    } else {
        v.items = msg.items
        v.ClearError()
    }
```

### 11. Modal Views

When creating modal overlays (like filter forms or confirmation dialogs):

1. Still show the header with updated title using `RenderHeader()`
2. Show instructions specific to the modal interaction using `RenderInstructions()`
3. Show the modal content
4. Show a footer with modal-specific controls using `RenderFooter()`

**Example Modal Structure:**

```go
func (v *TasksView) renderFilterModal() string {
    var b strings.Builder

    // 1. Header - USE REUSABLE COMPONENT
    header := RenderHeader("Filter Tasks", v.Ctx.ProfileName, v.Width)
    b.WriteString(header)
    b.WriteString("\n\n")

    // 2. Instructions - USE REUSABLE COMPONENT
    instructions := RenderInstructions("Configure filters and press Enter to apply", v.Width)
    b.WriteString(instructions)
    b.WriteString("\n\n")

    // 3. Modal content (form, etc.)
    b.WriteString(v.filterForm.View())
    b.WriteString("\n")

    // 4. Footer - USE REUSABLE COMPONENT
    footer := RenderFooter([]string{"Enter: Apply", "Esc: Cancel"}, v.Width)
    b.WriteString(footer)

    return b.String()
}
```

**Important**: Modals should maintain the same visual structure as regular views (header, instructions, content, footer) for consistency.

### 12. Form Integration

When using `huh` forms in views:

1. **Set the Hatchet theme**: `.WithTheme(styles.HatchetTheme())`
2. **Integrate forms directly into the main tea.Program** (don't run separate programs)
3. **Handle form completion** by checking `form.State == huh.StateCompleted`
4. **Pass ALL messages to the form** when it's active (not just key messages)

**Example:**

```go
import "github.com/charmbracelet/huh"

// In Update()
if v.showingFilter && v.filterForm != nil {
    // Pass ALL messages to form when active
    form, cmd := v.filterForm.Update(msg)
    v.filterForm = form.(*huh.Form)

    // Check if form completed
    if v.filterForm.State == huh.StateCompleted {
        v.showingFilter = false
        // Process form values
    }

    return v, cmd
}
```

### 13. Table Component

Using `github.com/charmbracelet/bubbles/table`:

```go
import "github.com/charmbracelet/bubbles/table"

// Define columns
columns := []table.Column{
    {Title: "Column1", Width: 20},
    {Title: "Column2", Width: 30},
}

// Create table
t := table.New(
    table.WithColumns(columns),
    table.WithFocused(true),
    table.WithHeight(20),
)

// Apply Hatchet styles
s := table.DefaultStyles()
s.Header = s.Header.
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(styles.AccentColor).
    BorderBottom(true).
    Bold(true).
    Foreground(styles.AccentColor)
s.Selected = s.Selected.
    Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0A1029"}).
    Background(styles.Blue).
    Bold(true)
t.SetStyles(s)

// Update rows
rows := make([]table.Row, len(items))
for i, item := range items {
    rows[i] = table.Row{item.Field1, item.Field2}
}
t.SetRows(rows)
```

## Common Patterns

### Formatting Utilities

#### Duration Formatting

```go
func formatDuration(ms int) string {
    duration := time.Duration(ms) * time.Millisecond
    if duration < time.Second {
        return fmt.Sprintf("%dms", ms)
    }
    seconds := duration.Seconds()
    if seconds < 60 {
        return fmt.Sprintf("%.1fs", seconds)
    }
    minutes := int(seconds / 60)
    secs := int(seconds) % 60
    return fmt.Sprintf("%dm%ds", minutes, secs)
}
```

#### ID Truncation

```go
func truncateID(id string, length int) string {
    if len(id) > length {
        return id[:length]
    }
    return id
}
```

#### Status Rendering

**IMPORTANT**: Do not manually style statuses. Use the status utility functions:

```go
// For V1TaskStatus (from REST API)
status := styles.RenderV1TaskStatus(task.Status)

// The utility automatically handles:
// - COMPLETED -> Green "Succeeded"
// - FAILED -> Red "Failed"
// - CANCELLED -> Orange "Cancelled"
// - RUNNING -> Yellow "Running"
// - QUEUED -> Gray "Queued"
// All colors match frontend badge variants
```

### Auto-refresh Pattern

```go
// Define tick message in your view file
type tickMsg time.Time

// Create tick command
func tick() tea.Cmd {
    return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

// Handle in Update
case tickMsg:
    // Refresh data
    return v, tea.Batch(v.fetchData(), tick())
```

### Debug Logging Pattern

**Important**: For views that make API calls or have complex state management, implement a debug logging system using a ring buffer to prevent memory leaks.

#### Step 1: Create Debug Logger (if not exists)

Create `cmd/hatchet-cli/cli/tui/debug.go`:

```go
package views

import (
	"fmt"
	"sync"
	"time"
)

// DebugLog represents a single debug log entry
type DebugLog struct {
	Timestamp time.Time
	Message   string
}

// DebugLogger is a fixed-size ring buffer for debug logs
type DebugLogger struct {
	mu       sync.RWMutex
	logs     []DebugLog
	capacity int
	index    int
	size     int
}

// NewDebugLogger creates a new debug logger with the specified capacity
func NewDebugLogger(capacity int) *DebugLogger {
	return &DebugLogger{
		logs:     make([]DebugLog, capacity),
		capacity: capacity,
		index:    0,
		size:     0,
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
```

#### Step 2: Integrate Debug Logger in Your View

```go
type YourView struct {
	BaseModel
	// ... other fields
	debugLogger *DebugLogger
	showDebug   bool // Whether to show debug overlay
}

func NewYourView(ctx ViewContext) *YourView {
	v := &YourView{
		BaseModel: BaseModel{
			Ctx: ctx,
		},
		debugLogger: NewDebugLogger(5000), // 5000 log entries max
		showDebug:   false,
	}

	v.debugLogger.Log("YourView initialized")

	return v
}
```

#### Step 3: Add Debug Logging Throughout View

```go
// Log important events
func (v *YourView) fetchData() tea.Cmd {
	return func() tea.Msg {
		v.debugLogger.Log("Fetching data...")

		// Make API call
		response, err := v.Ctx.Client.API().SomeEndpoint(...)

		if err != nil {
			v.debugLogger.Log("Error fetching data: %v", err)
			return dataMsg{err: err}
		}

		v.debugLogger.Log("Successfully fetched %d items", len(response.Items))
		return dataMsg{data: response.Items}
	}
}
```

#### Step 4: Add Toggle Key Handler

```go
func (v *YourView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "d":
			// Toggle debug view
			v.showDebug = !v.showDebug
			v.debugLogger.Log("Debug view toggled: %v", v.showDebug)
			return v, nil
		case "c":
			// Clear debug logs (only when in debug view)
			if v.showDebug {
				v.debugLogger.Clear()
				v.debugLogger.Log("Debug logs cleared")
			}
			return v, nil
		}
	}
	// ... rest of update logic
}
```

#### Step 5: Implement Debug View Rendering

```go
func (v *YourView) View() string {
	if v.Width == 0 {
		return "Initializing..."
	}

	// If debug view is enabled, show debug overlay
	if v.showDebug {
		return v.renderDebugView()
	}

	// ... normal view rendering
}

func (v *YourView) renderDebugView() string {
	logs := v.debugLogger.GetLogs()

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.AccentColor).
		Width(v.Width-4).
		Padding(0, 1)

	header := headerStyle.Render(fmt.Sprintf(
		"Debug Logs - %d/%d entries",
		v.debugLogger.Size(),
		v.debugLogger.Capacity(),
	))

	// Log entries
	logStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(v.Width - 4)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")

	// Calculate how many logs we can show
	maxLines := v.Height - 8 // Reserve space for header, footer, controls
	if maxLines < 1 {
		maxLines = 1
	}

	// Show most recent logs first
	startIdx := 0
	if len(logs) > maxLines {
		startIdx = len(logs) - maxLines
	}

	for i := startIdx; i < len(logs); i++ {
		log := logs[i]
		timestamp := log.Timestamp.Format("15:04:05.000")
		logLine := fmt.Sprintf("[%s] %s", timestamp, log.Message)
		b.WriteString(logStyle.Render(logLine))
		b.WriteString("\n")
	}

	// Footer with controls
	footerStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(styles.AccentColor).
		Width(v.Width-4).
		Padding(0, 1)

	controls := footerStyle.Render("d: Close Debug  |  c: Clear Logs  |  q: Quit")
	b.WriteString("\n")
	b.WriteString(controls)

	return b.String()
}
```

#### Step 6: Update Footer Controls

Add debug controls to your normal view footer:

```go
controls := footerStyle.Render("↑/↓: Navigate  |  r: Refresh  |  d: Debug  |  q: Quit")
```

**Benefits**:

- Fixed-size ring buffer prevents memory leaks
- Thread-safe with mutex protection
- Toggle on/off without restarting TUI
- Helps diagnose API issues and state changes
- No performance impact when not viewing logs

## Testing Approach

### Dummy Data Generation

During development, create dummy data generators in your view file:

```go
func generateDummyData() []YourDataType {
    now := time.Now()
    return []YourDataType{
        {
            Field1: "value1",
            Field2: "value2",
            CreatedAt: now.Add(-5 * time.Minute),
        },
        // ... more dummy items
    }
}
```

## Example Reference

### File Structure Example

```
cmd/hatchet-cli/cli/
├── tui.go                    # Root TUI command
└── views/
    ├── view.go               # View interface and base types
    ├── tasks.go              # Tasks view implementation
    └── workflows.go          # Workflows view implementation (future)
```

### Complete View Example

See `cmd/hatchet-cli/cli/tui/tasks.go` for a complete implementation.

## Compilation and Testing

**CRITICAL**: Always ensure the CLI binary compiles before considering work complete.

### Compilation Check

After implementing or modifying any view:

```bash
# Build the CLI binary
go build -o /tmp/hatchet-test ./cmd/hatchet-cli

# Check for errors
echo $?  # Should be 0 for success
```

### Common Compilation Issues

1. **UUID Type Mismatches**

   ```go
   // ❌ Wrong - string to UUID
   client.API().SomeMethod(ctx, client.TenantId(), ...)

   // ✅ Correct - parse and convert
   tenantUUID, err := uuid.Parse(client.TenantId())
   if err != nil {
       return msg{err: fmt.Errorf("invalid tenant ID: %w", err)}
   }
   client.API().SomeMethod(ctx, openapi_types.UUID(tenantUUID), ...)
   ```

2. **Required Imports**

   ```go
   import (
       "github.com/google/uuid"
       openapi_types "github.com/oapi-codegen/runtime/types"
   )
   ```

3. **Type Conversions for API Params**

   - `*int64` not `*int` for offset/limit
   - `time.Time` not `*time.Time` for Since parameter (check the generated types)
   - `openapi_types.UUID` for tenant and workflow IDs
   - Check `pkg/client/rest/gen.go` for exact parameter types

4. **Pointer Helper Functions**
   ```go
   func int64Ptr(i int64) *int64 {
       return &i
   }
   ```

### Testing Workflow

1. **Compilation Test**

   ```bash
   go build -o /tmp/hatchet-test ./cmd/hatchet-cli
   ```

2. **Linting Test**

   After the build succeeds, run the linting checks:

   ```bash
   task pre-commit-run
   ```

   Continue running this command until it succeeds. Fix any linting issues that are reported before proceeding.

3. **Basic Functionality Test**

   ```bash
   # Test with profile selection
   /tmp/hatchet-test tui

   # Test with specific profile
   /tmp/hatchet-test tui --profile your-profile
   ```

4. **Error Handling Test**

   - Try without profiles configured
   - Try with invalid profile
   - Test keyboard controls (q, r, arrows)

5. **Visual/Layout Testing**

   When implementing a new view:

   - Test at various terminal sizes (minimum 80x24)
   - Ensure header and footer are always visible
   - Verify instructions are clear and helpful
   - Check that navigation controls are consistent with other views
   - Test with both light and dark terminal backgrounds
   - Verify all reusable components render correctly

## Checklist for New TUI Views

### Before You Start

- [ ] **Find the corresponding frontend view** in `frontend/app/src/pages/main/v1/`
- [ ] Identify column structure and API calls from frontend
- [ ] Note the exact API endpoint and parameters used

### Creating the View

- [ ] Create new file in `cmd/hatchet-cli/cli/tui/{viewname}.go`
- [ ] Add required imports (including `uuid` and `openapi_types` if needed)
- [ ] Define view struct that embeds `BaseModel`
- [ ] Create `NewYourView(ctx ViewContext)` constructor
- [ ] Implement `Init()` method
- [ ] Implement `Update(msg tea.Msg)` method
- [ ] Implement `View()` method following standard structure
- [ ] Implement `SetSize(width, height int)` method
- [ ] **USE REUSABLE COMPONENTS**: `RenderHeader()`, `RenderInstructions()`, `RenderFooter()`
- [ ] **DO NOT** copy-paste header/footer styling code
- [ ] Apply Hatchet theme colors and styles for custom components only
- [ ] Implement view-specific keyboard controls
- [ ] Document all keyboard controls in footer using `RenderFooter()`
- [ ] Use appropriate REST API types
- [ ] Add error handling using `BaseModel.HandleError()`
- [ ] Add responsive layout (handle WindowSizeMsg)

### API Integration

- [ ] Parse tenant ID to UUID if needed
- [ ] Use correct parameter types (`*int64`, `time.Time`, etc.)
- [ ] Handle API response errors
- [ ] Format data for table display
- [ ] Add loading states and error display

### Integration and Testing

- [ ] Import view in `tui.go`
- [ ] Update `newTUIModel()` to instantiate your view
- [ ] **Compile the CLI binary** (`go build ./cmd/hatchet-cli`)
- [ ] Fix any compilation errors
- [ ] Test basic functionality with real profile
- [ ] Test error cases (no profile, invalid profile)
- [ ] Test keyboard controls (q, r, arrows)
- [ ] Update TUI command documentation

### Best Practices

- [ ] **CRITICAL**: Use reusable components (`RenderHeader`, `RenderInstructions`, `RenderFooter`)
- [ ] **NEVER** copy-paste header/footer styling code - this violates DRY principle
- [ ] Keep view logic isolated in the view file
- [ ] Use `ViewContext` to access client and profile info
- [ ] Handle all messages gracefully (return `v, nil` for unhandled)
- [ ] Always check `v.Width == 0` before rendering
- [ ] Use consistent styling with other views (use Hatchet theme colors)
- [ ] Document ALL keyboard controls in footer using `RenderFooter()`
- [ ] Follow the standard view structure (header, instructions, content, footer)
- [ ] **Always verify compilation before submitting**

### Post-Implementation

- [ ] **Update this skill document** with any new patterns or learnings discovered
- [ ] Document any issues encountered and their solutions
- [ ] Add any new utility functions or patterns to the appropriate sections
- [ ] Verify all code examples and file paths are accurate

## Lessons Learned & Updates

This section documents recent learnings and updates to maintain accuracy.

### Recent Updates

- **2026-01-09**: Added self-updating instructions and verification checklist
- Document initialized with comprehensive TUI view building guidelines

### Known Issues & Solutions

(This section will be populated as issues are discovered and resolved during implementation)

### Future Improvements

(This section will track potential improvements to the TUI system or this skill document)
