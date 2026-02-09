package dag

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Canvas represents an ASCII drawing surface for rendering the graph
type Canvas struct {
	cells  [][]Cell
	width  int
	height int

	// Track actual used dimensions (may be less than allocated)
	usedWidth  int
	usedHeight int
}

// Cell represents a single character on the canvas with optional styling
type Cell struct {
	Style lipgloss.Style
	Char  string
}

// CellPriority defines priority for character collision resolution
// Higher priority characters overwrite lower priority ones
type CellPriority int

const (
	PriorityEmpty    CellPriority = iota // Empty cell (space)
	PriorityLine                         // Line characters (─ │)
	PriorityArrow                        // Arrow characters (► ▼ ◄ ▲)
	PriorityJunction                     // Junction characters (┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼)
	PriorityBox                          // Box borders and content
)

// Box-drawing characters (Unicode)
const (
	CharHorizontal  = "─"
	CharVertical    = "│"
	CharTopLeft     = "┌"
	CharTopRight    = "┐"
	CharBottomLeft  = "└"
	CharBottomRight = "┘"
	CharArrowRight  = "►"
	CharArrowDown   = "▼"
	CharArrowLeft   = "◄"
	CharArrowUp     = "▲"
	CharSpace       = " "
	CharCross       = "┼" // Cross junction
	CharTeeRight    = "├" // T-junction pointing right
	CharTeeLeft     = "┤" // T-junction pointing left
	CharTeeDown     = "┬" // T-junction pointing down
	CharTeeUp       = "┴" // T-junction pointing up
)

// NewCanvas creates a new canvas with the specified dimensions
func NewCanvas(width, height int) *Canvas {
	// Allocate 2D cell array
	cells := make([][]Cell, width)
	for x := 0; x < width; x++ {
		cells[x] = make([]Cell, height)
		for y := 0; y < height; y++ {
			cells[x][y] = Cell{
				Char:  CharSpace,
				Style: lipgloss.NewStyle(),
			}
		}
	}

	return &Canvas{
		cells:      cells,
		width:      width,
		height:     height,
		usedWidth:  0,
		usedHeight: 0,
	}
}

// GetDimensions returns the actual used dimensions
func (c *Canvas) GetDimensions() (width, height int) {
	return c.usedWidth, c.usedHeight
}

// SetCell sets a cell using priority-based collision handling
func (c *Canvas) SetCell(x, y int, char string, style lipgloss.Style) {
	if x < 0 || x >= c.width || y < 0 || y >= c.height {
		return
	}

	if x >= c.usedWidth {
		c.usedWidth = x + 1
	}
	if y >= c.usedHeight {
		c.usedHeight = y + 1
	}

	existingPriority := c.getPriority(c.cells[x][y].Char)
	newPriority := c.getPriority(char)

	if newPriority >= existingPriority {
		c.cells[x][y] = Cell{
			Char:  char,
			Style: style,
		}
	}
}

// getPriority returns the priority of a character for collision resolution
func (c *Canvas) getPriority(char string) CellPriority {
	switch char {
	case CharSpace:
		return PriorityEmpty
	case CharHorizontal, CharVertical:
		return PriorityLine
	case CharArrowRight, CharArrowDown, CharArrowLeft, CharArrowUp:
		return PriorityArrow
	case CharTopLeft, CharTopRight, CharBottomLeft, CharBottomRight,
		CharCross, CharTeeRight, CharTeeLeft, CharTeeDown, CharTeeUp:
		return PriorityJunction
	default:
		return PriorityBox // Text and box content
	}
}

