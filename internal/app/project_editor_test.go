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

func TestSelectableEditorMenuAndWrap(t *testing.T) {
	m := newMenuModel(config.Default(), "unused", false)
	m, _ = pressMenu(m, tea.KeyDown)
	m, _ = pressMenu(m, tea.KeyDown)
	if !strings.Contains(m.View(), "> New scene / project") {
		t.Fatal(m.View())
	}
	opened, cmd := pressMenu(m, tea.KeyEnter)
	if cmd == nil || !opened.projectEditor || opened.start {
		t.Fatal("menu did not open editor")
	}
	m, _ = pressMenu(m, tea.KeyDown)
	if !strings.Contains(m.View(), "> Exit") {
		t.Fatal(m.View())
	}
	m, _ = pressMenu(m, tea.KeyDown)
	if m.selected != 0 {
		t.Fatal("down wrap")
	}
	m, _ = pressMenu(m, tea.KeyUp)
	if m.selected != 3 {
		t.Fatal("up wrap")
	}
}
func TestEditorHierarchyPickerAndSelection(t *testing.T) {
	p, err := project.Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	m := newProjectEditor(config.Default())
	if err = m.install(p); err != nil {
		t.Fatal(err)
	}
	m, _ = editorRune(m, "g")
	e, _ := m.selectedEntity()
	id := e.ID
	if err = m.document.Set(id, "body", "none"); err != nil {
		t.Fatal(err)
	}
	m.refresh()
	m, _ = editorRune(m, "h")
	if m.modal != "parent" || !strings.Contains(m.View(), "Scene root") {
		t.Fatal(m.View())
	}
	m, _ = editorRune(m, "p")
	if m.play != nil || m.modal != "parent" {
		t.Fatal("picker leaked play")
	}
	// Root, Floor, Box: choose Box.
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	e, _ = m.selectedEntity()
	if e.ID != id || e.Parent != "box" || m.modal != "" {
		t.Fatal("parent/selection", e, m.modal)
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyCtrlZ})
	e, _ = m.selectedEntity()
	if e.ID != id || e.Parent != "" {
		t.Fatal("undo selection")
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyCtrlY})
	// Duplicating a selected subtree selects the copied root, not its last child.
	m.selectID("box")
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyCtrlD})
	e, _ = m.selectedEntity()
	if e.ID == "box" || e.Name != "Box copy" || len(m.document.Document().SubtreeIDs(e.ID)) != 2 {
		t.Fatal("duplicate selection", e)
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyDelete})
	if len(m.document.Document().Entities) != 3 {
		t.Fatal("delete did not remove subtree")
	}
	m.selectID(id)
	m, _ = editorRune(m, "h")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 14})
	m = next.(projectEditorModel)
	if !strings.Contains(m.View(), "Enter apply") {
		t.Fatal("small picker", m.View())
	}
	m, _ = editorKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = editorRune(m, "p")
	m, _ = editorRune(m, "h")
	if m.modal != "" {
		t.Fatal("parenting allowed in play")
	}
}
