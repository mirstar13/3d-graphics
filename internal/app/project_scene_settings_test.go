package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"slices"
	"testing"
)

func TestSceneSettingsWorkflow(t *testing.T) {
	m := inputEditor(t)
	m = resourceKey(m, tea.KeyCtrlE)
	m = resourceKey(m, tea.KeyEnter)
	m.draft = `{"Near":400}`
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "scene_edit" || m.document.Dirty() {
		t.Fatal("accepted invalid clipping")
	}
	m.draft = `{"Position":{"X":11,"Y":8,"Z":-12}}`
	m = resourceKey(m, tea.KeyEnter)
	if m.modal != "scene" {
		t.Fatal(m.message)
	}
	camera := *m.document.Document().Camera
	m, _ = editorRune(m, "l")
	if len(m.document.Document().Lighting.Lights) != 2 {
		t.Fatal("add light")
	}
	m, _ = editorRune(m, "d")
	if !m.document.Undo() || len(m.document.Document().Lighting.Lights) != 2 || !m.document.Redo() {
		t.Fatal("light history")
	}
	m.save()
	p, err := project.Open(m.project.Root)
	if err != nil || *p.Scene.Camera != camera || len(p.Scene.Lighting.Lights) != 1 {
		t.Fatal("reopen", err)
	}
	m = resourceKey(m, tea.KeyEsc)
	m, _ = editorRune(m, "1")
	m, _ = editorRune(m, "+")
	m.startPlay()
	if m.play.Scene.Camera.GetPosition() != camera.Position {
		t.Fatal("Play camera")
	}
	m.view = 2
	m.zoom = 2
	m.resetPlay()
	if m.play.Scene.Camera.GetPosition() != camera.Position {
		t.Fatal("reset camera")
	}
	m.play = nil
	m.activeResource = &project.Resource{}
	m = resourceKey(m, tea.KeyCtrlE)
	if m.modal != "" {
		t.Fatal("resource opened settings")
	}
}
func TestLightingChangesMonochromeFrame(t *testing.T) {
	m := inputEditor(t)
	s := project.SceneSettings{Lighting: &project.LightingSettings{}}
	if err := m.document.SetSceneSettings(s); err != nil {
		t.Fatal(err)
	}
	m.refresh()
	dark := slices.Clone(m.frame)
	rig := project.DefaultLighting()
	rig.Intensity = 100
	s.Lighting = &rig
	if err := m.document.SetSceneSettings(s); err != nil {
		t.Fatal(err)
	}
	m.refresh()
	if slices.Equal(dark, m.frame) {
		t.Fatal("lighting should change shaded glyphs")
	}
}
