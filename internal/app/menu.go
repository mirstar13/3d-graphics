package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
)

type menuPage int

const (
	homePage menuPage = iota
	scenePage
	settingsPage
)

type menuModel struct {
	settings      config.Settings
	path          string
	page          menuPage
	selected      int
	scene         int
	width, height int
	editor        settingsModel
	start         bool
}

func newMenuModel(settings config.Settings, path string, openSettings bool) menuModel {
	model := menuModel{
		settings: settings, path: path, scene: settings.Demo - 1, width: 80, height: 24,
	}
	if openSettings {
		model.openSettings()
	}
	return model
}

func (m *menuModel) openSettings() {
	m.editor = newSettingsModel(m.settings, m.path)
	m.editor.width, m.editor.height = m.width, m.height
	m.editor.returnToMenu = true
	m.page = settingsPage
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := message.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.editor.width, m.editor.height = size.Width, size.Height
		return m, nil
	}
	if key, ok := message.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.page == settingsPage {
		// Escape during number editing cancels that edit, preserving the form.
		if key, ok := message.(tea.KeyMsg); ok && !m.editor.editing && (key.String() == "esc" || key.String() == "q") {
			m.settings = m.editor.settings
			m.page = homePage
			return m, nil
		}
		updated, cmd := m.editor.Update(message)
		m.editor = updated.(settingsModel)
		if m.editor.start {
			m.settings = m.editor.settings
			m.scene = m.settings.Demo - 1
			m.page = homePage
			return m, nil
		}
		return m, cmd
	}
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.page == scenePage {
		switch key.String() {
		case "esc", "q":
			m.page = homePage
		case "up", "k":
			m.scene = (m.scene + len(demo.Names) - 1) % len(demo.Names)
		case "down", "j", "tab":
			m.scene = (m.scene + 1) % len(demo.Names)
		case "enter":
			m.settings.Demo, m.start = m.scene+1, true
			return m, tea.Quit
		}
		return m, nil
	}
	switch key.String() {
	case "esc", "q":
		return m, tea.Quit
	case "up", "k":
		m.selected = (m.selected + 2) % 3
	case "down", "j", "tab":
		m.selected = (m.selected + 1) % 3
	case "enter":
		switch m.selected {
		case 0:
			m.scene, m.page = m.settings.Demo-1, scenePage
		case 1:
			m.openSettings()
		case 2:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	if m.page == settingsPage {
		return m.editor.View()
	}
	title, help := "3D Graphics", "Up/Down select | Enter open | Esc exit"
	items := []string{"New scene", "Settings", "Exit"}
	selected := m.selected
	if m.page == scenePage {
		title, help = "New scene", "Up/Down select | Enter start | Esc back"
		items, selected = demo.Names, m.scene
	}
	var out strings.Builder
	out.WriteString(title + "\n\n")
	visible := max(1, m.height-5)
	first := max(0, selected-visible+1)
	for index := first; index < min(len(items), first+visible); index++ {
		prefix := "  "
		if index == selected {
			prefix = "> "
		}
		fmt.Fprintf(&out, "%s%s\n", prefix, items[index])
	}
	out.WriteString("\n" + help)
	return fitView(out.String(), m.width, m.height)
}

func showStartMenu(settings config.Settings, path string, openSettings bool) (config.Settings, bool, error) {
	result, err := tea.NewProgram(newMenuModel(settings, path, openSettings), tea.WithAltScreen()).Run()
	if err != nil {
		return settings, false, err
	}
	model := result.(menuModel)
	return model.settings, model.start, nil
}
