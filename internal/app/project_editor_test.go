package app

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func editorKey(m projectEditorModel, key tea.KeyMsg) (projectEditorModel, tea.Cmd) {
	next, cmd := m.Update(key)
	return next.(projectEditorModel), cmd
}
func editorRune(m projectEditorModel, text string) (projectEditorModel, tea.Cmd) {
	return editorKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
}
func TestEditorCreateEditSavePlayStopReopen(t *testing.T) {
	m := newProjectEditor(config.Default())
	m, _ = editorRune(m, "n")
	root := filepath.Join(t.TempDir(), "Game")
	m, _ = editorRune(m, root)
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.project == nil || m.modal != "" {
		t.Fatal("create project", m.message)
	}
	m, _ = editorRune(m, "g")
	e, _ := m.selectedEntity()
	if e.Shape != "sphere" || !m.document.Dirty() {
		t.Fatal("spawn authoring command")
	}
	m.property = 4
	m, _ = editorRune(m, "e")
	m.draft = "7"
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	e, _ = m.selectedEntity()
	if e.Position.Y != 7 {
		t.Fatal("property edit")
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.document.Dirty() {
		t.Fatal("save", m.message)
	}
	authored := m.document.Document()
	var cmd tea.Cmd
	m, cmd = editorRune(m, "p")
	if m.play == nil || cmd == nil {
		t.Fatal("play not started")
	}
	for i := 0; i < 60; i++ {
		m.play.Advance(config.Default().Physics.WorldOptions().FixedStep)
	}
	m, _ = editorRune(m, "b")
	if !reflect.DeepEqual(authored, m.document.Document()) {
		t.Fatal("editing leaked during play")
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.play != nil || !reflect.DeepEqual(authored, m.document.Document()) {
		t.Fatal("stop modified document")
	}
	p, err := project.Open(root)
	if err != nil || !reflect.DeepEqual(p.Scene, authored) {
		t.Fatal("reopen", err)
	}
}
func TestEditorFocusUnsavedAndFailedOpen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Game")
	p, _ := project.Create(root)
	m := newProjectEditor(config.Default())
	if err := m.install(p); err != nil {
		t.Fatal(err)
	}
	m.property = 0
	m, _ = editorRune(m, "e")
	m, _ = editorRune(m, "p")
	if m.play != nil || m.modal != "property" {
		t.Fatal("text input triggered play")
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = editorRune(m, "b")
	before := m.document.Document()
	m, cmd := editorKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || m.modal != "discard" {
		t.Fatal("unsaved exit not guarded")
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyCtrlO})
	m, _ = editorRune(m, "d")
	m.draft = filepath.Join(root, "missing")
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !reflect.DeepEqual(before, m.document.Document()) || m.project.Root != root {
		t.Fatal("failed load replaced project")
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 14})
	m = next.(projectEditorModel)
	m.focus = 1
	m.property = 12
	if !strings.Contains(m.View(), "Mass") {
		t.Fatal("focused property hidden in narrow layout", m.View())
	}
}
func TestLauncherOpensProjectEditor(t *testing.T) {
	m := newMenuModel(config.Default(), "unused", false)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil || !next.(menuModel).projectEditor {
		t.Fatal("editor shortcut missing")
	}
}

func TestEditorUnicodeLayout(t *testing.T) {
	view := fitView("世界世界世界\nGame", 7, 2)
	if strings.Split(view, "\n")[0] != "世界世" {
		t.Fatal("wide characters overflow viewport", view)
	}
}
