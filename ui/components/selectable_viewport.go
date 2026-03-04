package components

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectionStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("33")).
			Foreground(lipgloss.Color("255"))
)

// SelectableViewport is a viewport that supports text selection with mouse.
type SelectableViewport struct {
	width      int
	height     int
	content    string
	lines      []string
	yOffset    int // 滚动偏移

	// 选择状态
	selecting  bool
	selectStartX int
	selectStartY int
	selectEndX   int
	selectEndY   int
	hasSelection bool

	// 鼠标拖拽状态
	dragging   bool
}

// NewSelectableViewport creates a new selectable viewport.
func NewSelectableViewport(width, height int) SelectableViewport {
	return SelectableViewport{
		width:  width,
		height: height,
		lines:  []string{},
	}
}

// SetSize updates the viewport dimensions.
func (v SelectableViewport) SetSize(width, height int) SelectableViewport {
	v.width = width
	v.height = height
	return v
}

// SetContent replaces all content.
func (v SelectableViewport) SetContent(content string) SelectableViewport {
	v.content = content
	// 注意：不在这里分割 lines，由 SetLines 单独设置
	return v
}

// SetLines sets the raw text lines for selection (without ANSI codes).
func (v SelectableViewport) SetLines(lines []string) SelectableViewport {
	v.lines = lines
	return v
}

// Clear resets the viewport.
func (v SelectableViewport) Clear() SelectableViewport {
	v.content = ""
	v.lines = []string{}
	v.hasSelection = false
	v.selecting = false
	v.dragging = false
	return v
}

// Update handles mouse and keyboard events.
func (v SelectableViewport) Update(msg tea.Msg) (SelectableViewport, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		return v.handleMouse(msg)

	case tea.KeyMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

// handleMouse processes mouse events for text selection.
func (v SelectableViewport) handleMouse(msg tea.MouseMsg) (SelectableViewport, tea.Cmd) {
	// 转换鼠标坐标到视口内容坐标
	mouseX := msg.X - 2 // 减去左边距
	mouseY := msg.Y - 4 // 减去顶部栏高度

	// 检查是否在视口区域内
	if mouseX < 0 || mouseX >= v.width || mouseY < 0 || mouseY >= v.height {
		if v.dragging {
			// 鼠标移出区域，结束拖拽
			v.dragging = false
			v.selecting = false
		}
		return v, nil
	}

	// 转换到内容坐标（考虑滚动）
	contentY := mouseY + v.yOffset
	if contentY < 0 || contentY >= len(v.lines) {
		return v, nil
	}

	switch msg.Type {
	case tea.MouseLeft:
		// 开始新的选择
		v.dragging = true
		v.selecting = true
		v.hasSelection = false
		v.selectStartX = mouseX
		v.selectStartY = contentY
		v.selectEndX = mouseX
		v.selectEndY = contentY

	case tea.MouseRelease:
		if v.dragging {
			v.dragging = false
			if v.selectStartX != v.selectEndX || v.selectStartY != v.selectEndY {
				v.hasSelection = true
			} else {
				v.hasSelection = false
				v.selecting = false
			}
		}

	case tea.MouseMotion:
		if v.dragging && v.selecting && msg.Button == tea.MouseButtonLeft {
			v.selectEndX = mouseX
			v.selectEndY = contentY
			v.hasSelection = true
		}
	}

	return v, nil
}

// handleKey processes keyboard events.
func (v SelectableViewport) handleKey(msg tea.KeyMsg) (SelectableViewport, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		// 如果有选中文本，复制到剪贴板
		if v.hasSelection {
			text := v.GetSelectedText()
			if text != "" {
				clipboard.WriteAll(text)
			}
		}
	case "pgup":
		v.yOffset -= v.height
		if v.yOffset < 0 {
			v.yOffset = 0
		}
	case "pgdown":
		v.yOffset += v.height
		maxOffset := len(v.lines) - v.height
		if maxOffset < 0 {
			maxOffset = 0
		}
		if v.yOffset > maxOffset {
			v.yOffset = maxOffset
		}
	case "up":
		if v.yOffset > 0 {
			v.yOffset--
		}
	case "down":
		maxOffset := len(v.lines) - v.height
		if maxOffset < 0 {
			maxOffset = 0
		}
		if v.yOffset < maxOffset {
			v.yOffset++
		}
	case "home":
		v.yOffset = 0
	case "end":
		maxOffset := len(v.lines) - v.height
		if maxOffset < 0 {
			maxOffset = 0
		}
		v.yOffset = maxOffset
	}
	return v, nil
}

