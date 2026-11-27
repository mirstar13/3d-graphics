package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/input"
)

func (m projectEditorModel) commands() []editorButton {
	common := []editorButton{
		{"Hierarchy", "ui.hierarchy"}, {"Inspector", "ui.inspector"}, {"Viewport", "ui.viewport"},
		{"Clear selection", input.Clear}, {"Zoom in", "+"}, {"Zoom out", "-"},
	}
	if m.play != nil {
		return append(common, []editorButton{
			{"Pause / resume", input.Pause}, {"Step", input.Step}, {"Impulse", input.Impulse},
			{"Play tools", "ctrl+l"}, {"Mute audio", "ctrl+y"}, {"Stop Play", "f8"},
		}...)
	}
	commands := append(common, []editorButton{
		{"Add box", "b"}, {"Add sphere", "g"}, {"Edit selected property", "enter"},
		{"Duplicate entity", "ctrl+d"}, {"Delete entity", "delete"}, {"Set parent", "h"},
		{"Undo", "ctrl+z"}, {"Redo", "ctrl+y"},
		{"Front camera", "1"}, {"Top camera", "2"}, {"Perspective camera", "3"},
		{"Audio cues", "ctrl+k"}, {"Mesh assets", "ctrl+x"}, {"Scene resources", "a"},
		{"Extract resource", "r"}, {"Refresh linked resources", "ctrl+r"},
		{"Unpack instance", "ctrl+u"}, {"Reset selected override", "ctrl+t"},
		{"Key bindings", "k"}, {"Play", input.Play}, {"Scripted Play", "ctrl+w"},
	}...)
	if m.activeResource == nil {
		commands = append(commands, []editorButton{
			{"Scene settings", "ctrl+e"}, {"Build game", "ctrl+b"}, {"Run build and capture output", "f6"},
			{"Run built game interactively", "ctrl+g"}, {"Export game", "f9"},
		}...)
	}
	return append(commands, []editorButton{{"Save", "ctrl+s"}, {"Open project", "ctrl+o"}, {"New project", "ctrl+n"}, {"Shortcuts", "?"}, {"Exit editor", "q"}}...)
}

func (m projectEditorModel) commandKey(key string) (tea.Model, tea.Cmd) {
	commands := m.commands()
	switch key {
	case "esc", "ctrl+c":
		m.modal = ""
	case "up":
		m.actionSelection = max(0, m.actionSelection-1)
	case "down", "tab":
		m.actionSelection = min(len(commands)-1, m.actionSelection+1)
	case "enter":
		action := commands[m.actionSelection]
		m.modal = ""
		return m.handleEditorKey(tea.KeyMsg{}, action.key)
	}
	return m, nil
}