// DrawBox renders a node box with status coloring and optional selection
func (c *Canvas) DrawBox(x, y, width, height int, text string, status rest.V1TaskStatus, selected bool) {
	if width < 3 || height < 1 {
		return
	}

	borderColor := getStatusColor(status)
	if selected {
		borderColor = styles.HighlightColor
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	c.SetCell(x, y, CharTopLeft, borderStyle)
	for i := 1; i < width-1; i++ {
		c.SetCell(x+i, y, CharHorizontal, borderStyle)
	}
	c.SetCell(x+width-1, y, CharTopRight, borderStyle)

	for row := 1; row < height-1; row++ {
		c.SetCell(x, y+row, CharVertical, borderStyle)
		c.SetCell(x+width-1, y+row, CharVertical, borderStyle)
		for col := 1; col < width-1; col++ {
			c.SetCell(x+col, y+row, CharSpace, lipgloss.NewStyle())
		}
	}

	c.SetCell(x, y+height-1, CharBottomLeft, borderStyle)
	for i := 1; i < width-1; i++ {
		c.SetCell(x+i, y+height-1, CharHorizontal, borderStyle)
	}
	c.SetCell(x+width-1, y+height-1, CharBottomRight, borderStyle)

	if height > 1 {
		c.drawCenteredText(x, y, width, height, text, status, selected)
	}
}

// drawCenteredText renders text centered within a box
func (c *Canvas) drawCenteredText(x, y, width, height int, text string, status rest.V1TaskStatus, selected bool) {
	maxTextWidth := width - 4
	if maxTextWidth < 1 {
		return
	}

	displayText := text
	if len(text) > maxTextWidth {
		if maxTextWidth < 5 {
			displayText = text[:maxTextWidth]
		} else {
			leftPart := (maxTextWidth - 3) / 2
			rightPart := maxTextWidth - 3 - leftPart
			displayText = text[:leftPart] + "..." + text[len(text)-rightPart:]
		}
	}

	textRow := y + height/2
	padding := (width - len(displayText)) / 2
	startX := x + padding

	textStyle := lipgloss.NewStyle()
	if selected {
		textStyle = textStyle.Bold(true).Foreground(styles.HighlightColor)
	} else {
		textStyle = textStyle.Foreground(getStatusColor(status))
	}

	for i, ch := range displayText {
		c.SetCell(startX+i, textRow, string(ch), textStyle)
	}
}

// getStatusColor returns the color for a task status
func getStatusColor(status rest.V1TaskStatus) lipgloss.AdaptiveColor {
	switch status {
	case rest.V1TaskStatusCOMPLETED:
		return styles.StatusSuccessColor
	case rest.V1TaskStatusFAILED:
		return styles.StatusFailedColor
	case rest.V1TaskStatusCANCELLED:
		return styles.StatusCancelledColor
	case rest.V1TaskStatusRUNNING:
		return styles.StatusInProgressColor
	case rest.V1TaskStatusQUEUED:
		return styles.StatusQueuedColor
	default:
		return styles.MutedColor
	}
}

// DrawHorizontalLine draws a horizontal line from (x1, y) to (x2, y)
func (c *Canvas) DrawHorizontalLine(x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1 // Swap
	}

	style := lipgloss.NewStyle().Foreground(styles.MutedColor)

	for x := x1; x <= x2; x++ {
		c.SetCell(x, y, CharHorizontal, style)
	}
}

// DrawVerticalLine draws a vertical line from (x, y1) to (x, y2)
func (c *Canvas) DrawVerticalLine(x, y1, y2 int) {
	if y1 > y2 {
		y1, y2 = y2, y1 // Swap
	}

	style := lipgloss.NewStyle().Foreground(styles.MutedColor)

	for y := y1; y <= y2; y++ {
		c.SetCell(x, y, CharVertical, style)
	}
}

// DrawArrow draws a directional arrow character at the given position
func (c *Canvas) DrawArrow(x, y int, direction Direction) {
	style := lipgloss.NewStyle().Foreground(styles.AccentColor)

	var char string
	switch direction {
	case DirectionRight:
		char = CharArrowRight
	case DirectionDown:
		char = CharArrowDown
	case DirectionLeft:
		char = CharArrowLeft
	case DirectionUp:
		char = CharArrowUp
	default:
		return
	}

	c.SetCell(x, y, char, style)
}

// DrawPath draws a path connecting a series of coordinates
func (c *Canvas) DrawPath(path []Coord) {
	if len(path) < 2 {
		return
	}

	// Draw line segments between consecutive points
	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]

		if from.X == to.X {
			// Vertical line
			c.DrawVerticalLine(from.X, from.Y, to.Y)
		} else if from.Y == to.Y {
			// Horizontal line
			c.DrawHorizontalLine(from.X, to.X, from.Y)
		}
		// Note: Diagonal lines not supported in this simple implementation
	}

	// Add arrow at the end
	if len(path) >= 2 {
		lastSeg := path[len(path)-1]
		prevSeg := path[len(path)-2]

		switch {
		case lastSeg.X > prevSeg.X:
			c.DrawArrow(lastSeg.X, lastSeg.Y, DirectionRight)
		case lastSeg.X < prevSeg.X:
			c.DrawArrow(lastSeg.X, lastSeg.Y, DirectionLeft)
		case lastSeg.Y > prevSeg.Y:
			c.DrawArrow(lastSeg.X, lastSeg.Y, DirectionDown)
		case lastSeg.Y < prevSeg.Y:
			c.DrawArrow(lastSeg.X, lastSeg.Y, DirectionUp)
		}
	}
}

// ToString converts the canvas to a styled string
func (c *Canvas) ToString() string {
	var b strings.Builder

	// Use actual used dimensions, not full canvas
	effectiveWidth := c.usedWidth
	effectiveHeight := c.usedHeight

	if effectiveWidth > c.width {
		effectiveWidth = c.width
	}
	if effectiveHeight > c.height {
		effectiveHeight = c.height
	}

	for y := 0; y < effectiveHeight; y++ {
		for x := 0; x < effectiveWidth; x++ {
			cell := c.cells[x][y]
			// Apply style and render character
			b.WriteString(cell.Style.Render(cell.Char))
		}
		if y < effectiveHeight-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// Clear resets the canvas to empty
func (c *Canvas) Clear() {
	for x := 0; x < c.width; x++ {
		for y := 0; y < c.height; y++ {
			c.cells[x][y] = Cell{
				Char:  CharSpace,
				Style: lipgloss.NewStyle(),
			}
		}
	}
	c.usedWidth = 0
	c.usedHeight = 0
}
