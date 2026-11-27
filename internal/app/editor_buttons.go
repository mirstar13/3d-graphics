package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/input"
)

type editorButton struct{ label, key string }

func (m projectEditorModel) buttonBar() (row int, buttons []editorButton) {
	switch {
	case m.project == nil:
		row, buttons = 2, []editorButton{{"New", "ctrl+n"}, {"Open", "ctrl+o"}, {"Exit", "q"}}
	case m.play != nil:
		row = 1
		buttons = []editorButton{{"Stop", "f8"}, {"Actions", "ui.commands"}}
		if m.gameViewActive() {
			row = strings.Count(objectiveHUD(m.play.Objective()), "\n")
		}
		if m.play.World.Stats().Paused {
			buttons = append(buttons, editorButton{"Resume", input.Pause}, editorButton{"Step", input.Step})
		} else {
			buttons = append(buttons, editorButton{"Pause", input.Pause})
		}
	default:
		row, buttons = 1, []editorButton{{"Play", input.Play}, {"Actions", "ui.commands"}, {"Save", "ctrl+s"}, {"Open", "ctrl+o"}, {"New", "ctrl+n"}}
	}
	if m.project != nil && m.width < 18 {
		buttons = []editorButton{{"Actions", "ui.commands"}}
	}
	end := 0
	for i, button := range buttons {
		end += len(button.label) + 2
		if end > max(1, m.width-1) {
			return row, buttons[:i]
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
	if m.modal != "" || msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft || msg.Y != row || row >= m.height {
		return m, nil, false
	}
	x := 0
	for _, button := range buttons {
		end := x + len(button.label) + 2
		if msg.X >= x && msg.X < end {
			next, cmd := m.handleEditorKey(tea.KeyMsg{}, button.key)
			return next, cmd, true
		}
		x = end + 1
	}
	return m, nil, false
}
