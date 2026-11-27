package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The same visible ASCII spans drive both drawing and hit testing.
type editorButton struct {
	label string
	key   tea.KeyType
}

func (m projectEditorModel) buttonBar() (row int, buttons []editorButton) {
	if m.project == nil {
		row, buttons = 2, []editorButton{{"New", tea.KeyCtrlN}, {"Open", tea.KeyCtrlO}}
	} else if m.play != nil {
		row, buttons = 1, []editorButton{{"Stop", tea.KeyF8}, {"Inspect", tea.KeyTab}}
		if m.gameViewActive() {
			row = strings.Count(objectiveHUD(m.play.Objective()), "\n")
		} else {
			buttons = buttons[:1]
		}
	} else {
		row, buttons = 1, []editorButton{{"Play", 0}, {"Save", tea.KeyCtrlS}, {"Open", tea.KeyCtrlO}, {"New", tea.KeyCtrlN}}
	}
	end := 0
	for i, button := range buttons {
		end += len(button.label) + 2
		if end > max(1, m.width-1) {
			buttons = buttons[:i]
			break
		}
		end++
	}
	return row, buttons
}

func (m projectEditorModel) buttonText() string {
	_, buttons := m.buttonBar()
	labels := make([]string, len(buttons))
	for i, b := range buttons {
		labels[i] = "[" + b.label + "]"
	}
	return strings.Join(labels, " ")
}

func (m projectEditorModel) clickButton(msg tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	row, buttons := m.buttonBar()
	if m.modal == "discard" && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && msg.X >= 0 && msg.X < m.width-1 {
		lines := strings.Split(m.View(), "\n")
		if msg.Y >= 0 && msg.Y < len(lines) && strings.HasPrefix(lines[msg.Y], "[Discard]") {
			var key tea.KeyMsg
			switch {
			case msg.X < len("[Discard]"):
				key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
			case msg.X >= len("[Discard] ") && msg.X < len("[Discard] [Cancel]") && strings.HasPrefix(lines[msg.Y], "[Discard] [Cancel]"):
				key = tea.KeyMsg{Type: tea.KeyEsc}
			default:
				return m, nil, false
			}
			next, cmd := m.Update(key)
			return next, cmd, true
		}
	}
	if m.modal != "" || msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft || msg.Y != row || row >= m.height {
		return m, nil, false
	}
	x := 0
	for _, button := range buttons {
		end := x + len(button.label) + 2
		if msg.X >= x && msg.X < end {
			if button.key == 0 {
				cmd := m.startPlay()
				return m, cmd, true
			}
			next, cmd := m.Update(tea.KeyMsg{Type: button.key})
			return next, cmd, true
		}
		x = end + 1
	}
	return m, nil, false
}
