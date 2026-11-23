package app

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/input"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func inputEditor(t *testing.T) projectEditorModel {
	t.Helper()
	p, err := project.Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	m := newProjectEditor(config.Default())
	if err = m.install(p); err != nil {
		t.Fatal(err)
	}
	return m
}
func TestEditorRebindingFocusAndQueuedImpulse(t *testing.T) {
	m := inputEditor(t)
	m, _ = editorRune(m, "k")
	m, _ = editorRune(m, "p")
	if m.play != nil || m.modal != "bindings" {
		t.Fatal("binding dialog leaked Play")
	}
	m = resourceKey(m, tea.KeyEnter)
	m.draft = "q"
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "binding" || len(m.project.Manifest.Bindings) != 0 {
		t.Fatal("invalid binding committed")
	}
	m.draft = "v"
	m = resourceKey(m, tea.KeyEnter)
	if input.Resolve(m.project.Manifest.Bindings, "v", false) != input.Play {
		t.Fatal("remap failed", m.message)
	}
	m = resourceKey(m, tea.KeyEsc)
	if !strings.Contains(m.View(), "v Play") {
		t.Fatal("help did not reflect binding")
	}
	m, _ = editorRune(m, "p")
	if m.play != nil {
		t.Fatal("removed default still works")
	}
	m, _ = editorRune(m, "e")
	m, _ = editorRune(m, "v")
	if m.play != nil {
		t.Fatal("text editing leaked action")
	}
	m = resourceKey(m, tea.KeyEsc)
	m.selectID("box")
	m, _ = editorRune(m, "v")
	m, _ = editorRune(m, " ")
	m, _ = editorRune(m, "i")
	if m.play == nil || !m.play.World.Stats().Paused || m.play.PendingCommands() != 1 {
		t.Fatal("semantic impulse dispatch")
	}
	tick := m.play.World.Stats().Tick
	m, _ = editorRune(m, "n")
	if m.play.World.Stats().Tick != tick+1 || m.play.PendingCommands() != 0 {
		t.Fatal("step did not consume once")
	}
	m, _ = editorRune(m, "n")
	if m.play.PendingCommands() != 0 {
		t.Fatal("impulse replayed")
	}
	m = resourceKey(m, tea.KeyEsc)
	reopened, err := project.Open(m.project.Root)
	if err != nil || input.Resolve(reopened.Manifest.Bindings, "v", false) != input.Play {
		t.Fatal("bindings not saved", err)
	}
}
func TestViewportReticlePickingClearingAndResize(t *testing.T) {
	m := inputEditor(t)
	before := m.document.Document()
	m = resourceKey(m, tea.KeyTab)
	m = resourceKey(m, tea.KeyTab)
	if m.focus != 2 {
		t.Fatal("viewport focus missing")
	}
	h, w := len(m.frame), len(m.frame[0])
	x, y := m.reticleCell(w, h)
	m = resourceKey(m, tea.KeyRight)
	nextX, nextY := m.reticleCell(w, h)
	if nextX != x+1 || nextY != y {
		t.Fatal("reticle move", x, y, nextX, nextY)
	}
	// Select a cell that analytically hits a fixture object, then dispatch Pick.
	found := false
	for row := 0; row < h && !found; row++ {
		for col := 0; col < w; col++ {
			m.reticleU = (float64(col) + .5) / float64(w)
			m.reticleV = (float64(row) + .5) / float64(h)
			m, _ = editorRune(m, "f")
			if strings.HasPrefix(m.message, "Picked ") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("no fixture geometry pickable")
	}
	m, _ = editorRune(m, "0")
	if _, ok := m.selectedEntity(); ok {
		t.Fatal("clear selection ignored")
	}
	m, _ = editorRune(m, "[")
	e, ok := m.selectedEntity()
	if !ok || e.ID != "box" {
		t.Fatal("previous from cleared")
	}
	m, _ = editorRune(m, "0")
	m, _ = editorRune(m, "]")
	e, ok = m.selectedEntity()
	if !ok || e.ID != "floor" {
		t.Fatal("next from cleared")
	}
	if !reflect.DeepEqual(before, m.document.Document()) || m.preview.World.Stats().Tick != 0 {
		t.Fatal("selection mutated scene")
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 14})
	m = next.(projectEditorModel)
	if len(m.frame)+1 > max(1, m.height-11) {
		t.Fatal("viewport reticle can be clipped")
	}
	if !strings.Contains(m.View(), "Viewport") {
		t.Fatal("narrow viewport hidden", m.View())
	}
	m = resourceKey(m, tea.KeyTab)
	u, v := m.reticleU, m.reticleV
	m = resourceKey(m, tea.KeyUp)
	if m.reticleU != u || m.reticleV != v {
		t.Fatal("hierarchy navigation moved reticle")
	}
}

func TestBatchedActionsAndPasteIsolation(t *testing.T) {
	m := inputEditor(t)
	m, _ = editorRune(m, "0]]")
	e, ok := m.selectedEntity()
	if !ok || e.ID != "box" {
		t.Fatal("batched keys lost", e)
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p"), Paste: true})
	if m.play != nil {
		t.Fatal("paste triggered action")
	}
	m, _ = editorRune(m, "p")
	m, _ = editorRune(m, " ")
	m, _ = editorRune(m, "ii")
	if m.play.PendingCommands() != 2 {
		t.Fatal("batched impulses lost or duplicated")
	}
	m, _ = editorRune(m, "n")
	if m.play.PendingCommands() != 0 || m.play.World.Stats().Tick != 1 {
		t.Fatal("queued batch not consumed once")
	}
}
