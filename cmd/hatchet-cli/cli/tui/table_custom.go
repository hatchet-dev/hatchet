package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// TableWithStyleFunc wraps bubbles table.Model and adds per-cell styling support
type TableWithStyleFunc struct {
	*table.Model
	styleFunc func(row, col int) lipgloss.Style
	styles    table.Styles
}

// NewTableWithStyleFunc creates a new table with StyleFunc support
func NewTableWithStyleFunc(opts ...table.Option) *TableWithStyleFunc {
	model := table.New(opts...)
	return &TableWithStyleFunc{
		Model:  &model,
		styles: table.DefaultStyles(),
	}
}

// SetStyles sets the table styles
func (t *TableWithStyleFunc) SetStyles(s table.Styles) {
	t.styles = s
	t.Model.SetStyles(s)
}

// SetStyleFunc sets the function that determines styling for each cell
func (t *TableWithStyleFunc) SetStyleFunc(fn func(row, col int) lipgloss.Style) {
	t.styleFunc = fn
}

// Update delegates to the underlying table model and handles all events including mouse
func (t *TableWithStyleFunc) Update(msg interface{}) (table.Model, tea.Cmd) {
	updatedModel, cmd := t.Model.Update(msg)
	*t.Model = updatedModel
	return updatedModel, cmd
}

// Cursor returns the current cursor position
func (t *TableWithStyleFunc) Cursor() int {
	return t.Model.Cursor()
}

// SetHeight sets the table height
func (t *TableWithStyleFunc) SetHeight(height int) {
	t.Model.SetHeight(height)
}

// SetRows sets the table rows
func (t *TableWithStyleFunc) SetRows(rows []table.Row) {
	t.Model.SetRows(rows)
}

// Rows returns the table rows
func (t *TableWithStyleFunc) Rows() []table.Row {
	return t.Model.Rows()
}

// Columns returns the table columns
func (t *TableWithStyleFunc) Columns() []table.Column {
	return t.Model.Columns()
}

// Height returns the table height
func (t *TableWithStyleFunc) Height() int {
	return t.Model.Height()
}

// View renders the table with per-cell styling
func (t *TableWithStyleFunc) View() string {
	if t.styleFunc == nil {
		// No custom styling, use default
		return t.Model.View()
	}

	// Custom rendering with StyleFunc
	return t.customView()
}

// customView renders the table with StyleFunc applied
// We need to pre-render the rows with styling, then let the viewport handle scrolling
func (t *TableWithStyleFunc) customView() string {
	// Pre-apply styling to all rows by creating styled row data
	// Then store it back and use the default View which handles viewport

	// Unfortunately, we can't easily inject into the viewport rendering
	// Instead, let's render rows with proper viewport scrolling logic

	rows := t.Rows()
	cols := t.Columns()
	cursor := t.Cursor()
	height := t.Height()

	// Calculate visible range (similar to bubbles table logic)
	start := 0
	end := len(rows)

	// If we have more rows than height, show only the viewport
	if len(rows) > height {
		// Center the cursor in the viewport
		half := height / 2
		start = cursor - half
		if start < 0 {
			start = 0
		}
		end = start + height
		if end > len(rows) {
			end = len(rows)
			start = end - height
			if start < 0 {
				start = 0
			}
		}
	}

	var s strings.Builder

	// Render header
	s.WriteString(t.renderHeader())
	s.WriteString("\n")

	// Render visible rows only
	for r := start; r < end; r++ {
		rowStr := t.renderRowWithStyle(r, rows[r], cols, r == cursor)
		s.WriteString(rowStr)
		if r < end-1 {
			s.WriteString("\n")
		}
	}

	return s.String()
}

// renderHeader renders the table header
func (t *TableWithStyleFunc) renderHeader() string {
	cols := t.Columns()

	s := make([]string, 0, len(cols))
	for _, col := range cols {
		if col.Width <= 0 {
			continue
		}
		// Truncate title to column width (leave room for space after ellipsis)
		truncated := runewidth.Truncate(col.Title, col.Width, "… ")

		// Pad to exact width to match row cells
		padded := runewidth.FillRight(truncated, col.Width)

		// Render with header style using inline
		headerStyled := t.styles.Header.Inline(true).Render(padded)

		// Wrap with cell style using inline
		rendered := t.styles.Cell.Inline(true).Render(headerStyled)

		s = append(s, rendered)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, s...)
}

// renderRowWithStyle renders a single row with StyleFunc applied
func (t *TableWithStyleFunc) renderRowWithStyle(rowIdx int, row table.Row, cols []table.Column, isSelected bool) string {
	s := make([]string, 0, len(cols))
	for colIdx, value := range row {
		if colIdx >= len(cols) || cols[colIdx].Width <= 0 {
			continue
		}

		// Get custom style for this cell
		cellStyle := t.styleFunc(rowIdx, colIdx)

		// Truncate to column width first (with space after ellipsis)
		truncated := runewidth.Truncate(value, cols[colIdx].Width, "… ")

		// Pad to exact width to ensure alignment
		padded := runewidth.FillRight(truncated, cols[colIdx].Width)

		// Apply cell style (adds ANSI codes for color)
		// Use Inline to prevent lipgloss from adding its own padding
		styledText := cellStyle.Inline(true).Render(padded)

		// Apply selection style only to first column
		if isSelected && colIdx == 0 {
			styledText = t.styles.Selected.Inline(true).Render(styledText)
		}

		// Wrap with table cell style (should be empty/no color)
		rendered := t.styles.Cell.Inline(true).Render(styledText)

		s = append(s, rendered)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, s...)
}
