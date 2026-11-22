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

func resourceKey(m projectEditorModel, key tea.KeyType) projectEditorModel {
	m, _ = editorKey(m, tea.KeyMsg{Type: key})
	return m
}
func TestResourceBrowserSourceEditingAndOverrides(t *testing.T) {
	p, err := project.Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	m := newProjectEditor(config.Default())
	if err = m.install(p); err != nil {
		t.Fatal(err)
	}
	m.selectID("box")
	m, _ = editorRune(m, "r")
	m.draft = "Assembly"
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "" || !strings.Contains(m.message, "Created resource") {
		t.Fatal(m.message)
	}
	for i := 0; i < 2; i++ {
		m, _ = editorRune(m, "a")
		m = resourceKey(m, tea.KeyDown)
		m = resourceKey(m, tea.KeyEnter)
	}
	if len(m.document.Document().Instances) != 2 {
		t.Fatal("instantiate", m.message)
	}
	second, _ := m.selectedEntity()
	m, _ = editorRune(m, "e")
	m.draft = "Custom"
	m = resourceKey(m, tea.KeyEnter)
	if !strings.Contains(m.View(), "Name *") {
		t.Fatal("missing override marker", m.View())
	}
	m = resourceKey(m, tea.KeyCtrlS)
	m, _ = editorRune(m, "a")
	m = resourceKey(m, tea.KeyDown)
	m, _ = editorRune(m, "e")
	if m.activeResource == nil {
		t.Fatal("edit source", m.message)
	}
	m, _ = editorRune(m, "e")
	m.draft = "SourceBox"
	m = resourceKey(m, tea.KeyEnter)
	// An attempted switch guards unsaved source edits. Cancelling retains them.
	before := m.document.Document()
	m, _ = editorRune(m, "a")
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "discard" {
		t.Fatal("source dirty guard")
	}
	m = resourceKey(m, tea.KeyEsc)
	if !reflect.DeepEqual(before, m.document.Document()) {
		t.Fatal("cancel lost source")
	}
	m = resourceKey(m, tea.KeyCtrlS)
	m, _ = editorRune(m, "a")
	m = resourceKey(m, tea.KeyEnter)
	if m.activeResource != nil || m.focus != 0 {
		t.Fatal("return to entry", m.message)
	}
	d := m.document.Document()
	first := d.Instances[0].Root
	m.selectID(first)
	e, _ := m.selectedEntity()
	if e.Name != "SourceBox" {
		t.Fatal("source changes not propagated")
	}
	m.selectID(second.ID)
	e, _ = m.selectedEntity()
	if e.Name != "Custom" {
		t.Fatal("override lost")
	}
	m = resourceKey(m, tea.KeyCtrlT)
	e, _ = m.selectedEntity()
	if e.Name != "SourceBox" {
		t.Fatal("reset override", m.message)
	}
	m = resourceKey(m, tea.KeyCtrlZ)
	e, _ = m.selectedEntity()
	if e.Name != "Custom" {
		t.Fatal("reset undo")
	}
	m = resourceKey(m, tea.KeyCtrlU)
	if len(m.document.Document().Instances) != 1 {
		t.Fatal("unpack", m.message)
	}
	before = m.document.Document()
	m, _ = editorRune(m, "p")
	m, _ = editorRune(m, "a")
	if m.modal != "" || m.play == nil {
		t.Fatal("browser leaked through Play")
	}
	m = resourceKey(m, tea.KeyEsc)
	if !reflect.DeepEqual(before, m.document.Document()) {
		t.Fatal("Play changed authored links")
	}
}
func TestResourceBrowserFocusAndFailedSwitch(t *testing.T) {
	p, err := project.Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	m := newProjectEditor(config.Default())
	if err = m.install(p); err != nil {
		t.Fatal(err)
	}
	m, _ = editorRune(m, "a")
	m, _ = editorRune(m, "p")
	if m.play != nil || m.modal != "assets" {
		t.Fatal("browser key leak")
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 14})
	m = next.(projectEditorModel)
	if !strings.Contains(m.View(), "Entry scene") {
		t.Fatal("narrow browser", m.View())
	}
	m = resourceKey(m, tea.KeyEsc)
	m, _ = editorRune(m, "b")
	before := m.document.Document()
	m.assetTarget = "resources/missing.yaml"
	m.begin("asset")
	if m.modal != "discard" {
		t.Fatal("dirty scene guard")
	}
	m, _ = editorRune(m, "d")
	if !reflect.DeepEqual(before, m.document.Document()) || m.activeResource != nil {
		t.Fatal("failed load lost edits")
	}
}
