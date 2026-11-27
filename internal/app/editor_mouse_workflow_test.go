package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/mirstar13/go-3d-graphics/internal/config"
)

func TestEditorMouseAuthorSaveAndBrowse(t *testing.T) {
	m := newProjectEditor(config.Default())
	t.Cleanup(m.audio.Close)
	click := func(label string) bool {
		for y, row := range strings.Split(m.View(), "\n") {
			if i := strings.Index(row, label); i >= 0 {
				msg := tea.MouseMsg{X: runewidth.StringWidth(row[:i]), Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
				next, _ := m.Update(msg)
				m = next.(projectEditorModel)
				return true
			}
		}
		return false
	}
	require := func(label string) {
		t.Helper()
		if !click(label) {
			t.Fatalf("missing clickable %q in\n%s", label, m.View())
		}
	}
	require("[New]")
	parent := t.TempDir()
	// Position the browser at the fixture directory, then use mouse clicks
	// for path selection, filename entry, authoring and saving.
	m.draft = parent
	require("[Browse]")
	require("[Use folder]")
	require("[Keys]")
	for _, letter := range "demo" {
		require("[" + string(letter) + "]")
	}
	require("[Confirm]")
	if m.project == nil || m.project.Root != filepath.Join(parent, "demo") {
		t.Fatalf("create project failed, %s", m.message)
	}
	require("[Actions]")
	require("Add box")
	require("[Actions]")
	require("Edit selected property")
	require("[Clear]")
	for _, letter := range "test" {
		require("[" + string(letter) + "]")
	}
	require("[Apply]")
	require("[Save]")
	data, err := os.ReadFile(filepath.Join(m.project.Root, "scenes", "main.yaml"))
	if err != nil || !strings.Contains(string(data), "name: test") {
		t.Fatalf("mouse edit not saved: %v\n%s", err, data)
	}
	require("[Play]")
	require("[Pause]")
	require("[Step]")
	if m.play.World.Stats().Tick != 1 {
		t.Fatal("step button did not advance paused play")
	}
	require("[Stop]")
	if m.play != nil {
		t.Fatal("stop button left Play running")
	}
}