// GetSelectedText returns the currently selected text.
func (v SelectableViewport) GetSelectedText() string {
	if !v.hasSelection {
		return ""
	}

	// 规范化选择范围
	startX, startY, endX, endY := v.normalizeSelection()

	var selected []string
	for y := startY; y <= endY; y++ {
		if y < 0 || y >= len(v.lines) {
			continue
		}
		line := v.lines[y]

		if y == startY && y == endY {
			// 单行选择
			if startX < len(line) {
				if endX > len(line) {
					endX = len(line)
				}
				selected = append(selected, line[startX:endX])
			}
		} else if y == startY {
			// 起始行
			if startX < len(line) {
				selected = append(selected, line[startX:])
			}
		} else if y == endY {
			// 结束行
			if endX > len(line) {
				endX = len(line)
			}
			selected = append(selected, line[:endX])
		} else {
			// 中间行
			selected = append(selected, line)
		}
	}

	return strings.Join(selected, "\n")
}

// normalizeSelection returns normalized selection coordinates.
func (v SelectableViewport) normalizeSelection() (startX, startY, endX, endY int) {
	startX = v.selectStartX
	startY = v.selectStartY
	endX = v.selectEndX
	endY = v.selectEndY

	// 确保 start 在 end 之前
	if startY > endY || (startY == endY && startX > endX) {
		startX, endX = endX, startX
		startY, endY = endY, startY
	}

	return
}

// ClearSelection clears the current selection.
func (v SelectableViewport) ClearSelection() SelectableViewport {
	v.hasSelection = false
	v.selecting = false
	v.dragging = false
	return v
}

// View renders the viewport with selection highlighting.
func (v SelectableViewport) View() string {
	if len(v.lines) == 0 {
		return ""
	}

	// 计算可见行范围
	startLine := v.yOffset
	endLine := v.yOffset + v.height
	if endLine > len(v.lines) {
		endLine = len(v.lines)
	}

	var visibleLines []string
	for i := startLine; i < endLine; i++ {
		line := v.lines[i]
		renderedLine := v.renderLine(line, i)
		visibleLines = append(visibleLines, "  "+renderedLine)
	}

	return strings.Join(visibleLines, "\n")
}

// renderLine renders a single line with selection highlighting.
func (v SelectableViewport) renderLine(line string, lineIndex int) string {
	if !v.hasSelection {
		return line
	}

	startX, startY, endX, endY := v.normalizeSelection()

	// 检查当前行是否在选择范围内
	if lineIndex < startY || lineIndex > endY {
		return line
	}

	var parts []string

	if lineIndex == startY && lineIndex == endY {
		// 单行选择
		if startX > 0 {
			parts = append(parts, line[:min(startX, len(line))])
		}
		if startX < len(line) {
			selectedEnd := min(endX, len(line))
			parts = append(parts, selectionStyle.Render(line[startX:selectedEnd]))
		}
		if endX < len(line) {
			parts = append(parts, line[endX:])
		}
	} else if lineIndex == startY {
		// 起始行
		if startX > 0 {
			parts = append(parts, line[:min(startX, len(line))])
		}
		if startX < len(line) {
			parts = append(parts, selectionStyle.Render(line[startX:]))
		}
	} else if lineIndex == endY {
		// 结束行
		if endX > 0 {
			selectedPart := line[:min(endX, len(line))]
			parts = append(parts, selectionStyle.Render(selectedPart))
		}
		if endX < len(line) {
			parts = append(parts, line[endX:])
		}
	} else {
		// 中间行 - 全部选中
		parts = append(parts, selectionStyle.Render(line))
	}

	return strings.Join(parts, "")
}

// AtBottom reports whether the viewport is scrolled to the bottom.
func (v SelectableViewport) AtBottom() bool {
	if len(v.lines) <= v.height {
		return true
	}
	return v.yOffset >= len(v.lines)-v.height
}

// GotoBottom scrolls to the bottom.
func (v SelectableViewport) GotoBottom() SelectableViewport {
	maxOffset := len(v.lines) - v.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	v.yOffset = maxOffset
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
