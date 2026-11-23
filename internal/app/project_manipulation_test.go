package app

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestPlayToolsSpawnDeleteResetIsolation(t *testing.T) {
	m := inputEditor(t)
	authored := m.document.Document()
	m.selectID("box")
	m, _ = editorRune(m, "p")
	m = resourceKey(m, tea.KeyCtrlL)
	if m.modal != "tools" || !m.play.World.Stats().Paused {
		t.Fatal("tools must pause")
	}
	m, _ = editorRune(m, "d")
	if !strings.Contains(m.message, "protected") {
		t.Fatal("authored deletion", m.message)
	}
	m, _ = editorRune(m, "g")
	if m.modal != "spawn_confirm" {
		t.Fatal("spawn preview", m.message)
	}
	m = resourceKey(m, tea.KeyEsc)
	if m.play.PendingCommands() != 0 || len(m.play.Hierarchy()) != 2 {
		t.Fatal("cancel spawned an object")
	}
	m = resourceKey(m, tea.KeyCtrlL)
	m, _ = editorRune(m, "g")
	m = resourceKey(m, tea.KeyEnter)
	if m.play.PendingCommands() != 1 || m.modal != "tools" {
		t.Fatal("spawn queue", m.message)
	}
	m = resourceKey(m, tea.KeyEsc)
	m, _ = editorRune(m, "n")
	selected, ok := m.selectedEntity()
	if !ok || !strings.HasPrefix(selected.Name, "Runtime") {
		t.Fatal("spawn selection", selected, m.message)
	}
	id, _ := m.play.Find(selected.ID)
	rows := strings.Join(m.inspectorRows(), "\n")
	if !strings.Contains(rows, "Runtime spawn: true") || !strings.Contains(rows, "Velocity:") || !strings.Contains(rows, "Sleeping:") {
		t.Fatal("live inspector", rows)
	}
	m = resourceKey(m, tea.KeyCtrlL)
	m, _ = editorRune(m, "d")
	m = resourceKey(m, tea.KeyEsc)
	m, _ = editorRune(m, "n")
	if _, ok = m.selectedEntity(); ok {
		t.Fatal("stale selection survived deletion")
	}
	if _, err := m.play.State(id); err == nil {
		t.Fatal("stale live handle")
	}
	m.selectID("box")
	old, _ := m.play.Find("box")
	m, _ = editorRune(m, "i")
	if m.play.PendingCommands() != 1 {
		t.Fatal("impulse not queued")
	}
	m = resourceKey(m, tea.KeyCtrlL)
	m, _ = editorRune(m, "r")
	if m.play.PendingCommands() != 0 || m.play.World.Stats().Tick != 0 || !m.play.World.Stats().Paused || m.runtimeSelection != "" {
		t.Fatal("reset state")
	}
	if _, err := m.play.State(old); err == nil {
		t.Fatal("reset kept old handle")
	}
	m = resourceKey(m, tea.KeyEsc)
	m = resourceKey(m, tea.KeyEsc)
	if m.play != nil || !reflect.DeepEqual(authored, m.document.Document()) {
		t.Fatal("Stop modified authoring")
	}
	e, _ := m.selectedEntity()
	if e.ID != "box" {
		t.Fatal("authored selection lost")
	}
}
func TestToolValidationHitModeAndModalFocus(t *testing.T) {
	m := inputEditor(t)
	m.selectID("box")
	m, _ = editorRune(m, "p")
	m = resourceKey(m, tea.KeyCtrlL)
	original := m.toolValues
	m = resourceKey(m, tea.KeyEnter)
	m.draft = "NaN"
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "tool_value" || m.toolValues != original {
		t.Fatal("invalid field applied")
	}
	m.draft = "3"
	m = resourceKey(m, tea.KeyEnter)
	if m.toolValues[0] != "3" {
		t.Fatal("value edit", m.message)
	}
	m.toolValues[3] = "hit"
	m, _ = editorRune(m, "i")
	if m.play.PendingCommands() != 0 || !strings.Contains(m.message, "Pick") {
		t.Fatal("hit mode without anchor")
	}
	id, _ := m.play.Find("box")
	state, _ := m.play.State(id)
	anchor, err := m.play.AnchorAt(id, engine.Point{X: state.Position.X + .5, Y: state.Position.Y, Z: state.Position.Z})
	if err != nil {
		t.Fatal(err)
	}
	m.hitAnchor = &anchor
	m, _ = editorRune(m, "i")
	if m.play.PendingCommands() != 1 {
		t.Fatal("anchored impulse not queued", m.message)
	}
	// Text input cannot spawn, step, reset, or apply impulses.
	m = resourceKey(m, tea.KeyEnter)
	m, _ = editorRune(m, "gnir")
	if m.play.PendingCommands() != 1 || m.play.World.Stats().Tick != 0 {
		t.Fatal("tool text leaked commands")
	}
	m = resourceKey(m, tea.KeyEsc)
	m = resourceKey(m, tea.KeyCtrlL)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 14})
	m = next.(projectEditorModel)
	if !strings.Contains(m.View(), "Play tools") || !strings.Contains(m.View(), "Esc close") {
		t.Fatal("narrow tools", m.View())
	}
}
