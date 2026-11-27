package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

type editorRow struct {
	text, entity string
	property     int // One-based so headers have no action.
}

func (m *projectEditorModel) sidebarRows() []editorRow {
	height := m.contentHeight()
	compact := m.width < 65 || height < 10
	label := "local"
	if m.play != nil {
		label = "live, read-only"
	}
	if compact && m.focus == 2 {
		rows := []editorRow{{text: "Viewport (Tab switches focus)"}}
		if m.width < 65 {
			for _, line := range m.frame {
				rows = append(rows, editorRow{text: line})
			}
		}
		return rows
	}
	if compact && m.focus == 1 {
		return append([]editorRow{{text: "Inspector (" + label + "; Tab: Viewport)"}}, m.inspectorSlice(max(1, height-1))...)
	}
	entities := m.hierarchy()
	selected := m.selectionIndex()
	heading, first, count := "── Hierarchy (Tab switches focus)", max(0, selected-2), 4
	if compact {
		heading, first, count = "Hierarchy (Tab: Inspector)", max(0, selected-height+2), height-1
	}
	rows := []editorRow{{text: heading}}
	for i := first; i < min(len(entities), first+count); i++ {
		e := entities[i]
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		if _, _, ok := m.document.Document().Link(e.ID); ok {
			e.Name += " [linked]"
		}
		rows = append(rows, editorRow{text: prefix + strings.Repeat("  ", min(e.Depth, 8)) + e.Name, entity: e.ID})
	}
	if compact {
		return rows
	}
	if len(entities) == 0 {
		rows = append(rows, editorRow{text: "(empty: B box / G sphere)"})
	}
	focus := []string{"Hierarchy", "Inspector", "Viewport"}[m.focus]
	rows = append(rows, editorRow{}, editorRow{text: "Inspector (" + label + ") - focus: " + focus})
	return append(rows, m.inspectorSlice(max(2, len(m.frame)-len(rows)))...)
}

func (m projectEditorModel) clickSidebar(msg tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	if m.document == nil || m.modal != "" || m.gameViewActive() || msg.X < 0 || msg.Y >= m.height || msg.Action != tea.MouseActionPress {
		return m, nil, false
	}
	top, x := 3, 0
	if m.width >= 65 {
		x = max(12, m.width-4-min(42, m.width/2)) + 3
	}
	if m.play != nil {
		top += strings.Count(objectiveHUD(m.play.Objective()), "\n")
	}
	y := msg.Y - top
	if y < 0 || y >= m.contentHeight() || msg.X < x || msg.X >= m.width-1 {
		return m, nil, false
	}
	rows := m.sidebarRows()
	if m.width < 65 && m.focus == 2 && y > 0 {
		return m, nil, false
	}
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		delta := 1
		if msg.Button == tea.MouseButtonWheelUp {
			delta = -1
		}
		inspector := m.focus == 1
		if y < len(rows) && m.width >= 65 {
			if rows[y].property > 0 || strings.HasPrefix(rows[y].text, "Inspector") {
				inspector = true
			} else if rows[y].entity != "" || strings.Contains(rows[y].text, "Hierarchy") {
				inspector = false
			}
		}
		if inspector {
			m.focus = 1
			m.property = max(0, min(len(m.inspectorRows())-1, m.property+delta))
		} else if m.focus != 2 || m.width >= 65 {
			m.focus = 0
			m.cycleSelection(delta)
		}
		return m, nil, true
	}
	if msg.Button != tea.MouseButtonLeft {
		return m, nil, false
	}
	if y >= len(rows) || msg.X-x >= runewidth.StringWidth(rows[y].text) {
		return m, nil, false
	}
	row := rows[y]
	if strings.Contains(row.text, "Tab") && row.entity == "" && row.property == 0 &&
		(strings.HasPrefix(row.text, "Hierarchy") || strings.HasPrefix(row.text, "Inspector") || strings.HasPrefix(row.text, "Viewport")) {
		m.focus = (m.focus + 1) % 3
		m.renderPreview()
		return m, nil, true
	}
	if row.entity != "" {
		m.selectID(row.entity)
		m.focus, m.property = 0, 0
		m.renderPreview()
		return m, nil, true
	}
	if row.property > 0 {
		m.focus, m.property = 1, row.property-1
		if m.play == nil {
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			return next, cmd, true
		}
		return m, nil, true
	}
	return m, nil, false
}
