package app

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
)

func pressMenu(m menuModel, key tea.KeyType) (menuModel, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: key})
	return updated.(menuModel), cmd
}

func TestStartMenuSceneSelection(t *testing.T) {
	settings := config.Default()
	settings.Demo = 3
	m := newMenuModel(settings, "unused.yaml", false)
	if !strings.Contains(m.View(), "> Load scene") {
		t.Fatal("launcher missing default selection")
	}
	m, _ = pressMenu(m, tea.KeyEnter)
	if !strings.Contains(m.View(), demo.PresetAt(2).Description) {
		t.Fatal("scene description missing")
	}
	if m.page != scenePage || m.scene != 2 {
		t.Fatal("scene picker lost configured demo")
	}
	m, _ = pressMenu(m, tea.KeyDown)
	m, cmd := pressMenu(m, tea.KeyEnter)
	if cmd == nil || !m.start || m.settings.Demo != 4 {
		t.Fatal("scene selection did not start chosen demo")
	}
}

func TestStartMenuSettingsApplyAndBack(t *testing.T) {
	m := newMenuModel(config.Default(), filepath.Join(t.TempDir(), "settings.yaml"), true)
	m.editor.fields[9].value = "85"
	m, cmd := pressMenu(m, tea.KeyEnter)
	if cmd != nil || m.page != homePage || m.settings.FOVX != 85 || m.start {
		t.Fatal("apply should return to menu")
	}
	m.selected = 1
	m, _ = pressMenu(m, tea.KeyEnter)
	m.editor.fields[9].value = "100"
	m, cmd = pressMenu(m, tea.KeyEsc)
	if cmd != nil || m.page != homePage || m.settings.FOVX != 85 {
		t.Fatal("back should discard unapplied edits")
	}
	m, _ = pressMenu(m, tea.KeyEnter)
	m.editor.fields[9].value = "95"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(menuModel)
	m, _ = pressMenu(m, tea.KeyEsc)
	if m.settings.FOVX != 95 {
		t.Fatal("saved settings lost on back")
	}
}

func TestStartMenuExitAndSmallScenePicker(t *testing.T) {
	m := newMenuModel(config.Default(), "unused.yaml", false)
	m.selected = 3
	m, cmd := pressMenu(m, tea.KeyEnter)
	if cmd == nil || m.start {
		t.Fatal("exit should quit without rendering")
	}
	m.page = scenePage
	m.scene = len(demo.Names) - 1
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 32, Height: 10})
	m = updated.(menuModel)
	if !strings.Contains(m.View(), "> "+demo.Names[len(demo.Names)-1]) {
		t.Fatal("selected scene hidden in small terminal")
	}
	m, cmd = pressMenu(m, tea.KeyEsc)
	if cmd != nil || m.page != homePage {
		t.Fatal("scene picker back should return home")
	}
}

func TestMenuEscapeCancelsNumberEdit(t *testing.T) {
	m := newMenuModel(config.Default(), "unused.yaml", true)
	m.editor.editing = true
	m, cmd := pressMenu(m, tea.KeyEsc)
	if cmd != nil || m.page != settingsPage || m.editor.editing {
		t.Fatal("escape closed form while editing number")
	}
}

func TestLauncherProjectRoutes(t *testing.T) {
	m := newMenuModel(config.Default(), "", false)
	m.selected = 2
	created, cmd := pressMenu(m, tea.KeyEnter)
	if cmd == nil || !created.projectEditor || created.editorAction != "new" {
		t.Fatal("new project route")
	}
	m.page = scenePage
	for i, action := range []string{"sample", "open"} {
		m.scene = len(demo.Names) + i
		loaded, cmd := pressMenu(m, tea.KeyEnter)
		if action == "sample" {
			if !loaded.playSample || loaded.projectEditor || cmd == nil {
				t.Fatal("sample should launch directly")
			}
			continue
		}
		if cmd == nil || !loaded.projectEditor || loaded.editorAction != action || loaded.start {
			t.Fatalf("load %s route", action)
		}
	}
}
