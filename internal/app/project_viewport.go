package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
	"strings"
)

// viewportRows returns the screen offset and visible height of the rendered frame.
func (m projectEditorModel) viewportRows() (top, visible int) {
	top = 3
	if m.play != nil {
		top += strings.Count(objectiveHUD(m.play.Objective()), "\n")
	}
	height := m.contentHeight()
	if m.width < 65 {
		if m.focus != 2 {
			return top, 0
		}
		height--
	}
	return top, min(len(m.frame), height, max(0, m.height-top))
}

func (m *projectEditorModel) viewportMouse(msg tea.MouseMsg) {
	if m.gameViewActive() || m.modal != "" || m.project == nil || msg.Action != tea.MouseActionPress {
		return
	}
	top, visible := m.viewportRows()
	y := msg.Y - top
	if y < 0 || y >= visible || msg.X < 0 || msg.X >= len(m.frame[y]) {
		return
	}
	switch msg.Button {
	case tea.MouseButtonLeft:
		m.focus = 2
		m.reticleU = (float64(msg.X) + .5) / float64(len(m.frame[y]))
		m.reticleV = (float64(y) + .5) / float64(len(m.frame))
		m.pickReticle()
		if m.transformTool != 0 {
			m.transformDrag = m.beginTransform(msg.X, msg.Y)
		}
	case tea.MouseButtonWheelUp:
		m.zoom = max(.25, m.zoom*.8)
		m.renderPreview()
	case tea.MouseButtonWheelDown:
		m.zoom = min(5, m.zoom*1.25)
		m.renderPreview()
	}
}

func (m *projectEditorModel) colorPreview(renderer *terminal.TerminalRenderer, x, y int) {
	m.coloredFrame = nil
	if m.colorMode != "" && m.colorMode != "none" {
		m.coloredFrame = coloredRows(renderer, x, y, m.colorMode)
	}
}

func (m projectEditorModel) stylePreview(view string) string {
	if m.modal == "" && m.project != nil && len(m.coloredFrame) > 0 {
		lines := strings.Split(view, "\n")
		top, visible := m.viewportRows()
		for i := 0; i < visible && top+i < len(lines); i++ {
			// Frame glyphs are single-column ASCII before coloring.
			if strings.HasPrefix(lines[top+i], m.frame[i]) {
				sidebar := lines[top+i][len(m.frame[i]):]
				if strings.HasPrefix(sidebar, " | > ") {
					sidebar = " | \x1b[1;33m" + sidebar[3:] + "\x1b[0m"
				}
				lines[top+i] = m.coloredFrame[i] + sidebar
			}
		}
		lines[0] = "\x1b[1;36m" + lines[0] + "\x1b[0m"
		view = strings.Join(lines, "\n")
	}
	return view
}

func coloredRows(renderer *terminal.TerminalRenderer, x, y int, mode string) []string {
	rows := make([]string, 0, len(renderer.Surface))
	codes := map[engine.Color]string{}
	for i, cells := range renderer.Surface {
		var line strings.Builder
		previous := ""
		for j, glyph := range cells {
			color := renderer.ColorBuffer[i][j]
			if i == y && j == x {
				glyph, color = '+', engine.ColorWhite
			}
			if glyph != ' ' {
				code, ok := codes[color]
				if !ok {
					code = terminal.Foreground(color, mode)
					if code == "\x1b[30m" {
						code = "\x1b[90m"
					}
					codes[color] = code
				}
				if code != previous {
					line.WriteString(code)
					previous = code
				}
				if x < 0 || glyph != '+' {
					glyph = engine.FILLED_CHAR
				}
			}
			line.WriteRune(glyph)
		}
		if previous != "" {
			line.WriteString("\x1b[0m")
		}
		rows = append(rows, line.String())
	}
	return rows
}
