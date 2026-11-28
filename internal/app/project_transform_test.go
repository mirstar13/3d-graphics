package app

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestViewportTransformUndoAndPausedIsolation(t *testing.T) {
	m := inputEditor(t)
	if _, ok := m.selectedEntity(); !ok {
		t.Fatal("expected a selected entity")
	}
	before := m.document.Document()
	m.transformTool, m.transformSnap = 1, true
	d := m.beginTransform(0, 0)
	if d == nil {
		t.Fatal("expected drag")
	}
	d.amount = .53
	m.commitTransform(*d)
	after := m.document.Document()
	if reflect.DeepEqual(after, before) || !m.document.Undo() || !reflect.DeepEqual(m.document.Document(), before) {
		t.Fatal("expected one undoable pose change")
	}
	if !m.document.Redo() {
		t.Fatal("expected redo")
	}
	m.refresh()
	m.startPlay()
	m.play.SetPaused(true)
	d = m.beginTransform(0, 0)
	if d == nil {
		t.Fatal("expected paused drag")
	}
	d.amount = .75
	want, _ := d.pose()
	m.commitTransform(*d)
	runtime, err := m.play.LocalPose(d.handle)
	if err != nil || runtime.Position != want {
		t.Fatalf("runtime pose %v, %v", runtime.Position, err)
	}
	tick := m.play.World.Stats().Tick
	if err := m.play.Step(); err != nil || m.play.World.Stats().Tick != tick+1 {
		t.Fatal("expected one tick", err)
	}
	m.transformDrag = m.beginTransform(0, 0)
	next, _ := m.Update(tea.BlurMsg{})
	m = next.(projectEditorModel)
	if m.transformDrag != nil {
		t.Fatal("blur must cancel preview")
	}
	m.stopPlay()
	if !reflect.DeepEqual(m.document.Document(), after) || !m.document.Dirty() {
		t.Fatal("Play changed authored edits")
	}
	if !m.document.Undo() || !reflect.DeepEqual(m.document.Document(), before) {
		t.Fatal("Play changed undo history")
	}
}
