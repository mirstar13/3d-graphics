package app

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/config"
)

func TestFileAndFlagPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "renderer.yaml")
	if err := os.WriteFile(path, []byte("fps: 15\nworkers: 2\nanti_aliasing: fxaa\nfov_x: 85\n"), 0644); err != nil {
		t.Fatal(err)
	}
	settings, _, err := readCommand([]string{"-config", path, "-fps", "60", "-debug=false"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if settings.FPS != 60 || settings.Workers != 2 || settings.FOVX != 85 || settings.AntiAliasing != "fxaa" || settings.ShowDebug {
		t.Fatalf("bad precedence: %+v", settings)
	}
	if _, _, err := readCommand([]string{"-config", path + ".missing"}, io.Discard); err == nil {
		t.Fatal("missing explicit config accepted")
	}
}

func TestSettingsEditorSaveAndApply(t *testing.T) {
	path := filepath.Join(t.TempDir(), "renderer.yaml")
	m := newSettingsModel(config.Default(), path)
	m.selected = 1
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = result.(settingsModel)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = result.(settingsModel)
	saved, err := config.Load(path)
	if err != nil || saved.RenderMode != "tiles" {
		t.Fatalf("save: %+v %v", saved, err)
	}
	m.fields[9].value = "90"
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(settingsModel)
	if cmd == nil || !m.start || m.settings.FOVX != 90 {
		t.Fatal("editor did not apply FOV")
	}
	m.start = false
	m.fields[9].value = "nan"
	result, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(settingsModel)
	if cmd != nil || m.start || m.message == "" {
		t.Fatal("editor accepted invalid FOV")
	}
}

func TestSettingsEditorCancelAndResize(t *testing.T) {
	m := newSettingsModel(config.Default(), "unused.yaml")
	m.selected = len(m.fields) - 1
	result, _ := m.Update(tea.WindowSizeMsg{Width: 32, Height: 12})
	m = result.(settingsModel)
	if m.View() == "" {
		t.Fatal("empty small-terminal view")
	}
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = result.(settingsModel)
	if cmd == nil || m.start {
		t.Fatal("cancel started rendering")
	}
}
